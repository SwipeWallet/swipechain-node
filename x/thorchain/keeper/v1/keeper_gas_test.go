package keeperv1

import (
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
)

type KeeperGasSuite struct{}

var _ = Suite(&KeeperGasSuite{})

func (s *KeeperGasSuite) TestGas(c *C) {
	ctx, k := setupKeeperForTest(c)

	bnbGas := []cosmos.Uint{
		cosmos.NewUint(37500),
		cosmos.NewUint(30000),
	}

	k.SetGas(ctx, common.BNBAsset, bnbGas)

	gas, err := k.GetGas(ctx, common.BNBAsset)
	c.Assert(err, IsNil)
	c.Assert(gas, HasLen, 2)
	c.Assert(gas[0].Equal(cosmos.NewUint(37500)), Equals, true)
	c.Assert(gas[1].Equal(cosmos.NewUint(30000)), Equals, true)

	c.Check(k.GetGasIterator(ctx), NotNil)
}
