package thorchain

import (
	"errors"

	"github.com/blang/semver"
	sdk "github.com/cosmos/cosmos-sdk/types"
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
	"gitlab.com/thorchain/thornode/x/thorchain/keeper"
)

type HandlerObserveNetworkFeeSuite struct{}

var _ = Suite(&HandlerObserveNetworkFeeSuite{})

type KeeperObserveNetworkFeeTest struct {
	keeper.Keeper
	errFailListActiveNodeAccount   bool
	errFailGetObservedNetworkVoter bool
	errFailSaveNetworkFee          bool
}

func NewKeeperObserveNetworkFeeTest(k keeper.Keeper) KeeperObserveNetworkFeeTest {
	return KeeperObserveNetworkFeeTest{Keeper: k}
}

func (k KeeperObserveNetworkFeeTest) ListActiveNodeAccounts(ctx cosmos.Context) (NodeAccounts, error) {
	if k.errFailListActiveNodeAccount {
		return NodeAccounts{}, kaboom
	}
	return k.Keeper.ListActiveNodeAccounts(ctx)
}

func (k KeeperObserveNetworkFeeTest) GetObservedNetworkFeeVoter(ctx cosmos.Context, height int64, chain common.Chain) (ObservedNetworkFeeVoter, error) {
	if k.errFailGetObservedNetworkVoter {
		return ObservedNetworkFeeVoter{}, kaboom
	}
	return k.Keeper.GetObservedNetworkFeeVoter(ctx, height, chain)
}

func (k KeeperObserveNetworkFeeTest) SaveNetworkFee(ctx cosmos.Context, chain common.Chain, networkFee NetworkFee) error {
	if k.errFailSaveNetworkFee {
		return kaboom
	}
	return k.Keeper.SaveNetworkFee(ctx, chain, networkFee)
}

func (h *HandlerObserveNetworkFeeSuite) TestHandlerObserveNetworkFee(c *C) {
	h.testHandlerObserveNetworkFeeWithVersion(c, constants.SWVersion)
	h.testHandlerObserveNetworkFeeWithVersion(c, semver.MustParse("0.13.0"))
}

func (*HandlerObserveNetworkFeeSuite) testHandlerObserveNetworkFeeWithVersion(c *C, ver semver.Version) {
	ctx, keeper := setupKeeperForTest(c)
	activeNodeAccount := GetRandomNodeAccount(NodeActive)
	c.Assert(keeper.SetNodeAccount(ctx, activeNodeAccount), IsNil)
	handler := NewNetworkFeeHandler(keeper, NewDummyMgr())
	msg := NewMsgNetworkFee(1024, common.BNBChain, 256, sdk.NewUint(100), activeNodeAccount.NodeAddress)
	constantsAccessor := constants.GetConstantValues(ver)
	result, err := handler.Run(ctx, msg, ver, constantsAccessor)
	c.Assert(err, IsNil)
	c.Assert(result, NotNil)

	// already signed not cause error
	result, err = handler.Run(ctx, msg, ver, constantsAccessor)
	c.Assert(err, IsNil)
	c.Assert(result, NotNil)

	// invalid version should return bad version
	result, err = handler.Run(ctx, msg, semver.MustParse("0.0.1"), constantsAccessor)
	c.Assert(result, IsNil)
	c.Assert(errors.Is(err, errBadVersion), Equals, true)

	// already processed
	msg1 := NewMsgNetworkFee(1024, common.BNBChain, 256, sdk.NewUint(100), activeNodeAccount.NodeAddress)
	result, err = handler.Run(ctx, msg1, ver, constantsAccessor)
	c.Assert(err, IsNil)
	c.Assert(result, NotNil)

	// fail list active node account should fail
	handler1 := NewNetworkFeeHandler(
		KeeperObserveNetworkFeeTest{
			Keeper:                       keeper,
			errFailListActiveNodeAccount: true,
		}, NewDummyMgr())
	result, err = handler1.Run(ctx, msg, ver, constantsAccessor)
	c.Assert(err, NotNil)
	c.Assert(result, IsNil)
	c.Assert(errors.Is(err, errInternal), Equals, true)

	// fail to get observed network fee voter should return an error
	handler2 := NewNetworkFeeHandler(
		KeeperObserveNetworkFeeTest{
			Keeper:                         keeper,
			errFailGetObservedNetworkVoter: true,
		}, NewDummyMgr())
	result, err = handler2.Run(ctx, msg, ver, constantsAccessor)
	c.Assert(err, NotNil)
	c.Assert(result, IsNil)

	// fail to save network fee should result in an error
	handler3 := NewNetworkFeeHandler(
		KeeperObserveNetworkFeeTest{
			Keeper:                keeper,
			errFailSaveNetworkFee: true,
		}, NewDummyMgr())
	msg2 := NewMsgNetworkFee(2056, common.BNBChain, 200, sdk.NewUint(102), activeNodeAccount.NodeAddress)
	result, err = handler3.Run(ctx, msg2, ver, constantsAccessor)
	c.Assert(err, NotNil)
	c.Assert(result, IsNil)

	// invalid message should return an error
	msg3 := NewMsgReserveContributor(GetRandomTx(), ReserveContributor{}, GetRandomBech32Addr())
	result, err = handler3.Run(ctx, msg3, ver, constantsAccessor)
	c.Check(result, IsNil)
	c.Check(err, NotNil)
	c.Check(errors.Is(err, errInvalidMessage), Equals, true)
}
