package thorchain

import (
	"errors"

	"github.com/blang/semver"
	se "github.com/cosmos/cosmos-sdk/types/errors"
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
	keeper "gitlab.com/thorchain/thornode/x/thorchain/keeper"
)

type HandlerReserveContributorSuite struct{}

var _ = Suite(&HandlerReserveContributorSuite{})

type reserveContributorKeeper struct {
	keeper.Keeper
	errGetReserveContributors bool
	errSetReserveContributors bool
	errGetVaultData           bool
	errSetVaultData           bool
}

func newReserveContributorKeeper(k keeper.Keeper) *reserveContributorKeeper {
	return &reserveContributorKeeper{
		Keeper: k,
	}
}

func (k *reserveContributorKeeper) GetReservesContributors(ctx cosmos.Context) (ReserveContributors, error) {
	if k.errGetReserveContributors {
		return ReserveContributors{}, kaboom
	}
	return k.Keeper.GetReservesContributors(ctx)
}

func (k *reserveContributorKeeper) SetReserveContributors(ctx cosmos.Context, contributors ReserveContributors) error {
	if k.errSetReserveContributors {
		return kaboom
	}
	return k.Keeper.SetReserveContributors(ctx, contributors)
}

func (k *reserveContributorKeeper) GetVaultData(ctx cosmos.Context) (VaultData, error) {
	if k.errGetVaultData {
		return VaultData{}, kaboom
	}
	return k.Keeper.GetVaultData(ctx)
}

func (k *reserveContributorKeeper) SetVaultData(ctx cosmos.Context, data VaultData) error {
	if k.errSetVaultData {
		return kaboom
	}
	return k.Keeper.SetVaultData(ctx, data)
}

type reserveContributorHandlerHelper struct {
	ctx                cosmos.Context
	version            semver.Version
	keeper             *reserveContributorKeeper
	nodeAccount        NodeAccount
	constAccessor      constants.ConstantValues
	reserveContributor ReserveContributor
}

func newReserveContributorHandlerHelper(c *C) reserveContributorHandlerHelper {
	ctx, k := setupKeeperForTest(c)
	ctx = ctx.WithBlockHeight(1023)

	version := constants.SWVersion
	keeper := newReserveContributorKeeper(k)

	// active account
	nodeAccount := GetRandomNodeAccount(NodeActive)
	nodeAccount.Bond = cosmos.NewUint(100 * common.One)
	c.Assert(keeper.SetNodeAccount(ctx, nodeAccount), IsNil)
	constAccessor := constants.GetConstantValues(version)

	reserveContributor := ReserveContributor{
		Address: GetRandomBNBAddress(),
		Amount:  cosmos.NewUint(100 * common.One),
	}
	return reserveContributorHandlerHelper{
		ctx:                ctx,
		version:            version,
		keeper:             keeper,
		nodeAccount:        nodeAccount,
		constAccessor:      constAccessor,
		reserveContributor: reserveContributor,
	}
}

