package types

import (
	"errors"

	se "github.com/cosmos/cosmos-sdk/types/errors"
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

type MsgSwitchSuite struct{}

var _ = Suite(&MsgSwitchSuite{})

func (MsgSwitchSuite) TestMsgSwitchSuite(c *C) {
	tx := GetRandomTx()
	tx.Coins = common.Coins{
		common.NewCoin(common.RuneAsset(), cosmos.NewUint(100*common.One)),
	}

	acc1 := GetRandomBNBAddress()
	acc2 := GetRandomBech32Addr()

	c.Assert(acc1.IsEmpty(), Equals, false)
	msg := NewMsgSwitch(tx, acc1, acc2)
	c.Assert(msg.Route(), Equals, RouterKey)
	c.Assert(msg.Type(), Equals, "switch")
	c.Assert(msg.ValidateBasic(), IsNil)
	c.Assert(len(msg.GetSignBytes()) > 0, Equals, true)
	c.Assert(msg.GetSigners(), NotNil)
	c.Assert(msg.GetSigners()[0].String(), Equals, acc2.String())

	msg1 := NewMsgSwitch(tx, acc1, cosmos.AccAddress{})
	err1 := msg1.ValidateBasic()
	c.Assert(err1, NotNil)
	c.Assert(errors.Is(err1, se.ErrInvalidAddress), Equals, true)

	msg2 := NewMsgSwitch(tx, "", acc2)
	err2 := msg2.ValidateBasic()
	c.Assert(err2, NotNil)
	c.Assert(errors.Is(err2, se.ErrInvalidAddress), Equals, true)
	// test too many coins
	tx.Coins = common.Coins{
		common.NewCoin(common.RuneAsset(), cosmos.NewUint(100*common.One)),
		common.NewCoin(common.BTCAsset, cosmos.NewUint(100*common.One)),
	}
	msg = NewMsgSwitch(tx, acc1, acc2)
	c.Assert(msg.ValidateBasic(), NotNil)

	// test too little coins
	tx.Coins = common.Coins{}
	msg = NewMsgSwitch(tx, acc1, acc2)
	c.Assert(msg.ValidateBasic(), NotNil)

	// test non rune token
	tx.Coins = common.Coins{
		common.NewCoin(common.BNBAsset, cosmos.NewUint(100*common.One)),
	}
	msg = NewMsgSwitch(tx, acc1, acc2)
	c.Assert(msg.ValidateBasic(), NotNil)
}
