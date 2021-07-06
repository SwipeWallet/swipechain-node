package thorchain

import (
	"fmt"

	"github.com/blang/semver"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
	"gitlab.com/thorchain/thornode/x/thorchain/keeper"
)

// VersionHandler is to handle Version message
type VersionHandler struct {
	keeper keeper.Keeper
	mgr    Manager
}

// NewVersionHandler create new instance of VersionHandler
func NewVersionHandler(keeper keeper.Keeper, mgr Manager) VersionHandler {
	return VersionHandler{
		keeper: keeper,
		mgr:    mgr,
	}
}

// Run it the main entry point to execute Version logic
func (h VersionHandler) Run(ctx cosmos.Context, m cosmos.Msg, version semver.Version, constAccessor constants.ConstantValues) (*cosmos.Result, error) {
	msg, ok := m.(MsgSetVersion)
	if !ok {
		return nil, errInvalidMessage
	}
	ctx.Logger().Info("receive version number",
		"version", msg.Version.String())
	if err := h.validate(ctx, msg, version, constAccessor); err != nil {
		ctx.Logger().Error("msg set version failed validation", "error", err)
		return nil, err
	}
	if err := h.handle(ctx, msg, version, constAccessor); err != nil {
		ctx.Logger().Error("fail to process msg set version", "error", err)
		return nil, err
	}

	return &cosmos.Result{}, nil
}

func (h VersionHandler) validate(ctx cosmos.Context, msg MsgSetVersion, version semver.Version, constAccessor constants.ConstantValues) error {
	if version.GTE(semver.MustParse("0.1.0")) {
		return h.validateV1(ctx, msg, constAccessor)
	}
	return errBadVersion
}

func (h VersionHandler) validateV1(ctx cosmos.Context, msg MsgSetVersion, constAccessor constants.ConstantValues) error {
	if err := msg.ValidateBasic(); err != nil {
		return err
	}

	nodeAccount, err := h.keeper.GetNodeAccount(ctx, msg.Signer)
	if err != nil {
		return cosmos.ErrUnauthorized(fmt.Sprintf("%s is not authorizaed", msg.Signer))
	}
	if nodeAccount.IsEmpty() {
		return cosmos.ErrUnauthorized(fmt.Sprintf("%s is not authorizaed", msg.Signer))
	}

	cost := constAccessor.GetInt64Value(constants.CliTxCost)
	if nodeAccount.Bond.LT(cosmos.NewUint(uint64(cost))) {
		return cosmos.ErrUnauthorized("not enough bond")
	}

	return nil
}

func (h VersionHandler) handle(ctx cosmos.Context, msg MsgSetVersion, version semver.Version, constAccessor constants.ConstantValues) error {
	ctx.Logger().Info("handleMsgSetVersion request", "Version:", msg.Version.String())
	if version.GTE(semver.MustParse("0.1.0")) {
		return h.handleV1(ctx, msg, constAccessor)
	}
	return errBadVersion
}

func (h VersionHandler) handleV1(ctx cosmos.Context, msg MsgSetVersion, constAccessor constants.ConstantValues) error {
	nodeAccount, err := h.keeper.GetNodeAccount(ctx, msg.Signer)
	if err != nil {
		return cosmos.ErrUnauthorized(fmt.Errorf("unable to find account(%s):%w", msg.Signer, err).Error())
	}

	if nodeAccount.Version.LT(msg.Version) {
		nodeAccount.Version = msg.Version
	}

	cost := cosmos.NewUint(uint64(constAccessor.GetInt64Value(constants.CliTxCost)))
	if cost.GT(nodeAccount.Bond) {
		cost = nodeAccount.Bond
	}

	nodeAccount.Bond = common.SafeSub(nodeAccount.Bond, cost)
	if err := h.keeper.SetNodeAccount(ctx, nodeAccount); err != nil {
		return fmt.Errorf("fail to save node account: %w", err)
	}

	// add bond to reserve
	if common.RuneAsset().Chain.Equals(common.THORChain) {
		coin := common.NewCoin(common.RuneNative, cost)
		if err := h.keeper.SendFromAccountToModule(ctx, msg.Signer, ReserveName, coin); err != nil {
			ctx.Logger().Error("fail to transfer funds from bond to reserve", "error", err)
			return err
		}
	} else {
		vaultData, err := h.keeper.GetVaultData(ctx)
		if err != nil {
			return fmt.Errorf("fail to get vault data: %w", err)
		}
		vaultData.TotalReserve = vaultData.TotalReserve.Add(cost)
		if err := h.keeper.SetVaultData(ctx, vaultData); err != nil {
			return fmt.Errorf("fail to save vault data: %w", err)
		}
	}

	ctx.EventManager().EmitEvent(
		cosmos.NewEvent("set_version",
			cosmos.NewAttribute("thor_address", msg.Signer.String()),
			cosmos.NewAttribute("version", msg.Version.String())))

	return nil
}
