package thorchain

import (
	"fmt"

	"github.com/blang/semver"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
	"gitlab.com/thorchain/thornode/x/thorchain/keeper"
)

// ObservedTxOutHandler process MsgObservedTxOut messages
type ObservedTxOutHandler struct {
	keeper keeper.Keeper
	mgr    Manager
}

// NewObservedTxOutHandler create a new instance of ObservedTxOutHandler
func NewObservedTxOutHandler(keeper keeper.Keeper, mgr Manager) ObservedTxOutHandler {
	return ObservedTxOutHandler{
		keeper: keeper,
		mgr:    mgr,
	}
}

// Run is the main entry point for ObservedTxOutHandler
func (h ObservedTxOutHandler) Run(ctx cosmos.Context, m cosmos.Msg, version semver.Version, constAccessor constants.ConstantValues) (*cosmos.Result, error) {
	msg, ok := m.(MsgObservedTxOut)
	if !ok {
		return nil, errInvalidMessage
	}
	if err := h.validate(ctx, msg, version); err != nil {
		ctx.Logger().Error("MsgObserveTxOut failed validation", "error", err)
		return nil, err
	}
	result, err := h.handle(ctx, msg, version, constAccessor)
	if err != nil {
		ctx.Logger().Error("fail to handle MsgObserveTxOut", "error", err)
	}
	return result, err
}

func (h ObservedTxOutHandler) validate(ctx cosmos.Context, msg MsgObservedTxOut, version semver.Version) error {
	if version.GTE(semver.MustParse("0.1.0")) {
		return h.validateV1(ctx, msg)
	}
	return errInvalidVersion
}

func (h ObservedTxOutHandler) validateV1(ctx cosmos.Context, msg MsgObservedTxOut) error {
	if err := msg.ValidateBasic(); err != nil {
		return err
	}

	if !isSignedByActiveNodeAccounts(ctx, h.keeper, msg.GetSigners()) {
		return cosmos.ErrUnauthorized(fmt.Sprintf("%+v are not authorized", msg.GetSigners()))
	}

	return nil
}

func (h ObservedTxOutHandler) handle(ctx cosmos.Context, msg MsgObservedTxOut, version semver.Version, constAccessor constants.ConstantValues) (*cosmos.Result, error) {
	if version.GTE(semver.MustParse("0.13.0")) {
		return h.handleV13(ctx, version, msg, constAccessor)
	} else if version.GTE(semver.MustParse("0.1.0")) {
		return h.handleV1(ctx, version, msg, constAccessor)
	}

	return nil, errBadVersion
}

func (h ObservedTxOutHandler) preflight(ctx cosmos.Context, voter ObservedTxVoter, nas NodeAccounts, tx ObservedTx, signer cosmos.AccAddress, version semver.Version, constAccessor constants.ConstantValues) (ObservedTxVoter, bool) {
	observeSlashPoints := constAccessor.GetInt64Value(constants.ObserveSlashPoints)
	observeFlex := constAccessor.GetInt64Value(constants.ObserveFlex)
	ok := false
	h.mgr.Slasher().IncSlashPoints(ctx, observeSlashPoints, signer)
	if !voter.Add(tx, signer) {
		// when the signer already sign it
		return voter, ok
	}
	if voter.HasConsensus(nas) {
		if voter.Height == 0 {
			ok = true
			voter.Height = common.BlockHeight(ctx)
			voter.Tx = voter.GetTx(nas)
			// tx has consensus now, so decrease the slashing point for all the signers whom voted for it
			h.mgr.Slasher().DecSlashPoints(ctx, observeSlashPoints, voter.Tx.Signers...)

		} else {
			// event the tx had been processed , given the signer just a bit late , so we still take away their slash points
			if common.BlockHeight(ctx) <= (voter.Height+observeFlex) && voter.Tx.Equals(tx) {
				h.mgr.Slasher().DecSlashPoints(ctx, observeSlashPoints, signer)
			}
		}
	}
	h.keeper.SetObservedTxOutVoter(ctx, voter)

	// Check to see if we have enough identical observations to process the transaction
	return voter, ok
}

