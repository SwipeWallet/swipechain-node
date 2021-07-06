package thorchain

import (
	"fmt"

	"github.com/blang/semver"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
	"gitlab.com/thorchain/thornode/x/thorchain/keeper"
)

// MsgHandler is an interface expect all handler to implement
type MsgHandler interface {
	Run(ctx cosmos.Context, msg cosmos.Msg, version semver.Version, constAccessor constants.ConstantValues) (*cosmos.Result, error)
}

// NewExternalHandler returns a handler for "thorchain" type messages.
func NewExternalHandler(keeper keeper.Keeper, mgr Manager) cosmos.Handler {
	return func(ctx cosmos.Context, msg cosmos.Msg) (*cosmos.Result, error) {
		ctx = ctx.WithEventManager(cosmos.NewEventManager())
		version := keeper.GetLowestActiveVersion(ctx)
		constantValues := constants.GetConstantValues(version)
		if constantValues == nil {
			return nil, errConstNotAvailable
		}
		handlerMap := getHandlerMapping(keeper, mgr)
		h, ok := handlerMap[msg.Type()]
		if !ok {
			errMsg := fmt.Sprintf("Unrecognized thorchain Msg type: %v", msg.Type())
			return nil, cosmos.ErrUnknownRequest(errMsg)
		}
		result, err := h.Run(ctx, msg, version, constantValues)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = &cosmos.Result{}
		}
		if len(ctx.EventManager().Events()) > 0 {
			result.Events = result.Events.AppendEvents(ctx.EventManager().Events())
		}
		return result, nil
	}
}

func getHandlerMapping(keeper keeper.Keeper, mgr Manager) map[string]MsgHandler {
	// New arch handlers
	m := make(map[string]MsgHandler)

	// consensus handlers
	m[MsgTssPool{}.Type()] = NewTssHandler(keeper, mgr)
	m[MsgObservedTxIn{}.Type()] = NewObservedTxInHandler(keeper, mgr)
	m[MsgObservedTxOut{}.Type()] = NewObservedTxOutHandler(keeper, mgr)
	m[MsgTssKeysignFail{}.Type()] = NewTssKeysignHandler(keeper, mgr)
	m[MsgErrataTx{}.Type()] = NewErrataTxHandler(keeper, mgr)
	m[MsgMimir{}.Type()] = NewMimirHandler(keeper, mgr)
	m[MsgBan{}.Type()] = NewBanHandler(keeper, mgr)
	m[MsgNetworkFee{}.Type()] = NewNetworkFeeHandler(keeper, mgr)

	// cli handlers (non-consensus)
	m[MsgSetNodeKeys{}.Type()] = NewSetNodeKeysHandler(keeper, mgr)
	m[MsgSetVersion{}.Type()] = NewVersionHandler(keeper, mgr)
	m[MsgSetIPAddress{}.Type()] = NewIPAddressHandler(keeper, mgr)

	// native handlers (non-consensus)
	m[MsgSend{}.Type()] = NewSendHandler(keeper, mgr)
	m[MsgNativeTx{}.Type()] = NewNativeTxHandler(keeper, mgr)
	return m
}

// NewInternalHandler returns a handler for "thorchain" internal type messages.
func NewInternalHandler(keeper keeper.Keeper, mgr Manager) cosmos.Handler {
	return func(ctx cosmos.Context, msg cosmos.Msg) (*cosmos.Result, error) {
		version := keeper.GetLowestActiveVersion(ctx)
		constantValues := constants.GetConstantValues(version)
		if constantValues == nil {
			return nil, errConstNotAvailable
		}
		handlerMap := getInternalHandlerMapping(keeper, mgr)
		h, ok := handlerMap[msg.Type()]
		if !ok {
			errMsg := fmt.Sprintf("Unrecognized thorchain Msg type: %v", msg.Type())
			return nil, cosmos.ErrUnknownRequest(errMsg)
		}
		return h.Run(ctx, msg, version, constantValues)
	}
}

