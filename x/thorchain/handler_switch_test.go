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

var _ = Suite(&HandlerSwitchSuite{})

type HandlerSwitchSuite struct{}

func (s *HandlerSwitchSuite) SetUpSuite(c *C) {
	SetupConfigForTest()
}

func (s *HandlerSwitchSuite) TestValidate(c *C) {
	ctx, k := setupKeeperForTest(c)

	na := GetRandomNodeAccount(NodeActive)
	c.Assert(k.SetNodeAccount(ctx, na), IsNil)
	tx := GetRandomTx()
	tx.Coins = common.Coins{
		common.NewCoin(common.Rune67CAsset, cosmos.NewUint(100*common.One)),
	}
	destination := GetRandomBNBAddress()

	handler := NewSwitchHandler(k, NewDummyMgr())

	constantAccessor := constants.GetConstantValues(constants.SWVersion)
	if common.RuneAsset().Chain.Equals(common.THORChain) {
		destination = GetRandomTHORAddress()
		// happy path
		msg := NewMsgSwitch(tx, destination, na.NodeAddress)
		result, err := handler.Run(ctx, msg, constants.SWVersion, constantAccessor)
		c.Assert(err, IsNil)
		c.Assert(result, NotNil)

		// invalid version
		result, err = handler.Run(ctx, msg, semver.Version{}, constantAccessor)
		c.Assert(err, Equals, errBadVersion)
		c.Assert(result, IsNil)

		// invalid msg
		msg = MsgSwitch{}
		result, err = handler.Run(ctx, msg, constants.SWVersion, constantAccessor)
		c.Assert(err, NotNil)
		c.Assert(result, IsNil)
	} else {
		// no swapping when using BEP2 rune
		msg := NewMsgSwitch(tx, destination, na.NodeAddress)
		result, err := handler.Run(ctx, msg, constants.SWVersion, constantAccessor)
		c.Assert(err, NotNil)
		c.Assert(result, IsNil)
		result, err = handler.Run(ctx, NewMsgMimir("whatever", 1, GetRandomBech32Addr()), constants.SWVersion, constantAccessor)
		c.Assert(err, NotNil)
		c.Assert(result, IsNil)
		c.Assert(errors.Is(err, errInvalidMessage), Equals, true)
	}
}

func (s *HandlerSwitchSuite) TestGettingNativeTokens(c *C) {
	ctx, k := setupKeeperForTest(c)

	na := GetRandomNodeAccount(NodeActive)
	c.Assert(k.SetNodeAccount(ctx, na), IsNil)
	tx := GetRandomTx()
	tx.Coins = common.Coins{
		common.NewCoin(common.Rune67CAsset, cosmos.NewUint(100*common.One)),
	}
	destination := GetRandomTHORAddress()

	handler := NewSwitchHandler(k, NewDummyMgr())

	msg := NewMsgSwitch(tx, destination, na.NodeAddress)
	_, err := handler.handle(ctx, msg, constants.SWVersion)
	c.Assert(err, IsNil)
	coin, err := common.NewCoin(common.RuneNative, cosmos.NewUint(100*common.One)).Native()
	c.Assert(err, IsNil)
	addr, err := cosmos.AccAddressFromBech32(destination.String())
	c.Assert(err, IsNil)
	c.Check(k.CoinKeeper().HasCoins(ctx, addr, cosmos.NewCoins(coin)), Equals, true)
	vaultData, err := k.GetVaultData(ctx)
	c.Assert(err, IsNil)
	c.Check(vaultData.TotalBEP2Rune.Equal(cosmos.NewUint(100*common.One)), Equals, true)

	// check that we can add more an account
	_, err = handler.handle(ctx, msg, constants.SWVersion)
	c.Assert(err, IsNil)
	coin, err = common.NewCoin(common.RuneNative, cosmos.NewUint(200*common.One)).Native()
	c.Assert(err, IsNil)
	c.Check(k.CoinKeeper().HasCoins(ctx, addr, cosmos.NewCoins(coin)), Equals, true)
	vaultData, err = k.GetVaultData(ctx)
	c.Assert(err, IsNil)
	c.Check(vaultData.TotalBEP2Rune.Equal(cosmos.NewUint(200*common.One)), Equals, true)
}

