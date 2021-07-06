package keeperv1

import (
	. "gopkg.in/check.v1"
)

type KeeperMimirSuite struct{}

var _ = Suite(&KeeperMimirSuite{})

func (s *KeeperMimirSuite) TestMimir(c *C) {
	ctx, k := setupKeeperForTest(c)

	k.SetMimir(ctx, "foo", 14)

	val, err := k.GetMimir(ctx, "foo")
	c.Assert(err, IsNil)
	c.Assert(val, Equals, int64(14))

	val, err = k.GetMimir(ctx, "bogus")
	c.Assert(err, IsNil)
	c.Check(val, Equals, int64(-1))

	// test that releasing the kraken removes previously set key/values
	k.SetMimir(ctx, KRAKEN, 0)
	val, err = k.GetMimir(ctx, "foo")
	c.Assert(err, IsNil)
	c.Assert(val, Equals, int64(-1))

	// test that we cannot put the kraken back in the cage
	k.SetMimir(ctx, KRAKEN, -1)
	k.SetMimir(ctx, "foo", 33)
	val, err = k.GetMimir(ctx, "foo")
	c.Assert(err, IsNil)
	c.Assert(val, Equals, int64(-1))
	c.Check(k.GetMimirIterator(ctx), NotNil)
}
