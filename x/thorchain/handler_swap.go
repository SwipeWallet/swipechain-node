package thorchain

import (
	"github.com/blang/semver"

	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
	"gitlab.com/thorchain/thornode/x/thorchain/keeper"
)

// SwapHandler is the handler to process swap request
type SwapHandler struct {
	keeper keeper.Keeper
	mgr    Manager
}

// NewSwapHandler create a new instance of swap handler
func NewSwapHandler(keeper keeper.Keeper, mgr Manager) SwapHandler {
	return SwapHandler{
		keeper: keeper,
		mgr:    mgr,
	}
}

// Run is the main entry point of swap message
func (h SwapHandler) Run(ctx cosmos.Context, m cosmos.Msg, version semver.Version, constAccessor constants.ConstantValues) (*cosmos.Result, error) {
	msg, ok := m.(MsgSwap)
	if !ok {
		return nil, errInvalidMessage
	}
	if err := h.validate(ctx, msg, version); err != nil {
		ctx.Logger().Error("MsgSwap failed validation", "error", err)
		return nil, err
	}
	result, err := h.handle(ctx, msg, version, constAccessor)
	if err != nil {
		ctx.Logger().Error("fail to handle MsgSwap", "error", err)
		return nil, err
	}
	return result, err
}

func (h SwapHandler) validate(ctx cosmos.Context, msg MsgSwap, version semver.Version) error {
	if version.GTE(semver.MustParse("0.1.0")) {
		return h.validateV1(ctx, msg)
	}
	return errInvalidVersion
}

func (h SwapHandler) validateV1(ctx cosmos.Context, msg MsgSwap) error {
	return msg.ValidateBasic()
}

func (h SwapHandler) handle(ctx cosmos.Context, msg MsgSwap, version semver.Version, constAccessor constants.ConstantValues) (*cosmos.Result, error) {
	ctx.Logger().Info("receive MsgSwap", "request tx hash", msg.Tx.ID, "source asset", msg.Tx.Coins[0].Asset, "target asset", msg.TargetAsset, "signer", msg.Signer.String())
	if version.GTE(semver.MustParse("0.1.0")) {
		return h.handleV1(ctx, msg, version, constAccessor)
	}
	return nil, errBadVersion
}

func (h SwapHandler) handleV1(ctx cosmos.Context, msg MsgSwap, version semver.Version, constAccessor constants.ConstantValues) (*cosmos.Result, error) {
	transactionFee := constAccessor.GetInt64Value(constants.TransactionFee)
	_, _, swapErr := swap(
		ctx,
		h.keeper,
		msg.Tx,
		msg.TargetAsset,
		msg.Destination,
		msg.TradeTarget,
		cosmos.NewUint(uint64(transactionFee)), h.mgr)
	if swapErr != nil {
		return nil, swapErr
	}

	return &cosmos.Result{}, nil
}
