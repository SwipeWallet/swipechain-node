package thorchain

import (
	"errors"
	"fmt"

	"github.com/blang/semver"
	se "github.com/cosmos/cosmos-sdk/types/errors"
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
	keeper "gitlab.com/thorchain/thornode/x/thorchain/keeper"
)

type HandlerSetNodeKeysSuite struct{}

type TestSetNodeKeysKeeper struct {
	keeper.KVStoreDummy
	na     NodeAccount
	ensure error
}

func (k *TestSetNodeKeysKeeper) SendFromAccountToModule(ctx cosmos.Context, from cosmos.AccAddress, to string, coin common.Coin) error {
	return nil
}

func (k *TestSetNodeKeysKeeper) GetNodeAccount(ctx cosmos.Context, signer cosmos.AccAddress) (NodeAccount, error) {
	return k.na, nil
}

func (k *TestSetNodeKeysKeeper) EnsureNodeKeysUnique(_ cosmos.Context, _ string, _ common.PubKeySet) error {
	return k.ensure
}

func (k *TestSetNodeKeysKeeper) GetVaultData(ctx cosmos.Context) (VaultData, error) {
	return NewVaultData(), nil
}

func (k *TestSetNodeKeysKeeper) SetVaultData(ctx cosmos.Context, data VaultData) error {
	return nil
}

func (k *TestSetNodeKeysKeeper) SetNodeAccount(ctx cosmos.Context, na NodeAccount) error {
	return nil
}

var _ = Suite(&HandlerSetNodeKeysSuite{})

func (s *HandlerSetNodeKeysSuite) TestValidate(c *C) {
	ctx, _ := setupKeeperForTest(c)

	keeper := &TestSetNodeKeysKeeper{
		na:     GetRandomNodeAccount(NodeStandby),
		ensure: nil,
	}

	handler := NewSetNodeKeysHandler(keeper, NewDummyMgr())

	// happy path
	ver := constants.SWVersion
	constAccessor := constants.GetConstantValues(ver)
	signer := GetRandomBech32Addr()
	c.Assert(signer.Empty(), Equals, false)
	consensPubKey := GetRandomBech32ConsensusPubKey()
	pubKeys := GetRandomPubKeySet()

	msg := NewMsgSetNodeKeys(pubKeys, consensPubKey, signer)
	err := handler.validate(ctx, msg, ver, constAccessor)
	c.Assert(err, IsNil)
	result, err := handler.Run(ctx, msg, ver, constAccessor)
	c.Assert(err, IsNil)
	c.Assert(result, NotNil)

	// cannot set node keys for active account
	keeper.na.Status = NodeActive
	msg = NewMsgSetNodeKeys(pubKeys, consensPubKey, keeper.na.NodeAddress)
	err = handler.validate(ctx, msg, ver, constAccessor)
	c.Assert(err, NotNil)

	// cannot set node keys for disabled account
	keeper.na.Status = NodeDisabled
	msg = NewMsgSetNodeKeys(pubKeys, consensPubKey, keeper.na.NodeAddress)
	err = handler.validate(ctx, msg, ver, constAccessor)
	c.Assert(err, NotNil)

	// cannot set node keys when duplicate
	keeper.na.Status = NodeStandby
	keeper.ensure = fmt.Errorf("duplicate keys")
	msg = NewMsgSetNodeKeys(keeper.na.PubKeySet, consensPubKey, keeper.na.NodeAddress)
	err = handler.validate(ctx, msg, ver, constAccessor)
	c.Assert(err, ErrorMatches, "duplicate keys")
	keeper.ensure = nil

	// new version GT
	err = handler.validate(ctx, msg, semver.MustParse("2.0.0"), constAccessor)
	c.Assert(err, IsNil)

	// invalid version
	err = handler.validate(ctx, msg, semver.Version{}, constAccessor)
	c.Assert(err, Equals, errInvalidVersion)
	result, err = handler.Run(ctx, msg, semver.Version{}, constAccessor)
	c.Check(err, NotNil)
	c.Check(result, IsNil)

	// invalid msg
	msg = MsgSetNodeKeys{}
	err = handler.validate(ctx, msg, ver, constAccessor)
	c.Assert(err, NotNil)
	result, err = handler.Run(ctx, msg, ver, constAccessor)
	c.Assert(err, NotNil)
	c.Assert(result, IsNil)

	result, err = handler.Run(ctx, NewMsgMimir("what", 1, GetRandomBech32Addr()), ver, constAccessor)
	c.Check(err, NotNil)
	c.Check(result, IsNil)
}

