package thorchain

import (
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
	keeper "gitlab.com/thorchain/thornode/x/thorchain/keeper"
)

type YggdrasilSuite struct{}

var _ = Suite(&YggdrasilSuite{})

func (s YggdrasilSuite) TestCalcTargetAmounts(c *C) {
	var pools []Pool
	p := NewPool()
	p.Asset = common.BNBAsset
	p.BalanceRune = cosmos.NewUint(1000 * common.One)
	p.BalanceAsset = cosmos.NewUint(500 * common.One)
	pools = append(pools, p)

	p = NewPool()
	p.Asset = common.BTCAsset
	p.BalanceRune = cosmos.NewUint(3000 * common.One)
	p.BalanceAsset = cosmos.NewUint(225 * common.One)
	pools = append(pools, p)

	ygg := GetRandomVault()
	ygg.Type = YggdrasilVault

	totalBond := cosmos.NewUint(8000 * common.One)
	bond := cosmos.NewUint(200 * common.One)
	ymgr := NewYggMgrV1(keeper.KVStoreDummy{})
	yggFundLimit := cosmos.NewUint(50)
	coins, err := ymgr.calcTargetYggCoins(pools, ygg, bond, totalBond, yggFundLimit)
	c.Assert(err, IsNil)
	c.Assert(coins, HasLen, 3)
	c.Check(coins[0].Asset.String(), Equals, common.BNBAsset.String())
	c.Check(coins[0].Amount.Uint64(), Equals, cosmos.NewUint(6.25*common.One).Uint64(), Commentf("%d vs %d", coins[0].Amount.Uint64(), cosmos.NewUint(6.25*common.One).Uint64()))
	c.Check(coins[1].Asset.String(), Equals, common.BTCAsset.String())
	c.Check(coins[1].Amount.Uint64(), Equals, cosmos.NewUint(2.8125*common.One).Uint64(), Commentf("%d vs %d", coins[1].Amount.Uint64(), cosmos.NewUint(2.8125*common.One).Uint64()))
	c.Check(coins[2].Asset.String(), Equals, common.RuneAsset().String())
	c.Check(coins[2].Amount.Uint64(), Equals, cosmos.NewUint(50*common.One).Uint64(), Commentf("%d vs %d", coins[2].Amount.Uint64(), cosmos.NewUint(50*common.One).Uint64()))
}

func (s YggdrasilSuite) TestCalcTargetAmounts2(c *C) {
	// Adding specific test per PR request
	// https://gitlab.com/thorchain/thornode/merge_requests/246#note_241913460
	var pools []Pool
	p := NewPool()
	p.Asset = common.BNBAsset
	p.BalanceRune = cosmos.NewUint(1000000 * common.One)
	p.BalanceAsset = cosmos.NewUint(1 * common.One)
	pools = append(pools, p)

	ygg := GetRandomVault()
	ygg.Type = YggdrasilVault

	totalBond := cosmos.NewUint(3000000 * common.One)
	bond := cosmos.NewUint(1000000 * common.One)
	ymgr := NewYggMgrV1(keeper.KVStoreDummy{})
	yggFundLimit := cosmos.NewUint(50)
	coins, err := ymgr.calcTargetYggCoins(pools, ygg, bond, totalBond, yggFundLimit)
	c.Assert(err, IsNil)
	c.Assert(coins, HasLen, 2)
	c.Check(coins[0].Asset.String(), Equals, common.BNBAsset.String())
	c.Check(coins[0].Amount.Uint64(), Equals, cosmos.NewUint(0.16666667*common.One).Uint64(), Commentf("%d vs %d", coins[0].Amount.Uint64(), cosmos.NewUint(0.16666667*common.One).Uint64()))
	c.Check(coins[1].Asset.String(), Equals, common.RuneAsset().String())
	c.Check(coins[1].Amount.Uint64(), Equals, cosmos.NewUint(166666.66666667*common.One).Uint64(), Commentf("%d vs %d", coins[1].Amount.Uint64(), cosmos.NewUint(166666.66666667*common.One).Uint64()))
}

func (s YggdrasilSuite) TestCalcTargetAmounts3(c *C) {
	// pre populate the yggdrasil vault with funds already, ensure we don't
	// double up on funds.
	var pools []Pool
	p := NewPool()
	p.Asset = common.BNBAsset
	p.BalanceRune = cosmos.NewUint(1000 * common.One)
	p.BalanceAsset = cosmos.NewUint(500 * common.One)
	pools = append(pools, p)

	p = NewPool()
	p.Asset = common.BTCAsset
	p.BalanceRune = cosmos.NewUint(3000 * common.One)
	p.BalanceAsset = cosmos.NewUint(225 * common.One)
	pools = append(pools, p)

	ygg := GetRandomVault()
	ygg.Type = YggdrasilVault
	ygg.Coins = common.Coins{
		common.NewCoin(common.BNBAsset, cosmos.NewUint(6.25*common.One)),
		common.NewCoin(common.BTCAsset, cosmos.NewUint(1.8125*common.One)),
		common.NewCoin(common.RuneAsset(), cosmos.NewUint(30*common.One)),
	}

	totalBond := cosmos.NewUint(8000 * common.One)
	bond := cosmos.NewUint(200 * common.One)
	ymgr := NewYggMgrV1(keeper.KVStoreDummy{})
	yggFundLimit := cosmos.NewUint(50)
	coins, err := ymgr.calcTargetYggCoins(pools, ygg, bond, totalBond, yggFundLimit)
	c.Assert(err, IsNil)
	c.Assert(coins, HasLen, 2, Commentf("%d", len(coins)))
	c.Check(coins[0].Asset.String(), Equals, common.BTCAsset.String())
	c.Check(coins[0].Amount.Uint64(), Equals, cosmos.NewUint(1*common.One).Uint64(), Commentf("%d vs %d", coins[0].Amount.Uint64(), cosmos.NewUint(2.8125*common.One).Uint64()))
	c.Check(coins[1].Asset.String(), Equals, common.RuneAsset().String())
	c.Check(coins[1].Amount.Uint64(), Equals, cosmos.NewUint(20*common.One).Uint64(), Commentf("%d vs %d", coins[1].Amount.Uint64(), cosmos.NewUint(50*common.One).Uint64()))
}

