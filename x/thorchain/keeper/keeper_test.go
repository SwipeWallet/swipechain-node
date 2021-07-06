package keeper

import (
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
)

func FundModule(c *C, ctx cosmos.Context, k Keeper, name string, amt uint64) {
	coin, err := common.NewCoin(common.RuneNative, cosmos.NewUint(amt*common.One)).Native()
	c.Assert(err, IsNil)
	err = k.Supply().MintCoins(ctx, ModuleName, cosmos.NewCoins(coin))
	c.Assert(err, IsNil)
	err = k.Supply().SendCoinsFromModuleToModule(ctx, ModuleName, name, cosmos.NewCoins(coin))
	c.Assert(err, IsNil)
}

func FundAccount(c *C, ctx cosmos.Context, k Keeper, addr cosmos.AccAddress, amt uint64) {
	coin, err := common.NewCoin(common.RuneNative, cosmos.NewUint(amt*common.One)).Native()
	c.Assert(err, IsNil)
	err = k.Supply().MintCoins(ctx, ModuleName, cosmos.NewCoins(coin))
	c.Assert(err, IsNil)
	err = k.Supply().SendCoinsFromModuleToAccount(ctx, ModuleName, addr, cosmos.NewCoins(coin))
	c.Assert(err, IsNil)
}

var (
	multiPerm    = "multiple permissions account"
	randomPerm   = "random permission"
	holder       = "holder"
	keyThorchain = cosmos.NewKVStoreKey(StoreKey)
)
