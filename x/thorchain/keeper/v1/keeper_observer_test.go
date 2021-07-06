package keeperv1

import (
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common/cosmos"
)

type KeeperObserverSuite struct{}

var _ = Suite(&KeeperObserverSuite{})

func (s *KeeperObserverSuite) TestObserver(c *C) {
	ctx, k := setupKeeperForTest(c)

	addr := GetRandomBech32Addr()

	k.AddObservingAddresses(ctx, []cosmos.AccAddress{addr})
	addrs, err := k.GetObservingAddresses(ctx)
	c.Assert(err, IsNil)
	c.Assert(addrs, HasLen, 1)
	c.Check(addrs[0].Equals(addr), Equals, true)

	k.ClearObservingAddresses(ctx)
	addrs, err = k.GetObservingAddresses(ctx)
	c.Assert(err, IsNil)
	c.Assert(addrs, HasLen, 0)
}
