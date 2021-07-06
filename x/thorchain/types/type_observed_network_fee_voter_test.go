package types

import (
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
)

type ObservedNetworkFeeVoterTestSuite struct{}

var _ = Suite(&ObservedNetworkFeeVoterTestSuite{})

func (ObservedNetworkFeeVoterTestSuite) TestObservedNetworkFeeVoter(c *C) {
	voter := NewObservedNetworkFeeVoter(1024, common.BTCChain)
	c.Check(voter.IsEmpty(), Equals, false)
	addr := GetRandomBech32Addr()
	c.Check(voter.HasSigned(addr), Equals, false)
	voter.Sign(addr)
	c.Check(voter.HasSigned(addr), Equals, true)
	voter.Sign(addr)
	c.Check(voter.Signers, HasLen, 1)
	c.Check(voter.HasConsensus(nil), Equals, false)
	nas := NodeAccounts{
		NodeAccount{NodeAddress: addr, Status: Active},
	}
	c.Check(voter.HasConsensus(nas), Equals, true)
	c.Check(len(voter.String()) > 0, Equals, true)

	voter1 := NewObservedNetworkFeeVoter(0, common.EmptyChain)
	c.Check(voter1.IsEmpty(), Equals, true)
}
