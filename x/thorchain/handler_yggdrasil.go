package thorchain

import (
	"errors"
	"fmt"

	"github.com/blang/semver"
	"github.com/hashicorp/go-multierror"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
	"gitlab.com/thorchain/thornode/x/thorchain/keeper"
	kvTypes "gitlab.com/thorchain/thornode/x/thorchain/keeper/types"
)

// YggdrasilHandler is to process yggdrasil messages
// When thorchain fund yggdrasil pool , observer should observe two transactions
// 1. outbound tx from asgard vault
// 2. inbound tx to yggdrasil vault
// when yggdrasil pool return fund , observer should observe two transactions as well
// 1. outbound tx from yggdrasil vault
// 2. inbound tx to asgard vault
type YggdrasilHandler struct {
	keeper keeper.Keeper
	mgr    Manager
}

// NewYggdrasilHandler create a new Yggdrasil handler
func NewYggdrasilHandler(keeper keeper.Keeper, mgr Manager) YggdrasilHandler {
	return YggdrasilHandler{
		keeper: keeper,
		mgr:    mgr,
	}
}

// Run execute the logic in Yggdrasil Handler
func (h YggdrasilHandler) Run(ctx cosmos.Context, m cosmos.Msg, version semver.Version, constAccessor constants.ConstantValues) (*cosmos.Result, error) {
	msg, ok := m.(MsgYggdrasil)
	if !ok {
		return nil, errInvalidMessage
	}
	if err := h.validate(ctx, msg, version); err != nil {
		ctx.Logger().Error("MsgYggdrasil failed validation", "error", err)
		return nil, err
	}
	result, err := h.handle(ctx, msg, version, constAccessor)
	if err != nil {
		ctx.Logger().Error("failed to process MsgYggdrasil", "error", err)
		return nil, err
	}
	return result, nil
}

func (h YggdrasilHandler) validate(ctx cosmos.Context, msg MsgYggdrasil, version semver.Version) error {
	if version.GTE(semver.MustParse("0.1.0")) {
		return h.validateV1(ctx, msg)
	}
	return errBadVersion
}

func (h YggdrasilHandler) validateV1(ctx cosmos.Context, msg MsgYggdrasil) error {
	return msg.ValidateBasic()
}

func (h YggdrasilHandler) handle(ctx cosmos.Context, msg MsgYggdrasil, version semver.Version, constAccessor constants.ConstantValues) (*cosmos.Result, error) {
	ctx.Logger().Info("receive MsgYggdrasil", "pubkey", msg.PubKey.String(), "add_funds", msg.AddFunds, "coins", msg.Coins)
	if version.GTE(semver.MustParse("0.17.0")) {
		return h.handleV3(ctx, msg, version)
	} else if version.GTE(semver.MustParse("0.8.0")) {
		return h.handleV2(ctx, msg, version)
	} else if version.GTE(semver.MustParse("0.1.0")) {
		return h.handleV1(ctx, msg, version)
	}
	return nil, errBadVersion
}

func (h YggdrasilHandler) slash(ctx cosmos.Context, version semver.Version, pk common.PubKey, coins common.Coins) error {
	var returnErr error
	for _, c := range coins {
		if err := h.mgr.Slasher().SlashNodeAccount(ctx, pk, c.Asset, c.Amount, h.mgr); err != nil {
			ctx.Logger().Error("fail to slash account", "error", err)
			if returnErr == nil {
				returnErr = err
			} else {
				returnErr = multierror.Append(returnErr, err)
			}
		}
	}
	return returnErr
}

