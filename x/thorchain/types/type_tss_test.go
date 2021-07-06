package types

import (
	"sort"

	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
)

type TypeTssSuite struct{}

var _ = Suite(&TypeTssSuite{})

func (s *TypeTssSuite) TestVoter(c *C) {
	pk := GetRandomPubKey()
	pks := common.PubKeys{
		GetRandomPubKey(), GetRandomPubKey(), GetRandomPubKey(),
	}
	tss := NewTssVoter(
		"hello",
		pks,
		pk,
	)
	c.Check(tss.IsEmpty(), Equals, false)
	c.Check(tss.String(), Equals, "hello")

	chains := common.Chains{common.BNBChain, common.BTCChain}

	addr, err := pks[0].GetThorAddress()
	c.Assert(err, IsNil)
	c.Check(tss.HasSigned(addr), Equals, false)
	tss.Sign(addr, chains)
	c.Check(tss.Signers, HasLen, 1)
	c.Check(tss.HasSigned(addr), Equals, true)
	tss.Sign(addr, chains) // ensure signing twice doesn't duplicate
	c.Check(tss.Signers, HasLen, 1)
	c.Check(tss.Chains, HasLen, 2)

	c.Check(tss.HasConsensus(), Equals, false)
	addr, err = pks[1].GetThorAddress()
	c.Assert(err, IsNil)
	tss.Sign(addr, chains)
	c.Check(tss.HasConsensus(), Equals, true)
	v1 := NewTssVoter("", common.PubKeys{}, common.EmptyPubKey)
	c.Check(v1.IsEmpty(), Equals, true)
}

func (s *TypeTssSuite) TestChainConsensus(c *C) {
	voter := TssVoter{
		PubKeys: common.PubKeys{
			GetRandomPubKey(),
			GetRandomPubKey(),
			GetRandomPubKey(),
			GetRandomPubKey(),
		},
		Chains: common.Chains{
			common.BNBChain, // 4 BNB chains
			common.BNBChain,
			common.BNBChain,
			common.BNBChain,
			common.BTCChain, // 3 BTC chains
			common.BTCChain,
			common.BTCChain,
			common.ETHChain, // 2 ETH chains
			common.ETHChain,
			common.THORChain, // 1 THOR chain and partridge in a pear tree
		},
	}

	chains := voter.ConsensusChains()
	sort.Slice(chains, func(i, j int) bool {
		return chains[i].String() < chains[j].String()
	})
	c.Check(chains, DeepEquals, common.Chains{common.BNBChain, common.BTCChain})
}
