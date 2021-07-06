package thorchain

import (
	"fmt"
	"strconv"

	"github.com/blang/semver"

	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
	"gitlab.com/thorchain/thornode/x/thorchain/keeper"
)

// MimirHandler is to handle admin messages
type MimirHandler struct {
	keeper keeper.Keeper
	mgr    Manager
}

// NewMimirHandler create new instance of MimirHandler
func NewMimirHandler(keeper keeper.Keeper, mgr Manager) MimirHandler {
	return MimirHandler{
		keeper: keeper,
		mgr:    mgr,
	}
}

// Run is the main entry point to execute mimir logic
func (h MimirHandler) Run(ctx cosmos.Context, m cosmos.Msg, version semver.Version, _ constants.ConstantValues) (*cosmos.Result, error) {
	msg, ok := m.(MsgMimir)
	if !ok {
		return nil, errInvalidMessage
	}
	ctx.Logger().Info("receive mimir", "key", msg.Key, "value", msg.Value)
	if err := h.validate(ctx, msg, version); err != nil {
		ctx.Logger().Error("msg mimir failed validation", "error", err)
		return nil, err
	}
	if err := h.handle(ctx, msg, version); err != nil {
		ctx.Logger().Error("fail to process msg set mimir", "error", err)
		return nil, err
	}

	return &cosmos.Result{}, nil
}

func (h MimirHandler) validate(ctx cosmos.Context, msg MsgMimir, version semver.Version) error {
	if version.GTE(semver.MustParse("0.1.0")) {
		return h.validateV1(ctx, msg)
	}
	return errBadVersion
}

func (h MimirHandler) validateV1(ctx cosmos.Context, msg MsgMimir) error {
	if err := msg.ValidateBasic(); err != nil {
		return err
	}

	for _, admin := range ADMINS {
		addr, err := cosmos.AccAddressFromBech32(admin)
		if msg.Signer.Equals(addr) && err == nil {
			return nil
		}
	}
	return cosmos.ErrUnauthorized(fmt.Sprintf("%s is not authorizaed", msg.Signer))
}

func (h MimirHandler) handle(ctx cosmos.Context, msg MsgMimir, version semver.Version) error {
	ctx.Logger().Info("handleMsgMimir request", "key", msg.Key, "value", msg.Value)
	if version.GTE(semver.MustParse("0.1.0")) {
		return h.handleV1(ctx, msg)
	}
	ctx.Logger().Error(errInvalidVersion.Error())
	return errBadVersion
}

func (h MimirHandler) handleV1(ctx cosmos.Context, msg MsgMimir) error {
	h.keeper.SetMimir(ctx, msg.Key, msg.Value)

	ctx.EventManager().EmitEvent(
		cosmos.NewEvent("set_mimir",
			cosmos.NewAttribute("key", msg.Key),
			cosmos.NewAttribute("value", strconv.FormatInt(msg.Value, 10))))

	return nil
}