func (h YggdrasilHandler) handleV1(ctx cosmos.Context, msg MsgYggdrasil, version semver.Version) (*cosmos.Result, error) {
	// update txOut record with our TxID that sent funds out of the pool
	txOut, err := h.keeper.GetTxOut(ctx, msg.BlockHeight)
	if err != nil {
		return nil, ErrInternal(err, "unable to get txOut record")
	}

	shouldSlash := true
	for i, tx := range txOut.TxArray {
		// yggdrasil is the memo used by thorchain to identify fund migration
		// to a yggdrasil vault.
		// it use yggdrasil+/-:{block height} to mark a tx out caused by vault
		// rotation
		// this type of tx out is special , because it doesn't have relevant tx
		// in to trigger it, it is trigger by thorchain itself.
		fromAddress, _ := tx.VaultPubKey.GetAddress(tx.Chain)
		if tx.InHash.Equals(common.BlankTxID) &&
			tx.OutHash.IsEmpty() &&
			tx.ToAddress.Equals(msg.Tx.ToAddress) &&
			fromAddress.Equals(msg.Tx.FromAddress) {

			// only need to check the coin if yggdrasil+
			if msg.AddFunds && !msg.Tx.Coins.Contains(tx.Coin) {
				continue
			}

			txOut.TxArray[i].OutHash = msg.Tx.ID
			shouldSlash = false

			if err := h.keeper.SetTxOut(ctx, txOut); nil != err {
				ctx.Logger().Error("fail to save tx out", "error", err)
			}

			break
		}
	}

	if shouldSlash {
		if err := h.slash(ctx, version, msg.PubKey, msg.Tx.Coins); err != nil {
			return nil, ErrInternal(err, "fail to slash account")
		}
	}

	vault, err := h.keeper.GetVault(ctx, msg.PubKey)
	if err != nil && !errors.Is(err, kvTypes.ErrVaultNotFound) {
		return nil, fmt.Errorf("fail to get yggdrasil: %w", err)
	}
	if len(vault.Type) == 0 {
		vault.Status = ActiveVault
		vault.Type = YggdrasilVault
	}

	if err := h.keeper.SetLastSignedHeight(ctx, msg.BlockHeight); err != nil {
		ctx.Logger().Error("fail to update last signed height", "error", err)
	}

	if msg.AddFunds {
		return h.handleYggdrasilFund(ctx, msg, vault)
	}
	return h.handleYggdrasilReturn(ctx, msg, vault, version)
}

func (h YggdrasilHandler) handleV2(ctx cosmos.Context, msg MsgYggdrasil, version semver.Version) (*cosmos.Result, error) {
	// update txOut record with our TxID that sent funds out of the pool
	txOut, err := h.keeper.GetTxOut(ctx, msg.BlockHeight)
	if err != nil {
		return nil, ErrInternal(err, "unable to get txOut record")
	}

	shouldSlash := true
	for i, tx := range txOut.TxArray {
		// yggdrasil is the memo used by thorchain to identify fund migration
		// to a yggdrasil vault.
		// it use yggdrasil+/-:{block height} to mark a tx out caused by vault
		// rotation
		// this type of tx out is special , because it doesn't have relevant tx
		// in to trigger it, it is trigger by thorchain itself.
		fromAddress, _ := tx.VaultPubKey.GetAddress(tx.Chain)
		if tx.InHash.Equals(common.BlankTxID) &&
			tx.OutHash.IsEmpty() &&
			tx.ToAddress.Equals(msg.Tx.ToAddress) &&
			fromAddress.Equals(msg.Tx.FromAddress) {

			// only need to check the coin if yggdrasil+
			if msg.AddFunds && !msg.Tx.Coins.Contains(tx.Coin) {
				continue
			}

			txOut.TxArray[i].OutHash = msg.Tx.ID
			shouldSlash = false

			if err := h.keeper.SetTxOut(ctx, txOut); nil != err {
				ctx.Logger().Error("fail to save tx out", "error", err)
			}

			break
		}
	}

	if shouldSlash {
		if err := h.slash(ctx, version, msg.PubKey, msg.Tx.Coins); err != nil {
			return nil, ErrInternal(err, "fail to slash account")
		}
	}

	vault, err := h.keeper.GetVault(ctx, msg.PubKey)
	if err != nil && !errors.Is(err, kvTypes.ErrVaultNotFound) {
		return nil, fmt.Errorf("fail to get yggdrasil: %w", err)
	}
	if len(vault.Type) == 0 {
		vault.Status = ActiveVault
		vault.Type = YggdrasilVault
	}

	if err := h.keeper.SetLastSignedHeight(ctx, msg.BlockHeight); err != nil {
		ctx.Logger().Error("fail to update last signed height", "error", err)
	}

	if msg.AddFunds {
		return h.handleYggdrasilFund(ctx, msg, vault)
	}
	return h.handleYggdrasilReturnV2(ctx, msg, vault, version)
}