// Handle a message to observe outbound tx
func (h ObservedTxOutHandler) handleV1(ctx cosmos.Context, version semver.Version, msg MsgObservedTxOut, constAccessor constants.ConstantValues) (*cosmos.Result, error) {
	activeNodeAccounts, err := h.keeper.ListActiveNodeAccounts(ctx)
	if err != nil {
		return nil, wrapError(ctx, err, "fail to get list of active node accounts")
	}

	handler := NewInternalHandler(h.keeper, h.mgr)

	for _, tx := range msg.Txs {
		// check we are sending from a valid vault
		if !h.keeper.VaultExists(ctx, tx.ObservedPubKey) {
			ctx.Logger().Info("Not valid Observed Pubkey", tx.ObservedPubKey)
			continue
		}

		voter, err := h.keeper.GetObservedTxOutVoter(ctx, tx.Tx.ID)
		if err != nil {
			ctx.Logger().Error("fail to get tx out voter", "error", err)
			continue
		}

		// check whether the tx has consensus
		voter, ok := h.preflight(ctx, voter, activeNodeAccounts, tx, msg.Signer, version, constAccessor)
		if !ok {
			if voter.Height == common.BlockHeight(ctx) {
				// we've already process the transaction, but we should still
				// update the observing addresses
				h.mgr.ObMgr().AppendObserver(tx.Tx.Chain, msg.GetSigners())
			}
			continue
		}
		tx.Tx.Memo = fetchMemo(ctx, constAccessor, h.keeper, tx.Tx)
		if len(tx.Tx.Memo) == 0 {
			// we didn't find our memo, it might be yggdrasil return. These are
			// tx markers without coin amounts because we allow yggdrasil to
			// figure out the coin amounts
			txYgg := tx.Tx
			txYgg.Coins = common.Coins{
				common.NewCoin(common.RuneAsset(), cosmos.ZeroUint()),
			}
			tx.Tx.Memo = fetchMemo(ctx, constAccessor, h.keeper, txYgg)
		}
		ctx.Logger().Info("handleMsgObservedTxOut request", "Tx:", tx.String())

		// if memo isn't valid or its an inbound memo, and its funds moving
		// from a yggdrasil vault, slash the node
		memo, _ := ParseMemo(tx.Tx.Memo)
		if memo.IsEmpty() || memo.IsInbound() {
			vault, err := h.keeper.GetVault(ctx, tx.ObservedPubKey)
			if err != nil {
				ctx.Logger().Error("fail to get vault", "error", err)
				continue
			}
			if vault.IsYggdrasil() {
				// a yggdrasil vault has apparently stolen funds, slash them
				for _, c := range append(tx.Tx.Coins, tx.Tx.Gas.ToCoins()...) {
					if err := h.mgr.Slasher().SlashNodeAccount(ctx, tx.ObservedPubKey, c.Asset, c.Amount, h.mgr); err != nil {
						ctx.Logger().Error("fail to slash account for sending extra fund", "error", err)
					}
				}
				vault.SubFunds(tx.Tx.Coins)
				vault.SubFunds(tx.Tx.Gas.ToCoins()) // we don't subsidize the gas when it's theft
				if err := h.keeper.SetVault(ctx, vault); err != nil {
					ctx.Logger().Error("fail to save vault", "error", err)
				}
			}

			continue
		}

		txOut := voter.GetTx(activeNodeAccounts) // get consensus tx, in case our for loop is incorrect
		txOut.Tx.Memo = tx.Tx.Memo
		m, err := processOneTxIn(ctx, h.keeper, txOut, msg.Signer)
		if err != nil || tx.Tx.Chain.IsEmpty() {
			ctx.Logger().Error("fail to process txOut",
				"error", err,
				"tx", tx.Tx.String())
			continue
		}

		// Apply Gas fees
		if err := AddGasFees(ctx, h.keeper, tx, h.mgr.GasMgr()); err != nil {
			ctx.Logger().Error("fail to add gas fee", "error", err)
			continue
		}

		// If sending from one of our vaults, decrement coins
		vault, err := h.keeper.GetVault(ctx, tx.ObservedPubKey)
		if err != nil {
			ctx.Logger().Error("fail to get vault", "error", err)
			continue
		}
		vault.SubFunds(tx.Tx.Coins)
		vault.OutboundTxCount++
		if vault.IsAsgard() && memo.IsType(TxMigrate) {
			// only remove the block height that had been specified in the memo
			vault.RemovePendingTxBlockHeights(memo.GetBlockHeight())
		}

		if !vault.HasFunds() && vault.Status == RetiringVault {
			// we have successfully removed all funds from a retiring vault,
			// mark it as inactive
			vault.Status = InactiveVault
		}
		if err := h.keeper.SetVault(ctx, vault); err != nil {
			ctx.Logger().Error("fail to save vault", "error", err)
			continue
		}

		// add addresses to observing addresses. This is used to detect
		// active/inactive observing node accounts
		h.mgr.ObMgr().AppendObserver(tx.Tx.Chain, txOut.Signers)

		_, err = handler(ctx, m)
		if err != nil {
			ctx.Logger().Error("handler failed:", "error", err)
			continue
		}
	}

	return &cosmos.Result{}, nil
}

