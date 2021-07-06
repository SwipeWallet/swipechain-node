package types

import (
	. "gopkg.in/check.v1"
)

type BanVoterSuite struct{}

var _ = Suite(&BanVoterSuite{})

func (s BanVoterSuite) TestVoter(c *C) {
	ban := BanVoter{}
	c.Check(ban.Valid(), NotNil)
	c.Check(ban.IsEmpty(), Equals, true)

	addr := GetRandomBech32Addr()
	ban = NewBanVoter(addr)
	ban.BlockHeight = 12

	c.Check(ban.Valid(), IsNil)
	c.Check(ban.IsEmpty(), Equals, false)
	c.Check(ban.String(), Equals, addr.String())

	c.Check(ban.HasSigned(addr), Equals, false)
	ban.Sign(addr)
	c.Check(ban.HasSigned(addr), Equals, true)

	nodes := NodeAccounts{
		GetRandomNodeAccount(Active),
		GetRandomNodeAccount(Active),
		GetRandomNodeAccount(Active),
		GetRandomNodeAccount(Active),
	}

	c.Check(ban.HasConsensus(nodes), Equals, false)
	ban.Sign(nodes[0].NodeAddress)
	ban.Sign(nodes[1].NodeAddress)
	c.Check(ban.HasConsensus(nodes), Equals, false)
	ban.Sign(nodes[2].NodeAddress)
	c.Check(ban.HasConsensus(nodes), Equals, true)
}
