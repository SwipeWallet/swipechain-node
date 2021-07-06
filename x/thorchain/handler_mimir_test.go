package thorchain

import (
	"github.com/blang/semver"
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
)

type HandlerMimirSuite struct{}

var _ = Suite(&HandlerMimirSuite{})

func (s *HandlerMimirSuite) SetUpSuite(c *C) {
	SetupConfigForTest()
}

func (s *HandlerMimirSuite) TestValidate(c *C) {
	ctx, keeper := setupKeeperForTest(c)

	addr, _ := cosmos.AccAddressFromBech32(ADMINS[0])
	handler := NewMimirHandler(keeper, NewDummyMgr())
	// happy path
	ver := constants.SWVersion
	msg := NewMsgMimir("foo", 44, addr)
	err := handler.validate(ctx, msg, ver)
	c.Assert(err, IsNil)

	// invalid version
	err = handler.validate(ctx, msg, semver.Version{})
	c.Assert(err, Equals, errBadVersion)

	// invalid msg
	msg = MsgMimir{}
	err = handler.validate(ctx, msg, ver)
	c.Assert(err, NotNil)
}

func (s *HandlerMimirSuite) TestHandle(c *C) {
	ctx, keeper := setupKeeperForTest(c)
	ver := constants.SWVersion

	handler := NewMimirHandler(keeper, NewDummyMgr())

	msg := NewMsgMimir("foo", 55, GetRandomBech32Addr())
	sdkErr := handler.handle(ctx, msg, ver)
	c.Assert(sdkErr, IsNil)
	val, err := keeper.GetMimir(ctx, "foo")
	c.Assert(err, IsNil)
	c.Check(val, Equals, int64(55))

	invalidMsg := NewMsgNetworkFee(ctx.BlockHeight(), common.BNBChain, 1, bnbSingleTxFee, GetRandomBech32Addr())
	result, err := handler.Run(ctx, invalidMsg, constants.SWVersion, constants.GetConstantValues(constants.SWVersion))
	c.Check(err, NotNil)
	c.Check(result, IsNil)

	result, err = handler.Run(ctx, msg, constants.SWVersion, constants.GetConstantValues(constants.SWVersion))
	c.Check(err, NotNil)
	c.Check(result, IsNil)
	addr, err := cosmos.AccAddressFromBech32(ADMINS[0])
	c.Check(err, IsNil)
	msg1 := NewMsgMimir("hello", 1, addr)
	result, err = handler.Run(ctx, msg1, constants.SWVersion, constants.GetConstantValues(constants.SWVersion))
	c.Check(err, IsNil)
	c.Check(result, NotNil)

	// invalid version should result an error
	c.Check(handler.handle(ctx, msg, semver.MustParse("0.0.1")), NotNil)
}
