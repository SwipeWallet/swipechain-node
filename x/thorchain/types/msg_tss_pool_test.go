package types

import (
	"errors"

	se "github.com/cosmos/cosmos-sdk/types/errors"
	blame "gitlab.com/thorchain/tss/go-tss/blame"
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
)

type MsgTssPoolSuite struct{}

var _ = Suite(&MsgTssPoolSuite{})

func (s *MsgTssPoolSuite) TestMsgTssPool(c *C) {
	pk := GetRandomPubKey()
	pks := common.PubKeys{
		GetRandomPubKey(), GetRandomPubKey(), GetRandomPubKey(),
	}
	addr := GetRandomBech32Addr()
	msg := NewMsgTssPool(pks, pk, AsgardKeygen, 1, blame.Blame{}, common.Chains{common.RuneAsset().Chain}, addr)
	c.Check(msg.Type(), Equals, "set_tss_pool")
	c.Assert(msg.ValidateBasic(), IsNil)
	EnsureMsgBasicCorrect(msg, c)

	c.Check(NewMsgTssPool(pks, pk, AsgardKeygen, 1, blame.Blame{}, common.Chains{common.RuneAsset().Chain}, nil).ValidateBasic(), NotNil)
	c.Check(NewMsgTssPool(nil, pk, AsgardKeygen, 1, blame.Blame{}, common.Chains{common.RuneAsset().Chain}, addr).ValidateBasic(), NotNil)
	c.Check(NewMsgTssPool(pks, "", AsgardKeygen, 1, blame.Blame{}, common.Chains{common.RuneAsset().Chain}, addr).ValidateBasic(), NotNil)
	c.Check(NewMsgTssPool(pks, "bogusPubkey", AsgardKeygen, 1, blame.Blame{}, common.Chains{common.RuneAsset().Chain}, addr).ValidateBasic(), NotNil)

	// fails on empty chain list
	msg = NewMsgTssPool(pks, pk, AsgardKeygen, 1, blame.Blame{}, common.Chains{}, addr)
	c.Check(msg.ValidateBasic(), NotNil)
	// fails on duplicates in chain list
	msg = NewMsgTssPool(pks, pk, AsgardKeygen, 1, blame.Blame{}, common.Chains{common.RuneAsset().Chain, common.RuneAsset().Chain}, addr)
	c.Check(msg.ValidateBasic(), NotNil)

	msg1 := NewMsgTssPool(pks, pk, AsgardKeygen, 1, blame.Blame{}, common.Chains{common.RuneAsset().Chain}, addr)
	msg1.ID = ""
	err1 := msg1.ValidateBasic()
	c.Assert(err1, NotNil)
	c.Check(errors.Is(err1, se.ErrUnknownRequest), Equals, true)

	msg2 := NewMsgTssPool(append(pks, ""), pk, AsgardKeygen, 1, blame.Blame{}, common.Chains{common.RuneAsset().Chain}, addr)
	err2 := msg2.ValidateBasic()
	c.Assert(err2, NotNil)
	c.Check(errors.Is(err2, se.ErrUnknownRequest), Equals, true)

	var allPks common.PubKeys
	for i := 0; i < 110; i++ {
		allPks = append(allPks, GetRandomPubKey())
	}
	msg3 := NewMsgTssPool(allPks, pk, AsgardKeygen, 1, blame.Blame{}, common.Chains{common.RuneAsset().Chain}, addr)
	err3 := msg3.ValidateBasic()
	c.Assert(err3, NotNil)
	c.Check(errors.Is(err3, se.ErrUnknownRequest), Equals, true)
}
