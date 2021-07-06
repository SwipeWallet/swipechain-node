package keeperv1

import (
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
)

type KeeperObservedNetworkFeeVoterSuite struct{}

var _ = Suite(&KeeperObservedNetworkFeeVoterSuite{})

func (*KeeperObservedNetworkFeeVoterSuite) TestObservedNetworkFeeVoter(c *C) {
	ctx, k := setupKeeperForTest(c)
	voter := NewObservedNetworkFeeVoter(1024, common.BNBChain)
	k.SetObservedNetworkFeeVoter(ctx, voter)
	voter, err := k.GetObservedNetworkFeeVoter(ctx, 1024, voter.Chain)
	c.Assert(err, IsNil)
	c.Check(voter.ReportBlockHeight, Equals, int64(1024))
	c.Check(voter.Chain.Equals(common.BNBChain), Equals, true)
	c.Check(k.GetObservedNetworkFeeVoterIterator(ctx), NotNil)

	voter1, err1 := k.GetObservedNetworkFeeVoter(ctx, 1028, common.BTCChain)
	c.Check(err1, IsNil)
	c.Check(voter1.IsEmpty(), Equals, false)
}
