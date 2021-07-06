package thorchain

import (
	"errors"
	"fmt"

	"github.com/blang/semver"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
	"gitlab.com/thorchain/thornode/x/thorchain/keeper"
)

// SwitchHandler is to handle Switch message
// MsgSwitch is used to switch from bep2 RUNE to native RUNE
type SwitchHandler struct {
	keeper keeper.Keeper
	mgr    Manager
}

// NewSwitchHandler create new instance of SwitchHandler
func NewSwitchHandler(keeper keeper.Keeper, mgr Manager) SwitchHandler {
	return SwitchHandler{
		keeper: keeper,
		mgr:    mgr,
	}
}

// Run it the main entry point to execute Switch logic
func (h SwitchHandler) Run(ctx cosmos.Context, m cosmos.Msg, version semver.Version, constAccessor constants.ConstantValues) (*cosmos.Result, error) {
	msg, ok := m.(MsgSwitch)
	if !ok {
		return nil, errInvalidMessage
	}
	if err := h.validate(ctx, msg, version); err != nil {
		ctx.Logger().Error("msg switch failed validation", "error", err)
		return nil, err
	}
	result, err := h.handle(ctx, msg, version)
	if err != nil {
		ctx.Logger().Error("failed to process msg switch", "error", err)
		return nil, err
	}
	return result, err
}

func (h SwitchHandler) validate(ctx cosmos.Context, msg MsgSwitch, version semver.Version) error {
	if version.GTE(semver.MustParse("0.1.0")) {
		return h.validateV1(ctx, msg)
	}
	return errBadVersion
}

func (h SwitchHandler) validateV1(ctx cosmos.Context, msg MsgSwitch) error {
	if err := msg.ValidateBasic(); err != nil {
		return err
	}

	// when using BEP2 rune, cannot swap rune
	if !common.RuneAsset().Chain.Equals(common.THORChain) {
		return errors.New("when using BEP2 rune, cannot swap rune assets")
	}

	// if we are getting a non-native asset, ensure its signed by an active
	// node account
	if !msg.Tx.Coins[0].IsNative() {
		if !isSignedByActiveNodeAccounts(ctx, h.keeper, msg.GetSigners()) {
			return cosmos.ErrUnauthorized(notAuthorized.Error())
		}
	}

	return nil
}

func (h SwitchHandler) handle(ctx cosmos.Context, msg MsgSwitch, version semver.Version) (*cosmos.Result, error) {
	ctx.Logger().Info("handleMsgSwitch request", "destination address", msg.Destination.String())
	if version.GTE(semver.MustParse("0.1.0")) {
		return h.handleV1(ctx, msg, version)
	}
	return nil, errBadVersion
}

func (h SwitchHandler) handleV1(ctx cosmos.Context, msg MsgSwitch, version semver.Version) (*cosmos.Result, error) {
	haltHeight, err := h.keeper.GetMimir(ctx, "HaltTHORChain")
	if err != nil {
		return nil, fmt.Errorf("failed to get mimir setting: %w", err)
	}
	if haltHeight > 0 && common.BlockHeight(ctx) > haltHeight {
		return nil, fmt.Errorf("mimir has halted THORChain transactions")
	}

	if msg.Tx.Coins[0].IsNative() {
		return h.toBEP2(ctx, msg)
	}

	return h.toNative(ctx, msg)
}

func (h SwitchHandler) toBEP2(ctx cosmos.Context, msg MsgSwitch) (*cosmos.Result, error) {
	bank := h.keeper.CoinKeeper()

	vaultData, err := h.keeper.GetVaultData(ctx)
	if err != nil {
		return nil, ErrInternal(err, "fail to get vault data")
	}
	coin, err := common.NewCoin(common.RuneNative, msg.Tx.Coins[0].Amount).Native()
	if err != nil {
		return nil, ErrInternal(err, "fail to get native coin")
	}

	// ensure we have enough BEP2 rune assets to fulfill the request
	if vaultData.TotalBEP2Rune.LT(msg.Tx.Coins[0].Amount) {
		return nil, ErrInternal(err, "not enough funds in the vault")
	}

	addr, err := cosmos.AccAddressFromBech32(msg.Tx.FromAddress.String())
	if err != nil {
		return nil, ErrInternal(err, "fail to parse thor address")
	}

	if !bank.HasCoins(ctx, addr, cosmos.NewCoins(coin)) {
		return nil, ErrInternal(err, "insufficient funds")
	}
	if _, err := bank.SubtractCoins(ctx, addr, cosmos.NewCoins(coin)); err != nil {
		return nil, ErrInternal(err, "fail to burn native rune coins")
	}

	vaultData.TotalBEP2Rune = common.SafeSub(vaultData.TotalBEP2Rune, msg.Tx.Coins[0].Amount)

	toi := &TxOutItem{
		Chain:     common.BNBChain, // BEP2 RUNE is always on binance chain
		InHash:    msg.Tx.ID,
		ToAddress: msg.Destination,
		Coin:      common.NewCoin(common.BEP2RuneAsset(), msg.Tx.Coins[0].Amount),
	}
	ok, err := h.mgr.TxOutStore().TryAddTxOutItem(ctx, h.mgr, toi)
	if err != nil {
		if !errors.Is(err, ErrNotEnoughToPayFee) {
			return nil, ErrInternal(err, "fail to add outbound tx")
		}
		ok = true
	}
	if !ok {
		return nil, errFailAddOutboundTx
	}

	if err := h.keeper.SetVaultData(ctx, vaultData); err != nil {
		return nil, ErrInternal(err, "fail to set vault data")
	}
	return &cosmos.Result{}, nil
}

func (h SwitchHandler) toNative(ctx cosmos.Context, msg MsgSwitch) (*cosmos.Result, error) {
	bank := h.keeper.CoinKeeper()

	vaultData, err := h.keeper.GetVaultData(ctx)
	if err != nil {
		return nil, ErrInternal(err, "fail to get vault data")
	}

	coin, err := common.NewCoin(common.RuneNative, msg.Tx.Coins[0].Amount).Native()
	if err != nil {
		return nil, ErrInternal(err, "fail to get native coin")
	}

	addr, err := cosmos.AccAddressFromBech32(msg.Destination.String())
	if err != nil {
		return nil, ErrInternal(err, "fail to parse thor address")
	}
	if _, err := bank.AddCoins(ctx, addr, cosmos.NewCoins(coin)); err != nil {
		return nil, ErrInternal(err, "fail to mint native rune coins")
	}
	vaultData.TotalBEP2Rune = vaultData.TotalBEP2Rune.Add(msg.Tx.Coins[0].Amount)

	if err := h.keeper.SetVaultData(ctx, vaultData); err != nil {
		return nil, ErrInternal(err, "fail to set vault data")
	}
	return &cosmos.Result{}, nil
}