func (h ObservedTxOutHandler) preflightV10(ctx cosmos.Context, voter ObservedTxVoter, nas NodeAccounts, tx ObservedTx, signer cosmos.AccAddress, version semver.Version, constAccessor constants.ConstantValues) (ObservedTxVoter, bool) {
	observeSlashPoints := constAccessor.GetInt64Value(constants.ObserveSlashPoints)
	observeFlex := constAccessor.GetInt64Value(constants.ObserveFlex)
	ok := false
	h.mgr.Slasher().IncSlashPoints(ctx, observeSlashPoints, signer)
	if !voter.Add(tx, signer) {
		// when the signer already sign it
		return voter, ok
	}
	if voter.HasConsensusV13(nas) {
		if voter.Height == 0 {
			ok = true
			voter.Height = common.BlockHeight(ctx)
			voter.Tx = voter.GetTxV13(nas)
			// tx has consensus now, so decrease the slashing point for all the signers whom voted for it
			h.mgr.Slasher().DecSlashPoints(ctx, observeSlashPoints, voter.Tx.Signers...)

		} else {
			// event the tx had been processed , given the signer just a bit late , so we still take away their slash points
			if common.BlockHeight(ctx) <= (voter.Height+observeFlex) && voter.Tx.Equals(tx) {
				h.mgr.Slasher().DecSlashPoints(ctx, observeSlashPoints, signer)
			}
		}
	}
	h.keeper.SetObservedTxOutVoter(ctx, voter)

	// Check to see if we have enough identical observations to process the transaction
	return voter, ok
}

