package thorchain

import (
	"fmt"

	"github.com/blang/semver"
	se "github.com/cosmos/cosmos-sdk/types/errors"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
	"gitlab.com/thorchain/thornode/x/thorchain/keeper"
)

// ObservedTxInHandler to handle MsgObservedTxIn
type ObservedTxInHandler struct {
	keeper keeper.Keeper
	mgr    Manager
}

// NewObservedTxInHandler create a new instance of ObservedTxInHandler
func NewObservedTxInHandler(keeper keeper.Keeper, mgr Manager) ObservedTxInHandler {
	return ObservedTxInHandler{
		keeper: keeper,
		mgr:    mgr,
	}
}

// Run is the main entry point of ObservedTxInHandler
func (h ObservedTxInHandler) Run(ctx cosmos.Context, m cosmos.Msg, version semver.Version, constAccessor constants.ConstantValues) (*cosmos.Result, error) {
	msg, ok := m.(MsgObservedTxIn)
	if !ok {
		return nil, errInvalidMessage
	}
	err := h.validate(ctx, msg, version)
	if err != nil {
		ctx.Logger().Error("MsgObservedTxIn failed validation", "error", err)
		return nil, err
	}

	result, err := h.handle(ctx, msg, version, constAccessor)
	if err != nil {
		ctx.Logger().Error("fail to handle MsgObservedTxIn message", "error", err)
	}
	return result, err
}

func (h ObservedTxInHandler) validate(ctx cosmos.Context, msg MsgObservedTxIn, version semver.Version) error {
	if version.GTE(semver.MustParse("0.1.0")) {
		return h.validateV1(ctx, msg)
	}
	return errInvalidVersion
}

func (h ObservedTxInHandler) validateV1(ctx cosmos.Context, msg MsgObservedTxIn) error {
	if err := msg.ValidateBasic(); err != nil {
		return err
	}

	if !isSignedByActiveNodeAccounts(ctx, h.keeper, msg.GetSigners()) {
		return cosmos.ErrUnauthorized(fmt.Sprintf("%+v are not authorized", msg.GetSigners()))
	}

	return nil
}

func (h ObservedTxInHandler) handle(ctx cosmos.Context, msg MsgObservedTxIn, version semver.Version, constAccessor constants.ConstantValues) (*cosmos.Result, error) {
	if version.GTE(semver.MustParse("0.13.0")) {
		return h.handleV13(ctx, version, msg, constAccessor)
	} else if version.GTE(semver.MustParse("0.1.0")) {
		return h.handleV1(ctx, version, msg, constAccessor)
	}
	return nil, errBadVersion
}

func (h ObservedTxInHandler) preflight(ctx cosmos.Context, voter ObservedTxVoter, nas NodeAccounts, tx ObservedTx, signer cosmos.AccAddress, version semver.Version, constAccessor constants.ConstantValues) (ObservedTxVoter, bool) {
	observeSlashPoints := constAccessor.GetInt64Value(constants.ObserveSlashPoints)
	observeFlex := constAccessor.GetInt64Value(constants.ObserveFlex)
	h.mgr.Slasher().IncSlashPoints(ctx, observeSlashPoints, signer)
	ok := false
	if !voter.Add(tx, signer) {
		return voter, ok
	}
	if voter.HasConsensus(nas) {
		if voter.Height == 0 {
			ok = true
			voter.Height = common.BlockHeight(ctx)
			// this is the tx that has consensus
			voter.Tx = voter.GetTx(nas)

			// tx has consensus now, so decrease the slashing points for all the signers whom had voted for it
			h.mgr.Slasher().DecSlashPoints(ctx, observeSlashPoints, voter.Tx.Signers...)
		} else {
			// event the tx had been processed , given the signer just a bit late , so still take away their slash points
			// but only when the tx signer are voting is the tx that already reached consensus
			if common.BlockHeight(ctx) <= (voter.Height+observeFlex) && voter.Tx.Equals(tx) {
				h.mgr.Slasher().DecSlashPoints(ctx, observeSlashPoints, signer)
			}
		}
	}
	h.keeper.SetObservedTxInVoter(ctx, voter)

	// Check to see if we have enough identical observations to process the transaction
	return voter, ok
}