func (h HandlerReserveContributorSuite) TestReserveContributorHandler(c *C) {
	testCases := []struct {
		name           string
		messageCreator func(helper reserveContributorHandlerHelper) cosmos.Msg
		runner         func(handler ReserveContributorHandler, helper reserveContributorHandlerHelper, msg cosmos.Msg) (*cosmos.Result, error)
		expectedResult error
		validator      func(helper reserveContributorHandlerHelper, msg cosmos.Msg, result *cosmos.Result, c *C)
	}{
		{
			name: "invalid message should return error",
			messageCreator: func(helper reserveContributorHandlerHelper) cosmos.Msg {
				return NewMsgNoOp(GetRandomObservedTx(), helper.nodeAccount.NodeAddress)
			},
			runner: func(handler ReserveContributorHandler, helper reserveContributorHandlerHelper, msg cosmos.Msg) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, helper.version, helper.constAccessor)
			},
			expectedResult: errInvalidMessage,
		},
		{
			name: "bad version should return an error",
			messageCreator: func(helper reserveContributorHandlerHelper) cosmos.Msg {
				return NewMsgReserveContributor(GetRandomTx(), helper.reserveContributor, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler ReserveContributorHandler, helper reserveContributorHandlerHelper, msg cosmos.Msg) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, semver.MustParse("0.0.1"), helper.constAccessor)
			},
			expectedResult: errBadVersion,
		},
		{
			name: "empty signer should return an error",
			messageCreator: func(helper reserveContributorHandlerHelper) cosmos.Msg {
				return NewMsgReserveContributor(GetRandomTx(), helper.reserveContributor, cosmos.AccAddress{})
			},
			runner: func(handler ReserveContributorHandler, helper reserveContributorHandlerHelper, msg cosmos.Msg) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, constants.SWVersion, helper.constAccessor)
			},
			expectedResult: se.ErrInvalidAddress,
		},
		{
			name: "empty contributor address should return an error",
			messageCreator: func(helper reserveContributorHandlerHelper) cosmos.Msg {
				return NewMsgReserveContributor(GetRandomTx(), ReserveContributor{
					Address: common.NoAddress,
					Amount:  cosmos.NewUint(100),
				}, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler ReserveContributorHandler, helper reserveContributorHandlerHelper, msg cosmos.Msg) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, constants.SWVersion, helper.constAccessor)
			},
			expectedResult: se.ErrUnknownRequest,
		},
		{
			name: "empty contributor amount should return an error",
			messageCreator: func(helper reserveContributorHandlerHelper) cosmos.Msg {
				return NewMsgReserveContributor(GetRandomTx(), ReserveContributor{
					Address: GetRandomBNBAddress(),
					Amount:  cosmos.ZeroUint(),
				}, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler ReserveContributorHandler, helper reserveContributorHandlerHelper, msg cosmos.Msg) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, constants.SWVersion, helper.constAccessor)
			},
			expectedResult: se.ErrUnknownRequest,
		},
		{
			name: "invalid tx should return an error",
			messageCreator: func(helper reserveContributorHandlerHelper) cosmos.Msg {
				tx := GetRandomTx()
				tx.ID = ""
				return NewMsgReserveContributor(tx, helper.reserveContributor, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler ReserveContributorHandler, helper reserveContributorHandlerHelper, msg cosmos.Msg) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, constants.SWVersion, helper.constAccessor)
			},
			expectedResult: se.ErrUnknownRequest,
		},
		{
			name: "fail to get reserve contributor should return an error",
			messageCreator: func(helper reserveContributorHandlerHelper) cosmos.Msg {
				return NewMsgReserveContributor(GetRandomTx(), helper.reserveContributor, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler ReserveContributorHandler, helper reserveContributorHandlerHelper, msg cosmos.Msg) (*cosmos.Result, error) {
				helper.keeper.errGetReserveContributors = true
				return handler.Run(helper.ctx, msg, constants.SWVersion, helper.constAccessor)
			},
			expectedResult: errInternal,
		},
		{
			name: "fail to set reserve contributor should return an error",
			messageCreator: func(helper reserveContributorHandlerHelper) cosmos.Msg {
				return NewMsgReserveContributor(GetRandomTx(), helper.reserveContributor, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler ReserveContributorHandler, helper reserveContributorHandlerHelper, msg cosmos.Msg) (*cosmos.Result, error) {
				helper.keeper.errSetReserveContributors = true
				return handler.Run(helper.ctx, msg, constants.SWVersion, helper.constAccessor)
			},
			expectedResult: errInternal,
		},
		{
			name: "fail to get vault data should return an error",
			messageCreator: func(helper reserveContributorHandlerHelper) cosmos.Msg {
				return NewMsgReserveContributor(GetRandomTx(), helper.reserveContributor, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler ReserveContributorHandler, helper reserveContributorHandlerHelper, msg cosmos.Msg) (*cosmos.Result, error) {
				helper.keeper.errGetVaultData = true
				return handler.Run(helper.ctx, msg, constants.SWVersion, helper.constAccessor)
			},
			expectedResult: errInternal,
		},
		{
			name: "fail to set vault data should return an error",
			messageCreator: func(helper reserveContributorHandlerHelper) cosmos.Msg {
				return NewMsgReserveContributor(GetRandomTx(), helper.reserveContributor, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler ReserveContributorHandler, helper reserveContributorHandlerHelper, msg cosmos.Msg) (*cosmos.Result, error) {
				helper.keeper.errSetVaultData = true
				return handler.Run(helper.ctx, msg, constants.SWVersion, helper.constAccessor)
			},
			expectedResult: errInternal,
		},
		{
			name: "normal reserve contribute message should return success",
			messageCreator: func(helper reserveContributorHandlerHelper) cosmos.Msg {
				return NewMsgReserveContributor(GetRandomTx(), helper.reserveContributor, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler ReserveContributorHandler, helper reserveContributorHandlerHelper, msg cosmos.Msg) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, constants.SWVersion, helper.constAccessor)
			},
			expectedResult: nil,
		},
	}
	for _, tc := range testCases {
		helper := newReserveContributorHandlerHelper(c)
		mgr := NewManagers(helper.keeper)
		c.Assert(mgr.BeginBlock(helper.ctx), IsNil)
		handler := NewReserveContributorHandler(helper.keeper, mgr)
		msg := tc.messageCreator(helper)
		result, err := tc.runner(handler, helper, msg)
		if tc.expectedResult == nil {
			c.Check(err, IsNil)
		} else {
			c.Check(errors.Is(err, tc.expectedResult), Equals, true, Commentf("name:%s", tc.name))
		}
		if tc.validator != nil {
			tc.validator(helper, msg, result, c)
		}
	}
}
