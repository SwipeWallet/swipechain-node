package thorchain

import (
	"errors"

	"github.com/blang/semver"
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
	"gitlab.com/thorchain/thornode/x/thorchain/keeper"
)

type HandlerIPAddressSuite struct{}

type TestIPAddresslKeeper struct {
	keeper.KVStoreDummy
	na NodeAccount
}

func (k *TestIPAddresslKeeper) SendFromAccountToModule(ctx cosmos.Context, from cosmos.AccAddress, to string, coin common.Coin) error {
	return nil
}

func (k *TestIPAddresslKeeper) GetNodeAccount(_ cosmos.Context, _ cosmos.AccAddress) (NodeAccount, error) {
	return k.na, nil
}

func (k *TestIPAddresslKeeper) SetNodeAccount(_ cosmos.Context, na NodeAccount) error {
	k.na = na
	return nil
}

func (k *TestIPAddresslKeeper) GetVaultData(ctx cosmos.Context) (VaultData, error) {
	return NewVaultData(), nil
}

func (k *TestIPAddresslKeeper) SetVaultData(ctx cosmos.Context, data VaultData) error {
	return nil
}

var _ = Suite(&HandlerIPAddressSuite{})

func (s *HandlerIPAddressSuite) TestValidate(c *C) {
	ctx, _ := setupKeeperForTest(c)

	keeper := &TestIPAddresslKeeper{
		na: GetRandomNodeAccount(NodeActive),
	}

	handler := NewIPAddressHandler(keeper, NewDummyMgr())
	// happy path
	ver := constants.SWVersion
	constAccessor := constants.GetConstantValues(ver)
	msg := NewMsgSetIPAddress("8.8.8.8", keeper.na.NodeAddress)
	err := handler.validate(ctx, msg, ver, constAccessor)
	c.Assert(err, IsNil)

	// invalid version
	err = handler.validate(ctx, msg, semver.Version{}, constAccessor)
	c.Assert(err, Equals, errBadVersion)

	// invalid msg
	msg = MsgSetIPAddress{}
	err = handler.validate(ctx, msg, ver, constAccessor)
	c.Assert(err, NotNil)
}

func (s *HandlerIPAddressSuite) TestHandle(c *C) {
	ctx, _ := setupKeeperForTest(c)
	ver := constants.SWVersion
	constAccessor := constants.GetConstantValues(ver)

	keeper := &TestIPAddresslKeeper{
		na: GetRandomNodeAccount(NodeActive),
	}

	handler := NewIPAddressHandler(keeper, NewDummyMgr())

	msg := NewMsgSetIPAddress("192.168.0.1", GetRandomBech32Addr())
	err := handler.handle(ctx, msg, ver, constAccessor)
	c.Assert(err, IsNil)
	c.Check(keeper.na.IPAddress, Equals, "192.168.0.1")

	err1 := handler.handle(ctx, msg, semver.MustParse("0.0.1"), constAccessor)
	c.Check(err1, NotNil)
	c.Check(errors.Is(err1, errBadVersion), Equals, true)
}

type HandlerIPAddressTestHelper struct {
	keeper.Keeper
	failGetNodeAccount  bool
	failSaveNodeAccount bool
}

func NewHandlerIPAddressTestHelper(k keeper.Keeper) *HandlerIPAddressTestHelper {
	return &HandlerIPAddressTestHelper{
		Keeper: k,
	}
}

func (h *HandlerIPAddressTestHelper) GetNodeAccount(ctx cosmos.Context, addr cosmos.AccAddress) (NodeAccount, error) {
	if h.failGetNodeAccount {
		return NodeAccount{}, kaboom
	}
	return h.Keeper.GetNodeAccount(ctx, addr)
}

func (h *HandlerIPAddressTestHelper) SetNodeAccount(ctx cosmos.Context, na NodeAccount) error {
	if h.failSaveNodeAccount {
		return kaboom
	}
	return h.Keeper.SetNodeAccount(ctx, na)
}

func (s *HandlerIPAddressSuite) TestHandlerSetIPAddress_validation(c *C) {
	testCases := []struct {
		name            string
		messageProvider func(ctx cosmos.Context, helper *HandlerIPAddressTestHelper) cosmos.Msg
		validator       func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *HandlerIPAddressTestHelper, name string)
	}{
		{
			name: "invalid message should return an error",
			messageProvider: func(ctx cosmos.Context, helper *HandlerIPAddressTestHelper) cosmos.Msg {
				return NewMsgNetworkFee(1024, common.BTCChain, 1, bnbSingleTxFee, GetRandomBech32Addr())
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *HandlerIPAddressTestHelper, name string) {
				c.Assert(err, NotNil)
				c.Assert(result, IsNil)
			},
		},
		{
			name: "message fail validation should return an error",
			messageProvider: func(ctx cosmos.Context, helper *HandlerIPAddressTestHelper) cosmos.Msg {
				return NewMsgSetIPAddress("whatever", GetRandomBech32Addr())
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *HandlerIPAddressTestHelper, name string) {
				c.Assert(err, NotNil)
				c.Assert(result, IsNil)
			},
		},
		{
			name: "fail to get node account should return an error",
			messageProvider: func(ctx cosmos.Context, helper *HandlerIPAddressTestHelper) cosmos.Msg {
				helper.failGetNodeAccount = true
				return NewMsgSetIPAddress("192.168.0.1", GetRandomBech32Addr())
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *HandlerIPAddressTestHelper, name string) {
				c.Assert(err, NotNil)
				c.Assert(result, IsNil)
			},
		},
		{
			name: "empty node account should return an error",
			messageProvider: func(ctx cosmos.Context, helper *HandlerIPAddressTestHelper) cosmos.Msg {
				return NewMsgSetIPAddress("192.168.0.1", GetRandomBech32Addr())
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *HandlerIPAddressTestHelper, name string) {
				c.Assert(err, NotNil)
				c.Assert(result, IsNil)
			},
		},
		{
			name: "fail to save node account should return an error",
			messageProvider: func(ctx cosmos.Context, helper *HandlerIPAddressTestHelper) cosmos.Msg {
				helper.failSaveNodeAccount = true
				nodeAccount := GetRandomNodeAccount(NodeWhiteListed)
				helper.Keeper.SetNodeAccount(ctx, nodeAccount)
				return NewMsgSetIPAddress("192.168.0.1", nodeAccount.NodeAddress)
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *HandlerIPAddressTestHelper, name string) {
				c.Assert(err, NotNil)
				c.Assert(result, IsNil)
			},
		},
		{
			name: "all good - happy path",
			messageProvider: func(ctx cosmos.Context, helper *HandlerIPAddressTestHelper) cosmos.Msg {
				nodeAccount := GetRandomNodeAccount(NodeWhiteListed)
				helper.Keeper.SetNodeAccount(ctx, nodeAccount)
				return NewMsgSetIPAddress("192.168.0.1", nodeAccount.NodeAddress)
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *HandlerIPAddressTestHelper, name string) {
				c.Assert(err, IsNil)
				c.Assert(result, NotNil)
			},
		},
	}
	for _, tc := range testCases {
		ctx, k := setupKeeperForTest(c)
		helper := NewHandlerIPAddressTestHelper(k)
		mgr := NewManagers(helper)
		mgr.BeginBlock(ctx)
		handler := NewIPAddressHandler(helper, mgr)
		msg := tc.messageProvider(ctx, helper)
		constantAccessor := constants.GetConstantValues(constants.SWVersion)
		result, err := handler.Run(ctx, msg, semver.MustParse("0.1.0"), constantAccessor)
		tc.validator(c, ctx, result, err, helper, tc.name)

	}
}
