package thorchain

import (
	"errors"
	"fmt"

	"github.com/blang/semver"
	se "github.com/cosmos/cosmos-sdk/types/errors"
	tmtypes "github.com/tendermint/tendermint/types"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
	"gitlab.com/thorchain/thornode/x/thorchain/keeper"
)

// NativeTxHandler is to process native messages on THORChain
type NativeTxHandler struct {
	keeper keeper.Keeper
	mgr    Manager
}

// NewNativeTxHandler create a new instance of NativeTxHandler
func NewNativeTxHandler(keeper keeper.Keeper, mgr Manager) NativeTxHandler {
	return NativeTxHandler{
		keeper: keeper,
		mgr:    mgr,
	}
}

// Run is the main entry of NativeTxHandler
func (h NativeTxHandler) Run(ctx cosmos.Context, m cosmos.Msg, version semver.Version, constAccessor constants.ConstantValues) (*cosmos.Result, error) {
	msg, ok := m.(MsgNativeTx)
	if !ok {
		return nil, errInvalidMessage
	}
	if err := h.validate(ctx, msg, version); err != nil {
		ctx.Logger().Error("MsgNativeTx failed validation", "error", err)
		return nil, err
	}
	result, err := h.handle(ctx, msg, version, constAccessor)
	if err != nil {
		ctx.Logger().Error("fail to process MsgNativeTx", "error", err)
		return nil, err
	}
	return result, nil
}

func (h NativeTxHandler) validate(ctx cosmos.Context, msg MsgNativeTx, version semver.Version) error {
	if version.GTE(semver.MustParse("0.1.0")) {
		return h.validateV1(ctx, msg)
	}
	return errInvalidVersion
}

func (h NativeTxHandler) validateV1(ctx cosmos.Context, msg MsgNativeTx) error {
	return msg.ValidateBasic()
}

func (h NativeTxHandler) handle(ctx cosmos.Context, msg MsgNativeTx, version semver.Version, constAccessor constants.ConstantValues) (*cosmos.Result, error) {
	ctx.Logger().Info("receive MsgNativeTx", "from", msg.GetSigners()[0], "coins", msg.Coins, "memo", msg.Memo)
	if version.GTE(semver.MustParse("0.1.0")) {
		return h.handleV1(ctx, msg, version, constAccessor)
	}
	return nil, errInvalidVersion
}

func (h NativeTxHandler) handleV1(ctx cosmos.Context, msg MsgNativeTx, version semver.Version, constAccessor constants.ConstantValues) (*cosmos.Result, error) {
	banker := h.keeper.CoinKeeper()
	supplier := h.keeper.Supply()
	// TODO: this shouldn't be tied to swaps, and should be cheaper. But
	// TransactionFee will be fine for now.
	transactionFee := constAccessor.GetInt64Value(constants.TransactionFee)

	gas := common.NewCoin(common.RuneNative, cosmos.NewUint(uint64(transactionFee)))
	gasFee, err := gas.Native()
	if err != nil {
		return nil, fmt.Errorf("fail to get gas fee: %w", err)
	}

	coins, err := msg.Coins.Native()
	if err != nil {
		return nil, ErrInternal(err, "coins are native to THORChain")
	}

	totalCoins := cosmos.NewCoins(gasFee).Add(coins...)
	if !banker.HasCoins(ctx, msg.GetSigners()[0], totalCoins) {
		return nil, cosmos.ErrInsufficientCoins(err, "insufficient funds")
	}

	// send gas to reserve
	sdkErr := supplier.SendCoinsFromAccountToModule(ctx, msg.GetSigners()[0], ReserveName, cosmos.NewCoins(gasFee))
	if sdkErr != nil {
		return nil, fmt.Errorf("unable to send gas to reserve: %w", sdkErr)
	}

	hash := tmtypes.Tx(ctx.TxBytes()).Hash()
	txID, err := common.NewTxID(fmt.Sprintf("%X", hash))
	if err != nil {
		return nil, fmt.Errorf("fail to get tx hash: %w", err)
	}
	from, err := common.NewAddress(msg.GetSigners()[0].String())
	if err != nil {
		return nil, fmt.Errorf("fail to get from address: %w", err)
	}
	to, err := common.NewAddress(supplier.GetModuleAddress(AsgardName).String())
	if err != nil {
		return nil, fmt.Errorf("fail to get to address: %w", err)
	}

	tx := common.NewTx(txID, from, to, msg.Coins, common.Gas{gas}, msg.Memo)

	handler := NewInternalHandler(h.keeper, h.mgr)

	memo, _ := ParseMemo(msg.Memo) // ignore err
	var targetModule string
	switch memo.GetType() {
	case TxBond:
		targetModule = BondName
	case TxReserve:
		targetModule = ReserveName
	default:
		targetModule = AsgardName
	}
	// send funds to target module
	sdkErr = supplier.SendCoinsFromAccountToModule(ctx, msg.GetSigners()[0], targetModule, coins)
	if sdkErr != nil {
		return nil, sdkErr
	}

	// construct msg from memo
	txIn := ObservedTx{Tx: tx}
	m, txErr := processOneTxIn(ctx, h.keeper, txIn, msg.Signer)
	if txErr != nil {
		ctx.Logger().Error("fail to process native inbound tx", "error", txErr.Error(), "tx hash", tx.ID.String())
		if newErr := refundTx(ctx, txIn, h.mgr, h.keeper, constAccessor, CodeInvalidMemo, txErr.Error(), targetModule); nil != newErr {
			return nil, newErr
		}

		return &cosmos.Result{}, nil
	}

	// check if we've halted trading
	_, isSwap := m.(MsgSwap)
	_, isStake := m.(MsgStake)
	haltTrading, err := h.keeper.GetMimir(ctx, "HaltTrading")
	if isSwap || isStake {
		if (haltTrading > 0 && haltTrading < common.BlockHeight(ctx) && err == nil) || h.keeper.RagnarokInProgress(ctx) {
			ctx.Logger().Info("trading is halted!!")
			if newErr := refundTx(ctx, txIn, h.mgr, h.keeper, constAccessor, se.ErrUnauthorized.ABCICode(), "trading halted", targetModule); nil != newErr {
				return nil, ErrInternal(newErr, "trading is halted, fail to refund")
			}
			return &cosmos.Result{}, nil
		}
	}

	// if its a swap, send it to our queue for processing later
	if isSwap {
		if err := h.keeper.SetSwapQueueItem(ctx, m.(MsgSwap)); err != nil {
			return nil, err
		}
		return &cosmos.Result{}, nil
	}

	result, err := handler(ctx, m)
	if err != nil {
		code := uint32(1)
		var e se.Error
		if errors.As(err, &e) {
			code = e.ABCICode()
		}
		if err := refundTx(ctx, txIn, h.mgr, h.keeper, constAccessor, code, err.Error(), targetModule); err != nil {
			return nil, fmt.Errorf("fail to refund tx: %w", err)
		}
	}
	return result, nil
}
