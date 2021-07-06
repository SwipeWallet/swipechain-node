package types

import (
	"gitlab.com/thorchain/thornode/common"
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
	. "gopkg.in/check.v1"
)

type MsgUnBondSuite struct{}

var _ = Suite(&MsgUnBondSuite{})

func (mas *MsgUnBondSuite) SetUpSuite(c *C) {
	SetupConfigForTest()
}

func (MsgUnBondSuite) TestMsgUnBond(c *C) {
	nodeAddr := GetRandomBech32Addr()
	txId := GetRandomTxHash()
	c.Check(txId.IsEmpty(), Equals, false)
	signerAddr := GetRandomBech32Addr()
	bondAddr := GetRandomBNBAddress()
	txin := GetRandomTx()
	txinNoID := txin
	txinNoID.ID = ""
	msgApply := NewMsgUnBond(txin, nodeAddr, cosmos.NewUint(common.One), bondAddr, signerAddr)
	c.Assert(msgApply.ValidateBasic(), IsNil)
	c.Assert(msgApply.Route(), Equals, RouterKey)
	c.Assert(msgApply.Type(), Equals, "unbond")
	c.Assert(msgApply.GetSignBytes(), NotNil)
	c.Assert(len(msgApply.GetSigners()), Equals, 1)
	c.Assert(msgApply.GetSigners()[0].Equals(signerAddr), Equals, true)
	c.Assert(NewMsgUnBond(txin, cosmos.AccAddress{}, cosmos.NewUint(common.One), bondAddr, signerAddr).ValidateBasic(), NotNil)
	c.Assert(NewMsgUnBond(txin, nodeAddr, cosmos.ZeroUint(), bondAddr, signerAddr).ValidateBasic(), NotNil)
	c.Assert(NewMsgUnBond(txinNoID, nodeAddr, cosmos.NewUint(common.One), bondAddr, signerAddr).ValidateBasic(), NotNil)
	c.Assert(NewMsgUnBond(txin, nodeAddr, cosmos.NewUint(common.One), "", signerAddr).ValidateBasic(), NotNil)
	c.Assert(NewMsgUnBond(txin, nodeAddr, cosmos.NewUint(common.One), bondAddr, cosmos.AccAddress{}).ValidateBasic(), NotNil)
}