func getInternalHandlerMapping(keeper keeper.Keeper, mgr Manager) map[string]MsgHandler {
	// New arch handlers
	m := make(map[string]MsgHandler)
	m[MsgOutboundTx{}.Type()] = NewOutboundTxHandler(keeper, mgr)
	m[MsgYggdrasil{}.Type()] = NewYggdrasilHandler(keeper, mgr)
	m[MsgSwap{}.Type()] = NewSwapHandler(keeper, mgr)
	m[MsgReserveContributor{}.Type()] = NewReserveContributorHandler(keeper, mgr)
	m[MsgBond{}.Type()] = NewBondHandler(keeper, mgr)
	m[MsgUnBond{}.Type()] = NewUnBondHandler(keeper, mgr)
	m[MsgLeave{}.Type()] = NewLeaveHandler(keeper, mgr)
	m[MsgAdd{}.Type()] = NewAddHandler(keeper, mgr)
	m[MsgUnStake{}.Type()] = NewUnstakeHandler(keeper, mgr)
	m[MsgStake{}.Type()] = NewStakeHandler(keeper, mgr)
	m[MsgRefundTx{}.Type()] = NewRefundHandler(keeper, mgr)
	m[MsgMigrate{}.Type()] = NewMigrateHandler(keeper, mgr)
	m[MsgRagnarok{}.Type()] = NewRagnarokHandler(keeper, mgr)
	m[MsgSwitch{}.Type()] = NewSwitchHandler(keeper, mgr)
	return m
}

func fetchMemo(ctx cosmos.Context, constAccessor constants.ConstantValues, keeper keeper.Keeper, tx common.Tx) string {
	if len(tx.Memo) > 0 {
		return tx.Memo
	}

	var memo string
	// attempt to pull memo from tx marker
	hash := tx.Hash()
	marks, err := keeper.ListTxMarker(ctx, hash)
	if err != nil {
		ctx.Logger().Error("fail to get tx marker", "error", err)
	}
	if len(marks) > 0 {
		// filter out expired tx markers
		period := constAccessor.GetInt64Value(constants.SigningTransactionPeriod) * 3
		marks = marks.FilterByMinHeight(common.BlockHeight(ctx) - period)

		// if we still have a marker, add the memo
		if len(marks) > 0 {
			var mark TxMarker
			mark, marks = marks.Pop()
			memo = mark.Memo
		}

		// update our marker list
		if err := keeper.SetTxMarkers(ctx, hash, marks); err != nil {
			ctx.Logger().Error("fail to set tx markers", "error", err)
		}
	}
	return memo
}

func processOneTxIn(ctx cosmos.Context, keeper keeper.Keeper, tx ObservedTx, signer cosmos.AccAddress) (cosmos.Msg, error) {
	if len(tx.Tx.Coins) == 0 {
		return nil, cosmos.ErrUnknownRequest("no coin found")
	}

	memo, err := ParseMemo(tx.Tx.Memo)
	if err != nil {
		ctx.Logger().Error("fail to parse memo", "error", err)
		return nil, err
	}
	// THORNode should not have one tx across chain, if it is cross chain it should be separate tx
	var newMsg cosmos.Msg
	// interpret the memo and initialize a corresponding msg event
	switch m := memo.(type) {
	case StakeMemo:
		newMsg, err = getMsgStakeFromMemo(ctx, m, tx, signer)
	case UnstakeMemo:
		newMsg, err = getMsgUnstakeFromMemo(m, tx, signer)
	case SwapMemo:
		newMsg, err = getMsgSwapFromMemo(m, tx, signer)
	case AddMemo:
		newMsg, err = getMsgAddFromMemo(m, tx, signer)
	case RefundMemo:
		newMsg, err = getMsgRefundFromMemo(m, tx, signer)
	case OutboundMemo:
		newMsg, err = getMsgOutboundFromMemo(m, tx, signer)
	case MigrateMemo:
		newMsg, err = getMsgMigrateFromMemo(m, tx, signer)
	case BondMemo:
		newMsg, err = getMsgBondFromMemo(m, tx, signer)
	case UnbondMemo:
		newMsg, err = getMsgUnbondFromMemo(m, tx, signer)
	case RagnarokMemo:
		newMsg, err = getMsgRagnarokFromMemo(m, tx, signer)
	case LeaveMemo:
		newMsg, err = getMsgLeaveFromMemo(m, tx, signer)
	case YggdrasilFundMemo:
		newMsg = NewMsgYggdrasil(tx.Tx, tx.ObservedPubKey, m.GetBlockHeight(), true, tx.Tx.Coins, signer)
	case YggdrasilReturnMemo:
		newMsg = NewMsgYggdrasil(tx.Tx, tx.ObservedPubKey, m.GetBlockHeight(), false, tx.Tx.Coins, signer)
	case ReserveMemo:
		res := NewReserveContributor(tx.Tx.FromAddress, tx.Tx.Coins.GetCoin(common.RuneAsset()).Amount)
		newMsg = NewMsgReserveContributor(tx.Tx, res, signer)
	case SwitchMemo:
		newMsg = NewMsgSwitch(tx.Tx, memo.GetDestination(), signer)
	default:
		return nil, errInvalidMemo
	}

	if err != nil {
		return newMsg, err
	}
	return newMsg, newMsg.ValidateBasic()
}

