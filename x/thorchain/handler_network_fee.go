package thorchain

import (
	"github.com/blang/semver"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
	"gitlab.com/thorchain/thornode/x/thorchain/keeper"
)

// NetworkFeeHandler a handler to process MsgNetworkFee messages
type NetworkFeeHandler struct {
	keeper keeper.Keeper
	mgr    Manager
}

// NewNetworkFeeHandler create a new instance of network fee handler
func NewNetworkFeeHandler(keeper keeper.Keeper, mgr Manager) NetworkFeeHandler {
	return NetworkFeeHandler{keeper: keeper, mgr: mgr}
}

// Run is the main entry point for network fee logic
func (h NetworkFeeHandler) Run(ctx cosmos.Context, m cosmos.Msg, version semver.Version, constAccessor constants.ConstantValues) (*cosmos.Result, error) {
	msg, ok := m.(MsgNetworkFee)
	if !ok {
		return nil, errInvalidMessage
	}
	ctx.Logger().Info("receive msg network fee")
	if err := h.validate(ctx, msg, version); err != nil {
		ctx.Logger().Error("MsgNetworkFee failed validation", "error", err)
		return nil, err
	}
	result, err := h.handle(ctx, msg, version, constAccessor)
	if err != nil {
		ctx.Logger().Error("fail to process MsgNetworkFee", "error", err)
	}
	return result, err
}

func (h NetworkFeeHandler) validate(ctx cosmos.Context, msg MsgNetworkFee, version semver.Version) error {
	if version.GTE(semver.MustParse("0.1.0")) {
		return h.validateV1(ctx, msg)
	}
	return errBadVersion
}

func (h NetworkFeeHandler) validateV1(ctx cosmos.Context, msg MsgNetworkFee) error {
	if err := msg.ValidateBasic(); err != nil {
		return err
	}
	if !isSignedByActiveNodeAccounts(ctx, h.keeper, msg.GetSigners()) {
		return cosmos.ErrUnauthorized(notAuthorized.Error())
	}
	return nil
}

// handle process MsgNetworkFee
func (h NetworkFeeHandler) handle(ctx cosmos.Context, msg MsgNetworkFee, version semver.Version, constAccessor constants.ConstantValues) (*cosmos.Result, error) {
	ctx.Logger().Info("handle MsgNetworkFee request", "chain", msg.Chain, "block height", msg.BlockHeight)
	if version.GTE(semver.MustParse("0.13.0")) {
		return h.handleV13(ctx, msg, version, constAccessor)
	} else if version.GTE(semver.MustParse("0.1.0")) {
		return h.handleV1(ctx, msg, version, constAccessor)
	}
	return nil, errBadVersion
}