func (s *HandlerSwitchSuite) TestGettingBEP2Tokens(c *C) {
	ctx, k := setupKeeperForTest(c)

	vaultData := NewVaultData()
	vaultData.TotalBEP2Rune = cosmos.NewUint(500 * common.One)
	c.Assert(k.SetVaultData(ctx, vaultData), IsNil)

	na := GetRandomNodeAccount(NodeActive)
	c.Assert(k.SetNodeAccount(ctx, na), IsNil)

	from := GetRandomBech32Addr()
	tx := GetRandomTx()
	tx.FromAddress = common.Address(from.String())
	tx.Coins = common.Coins{
		common.NewCoin(common.RuneNative, cosmos.NewUint(100*common.One)),
	}
	destination := GetRandomBNBAddress()

	coin, err := common.NewCoin(common.RuneNative, cosmos.NewUint(800*common.One)).Native()
	c.Assert(err, IsNil)
	k.CoinKeeper().AddCoins(ctx, from, cosmos.NewCoins(coin))

	mgr := NewDummyMgr()
	handler := NewSwitchHandler(k, mgr)

	msg := NewMsgSwitch(tx, destination, na.NodeAddress)
	_, err = handler.handle(ctx, msg, constants.SWVersion)
	c.Assert(err, IsNil)

	coin, err = common.NewCoin(common.RuneNative, cosmos.NewUint(700*common.One)).Native()
	c.Assert(err, IsNil)
	c.Check(k.CoinKeeper().HasCoins(ctx, from, cosmos.NewCoins(coin)), Equals, true)
	vaultData, err = k.GetVaultData(ctx)
	c.Assert(err, IsNil)
	c.Check(vaultData.TotalBEP2Rune.Equal(cosmos.NewUint(400*common.One)), Equals, true)
	items, err := mgr.TxOutStore().GetOutboundItems(ctx)
	c.Assert(err, IsNil)
	c.Assert(items, HasLen, 1)

	// check that we can subtract more an account
	_, err = handler.handle(ctx, msg, constants.SWVersion)
	c.Assert(err, IsNil)
	coin, err = common.NewCoin(common.RuneNative, cosmos.NewUint(600*common.One)).Native()
	c.Assert(err, IsNil)
	c.Check(k.CoinKeeper().HasCoins(ctx, from, cosmos.NewCoins(coin)), Equals, true)
	vaultData, err = k.GetVaultData(ctx)
	c.Assert(err, IsNil)
	c.Check(vaultData.TotalBEP2Rune.Equal(cosmos.NewUint(300*common.One)), Equals, true, Commentf("%d", vaultData.TotalBEP2Rune.Uint64()))
	items, err = mgr.TxOutStore().GetOutboundItems(ctx)
	c.Assert(err, IsNil)
	c.Assert(items, HasLen, 2)

	// check that we can't overdraw
	msg.Tx.Coins[0].Amount = cosmos.NewUint(400 * common.One)
	_, err = handler.handle(ctx, msg, constants.SWVersion)
	c.Assert(err, NotNil)
	coin, err = common.NewCoin(common.RuneNative, cosmos.NewUint(600*common.One)).Native()
	c.Assert(err, IsNil)
	c.Check(k.CoinKeeper().HasCoins(ctx, from, cosmos.NewCoins(coin)), Equals, true)
	vaultData, err = k.GetVaultData(ctx)
	c.Assert(err, IsNil)
	c.Check(vaultData.TotalBEP2Rune.Equal(cosmos.NewUint(300*common.One)), Equals, true, Commentf("%d", vaultData.TotalBEP2Rune.Uint64()))
	items, err = mgr.TxOutStore().GetOutboundItems(ctx)
	c.Assert(err, IsNil)

	c.Assert(items, HasLen, 2)
}

type HandlerSwitchTestHelper struct {
	keeper.Keeper
	failGetVaultData  bool
	failSaveVaultData bool
}

func NewHandlerSwitchTestHelper(k keeper.Keeper) *HandlerSwitchTestHelper {
	return &HandlerSwitchTestHelper{
		Keeper: k,
	}
}

func (h *HandlerSwitchTestHelper) GetVaultData(ctx cosmos.Context) (VaultData, error) {
	if h.failGetVaultData {
		return VaultData{}, kaboom
	}
	return h.Keeper.GetVaultData(ctx)
}

func (h *HandlerSwitchTestHelper) SetVaultData(ctx cosmos.Context, vaultData VaultData) error {
	if h.failSaveVaultData {
		return kaboom
	}
	return h.Keeper.SetVaultData(ctx, vaultData)
}

func (s *HandlerSwitchSuite) getAValidSwitchMsg(ctx cosmos.Context, helper *HandlerSwitchTestHelper) MsgSwitch {
	na := GetRandomNodeAccount(NodeActive)
	from := GetRandomBech32Addr()
	tx := GetRandomTx()
	tx.FromAddress = common.Address(from.String())
	tx.Coins = common.Coins{
		common.NewCoin(common.RuneNative, cosmos.NewUint(100*common.One)),
	}
	destination := GetRandomBNBAddress()
	helper.Keeper.SetNodeAccount(ctx, na)
	vaultData, _ := helper.Keeper.GetVaultData(ctx)
	vaultData.TotalBEP2Rune = cosmos.NewUint(common.One * 100)
	helper.Keeper.SetVaultData(ctx, vaultData)
	coin, _ := common.NewCoin(common.RuneNative, cosmos.NewUint(800*common.One)).Native()
	helper.Keeper.CoinKeeper().AddCoins(ctx, from, cosmos.NewCoins(coin))
	vault := GetRandomVault()
	vault.Type = AsgardVault
	vault.Status = ActiveVault
	vault.AddFunds(common.Coins{
		common.NewCoin(common.BEP2RuneAsset(), cosmos.NewUint(100*common.One)),
	})
	helper.Keeper.SetVault(ctx, vault)
	return NewMsgSwitch(tx, destination, na.NodeAddress)
}

