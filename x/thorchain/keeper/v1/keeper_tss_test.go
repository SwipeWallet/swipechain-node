package keeperv1

import (
	. "gopkg.in/check.v1"
)

type KeeperTssSuite struct{}

var _ = Suite(&KeeperTssSuite{})

func (s *KeeperTssSuite) TestTssVoter(c *C) {
	ctx, k := setupKeeperForTest(c)

	pk := GetRandomPubKey()
	voter := NewTssVoter("hello", nil, pk)

	v, err1 := k.GetTssVoter(ctx, voter.ID)
	c.Check(err1, IsNil)
	c.Check(v.IsEmpty(), Equals, true)

	k.SetTssVoter(ctx, voter)
	voter, err := k.GetTssVoter(ctx, voter.ID)
	c.Assert(err, IsNil)
	c.Check(voter.ID, Equals, "hello")
	c.Check(voter.PoolPubKey.Equals(pk), Equals, true)
	iter := k.GetTssVoterIterator(ctx)
	c.Check(iter, NotNil)
	iter.Close()
}