// Handle a message to observe outbound tx
func (h ObservedTxOutHandler) handleV13(ctx cosmos.Context, version semver.Version, msg MsgObservedTxOut, constAccessor constants.ConstantValues) (*cosmos.Result, error) {
	activeNodeAccounts, err := h.keeper.ListActiveNodeAccounts(ctx)
	if err != nil {
		return nil, wrapError(ctx, err, "fail to get list of active node accounts")
	}

	handler := NewInternalHandler(h.keeper, h.mgr)

	for _, tx := range msg.Txs {
		// check we are sending from a valid vault
		if !h.keeper.VaultExists(ctx, tx.ObservedPubKey) {
			ctx.Logger().Info("Not valid Observed Pubkey", tx.ObservedPubKey)
			continue
		}

		voter, err := h.keeper.GetObservedTxOutVoter(ctx, tx.Tx.ID)
		if err != nil {
			ctx.Logger().Error("fail to get tx out voter", "error", err)
			continue
		}

		// check whether the tx has consensus
		voter, ok := h.preflightV10(ctx, voter, activeNodeAccounts, tx, msg.Signer, version, constAccessor)
		if !ok {
			if voter.Height == common.BlockHeight(ctx) {
				// we've already process the transaction, but we should still
				// update the observing addresses
				h.mgr.ObMgr().AppendObserver(tx.Tx.Chain, msg.GetSigners())
			}
			continue
		}
		tx.Tx.Memo = fetchMemo(ctx, constAccessor, h.keeper, tx.Tx)
		if len(tx.Tx.Memo) == 0 {
			// we didn't find our memo, it might be yggdrasil return. These are
			// tx markers without coin amounts because we allow yggdrasil to
			// figure out the coin amounts
			txYgg := tx.Tx
			txYgg.Coins = common.Coins{
				common.NewCoin(common.RuneAsset(), cosmos.ZeroUint()),
			}
			tx.Tx.Memo = fetchMemo(ctx, constAccessor, h.keeper, txYgg)
		}
		ctx.Logger().Info("handleMsgObservedTxOut request", "Tx:", tx.String())

		// if memo isn't valid or its an inbound memo, and its funds moving
		// from a yggdrasil vault, slash the node
		memo, _ := ParseMemo(tx.Tx.Memo)
		if memo.IsEmpty() || memo.IsInbound() {
			vault, err := h.keeper.GetVault(ctx, tx.ObservedPubKey)
			if err != nil {
				ctx.Logger().Error("fail to get vault", "error", err)
				continue
			}
			if vault.IsYggdrasil() {
				// a yggdrasil vault has apparently stolen funds, slash them
				for _, c := range append(tx.Tx.Coins, tx.Tx.Gas.ToCoins()...) {
					if err := h.mgr.Slasher().SlashNodeAccount(ctx, tx.ObservedPubKey, c.Asset, c.Amount, h.mgr); err != nil {
						ctx.Logger().Error("fail to slash account for sending extra fund", "error", err)
					}
				}
				vault.SubFunds(tx.Tx.Coins)
				vault.SubFunds(tx.Tx.Gas.ToCoins()) // we don't subsidize the gas when it's theft
				if err := h.keeper.SetVault(ctx, vault); err != nil {
					ctx.Logger().Error("fail to save vault", "error", err)
				}
			}

			continue
		}

		txOut := voter.GetTxV13(activeNodeAccounts) // get consensus tx, in case our for loop is incorrect
		txOut.Tx.Memo = tx.Tx.Memo
		m, err := processOneTxIn(ctx, h.keeper, txOut, msg.Signer)
		if err != nil || tx.Tx.Chain.IsEmpty() {
			ctx.Logger().Error("fail to process txOut",
				"error", err,
				"tx", tx.Tx.String())
			continue
		}

		// Apply Gas fees
		if err := AddGasFees(ctx, h.keeper, tx, h.mgr.GasMgr()); err != nil {
			ctx.Logger().Error("fail to add gas fee", "error", err)
			continue
		}

		// If sending from one of our vaults, decrement coins
		vault, err := h.keeper.GetVault(ctx, tx.ObservedPubKey)
		if err != nil {
			ctx.Logger().Error("fail to get vault", "error", err)
			continue
		}
		vault.SubFunds(tx.Tx.Coins)
		vault.OutboundTxCount++
		if vault.IsAsgard() && memo.IsType(TxMigrate) {
			// only remove the block height that had been specified in the memo
			vault.RemovePendingTxBlockHeights(memo.GetBlockHeight())
		}

		if !vault.HasFunds() && vault.Status == RetiringVault {
			// we have successfully removed all funds from a retiring vault,
			// mark it as inactive
			vault.Status = InactiveVault
		}
		if err := h.keeper.SetVault(ctx, vault); err != nil {
			ctx.Logger().Error("fail to save vault", "error", err)
			continue
		}

		// add addresses to observing addresses. This is used to detect
		// active/inactive observing node accounts
		h.mgr.ObMgr().AppendObserver(tx.Tx.Chain, txOut.Signers)

		_, err = handler(ctx, m)
		if err != nil {
			ctx.Logger().Error("handler failed:", "error", err)
			continue
		}
	}

	return &cosmos.Result{}, nil
}
