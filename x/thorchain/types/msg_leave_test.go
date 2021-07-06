package types

import (
	"errors"

	se "github.com/cosmos/cosmos-sdk/types/errors"
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
)

type MsgLeaveSuite struct{}

var _ = Suite(&MsgLeaveSuite{})

func (*MsgLeaveSuite) SetupSuite(c *C) {
	SetupConfigForTest()
}

func (MsgLeaveSuite) TestMsgLeave(c *C) {
	nodeAddr := GetRandomBech32Addr()
	txId := GetRandomTxHash()
	senderBNBAddr := GetRandomBNBAddress()
	tx := GetRandomTx()
	tx.ID = txId
	tx.FromAddress = senderBNBAddr
	msgLeave := NewMsgLeave(tx, nodeAddr, nodeAddr)
	EnsureMsgBasicCorrect(msgLeave, c)
	c.Assert(msgLeave.ValidateBasic(), IsNil)
	c.Assert(msgLeave.Type(), Equals, "leave")

	msgLeave1 := NewMsgLeave(tx, nodeAddr, nodeAddr)
	c.Assert(msgLeave1.ValidateBasic(), IsNil)
	msgLeave2 := NewMsgLeave(common.Tx{ID: "", FromAddress: senderBNBAddr}, nodeAddr, nodeAddr)
	c.Assert(msgLeave2.ValidateBasic(), NotNil)
	msgLeave3 := NewMsgLeave(tx, nodeAddr, cosmos.AccAddress{})
	c.Assert(msgLeave3.ValidateBasic(), NotNil)
	msgLeave4 := NewMsgLeave(common.Tx{ID: txId, FromAddress: ""}, nodeAddr, nodeAddr)
	c.Assert(msgLeave4.ValidateBasic(), NotNil)

	msgLeave5 := NewMsgLeave(tx, cosmos.AccAddress{}, nodeAddr)
	err5 := msgLeave5.ValidateBasic()
	c.Assert(err5, NotNil)
	c.Assert(errors.Is(err5, se.ErrInvalidAddress), Equals, true)
}