func (h YggdrasilHandler) handleYggdrasilFund(ctx cosmos.Context, msg MsgYggdrasil, vault Vault) (*cosmos.Result, error) {
	switch vault.Type {
	case AsgardVault:
		ctx.EventManager().EmitEvent(
			cosmos.NewEvent("asgard_fund_yggdrasil",
				cosmos.NewAttribute("pubkey", vault.PubKey.String()),
				cosmos.NewAttribute("coins", msg.Coins.String()),
				cosmos.NewAttribute("tx", msg.Tx.ID.String())))
	case YggdrasilVault:
		ctx.EventManager().EmitEvent(
			cosmos.NewEvent("yggdrasil_receive_fund",
				cosmos.NewAttribute("pubkey", vault.PubKey.String()),
				cosmos.NewAttribute("coins", msg.Coins.String()),
				cosmos.NewAttribute("tx", msg.Tx.ID.String())))
	}
	// Yggdrasil usually comes from Asgard , Asgard --> Yggdrasil
	// It will be an outbound tx from Asgard pool , and it will be an Inbound tx form Yggdrasil pool
	// incoming fund will be added to Vault as part of ObservedTxInHandler
	// Yggdrasil handler doesn't need to do anything
	return &cosmos.Result{}, nil
}

func (h YggdrasilHandler) handleYggdrasilReturn(ctx cosmos.Context, msg MsgYggdrasil, vault Vault, version semver.Version) (*cosmos.Result, error) {
	// observe an outbound tx from yggdrasil vault
	switch vault.Type {
	case YggdrasilVault:
		asgardVaults, err := h.keeper.GetAsgardVaultsByStatus(ctx, ActiveVault)
		if err != nil {
			return nil, ErrInternal(err, "unable to get asgard vaults")
		}
		isAsgardReceipient, err := asgardVaults.HasAddress(msg.Tx.Chain, msg.Tx.ToAddress)
		if err != nil {
			return nil, ErrInternal(err, fmt.Sprintf("unable to determinate whether %s is an Asgard vault", msg.Tx.ToAddress))
		}

		if !isAsgardReceipient {
			// not sending to asgard , slash the node account
			if err := h.slash(ctx, version, msg.PubKey, msg.Tx.Coins); err != nil {
				return nil, ErrInternal(err, "fail to slash account for sending fund to a none asgard vault using yggdrasil-")
			}
		}

		return &cosmos.Result{}, nil

	case AsgardVault:
		// when vault.Type is asgard, that means this tx is observed on an asgard pool and it is an inbound tx
		// Yggdrasil return fund back to Asgard
		ctx.EventManager().EmitEvent(
			cosmos.NewEvent("yggdrasil_return",
				cosmos.NewAttribute("pubkey", vault.PubKey.String()),
				cosmos.NewAttribute("coins", msg.Coins.String()),
				cosmos.NewAttribute("tx", msg.Tx.ID.String())))
	}
	return &cosmos.Result{}, nil
}

func (h YggdrasilHandler) handleYggdrasilReturnV2(ctx cosmos.Context, msg MsgYggdrasil, vault Vault, version semver.Version) (*cosmos.Result, error) {
	// observe an outbound tx from yggdrasil vault
	switch vault.Type {
	case YggdrasilVault:
		asgardVaults, err := h.keeper.GetAsgardVaults(ctx)
		if err != nil {
			return nil, ErrInternal(err, "unable to get asgard vaults")
		}
		vaults := Vaults{}
		for _, v := range asgardVaults {
			// make sure vaults have both active asgard vault , and also retiring asgard vault
			if v.Status == ActiveVault || v.Status == RetiringVault {
				vaults = append(vaults, v)
			}
		}

		isAsgardReceipient, err := vaults.HasAddress(msg.Tx.Chain, msg.Tx.ToAddress)
		if err != nil {
			return nil, ErrInternal(err, fmt.Sprintf("unable to determinate whether %s is an Asgard vault", msg.Tx.ToAddress))
		}

		if !isAsgardReceipient {
			ctx.Logger().Info("yggdrasil send fund to none asgard address")
			// not sending to asgard , slash the node account
			if err := h.slash(ctx, version, msg.PubKey, msg.Tx.Coins); err != nil {
				return nil, ErrInternal(err, "fail to slash account for sending fund to a none asgard vault using yggdrasil-")
			}
		}

		return &cosmos.Result{}, nil

	case AsgardVault:
		// when vault.Type is asgard, that means this tx is observed on an asgard pool and it is an inbound tx
		// Yggdrasil return fund back to Asgard
		ctx.EventManager().EmitEvent(
			cosmos.NewEvent("yggdrasil_return",
				cosmos.NewAttribute("pubkey", vault.PubKey.String()),
				cosmos.NewAttribute("coins", msg.Coins.String()),
				cosmos.NewAttribute("tx", msg.Tx.ID.String())))
	}
	return &cosmos.Result{}, nil
}

