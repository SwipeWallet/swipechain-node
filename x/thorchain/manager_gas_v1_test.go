package thorchain

import (
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/x/thorchain/keeper"
)

type GasManagerTestSuite struct{}

var _ = Suite(&GasManagerTestSuite{})

func (GasManagerTestSuite) TestGasManagerV1(c *C) {
	ctx, k := setupKeeperForTest(c)
	gasMgr := NewGasMgrV1()
	gasEvent := gasMgr.gasEvent
	c.Assert(gasMgr, NotNil)
	gasMgr.BeginBlock()
	c.Assert(gasEvent != gasMgr.gasEvent, Equals, true)

	pool := NewPool()
	pool.Asset = common.BNBAsset
	c.Assert(k.SetPool(ctx, pool), IsNil)
	pool.Asset = common.BTCAsset
	c.Assert(k.SetPool(ctx, pool), IsNil)

	gasMgr.AddGasAsset(common.Gas{
		common.NewCoin(common.BNBAsset, cosmos.NewUint(37500)),
		common.NewCoin(common.BTCAsset, cosmos.NewUint(1000)),
	})
	c.Assert(gasMgr.GetGas(), HasLen, 2)
	gasMgr.AddGasAsset(common.Gas{
		common.NewCoin(common.BNBAsset, cosmos.NewUint(38500)),
		common.NewCoin(common.BTCAsset, cosmos.NewUint(2000)),
	})
	c.Assert(gasMgr.GetGas(), HasLen, 2)
	gasMgr.AddGasAsset(common.Gas{
		common.NewCoin(common.ETHAsset, cosmos.NewUint(38500)),
	})
	c.Assert(gasMgr.GetGas(), HasLen, 3)
	eventMgr := NewEventMgrV1()
	gasMgr.EndBlock(ctx, k, eventMgr)
}

type gasManagerTestHelper struct {
	keeper.Keeper
	failGetVaultData bool
	failGetPool      bool
	failSetPool      bool
}

func newGasManagerTestHelper(k keeper.Keeper) *gasManagerTestHelper {
	return &gasManagerTestHelper{
		Keeper: k,
	}
}

func (g *gasManagerTestHelper) GetVaultData(ctx cosmos.Context) (VaultData, error) {
	if g.failGetVaultData {
		return VaultData{}, kaboom
	}
	return g.Keeper.GetVaultData(ctx)
}

func (g *gasManagerTestHelper) GetPool(ctx cosmos.Context, asset common.Asset) (Pool, error) {
	if g.failGetPool {
		return NewPool(), kaboom
	}
	return g.Keeper.GetPool(ctx, asset)
}

func (g *gasManagerTestHelper) SetPool(ctx cosmos.Context, p Pool) error {
	if g.failSetPool {
		return kaboom
	}
	return g.Keeper.SetPool(ctx, p)
}

func (GasManagerTestSuite) TestDifferentValidations(c *C) {
	ctx, k := setupKeeperForTest(c)
	gasMgr := NewGasMgrV1()
	gasMgr.BeginBlock()
	helper := newGasManagerTestHelper(k)
	eventMgr := NewEventMgrV1()
	gasMgr.EndBlock(ctx, helper, eventMgr)

	helper.failGetVaultData = true
	gasMgr.EndBlock(ctx, helper, eventMgr)
	helper.failGetVaultData = false

	helper.failGetPool = true
	gasMgr.AddGasAsset(common.Gas{
		common.NewCoin(common.BNBAsset, cosmos.NewUint(37500)),
		common.NewCoin(common.BTCAsset, cosmos.NewUint(1000)),
		common.NewCoin(common.ETHAsset, cosmos.ZeroUint()),
	})
	gasMgr.EndBlock(ctx, helper, eventMgr)
	helper.failGetPool = false
	helper.failSetPool = true
	p := NewPool()
	p.Asset = common.BNBAsset
	p.BalanceAsset = cosmos.NewUint(common.One * 100)
	p.BalanceRune = cosmos.NewUint(common.One * 100)
	p.Status = PoolEnabled
	c.Assert(helper.Keeper.SetPool(ctx, p), IsNil)
	gasMgr.AddGasAsset(common.Gas{
		common.NewCoin(common.BNBAsset, cosmos.NewUint(37500)),
	})
	gasMgr.EndBlock(ctx, helper, eventMgr)
}