func (s *HandlerSwitchSuite) TestSwitchHandlerDifferentValidations(c *C) {
	testCases := []struct {
		name            string
		messageProvider func(c *C, ctx cosmos.Context, helper *HandlerSwitchTestHelper) cosmos.Msg
		validator       func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *HandlerSwitchTestHelper, name string)
		nativeRune      bool
	}{
		{
			name: "invalid msg should return an error",
			messageProvider: func(c *C, ctx cosmos.Context, helper *HandlerSwitchTestHelper) cosmos.Msg {
				return NewMsgMimir("what", 1, GetRandomBech32Addr())
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *HandlerSwitchTestHelper, name string) {
				c.Check(err, NotNil, Commentf(name))
				c.Check(result, IsNil, Commentf(name))
			},
		},
		{
			name: "fail get vault data should return an error",
			messageProvider: func(c *C, ctx cosmos.Context, helper *HandlerSwitchTestHelper) cosmos.Msg {
				helper.failGetVaultData = true
				return s.getAValidSwitchMsg(ctx, helper)
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *HandlerSwitchTestHelper, name string) {
				c.Check(err, NotNil, Commentf(name))
				c.Check(result, IsNil, Commentf(name))
			},
		},
		{
			name: "fail get vault data should return an error(to native RUNE)",
			messageProvider: func(c *C, ctx cosmos.Context, helper *HandlerSwitchTestHelper) cosmos.Msg {
				helper.failGetVaultData = true
				m := s.getAValidSwitchMsg(ctx, helper)
				m.Tx.Coins[0].Asset = common.BEP2RuneAsset()
				return m
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *HandlerSwitchTestHelper, name string) {
				c.Check(err, NotNil, Commentf(name))
				c.Check(result, IsNil, Commentf(name))
			},
		},
		{
			name: "Not enough RUNE to pay for fees should not fail",
			messageProvider: func(c *C, ctx cosmos.Context, helper *HandlerSwitchTestHelper) cosmos.Msg {
				m := s.getAValidSwitchMsg(ctx, helper)
				m.Tx.Coins[0].Amount = cosmos.NewUint(common.One).QuoUint64(2)
				return m
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *HandlerSwitchTestHelper, name string) {
				c.Check(err, IsNil, Commentf(name))
				c.Check(result, NotNil, Commentf(name))
			},
			nativeRune: true,
		},
		{
			name: "fail to save vault data should result an error",
			messageProvider: func(c *C, ctx cosmos.Context, helper *HandlerSwitchTestHelper) cosmos.Msg {
				helper.failSaveVaultData = true
				return s.getAValidSwitchMsg(ctx, helper)
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *HandlerSwitchTestHelper, name string) {
				c.Check(err, NotNil, Commentf(name))
				c.Check(result, IsNil, Commentf(name))
			},
		},
		{
			name: "fail to save vault data should result an error(to native RUNE)",
			messageProvider: func(c *C, ctx cosmos.Context, helper *HandlerSwitchTestHelper) cosmos.Msg {
				helper.failSaveVaultData = true
				m := s.getAValidSwitchMsg(ctx, helper)
				m.Tx.Coins[0].Asset = common.BEP2RuneAsset()
				return m
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *HandlerSwitchTestHelper, name string) {
				c.Check(err, NotNil, Commentf(name))
				c.Check(result, IsNil, Commentf(name))
			},
		},
	}

	for _, tc := range testCases {
		if len(common.RuneAsset().Native()) == 0 && tc.nativeRune {
			continue
		}

		ctx, k := setupKeeperForTest(c)
		helper := NewHandlerSwitchTestHelper(k)
		mgr := NewManagers(helper)
		mgr.BeginBlock(ctx)
		handler := NewSwitchHandler(helper, mgr)
		msg := tc.messageProvider(c, ctx, helper)
		constantAccessor := constants.GetConstantValues(constants.SWVersion)
		result, err := handler.Run(ctx, msg, semver.MustParse("0.1.0"), constantAccessor)
		tc.validator(c, ctx, result, err, helper, tc.name)
	}
}