func (h YggdrasilHandler) handleV3(ctx cosmos.Context, msg MsgYggdrasil, version semver.Version) (*cosmos.Result, error) {
	// update txOut record with our TxID that sent funds out of the pool
	txOut, err := h.keeper.GetTxOut(ctx, msg.BlockHeight)
	if err != nil {
		return nil, ErrInternal(err, "unable to get txOut record")
	}

	shouldSlash := true
	// if ygg is returning funds, don't slash if they are sending to asgard vault
	if !msg.AddFunds {
		// check if the node account is active, it shouldn't be
		na, err := h.keeper.GetNodeAccountByPubKey(ctx, msg.PubKey)
		if err != nil {
			ctx.Logger().Error("fail to get node account", "error", err)
		}
		if na.Status != NodeActive {
			active, err := h.keeper.GetAsgardVaultsByStatus(ctx, ActiveVault)
			if err != nil {
				ctx.Logger().Error("fail to get vaults", "error", err)
			}
			retiring, err := h.keeper.GetAsgardVaultsByStatus(ctx, RetiringVault)
			if err != nil {
				ctx.Logger().Error("fail to get vaults", "error", err)
			}
			for _, v := range append(active, retiring...) {
				addr, err := v.PubKey.GetAddress(msg.Tx.Chain)
				if err != nil {
					ctx.Logger().Error("fail to get address from pubkey", "error", err)
				}
				if !addr.IsEmpty() && addr.Equals(msg.Tx.ToAddress) {
					ctx.Logger().Info("yggdrasil vault return fund to asgard , should not be slashed")
					shouldSlash = false
					break
				}
			}
		}
	}

	for i, tx := range txOut.TxArray {
		// yggdrasil is the memo used by thorchain to identify fund migration
		// to a yggdrasil vault.
		// it use yggdrasil+/-:{block height} to mark a tx out caused by vault
		// rotation
		// this type of tx out is special , because it doesn't have relevant tx
		// in to trigger it, it is trigger by thorchain itself.
		fromAddress, _ := tx.VaultPubKey.GetAddress(tx.Chain)
		if tx.InHash.Equals(common.BlankTxID) &&
			tx.OutHash.IsEmpty() &&
			tx.ToAddress.Equals(msg.Tx.ToAddress) &&
			fromAddress.Equals(msg.Tx.FromAddress) {

			// only need to check the coin if yggdrasil+
			if msg.AddFunds && !msg.Tx.Coins.Contains(tx.Coin) {
				continue
			}

			txOut.TxArray[i].OutHash = msg.Tx.ID
			shouldSlash = false

			if err := h.keeper.SetTxOut(ctx, txOut); nil != err {
				ctx.Logger().Error("fail to save tx out", "error", err)
			}

			break
		}
	}

	if shouldSlash {
		if err := h.slash(ctx, version, msg.PubKey, msg.Tx.Coins); err != nil {
			return nil, ErrInternal(err, "fail to slash account")
		}
	}

	vault, err := h.keeper.GetVault(ctx, msg.PubKey)
	if err != nil && !errors.Is(err, kvTypes.ErrVaultNotFound) {
		return nil, fmt.Errorf("fail to get yggdrasil: %w", err)
	}
	if len(vault.Type) == 0 {
		vault.Status = ActiveVault
		vault.Type = YggdrasilVault
	}

	if err := h.keeper.SetLastSignedHeight(ctx, msg.BlockHeight); err != nil {
		ctx.Logger().Error("fail to update last signed height", "error", err)
	}

	if msg.AddFunds {
		return h.handleYggdrasilFund(ctx, msg, vault)
	}
	return h.handleYggdrasilReturnV2(ctx, msg, vault, version)
}
