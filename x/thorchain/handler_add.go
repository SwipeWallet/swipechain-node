package thorchain

import (
	"fmt"

	"github.com/blang/semver"

	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
	"gitlab.com/thorchain/thornode/x/thorchain/keeper"
)

// AddHandler is to handle Add message
type AddHandler struct {
	keeper keeper.Keeper
	mgr    Manager
}

// NewAddHandler create a new instance of AddHandler
func NewAddHandler(keeper keeper.Keeper, mgr Manager) AddHandler {
	return AddHandler{
		keeper: keeper,
		mgr:    mgr,
	}
}

// Run is the main entry point to execute Add logic
func (h AddHandler) Run(ctx cosmos.Context, m cosmos.Msg, version semver.Version, _ constants.ConstantValues) (*cosmos.Result, error) {
	msg, ok := m.(MsgAdd)
	if !ok {
		return nil, errInvalidMessage
	}
	ctx.Logger().Info(fmt.Sprintf("receive msg add %s", msg.Tx.ID))
	if err := h.validate(ctx, msg, version); err != nil {
		ctx.Logger().Error("msg add failed validation", "error", err)
		return nil, err
	}
	return h.handle(ctx, msg, version)
}

func (h AddHandler) validate(ctx cosmos.Context, msg MsgAdd, version semver.Version) error {
	if version.GTE(semver.MustParse("0.1.0")) {
		return h.validateV1(ctx, msg)
	}
	return errBadVersion
}

func (h AddHandler) validateV1(ctx cosmos.Context, msg MsgAdd) error {
	if err := msg.ValidateBasic(); err != nil {
		return err
	}
	return nil
}

// handle process MsgAdd, MsgAdd add asset and RUNE to the asset pool
// it simply increase the pool asset/RUNE balance but without taking any of the pool units
func (h AddHandler) handle(ctx cosmos.Context, msg MsgAdd, version semver.Version) (*cosmos.Result, error) {
	pool, err := h.keeper.GetPool(ctx, msg.Asset)
	if err != nil {
		return nil, ErrInternal(err, fmt.Sprintf("fail to get pool for (%s)", msg.Asset))
	}
	if pool.Asset.IsEmpty() {
		return nil, cosmos.ErrUnknownRequest(fmt.Sprintf("pool %s not exist", msg.Asset.String()))
	}
	pool.BalanceAsset = pool.BalanceAsset.Add(msg.AssetAmount)
	pool.BalanceRune = pool.BalanceRune.Add(msg.RuneAmount)

	if err := h.keeper.SetPool(ctx, pool); err != nil {
		return nil, ErrInternal(err, fmt.Sprintf("fail to set pool(%s)", pool))
	}
	// emit event
	addEvt := NewEventAdd(pool.Asset, msg.Tx)
	if err := h.mgr.EventMgr().EmitEvent(ctx, addEvt); err != nil {
		return nil, cosmos.Wrapf(errFailSaveEvent, "fail to save add events: %w", err)
	}
	return &cosmos.Result{}, nil
}