// Handle a message to observe inbound tx
func (h ObservedTxInHandler) handleV1(ctx cosmos.Context, version semver.Version, msg MsgObservedTxIn, constAccessor constants.ConstantValues) (*cosmos.Result, error) {
	activeNodeAccounts, err := h.keeper.ListActiveNodeAccounts(ctx)
	if err != nil {
		return nil, wrapError(ctx, err, "fail to get list of active node accounts")
	}
	handler := NewInternalHandler(h.keeper, h.mgr)
	for _, tx := range msg.Txs {
		// check we are sending to a valid vault
		if !h.keeper.VaultExists(ctx, tx.ObservedPubKey) {
			ctx.Logger().Info("Not valid Observed Pubkey", "observed pub key", tx.ObservedPubKey)
			continue
		}

		voter, err := h.keeper.GetObservedTxInVoter(ctx, tx.Tx.ID)
		if err != nil {
			ctx.Logger().Error("fail to get tx out voter", "error", err)
			continue
		}

		voter, ok := h.preflight(ctx, voter, activeNodeAccounts, tx, msg.Signer, version, constAccessor)
		if !ok {
			if voter.Height == common.BlockHeight(ctx) {
				// we've already process the transaction, but we should still
				// update the observing addresses
				h.mgr.ObMgr().AppendObserver(tx.Tx.Chain, msg.GetSigners())
			}
			continue
		}

		// all logic after this  is after consensus
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

		ctx.Logger().Info("handleMsgObservedTxIn request", "Tx:", tx.String())

		txIn := voter.GetTx(activeNodeAccounts)
		txIn.Tx.Memo = tx.Tx.Memo
		vault, err := h.keeper.GetVault(ctx, tx.ObservedPubKey)
		if err != nil {
			ctx.Logger().Error("fail to get vault", "error", err)
			continue
		}

		vault.AddFunds(tx.Tx.Coins)
		vault.InboundTxCount++
		memo, _ := ParseMemo(tx.Tx.Memo) // ignore err
		if vault.IsYggdrasil() && memo.IsType(TxYggdrasilFund) {
			vault.RemovePendingTxBlockHeights(memo.GetBlockHeight())
		}
		if err := h.keeper.SetVault(ctx, vault); err != nil {
			ctx.Logger().Error("fail to set vault", "error", err)
			continue
		}

		if !vault.IsAsgard() {
			ctx.Logger().Info("Vault is not an Asgard vault, transaction ignored.")
			continue
		}
		if vault.Status == InactiveVault {
			ctx.Logger().Error("Vault is inactive, transaction ignored.")
			continue
		}

		if memo.IsOutbound() || memo.IsInternal() {
			// do not process outbound handlers here, or internal handlers
			continue
		}

		// construct msg from memo
		m, txErr := processOneTxIn(ctx, h.keeper, txIn, msg.Signer)
		if txErr != nil {
			ctx.Logger().Error("fail to process inbound tx", "error", txErr.Error(), "tx hash", tx.Tx.ID.String())
			if newErr := refundTx(ctx, tx, h.mgr, h.keeper, constAccessor, CodeInvalidMemo, txErr.Error(), ""); nil != newErr {
				ctx.Logger().Error("fail to refund", "error", err)
			}
			continue
		}

		if err := h.keeper.SetLastChainHeight(ctx, tx.Tx.Chain, tx.BlockHeight); err != nil {
			ctx.Logger().Error("fail to set last chain height", "error", err)
		}

		// add addresses to observing addresses. This is used to detect
		// active/inactive observing node accounts
		h.mgr.ObMgr().AppendObserver(tx.Tx.Chain, txIn.Signers)

		// check if we've halted trading
		_, isSwap := m.(MsgSwap)
		_, isStake := m.(MsgStake)
		haltTrading, err := h.keeper.GetMimir(ctx, "HaltTrading")
		if isSwap || isStake {
			if (haltTrading > 0 && haltTrading < common.BlockHeight(ctx) && err == nil) || h.keeper.RagnarokInProgress(ctx) {
				ctx.Logger().Info("trading is halted!!")
				if newErr := refundTx(ctx, tx, h.mgr, h.keeper, constAccessor, se.ErrUnauthorized.ABCICode(), "trading halted", ""); nil != newErr {
					ctx.Logger().Error("fail to refund for halted trading", "error", err)
				}
				continue
			}
		}

		// if its a swap, send it to our queue for processing later
		if isSwap {
			if err := h.keeper.SetSwapQueueItem(ctx, m.(MsgSwap)); err != nil {
				ctx.Logger().Error("fail to add swap to queue", "error", err)
			}
			continue
		}

		_, err = handler(ctx, m)
		if err != nil {
			if err := refundTx(ctx, tx, h.mgr, h.keeper, constAccessor, CodeTxFail, err.Error(), ""); err != nil {
				ctx.Logger().Error("fail to refund", "error", err)
			}
		}
	}
	return &cosmos.Result{}, nil
}

