package keeperv1

import (
	. "gopkg.in/check.v1"
)

type KeeperBanSuite struct{}

var _ = Suite(&KeeperBanSuite{})

func (s *KeeperBanSuite) TestBanVoter(c *C) {
	ctx, k := setupKeeperForTest(c)
	addr := GetRandomBech32Addr()
	voter := NewBanVoter(addr)
	k.SetBanVoter(ctx, voter)
	voter, err := k.GetBanVoter(ctx, addr)
	c.Assert(err, IsNil)
	c.Check(voter.NodeAddress.Equals(addr), Equals, true)

	voter1, err := k.GetBanVoter(ctx, GetRandomBech32Addr())
	c.Check(err, IsNil)
	c.Check(voter1.IsEmpty(), Equals, false)
	iter := k.GetBanVoterIterator(ctx)
	c.Check(iter, NotNil)
	iter.Close()
}