type TestSetNodeKeysHandleKeeper struct {
	keeper.Keeper
	failGetNodeAccount bool
	failSetNodeAccount bool
	failGetVaultData   bool
	failSetVaultData   bool
}

func NewTestSetNodeKeysHandleKeeper(k keeper.Keeper) *TestSetNodeKeysHandleKeeper {
	return &TestSetNodeKeysHandleKeeper{
		Keeper: k,
	}
}

func (k *TestSetNodeKeysHandleKeeper) SendFromAccountToModule(ctx cosmos.Context, from cosmos.AccAddress, to string, coin common.Coin) error {
	return nil
}

func (k *TestSetNodeKeysHandleKeeper) GetNodeAccount(ctx cosmos.Context, signer cosmos.AccAddress) (NodeAccount, error) {
	if k.failGetNodeAccount {
		return NodeAccount{}, kaboom
	}
	return k.Keeper.GetNodeAccount(ctx, signer)
}

func (k *TestSetNodeKeysHandleKeeper) SetNodeAccount(ctx cosmos.Context, na NodeAccount) error {
	if k.failSetNodeAccount {
		return kaboom
	}
	return k.Keeper.SetNodeAccount(ctx, na)
}

func (k *TestSetNodeKeysHandleKeeper) GetVaultData(ctx cosmos.Context) (VaultData, error) {
	if k.failGetVaultData {
		return VaultData{}, kaboom
	}
	return k.Keeper.GetVaultData(ctx)
}

func (k *TestSetNodeKeysHandleKeeper) SetVaultData(ctx cosmos.Context, data VaultData) error {
	if k.failSetVaultData {
		return kaboom
	}
	return k.Keeper.SetVaultData(ctx, data)
}

func (k *TestSetNodeKeysHandleKeeper) EnsureNodeKeysUnique(_ cosmos.Context, consensPubKey string, pubKeys common.PubKeySet) error {
	return nil
}

