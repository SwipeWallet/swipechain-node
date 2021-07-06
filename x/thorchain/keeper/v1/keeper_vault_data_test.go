package keeperv1

import (
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

type KeeperVaultDataSuite struct{}

var _ = Suite(&KeeperVaultDataSuite{})

func (KeeperVaultDataSuite) TestVaultData(c *C) {
	ctx, k := setupKeeperForTest(c)
	vd, err := k.GetVaultData(ctx)
	c.Check(err, IsNil)
	c.Check(vd.BondRewardRune.Equal(cosmos.ZeroUint()), Equals, true)

	vd1 := NewVaultData()
	vd1.BondRewardRune = cosmos.NewUint(common.One * 100)
	err1 := k.SetVaultData(ctx, vd1)
	c.Assert(err1, IsNil)

	vd2, err2 := k.GetVaultData(ctx)
	c.Check(err2, IsNil)
	c.Check(vd2.BondRewardRune.Equal(vd1.BondRewardRune), Equals, true)
}