func (h ObservedTxInHandler) preflightV10(ctx cosmos.Context, voter ObservedTxVoter, nas NodeAccounts, tx ObservedTx, signer cosmos.AccAddress, version semver.Version, constAccessor constants.ConstantValues) (ObservedTxVoter, bool) {
	observeSlashPoints := constAccessor.GetInt64Value(constants.ObserveSlashPoints)
	observeFlex := constAccessor.GetInt64Value(constants.ObserveFlex)
	h.mgr.Slasher().IncSlashPoints(ctx, observeSlashPoints, signer)
	ok := false
	if !voter.Add(tx, signer) {
		return voter, ok
	}
	if voter.HasConsensusV13(nas) {
		if voter.Height == 0 {
			ok = true
			voter.Height = common.BlockHeight(ctx)
			// this is the tx that has consensus
			voter.Tx = voter.GetTxV13(nas)

			// tx has consensus now, so decrease the slashing points for all the signers whom had voted for it
			h.mgr.Slasher().DecSlashPoints(ctx, observeSlashPoints, voter.Tx.Signers...)
		} else {
			// event the tx had been processed , given the signer just a bit late , so still take away their slash points
			// but only when the tx signer are voting is the tx that already reached consensus
			if common.BlockHeight(ctx) <= (voter.Height+observeFlex) && voter.Tx.Equals(tx) {
				h.mgr.Slasher().DecSlashPoints(ctx, observeSlashPoints, signer)
			}
		}
	}
	h.keeper.SetObservedTxInVoter(ctx, voter)

	// Check to see if we have enough identical observations to process the transaction
	return voter, ok
}

