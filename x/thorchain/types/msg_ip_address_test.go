package types

import (
	"errors"

	se "github.com/cosmos/cosmos-sdk/types/errors"
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common/cosmos"
)

type MsgSetIPAddressSuite struct{}

var _ = Suite(&MsgSetIPAddressSuite{})

func (MsgSetIPAddressSuite) TestMsgSetIPAddressSuite(c *C) {
	acc1 := GetRandomBech32Addr()
	c.Assert(acc1.Empty(), Equals, false)
	msg := NewMsgSetIPAddress("192.168.0.1", acc1)
	c.Assert(msg.Route(), Equals, RouterKey)
	c.Assert(msg.Type(), Equals, "set_ip_address")
	c.Assert(msg.ValidateBasic(), IsNil)
	c.Assert(len(msg.GetSignBytes()) > 0, Equals, true)
	c.Assert(msg.GetSigners(), NotNil)
	c.Assert(msg.GetSigners()[0].String(), Equals, acc1.String())

	msg1 := NewMsgSetIPAddress("192.168.0.1", cosmos.AccAddress{})
	err1 := msg1.ValidateBasic()
	c.Assert(err1, NotNil)
	c.Assert(errors.Is(err1, se.ErrInvalidAddress), Equals, true)

	msg2 := NewMsgSetIPAddress("whatever", acc1)
	err2 := msg2.ValidateBasic()
	c.Assert(err2, NotNil)
	c.Assert(errors.Is(err2, se.ErrUnknownRequest), Equals, true)
}
