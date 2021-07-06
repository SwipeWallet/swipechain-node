package types

import (
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

type NetworkFeeSuite struct{}

var _ = Suite(&NetworkFeeSuite{})

func (NetworkFeeSuite) TestNetworkFee(c *C) {
	n := NewNetworkFee(common.BNBChain, 1, bnbSingleTxFee)
	c.Check(n.Valid(), IsNil)
	n1 := NewNetworkFee(common.EmptyChain, 1, bnbSingleTxFee)
	c.Check(n1.Valid(), NotNil)
	n2 := NewNetworkFee(common.BNBChain, -1, bnbSingleTxFee)
	c.Check(n2.Valid(), NotNil)

	n3 := NewNetworkFee(common.BNBChain, 1, cosmos.ZeroUint())
	c.Check(n3.Valid(), NotNil)
}