func (s *HandlerSetNodeKeysSuite) TestHandle(c *C) {
	ctx, k := setupKeeperForTest(c)
	helper := NewTestSetNodeKeysHandleKeeper(k)
	mgr := NewManagers(helper)
	mgr.BeginBlock(ctx)
	handler := NewSetNodeKeysHandler(helper, mgr)
	ver := constants.SWVersion
	constAccessor := constants.GetConstantValues(ver)
	ctx = ctx.WithBlockHeight(1)
	signer := GetRandomBech32Addr()

	// add observer
	bepConsPubKey := GetRandomBech32ConsensusPubKey()
	bondAddr := GetRandomBNBAddress()
	pubKeys := GetRandomPubKeySet()
	emptyPubKeySet := common.PubKeySet{}

	msgNodeKeys := NewMsgSetNodeKeys(pubKeys, bepConsPubKey, signer)

	bond := cosmos.NewUint(common.One * 100)
	nodeAccount := NewNodeAccount(signer, NodeActive, emptyPubKeySet, "", bond, bondAddr, common.BlockHeight(ctx))
	c.Assert(helper.Keeper.SetNodeAccount(ctx, nodeAccount), IsNil)

	nodeAccount = NewNodeAccount(signer, NodeWhiteListed, emptyPubKeySet, "", bond, bondAddr, common.BlockHeight(ctx))
	c.Assert(helper.Keeper.SetNodeAccount(ctx, nodeAccount), IsNil)

	// happy path
	_, err := handler.handle(ctx, msgNodeKeys, ver, constAccessor)
	c.Assert(err, IsNil)
	na, err := helper.Keeper.GetNodeAccount(ctx, msgNodeKeys.Signer)
	c.Assert(err, IsNil)
	c.Assert(na.PubKeySet, Equals, pubKeys)
	c.Assert(na.ValidatorConsPubKey, Equals, bepConsPubKey)
	c.Assert(na.Status, Equals, NodeStandby)
	c.Assert(na.StatusSince, Equals, int64(1))

	testCases := []struct {
		name              string
		messageProvider   func(c *C, ctx cosmos.Context, helper *TestSetNodeKeysHandleKeeper) cosmos.Msg
		validator         func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *TestSetNodeKeysHandleKeeper, name string)
		skipForNativeRune bool
	}{
		{
			name: "fail to get node account should return an error",
			messageProvider: func(c *C, ctx cosmos.Context, helper *TestSetNodeKeysHandleKeeper) cosmos.Msg {
				helper.failGetNodeAccount = true
				return NewMsgSetNodeKeys(GetRandomPubKeySet(), GetRandomBech32ConsensusPubKey(), GetRandomBech32Addr())
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *TestSetNodeKeysHandleKeeper, name string) {
				c.Check(result, IsNil, Commentf(name))
				c.Check(err, NotNil, Commentf(name))
				c.Check(errors.Is(err, se.ErrUnauthorized), Equals, true)
			},
		},
		{
			name: "node account is empty should return an error",
			messageProvider: func(c *C, ctx cosmos.Context, helper *TestSetNodeKeysHandleKeeper) cosmos.Msg {
				return NewMsgSetNodeKeys(GetRandomPubKeySet(), GetRandomBech32ConsensusPubKey(), GetRandomBech32Addr())
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *TestSetNodeKeysHandleKeeper, name string) {
				c.Check(result, IsNil, Commentf(name))
				c.Check(err, NotNil, Commentf(name))
				c.Check(errors.Is(err, se.ErrUnauthorized), Equals, true)
			},
		},
		{
			name: "fail to save node account should return an error",
			messageProvider: func(c *C, ctx cosmos.Context, helper *TestSetNodeKeysHandleKeeper) cosmos.Msg {
				nodeAcct := GetRandomNodeAccount(NodeWhiteListed)
				c.Assert(helper.Keeper.SetNodeAccount(ctx, nodeAcct), IsNil)
				helper.failSetNodeAccount = true
				return NewMsgSetNodeKeys(nodeAcct.PubKeySet, nodeAcct.ValidatorConsPubKey, nodeAcct.NodeAddress)
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *TestSetNodeKeysHandleKeeper, name string) {
				c.Check(result, IsNil, Commentf(name))
				c.Check(err, NotNil, Commentf(name))
			},
		},
		{
			name: "fail to get vault data should return an error",
			messageProvider: func(c *C, ctx cosmos.Context, helper *TestSetNodeKeysHandleKeeper) cosmos.Msg {
				nodeAcct := GetRandomNodeAccount(NodeWhiteListed)
				c.Assert(helper.Keeper.SetNodeAccount(ctx, nodeAcct), IsNil)
				helper.failGetVaultData = true
				return NewMsgSetNodeKeys(nodeAcct.PubKeySet, nodeAcct.ValidatorConsPubKey, nodeAcct.NodeAddress)
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *TestSetNodeKeysHandleKeeper, name string) {
				c.Check(result, IsNil, Commentf(name))
				c.Check(err, NotNil, Commentf(name))
			},
			skipForNativeRune: true,
		},
		{
			name: "fail to set vault data should return an error",
			messageProvider: func(c *C, ctx cosmos.Context, helper *TestSetNodeKeysHandleKeeper) cosmos.Msg {
				nodeAcct := GetRandomNodeAccount(NodeWhiteListed)
				c.Assert(helper.Keeper.SetNodeAccount(ctx, nodeAcct), IsNil)
				helper.failSetVaultData = true
				return NewMsgSetNodeKeys(nodeAcct.PubKeySet, nodeAcct.ValidatorConsPubKey, nodeAcct.NodeAddress)
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *TestSetNodeKeysHandleKeeper, name string) {
				c.Check(result, IsNil, Commentf(name))
				c.Check(err, NotNil, Commentf(name))
			},
			skipForNativeRune: true,
		},
	}
	for _, tc := range testCases {
		if common.RuneAsset().Native() != "" && tc.skipForNativeRune {
			continue
		}
		ctx, k := setupKeeperForTest(c)
		helper := NewTestSetNodeKeysHandleKeeper(k)
		mgr := NewManagers(helper)
		mgr.BeginBlock(ctx)
		handler := NewSetNodeKeysHandler(helper, mgr)
		msg := tc.messageProvider(c, ctx, helper)
		constantAccessor := constants.GetConstantValues(constants.SWVersion)
		result, err := handler.Run(ctx, msg, semver.MustParse("0.1.0"), constantAccessor)
		tc.validator(c, ctx, result, err, helper, tc.name)
	}
}