// Handle a message to observe inbound tx
func (h ObservedTxInHandler) handleV13(ctx cosmos.Context, version semver.Version, msg MsgObservedTxIn, constAccessor constants.ConstantValues) (*cosmos.Result, error) {
	activeNodeAccounts, err := h.keeper.ListActiveNodeAccounts(ctx)
	if err != nil {
		return nil, wrapError(ctx, err, "fail to get list of active node accounts")
	}
	handler := NewInternalHandler(h.keeper, h.mgr)
	for _, tx := range msg.Txs {
		// check we are sending to a valid vault
		if !h.keeper.VaultExists(ctx, tx.ObservedPubKey) {
			ctx.Logger().Info("Not valid Observed Pubkey", "observed pub key", tx.ObservedPubKey)
			continue
		}

		voter, err := h.keeper.GetObservedTxInVoter(ctx, tx.Tx.ID)
		if err != nil {
			ctx.Logger().Error("fail to get tx out voter", "error", err)
			continue
		}

		voter, ok := h.preflightV10(ctx, voter, activeNodeAccounts, tx, msg.Signer, version, constAccessor)
		if !ok {
			if voter.Height == common.BlockHeight(ctx) {
				// we've already process the transaction, but we should still
				// update the observing addresses
				h.mgr.ObMgr().AppendObserver(tx.Tx.Chain, msg.GetSigners())
			}
			continue
		}

		// all logic after this  is after consensus
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

		ctx.Logger().Info("handleMsgObservedTxIn request", "Tx:", tx.String())

		txIn := voter.GetTxV13(activeNodeAccounts)
		txIn.Tx.Memo = tx.Tx.Memo
		vault, err := h.keeper.GetVault(ctx, tx.ObservedPubKey)
		if err != nil {
			ctx.Logger().Error("fail to get vault", "error", err)
			continue
		}

		vault.AddFunds(tx.Tx.Coins)
		vault.InboundTxCount++
		memo, _ := ParseMemo(tx.Tx.Memo) // ignore err
		if vault.IsYggdrasil() && memo.IsType(TxYggdrasilFund) {
			vault.RemovePendingTxBlockHeights(memo.GetBlockHeight())
		}
		if err := h.keeper.SetVault(ctx, vault); err != nil {
			ctx.Logger().Error("fail to set vault", "error", err)
			continue
		}

		if !vault.IsAsgard() {
			ctx.Logger().Info("Vault is not an Asgard vault, transaction ignored.")
			continue
		}
		if vault.Status == InactiveVault {
			ctx.Logger().Error("Vault is inactive, transaction ignored.")
			continue
		}

		if memo.IsOutbound() || memo.IsInternal() {
			// do not process outbound handlers here, or internal handlers
			continue
		}

		// construct msg from memo
		m, txErr := processOneTxIn(ctx, h.keeper, txIn, msg.Signer)
		if txErr != nil {
			ctx.Logger().Error("fail to process inbound tx", "error", txErr.Error(), "tx hash", tx.Tx.ID.String())
			if newErr := refundTx(ctx, tx, h.mgr, h.keeper, constAccessor, CodeInvalidMemo, txErr.Error(), ""); nil != newErr {
				ctx.Logger().Error("fail to refund", "error", err)
			}
			continue
		}

		if err := h.keeper.SetLastChainHeight(ctx, tx.Tx.Chain, tx.BlockHeight); err != nil {
			ctx.Logger().Error("fail to set last chain height", "error", err)
		}

		// add addresses to observing addresses. This is used to detect
		// active/inactive observing node accounts
		h.mgr.ObMgr().AppendObserver(tx.Tx.Chain, txIn.Signers)

		// check if we've halted trading
		_, isSwap := m.(MsgSwap)
		_, isStake := m.(MsgStake)
		haltTrading, err := h.keeper.GetMimir(ctx, "HaltTrading")
		if isSwap || isStake {
			if (haltTrading > 0 && haltTrading < common.BlockHeight(ctx) && err == nil) || h.keeper.RagnarokInProgress(ctx) {
				ctx.Logger().Info("trading is halted!!")
				if newErr := refundTx(ctx, tx, h.mgr, h.keeper, constAccessor, se.ErrUnauthorized.ABCICode(), "trading halted", ""); nil != newErr {
					ctx.Logger().Error("fail to refund for halted trading", "error", err)
				}
				continue
			}
		}

		// if its a swap, send it to our queue for processing later
		if isSwap {
			if err := h.keeper.SetSwapQueueItem(ctx, m.(MsgSwap)); err != nil {
				ctx.Logger().Error("fail to add swap to queue", "error", err)
			}
			continue
		}

		_, err = handler(ctx, m)
		if err != nil {
			if err := refundTx(ctx, tx, h.mgr, h.keeper, constAccessor, CodeTxFail, err.Error(), ""); err != nil {
				ctx.Logger().Error("fail to refund", "error", err)
			}
		}
	}
	return &cosmos.Result{}, nil
}