//  handleV1 process MsgNetworkFee
func (h NetworkFeeHandler) handleV1(ctx cosmos.Context, msg MsgNetworkFee, version semver.Version, constAccessor constants.ConstantValues) (*cosmos.Result, error) {
	active, err := h.keeper.ListActiveNodeAccounts(ctx)
	if err != nil {
		err = wrapError(ctx, err, "fail to get list of active node accounts")
		return nil, err
	}

	voter, err := h.keeper.GetObservedNetworkFeeVoter(ctx, msg.BlockHeight, msg.Chain)
	if err != nil {
		return nil, err
	}
	observeSlashPoints := constAccessor.GetInt64Value(constants.ObserveSlashPoints)
	observeFlex := constAccessor.GetInt64Value(constants.ObserveFlex)
	h.mgr.Slasher().IncSlashPoints(ctx, observeSlashPoints, msg.Signer)
	if !voter.Sign(msg.Signer) {
		ctx.Logger().Info("signer already signed MsgNetworkFee", "signer", msg.Signer.String(), "block height", msg.BlockHeight, "chain", msg.Chain.String())
		return &cosmos.Result{}, nil
	}
	h.keeper.SetObservedNetworkFeeVoter(ctx, voter)
	// doesn't have consensus yet
	if !voter.HasConsensus(active) {
		ctx.Logger().Info("not having consensus yet, return")
		return &cosmos.Result{}, nil
	}

	if voter.BlockHeight > 0 {
		if (voter.BlockHeight + observeFlex) >= common.BlockHeight(ctx) {
			h.mgr.Slasher().DecSlashPoints(ctx, observeSlashPoints, msg.Signer)
		}
		// MsgNetworkFee tx already processed
		return &cosmos.Result{}, nil
	}

	voter.BlockHeight = common.BlockHeight(ctx)
	h.keeper.SetObservedNetworkFeeVoter(ctx, voter)
	// decrease the slash points
	h.mgr.Slasher().DecSlashPoints(ctx, observeSlashPoints, voter.Signers...)
	ctx.Logger().Info("update network fee", "chain", msg.Chain.String(), "transaction-size", msg.TransactionSize, "fee-rate", msg.TransactionFeeRate.String())
	if err := h.keeper.SaveNetworkFee(ctx, msg.Chain, NetworkFee{
		Chain:              msg.Chain,
		TransactionSize:    msg.TransactionSize,
		TransactionFeeRate: msg.TransactionFeeRate,
	}); err != nil {
		return nil, ErrInternal(err, "fail to save network fee")
	}
	return &cosmos.Result{}, nil
}

//  handleV13 process MsgNetworkFee
func (h NetworkFeeHandler) handleV13(ctx cosmos.Context, msg MsgNetworkFee, version semver.Version, constAccessor constants.ConstantValues) (*cosmos.Result, error) {
	active, err := h.keeper.ListActiveNodeAccounts(ctx)
	if err != nil {
		err = wrapError(ctx, err, "fail to get list of active node accounts")
		return nil, err
	}

	voter, err := h.keeper.GetObservedNetworkFeeVoter(ctx, msg.BlockHeight, msg.Chain)
	if err != nil {
		return nil, err
	}
	observeSlashPoints := constAccessor.GetInt64Value(constants.ObserveSlashPoints)
	observeFlex := constAccessor.GetInt64Value(constants.ObserveFlex)
	h.mgr.Slasher().IncSlashPoints(ctx, observeSlashPoints, msg.Signer)
	if !voter.Sign(msg.Signer) {
		ctx.Logger().Info("signer already signed MsgNetworkFee", "signer", msg.Signer.String(), "block height", msg.BlockHeight, "chain", msg.Chain.String())
		return &cosmos.Result{}, nil
	}
	h.keeper.SetObservedNetworkFeeVoter(ctx, voter)
	// doesn't have consensus yet
	if !voter.HasConsensusV13(active) {
		ctx.Logger().Info("not having consensus yet, return")
		return &cosmos.Result{}, nil
	}

	if voter.BlockHeight > 0 {
		if (voter.BlockHeight + observeFlex) >= common.BlockHeight(ctx) {
			h.mgr.Slasher().DecSlashPoints(ctx, observeSlashPoints, msg.Signer)
		}
		// MsgNetworkFee tx already processed
		return &cosmos.Result{}, nil
	}

	voter.BlockHeight = common.BlockHeight(ctx)
	h.keeper.SetObservedNetworkFeeVoter(ctx, voter)
	// decrease the slash points
	h.mgr.Slasher().DecSlashPoints(ctx, observeSlashPoints, voter.Signers...)
	ctx.Logger().Info("update network fee", "chain", msg.Chain.String(), "transaction-size", msg.TransactionSize, "fee-rate", msg.TransactionFeeRate.String())
	if err := h.keeper.SaveNetworkFee(ctx, msg.Chain, NetworkFee{
		Chain:              msg.Chain,
		TransactionSize:    msg.TransactionSize,
		TransactionFeeRate: msg.TransactionFeeRate,
	}); err != nil {
		return nil, ErrInternal(err, "fail to save network fee")
	}
	return &cosmos.Result{}, nil
}
