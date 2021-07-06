package keeperv1

import (
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

type KeeperNetworkFeeSuite struct{}

var _ = Suite(&KeeperNetworkFeeSuite{})

func (KeeperNetworkFeeSuite) TestNetworkFee(c *C) {
	ctx, k := setupKeeperForTest(c)
	networkFee := NewNetworkFee(common.BNBChain, 1, cosmos.NewUint(37500))
	c.Check(k.SaveNetworkFee(ctx, common.BNBChain, networkFee), IsNil)

	networkFee1 := NewNetworkFee(common.BNBChain, -1, cosmos.NewUint(37500))
	c.Check(k.SaveNetworkFee(ctx, common.BNBChain, networkFee1), NotNil)

	networkFee2, err := k.GetNetworkFee(ctx, common.ETHChain)
	c.Check(err, IsNil)
	c.Check(networkFee2.Valid(), NotNil)
	c.Check(k.GetNetworkFeeIterator(ctx), NotNil)
	networkFee3, err := k.GetNetworkFee(ctx, common.BNBChain)
	c.Check(err, IsNil)
	c.Check(networkFee3.Valid(), IsNil)
}
