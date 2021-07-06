package thorchain

import (
	"github.com/blang/semver"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
	"gitlab.com/thorchain/thornode/x/thorchain/keeper"
)

// CommonOutboundTxHandler is the place where those common logic can be shared
// between multiple different kind of outbound tx handler
// at the moment, handler_refund, and handler_outbound_tx are largely the same
// , only some small difference
type CommonOutboundTxHandler struct {
	keeper keeper.Keeper
	mgr    Manager
}

// NewCommonOutboundTxHandler create a new instance of the CommonOutboundTxHandler
func NewCommonOutboundTxHandler(k keeper.Keeper, mgr Manager) CommonOutboundTxHandler {
	return CommonOutboundTxHandler{
		keeper: k,
		mgr:    mgr,
	}
}

func (h CommonOutboundTxHandler) slash(ctx cosmos.Context, version semver.Version, tx ObservedTx) error {
	var returnErr error
	for _, c := range tx.Tx.Coins {
		if err := h.mgr.Slasher().SlashNodeAccount(ctx, tx.ObservedPubKey, c.Asset, c.Amount, h.mgr); err != nil {
			ctx.Logger().Error("fail to slash account", "error", err)
			returnErr = err
		}
	}
	return returnErr
}

func (h CommonOutboundTxHandler) handle(ctx cosmos.Context, version semver.Version, tx ObservedTx, inTxID common.TxID, constAccessor constants.ConstantValues) (*cosmos.Result, error) {
	// note: Outbound tx usually it is related to an inbound tx except migration
	// thus here try to get the ObservedTxInVoter,  and set the tx out hash accordingly
	voter, err := h.keeper.GetObservedTxInVoter(ctx, inTxID)
	if err != nil {
		return nil, ErrInternal(err, "fail to get observed tx voter")
	}
	voter.AddOutTx(tx.Tx)
	h.keeper.SetObservedTxInVoter(ctx, voter)

	// complete events
	if voter.IsDone() {
		for _, item := range voter.OutTxs {
			if err := h.mgr.EventMgr().EmitEvent(ctx, NewEventOutbound(inTxID, item)); err != nil {
				return nil, ErrInternal(err, "fail to emit outbound event")
			}
		}
	}

	shouldSlash := true
	signingTransPeriod := constAccessor.GetInt64Value(constants.SigningTransactionPeriod)
	// every Signing Transaction Period , THORNode will check whether a
	// TxOutItem had been sent by signer or not
	// if a txout item that is older than SigningTransactionPeriod, but has not
	// been sent out by signer , then it will create a new TxOutItem
	// here THORNode will have to mark all the TxOutItem to complete one the tx
	// get processed
	for height := voter.Height; height <= common.BlockHeight(ctx); height += signingTransPeriod {

		// update txOut record with our TxID that sent funds out of the pool
		txOut, err := h.keeper.GetTxOut(ctx, height)
		if err != nil {
			ctx.Logger().Error("unable to get txOut record", "error", err)
			return nil, cosmos.ErrUnknownRequest(err.Error())
		}

		// Save TxOut back with the TxID only when the TxOut on the block height is
		// not empty
		for i, txOutItem := range txOut.TxArray {
			// withdraw , refund etc, one inbound tx might result two outbound
			// txes, THORNode have to correlate outbound tx back to the
			// inbound, and also txitem , thus THORNode could record both
			// outbound tx hash correctly given every tx item will only have
			// one coin in it , THORNode could use that to identify which tx it
			// is
			if txOutItem.InHash.Equals(inTxID) &&
				txOutItem.OutHash.IsEmpty() &&
				tx.Tx.Coins.Equals(common.Coins{txOutItem.Coin}) &&
				tx.Tx.Chain.Equals(txOutItem.Chain) &&
				tx.Tx.ToAddress.Equals(txOutItem.ToAddress) &&
				tx.ObservedPubKey.Equals(txOutItem.VaultPubKey) {

				txOut.TxArray[i].OutHash = tx.Tx.ID
				shouldSlash = false

				if err := h.keeper.SetTxOut(ctx, txOut); err != nil {
					ctx.Logger().Error("fail to save tx out", "error", err)
				}
				break
			}
		}
	}

	// clear any tx markers
	// markers usually clear themselves, but in rare situations, where we get
	// two different transactions with matching hashes (but not matching
	// memos), we want to clear the markers with the memo we just used cleared.
	hash := tx.Tx.Hash()
	marks, err := h.keeper.ListTxMarker(ctx, hash)
	if err != nil {
		ctx.Logger().Error("fail to get tx marker", "error", err)
	}
	var newMarks TxMarkers
	for _, mark := range marks {
		if mark.Memo != tx.Tx.Memo {
			newMarks = append(newMarks, mark)
		}
	}
	// update our marker list
	if err := h.keeper.SetTxMarkers(ctx, hash, newMarks); err != nil {
		ctx.Logger().Error("fail to set tx markers", "error", err)
	}

	if shouldSlash {
		ctx.Logger().Info("slash node account, no matched tx out item", "inbound txid", inTxID, "outbound tx", tx.Tx)
		if err := h.slash(ctx, version, tx); err != nil {
			return nil, ErrInternal(err, "fail to slash account")
		}
	}

	if err := h.keeper.SetLastSignedHeight(ctx, voter.Height); err != nil {
		ctx.Logger().Error("fail to update last signed height", "error", err)
	}

	return &cosmos.Result{}, nil
}