func getMsgSwapFromMemo(memo SwapMemo, tx ObservedTx, signer cosmos.AccAddress) (cosmos.Msg, error) {
	if memo.Destination.IsEmpty() {
		memo.Destination = tx.Tx.FromAddress
	}
	return NewMsgSwap(tx.Tx, memo.GetAsset(), memo.Destination, memo.SlipLimit, signer), nil
}

func getMsgUnstakeFromMemo(memo UnstakeMemo, tx ObservedTx, signer cosmos.AccAddress) (cosmos.Msg, error) {
	withdrawAmount := cosmos.NewUint(MaxUnstakeBasisPoints)
	if !memo.GetAmount().IsZero() {
		withdrawAmount = memo.GetAmount()
	}
	return NewMsgUnStake(tx.Tx, tx.Tx.FromAddress, withdrawAmount, memo.GetAsset(), signer), nil
}

func getMsgStakeFromMemo(ctx cosmos.Context, memo StakeMemo, tx ObservedTx, signer cosmos.AccAddress) (cosmos.Msg, error) {
	// Extract the Rune amount and the asset amount from the transaction. At least one of them must be
	// nonzero. If THORNode saw two types of coins, one of them must be the asset coin.
	runeCoin := tx.Tx.Coins.GetCoin(common.RuneAsset())
	assetCoin := tx.Tx.Coins.GetCoin(memo.GetAsset())

	runeAddr := tx.Tx.FromAddress
	assetAddr := memo.GetDestination()
	// this is to cover multi-chain scenario, for example BTC , staker who
	// would like to stake in BTC pool,  will have to complete
	// the stake operation by sending in two asymmetric stake tx, one tx on BTC
	// chain with memo stake:BTC:<RUNE address> ,
	// and another one on Binance chain with stake:BTC , with only RUNE as the coin
	// Thorchain will use the <RUNE address> to match these two together , and
	// consider it as one stake.
	if !runeAddr.IsChain(common.RuneAsset().Chain) {
		runeAddr = memo.GetDestination()
		assetAddr = tx.Tx.FromAddress
	} else {
		// if it is on THOR chain , while the asset addr is empty, then the asset addr is runeAddr
		if assetAddr.IsEmpty() {
			assetAddr = runeAddr
		}
	}

	return NewMsgStake(tx.Tx, memo.GetAsset(), runeCoin.Amount, assetCoin.Amount, runeAddr, assetAddr, signer), nil
}

func getMsgAddFromMemo(memo AddMemo, tx ObservedTx, signer cosmos.AccAddress) (cosmos.Msg, error) {
	runeCoin := tx.Tx.Coins.GetCoin(common.RuneAsset())
	assetCoin := tx.Tx.Coins.GetCoin(memo.GetAsset())
	return NewMsgAdd(tx.Tx, memo.GetAsset(), runeCoin.Amount, assetCoin.Amount, signer), nil
}

func getMsgRefundFromMemo(memo RefundMemo, tx ObservedTx, signer cosmos.AccAddress) (cosmos.Msg, error) {
	return NewMsgRefundTx(tx, memo.GetTxID(), signer), nil
}

func getMsgOutboundFromMemo(memo OutboundMemo, tx ObservedTx, signer cosmos.AccAddress) (cosmos.Msg, error) {
	return NewMsgOutboundTx(tx, memo.GetTxID(), signer), nil
}

func getMsgMigrateFromMemo(memo MigrateMemo, tx ObservedTx, signer cosmos.AccAddress) (cosmos.Msg, error) {
	return NewMsgMigrate(tx, memo.GetBlockHeight(), signer), nil
}

func getMsgRagnarokFromMemo(memo RagnarokMemo, tx ObservedTx, signer cosmos.AccAddress) (cosmos.Msg, error) {
	return NewMsgRagnarok(tx, memo.GetBlockHeight(), signer), nil
}

func getMsgLeaveFromMemo(memo LeaveMemo, tx ObservedTx, signer cosmos.AccAddress) (cosmos.Msg, error) {
	return NewMsgLeave(tx.Tx, memo.GetAccAddress(), signer), nil
}

func getMsgBondFromMemo(memo BondMemo, tx ObservedTx, signer cosmos.AccAddress) (cosmos.Msg, error) {
	coin := tx.Tx.Coins.GetCoin(common.RuneAsset())
	return NewMsgBond(tx.Tx, memo.GetAccAddress(), coin.Amount, tx.Tx.FromAddress, signer), nil
}

func getMsgUnbondFromMemo(memo UnbondMemo, tx ObservedTx, signer cosmos.AccAddress) (cosmos.Msg, error) {
	return NewMsgUnBond(tx.Tx, memo.GetAccAddress(), memo.GetAmount(), tx.Tx.FromAddress, signer), nil
}
