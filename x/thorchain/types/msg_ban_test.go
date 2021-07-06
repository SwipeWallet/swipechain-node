package types

import (
	"errors"

	se "github.com/cosmos/cosmos-sdk/types/errors"
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common/cosmos"
)

type MsgBanSuite struct{}

var _ = Suite(&MsgBanSuite{})

func (MsgBanSuite) TestMsgBanSuite(c *C) {
	acc1 := GetRandomBech32Addr()
	acc2 := GetRandomBech32Addr()
	c.Assert(acc1.Empty(), Equals, false)
	msg := NewMsgBan(acc1, acc2)
	c.Assert(msg.Route(), Equals, RouterKey)
	c.Assert(msg.Type(), Equals, "ban")
	c.Assert(msg.ValidateBasic(), IsNil)
	c.Assert(len(msg.GetSignBytes()) > 0, Equals, true)
	c.Assert(msg.GetSigners(), NotNil)
	c.Assert(msg.GetSigners()[0].String(), Equals, acc2.String())
	msg1 := NewMsgBan(acc1, cosmos.AccAddress{})
	err1 := msg1.ValidateBasic()
	c.Assert(err1, NotNil)
	c.Assert(errors.Is(err1, se.ErrInvalidAddress), Equals, true)

	msg2 := NewMsgBan(cosmos.AccAddress{}, acc1)
	err2 := msg2.ValidateBasic()
	c.Assert(err2, NotNil)
	c.Assert(errors.Is(err2, se.ErrInvalidAddress), Equals, true)
}
