package keeperv1

import (
	. "gopkg.in/check.v1"
)

type KeeperTssKeysignFailureSuite struct{}

var _ = Suite(&KeeperTssKeysignFailureSuite{})

func (KeeperTssKeysignFailureSuite) TestTssKeysignFailVoter(c *C) {
	ctx, k := setupKeeperForTest(c)
	id := GetRandomTxHash().String()
	voter, err := k.GetTssKeysignFailVoter(ctx, id)
	c.Check(err, IsNil)
	c.Check(voter.Empty(), Equals, true)

	k.SetTssKeysignFailVoter(ctx, NewTssKeysignFailVoter(id, 1024))
	voter1, err1 := k.GetTssKeysignFailVoter(ctx, id)
	c.Check(err1, IsNil)
	c.Check(voter1.Empty(), Equals, false)
	c.Check(k.GetTssKeysignFailVoterIterator(ctx), NotNil)
}