func (s YggdrasilSuite) TestCalcTargetAmounts4(c *C) {
	// test under bonded scenario
	var pools []Pool
	p := NewPool()
	p.Asset = common.BNBAsset
	p.BalanceRune = cosmos.NewUint(1000 * common.One)
	p.BalanceAsset = cosmos.NewUint(500 * common.One)
	pools = append(pools, p)

	p = NewPool()
	p.Asset = common.BTCAsset
	p.BalanceRune = cosmos.NewUint(3000 * common.One)
	p.BalanceAsset = cosmos.NewUint(225 * common.One)
	pools = append(pools, p)

	ygg := GetRandomVault()
	ygg.Type = YggdrasilVault
	ygg.Coins = common.Coins{
		common.NewCoin(common.BNBAsset, cosmos.NewUint(6.25*common.One)),
		common.NewCoin(common.BTCAsset, cosmos.NewUint(1.8125*common.One)),
		common.NewCoin(common.RuneAsset(), cosmos.NewUint(30*common.One)),
	}

	totalBond := cosmos.NewUint(2000 * common.One)
	bond := cosmos.NewUint(200 * common.One)
	ymgr := NewYggMgrV1(keeper.KVStoreDummy{})
	yggFundLimit := cosmos.NewUint(50)
	coins, err := ymgr.calcTargetYggCoins(pools, ygg, bond, totalBond, yggFundLimit)
	c.Assert(err, IsNil)
	c.Assert(coins, HasLen, 2, Commentf("%d", len(coins)))
	c.Check(coins[0].Asset.String(), Equals, common.BTCAsset.String())
	c.Check(coins[0].Amount.Uint64(), Equals, cosmos.NewUint(1*common.One).Uint64(), Commentf("%d vs %d", coins[0].Amount.Uint64(), cosmos.NewUint(2.8125*common.One).Uint64()))
	c.Check(coins[1].Asset.String(), Equals, common.RuneAsset().String())
	c.Check(coins[1].Amount.Uint64(), Equals, cosmos.NewUint(20*common.One).Uint64(), Commentf("%d vs %d", coins[1].Amount.Uint64(), cosmos.NewUint(50*common.One).Uint64()))
}

func (s YggdrasilSuite) TestFund(c *C) {
	ctx, k := setupKeeperForTest(c)

	vault := GetRandomVault()
	vault.Coins = common.Coins{
		common.NewCoin(common.RuneAsset(), cosmos.NewUint(10000*common.One)),
		common.NewCoin(common.BNBAsset, cosmos.NewUint(10000*common.One)),
	}
	k.SetVault(ctx, vault)
	mgr := NewDummyMgr()

	// setup 6 active nodes
	for i := 0; i < 6; i++ {
		na := GetRandomNodeAccount(NodeActive)
		na.Bond = cosmos.NewUint(common.One * 1000000)
		c.Assert(k.SetNodeAccount(ctx, na), IsNil)
	}
	constAccessor := constants.GetConstantValues(constants.SWVersion)
	ymgr := NewYggMgrV1(k)
	err := ymgr.Fund(ctx, mgr, constAccessor)
	c.Assert(err, IsNil)
	na1 := GetRandomNodeAccount(NodeActive)
	na1.Bond = cosmos.NewUint(1000000 * common.One)
	c.Assert(k.SetNodeAccount(ctx, na1), IsNil)
	bnbPool := NewPool()
	bnbPool.Asset = common.BNBAsset
	bnbPool.BalanceAsset = cosmos.NewUint(100000 * common.One)
	bnbPool.BalanceRune = cosmos.NewUint(100000 * common.One)
	c.Assert(k.SetPool(ctx, bnbPool), IsNil)
	err1 := ymgr.Fund(ctx, mgr, constAccessor)
	c.Assert(err1, IsNil)
	items, err := mgr.TxOutStore().GetOutboundItems(ctx)
	c.Assert(err, IsNil)
	if common.RuneAsset().Chain.Equals(common.THORChain) {
		c.Assert(items, HasLen, 1)
	} else {
		c.Assert(items, HasLen, 2)
	}
}
