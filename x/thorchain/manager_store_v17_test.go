package thorchain

import (
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

type ManagerStoreV17TestSuite struct{}

var _ = Suite(&ManagerStoreV17TestSuite{})

func (s *ManagerStoreV17TestSuite) SetUpSuite(c *C) {
	SetupConfigForTest()
}

func (*ManagerStoreV17TestSuite) TestCalibrateVaultToPool(c *C) {
	ctx, keeper := setupKeeperForTest(c)
	helper := NewVaultGenesisSetupTestHelper(keeper)
	mgr := NewManagers(helper)
	mgr.BeginBlock(ctx)
	storeMgr := NewStoreMgr(keeper)

	for k := range assetsToPool {
		pool := NewPool()
		asset, err := common.NewAsset(k)
		c.Assert(err, IsNil)
		pool.Asset = asset
		pool.BalanceAsset = cosmos.NewUint(100 * common.One)
		pool.BalanceRune = cosmos.NewUint(100 * common.One)
		c.Assert(keeper.SetPool(ctx, pool), IsNil)
	}
	c.Assert(storeMgr.calibrateVaultToPool(ctx), IsNil)

	for k, v := range assetsToPool {
		asset, err := common.NewAsset(k)
		c.Assert(err, IsNil)
		p, err := keeper.GetPool(ctx, asset)
		c.Assert(err, IsNil)
		gap := p.BalanceAsset.Sub(cosmos.NewUint(100 * common.One))
		c.Assert(int64(gap.Uint64()), Equals, v)
		c.Assert(p.BalanceRune.Equal(cosmos.NewUint(100*common.One)), Equals, true)
	}
}
