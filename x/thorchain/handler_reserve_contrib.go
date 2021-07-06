package thorchain

import (
	"fmt"

	"github.com/blang/semver"

	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
	"gitlab.com/thorchain/thornode/x/thorchain/keeper"
)

// ReserveContributorHandler is handler to process MsgReserveContributor
type ReserveContributorHandler struct {
	keeper keeper.Keeper
	mgr    Manager
}

// NewReserveContributorHandler create a new instance of ReserveContributorHandler
func NewReserveContributorHandler(keeper keeper.Keeper, mgr Manager) ReserveContributorHandler {
	return ReserveContributorHandler{
		keeper: keeper,
		mgr:    mgr,
	}
}

// Run is the main entry point for ReserveContributorHandler
func (h ReserveContributorHandler) Run(ctx cosmos.Context, m cosmos.Msg, version semver.Version, _ constants.ConstantValues) (*cosmos.Result, error) {
	msg, ok := m.(MsgReserveContributor)
	if !ok {
		return nil, errInvalidMessage
	}
	if err := h.validate(ctx, msg, version); err != nil {
		ctx.Logger().Error("MsgReserveContributor failed validation", "error", err)
		return nil, err
	}
	result, err := h.handle(ctx, msg, version)
	if err != nil {
		ctx.Logger().Error("fail to process MsgReserveContributor", "error", err)
	}
	return result, err
}

func (h ReserveContributorHandler) validate(ctx cosmos.Context, msg MsgReserveContributor, version semver.Version) error {
	if version.GTE(semver.MustParse("0.1.0")) {
		return h.validateV1(ctx, msg)
	}
	return errBadVersion
}

func (h ReserveContributorHandler) validateV1(ctx cosmos.Context, msg MsgReserveContributor) error {
	return msg.ValidateBasic()
}

func (h ReserveContributorHandler) handle(ctx cosmos.Context, msg MsgReserveContributor, version semver.Version) (*cosmos.Result, error) {
	ctx.Logger().Info("handleMsgReserveContributor request")
	if version.GTE(semver.MustParse("0.1.0")) {
		if err := h.HandleV1(ctx, msg, version); err != nil {
			return nil, ErrInternal(err, "fail to process reserve contributor")
		}
		return &cosmos.Result{}, nil
	}
	return nil, errBadVersion
}

// HandleV1  process MsgReserveContributor
func (h ReserveContributorHandler) HandleV1(ctx cosmos.Context, msg MsgReserveContributor, version semver.Version) error {
	reses, err := h.keeper.GetReservesContributors(ctx)
	if err != nil {
		return err
	}

	reses = reses.Add(msg.Contributor)
	if err := h.keeper.SetReserveContributors(ctx, reses); err != nil {
		return fmt.Errorf("fail to save reserve contributors: %w", err)
	}

	vault, err := h.keeper.GetVaultData(ctx)
	if err != nil {
		return fmt.Errorf("fail to get vault data: %w", err)
	}

	vault.TotalReserve = vault.TotalReserve.Add(msg.Contributor.Amount)
	if err := h.keeper.SetVaultData(ctx, vault); err != nil {
		return fmt.Errorf("fail to save vault data: %w", err)
	}
	reserveEvent := NewEventReserve(msg.Contributor, msg.Tx)
	if err := h.mgr.EventMgr().EmitEvent(ctx, reserveEvent); err != nil {
		return fmt.Errorf("fail to emit reserve event: %w", err)
	}
	return nil
}
