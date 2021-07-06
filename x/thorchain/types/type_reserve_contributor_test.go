package types

import (
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
)

type RservesSuite struct{}

var _ = Suite(&RservesSuite{})

func (s *RservesSuite) TestReserveContributors(c *C) {
	addr := GetRandomBNBAddress()
	res := NewReserveContributor(
		addr,
		cosmos.NewUint(32*common.One),
	)
	c.Check(res.Address.Equals(addr), Equals, true)
	c.Check(res.Amount.Equal(cosmos.NewUint(32*common.One)), Equals, true)

	reses := ReserveContributors{res}

	res = NewReserveContributor(
		GetRandomBNBAddress(),
		cosmos.NewUint(10*common.One),
	)

	reses = reses.Add(res)
	c.Assert(reses, HasLen, 2)
	c.Check(reses[1].Amount.Equal(cosmos.NewUint(10*common.One)), Equals, true)
	reses = reses.Add(res)
	c.Assert(reses, HasLen, 2)
	c.Check(reses[1].Amount.Equal(cosmos.NewUint(20*common.One)), Equals, true)

	res1 := NewReserveContributor(common.NoAddress, cosmos.NewUint(1*common.One))
	c.Check(res1.Valid(), NotNil)
}
