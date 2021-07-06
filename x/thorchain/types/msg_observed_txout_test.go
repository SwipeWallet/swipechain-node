package types

import (
	"errors"

	"gitlab.com/thorchain/thornode/common"
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"

	se "github.com/cosmos/cosmos-sdk/types/errors"
	. "gopkg.in/check.v1"
)

type MsgObservedTxOutSuite struct{}

var _ = Suite(&MsgObservedTxOutSuite{})

func (s *MsgObservedTxOutSuite) TestMsgObservedTxOut(c *C) {
	var err error
	pk := GetRandomPubKey()
	tx := NewObservedTx(GetRandomTx(), 55, pk)
	tx.Tx.FromAddress, err = pk.GetAddress(tx.Tx.Coins[0].Asset.Chain)
	c.Assert(err, IsNil)
	acc := GetRandomBech32Addr()

	m := NewMsgObservedTxOut(ObservedTxs{tx}, acc)
	EnsureMsgBasicCorrect(m, c)
	c.Check(m.Type(), Equals, "set_observed_txout")

	m1 := NewMsgObservedTxOut(nil, acc)
	c.Assert(m1.ValidateBasic(), NotNil)
	m2 := NewMsgObservedTxOut(ObservedTxs{tx}, cosmos.AccAddress{})
	c.Assert(m2.ValidateBasic(), NotNil)

	// will not accept observations with pre-determined signers. This is
	// important to ensure an observer can fake signers from other node accounts
	// *IMPORTANT* DON'T REMOVE THIS CHECK
	tx.Signers = append(tx.Signers, GetRandomBech32Addr())
	m3 := NewMsgObservedTxOut(ObservedTxs{tx}, acc)
	c.Assert(m3.ValidateBasic(), NotNil)

	tx4 := NewObservedTx(GetRandomTx(), 1, pk)
	m4 := NewMsgObservedTxOut(ObservedTxs{tx4}, acc)
	err4 := m4.ValidateBasic()
	c.Assert(err4, NotNil)
	c.Assert(errors.Is(err4, se.ErrUnknownRequest), Equals, true)

	tx5 := NewObservedTx(GetRandomTx(), 1, pk)
	tx5.Tx.FromAddress, err = pk.GetAddress(tx.Tx.Coins[0].Asset.Chain)
	tx5.OutHashes = common.TxIDs{
		GetRandomTxHash(),
	}
	m5 := NewMsgObservedTxOut(ObservedTxs{tx5}, acc)
	err5 := m5.ValidateBasic()
	c.Assert(err5, NotNil)
	c.Assert(errors.Is(err4, se.ErrUnknownRequest), Equals, true)

	tx6 := NewObservedTx(GetRandomTx(), 1, pk)
	tx6.Tx.FromAddress = common.Address("")
	m6 := NewMsgObservedTxOut(ObservedTxs{tx6}, acc)
	err6 := m6.ValidateBasic()
	c.Assert(err6, NotNil)
	c.Assert(errors.Is(err4, se.ErrUnknownRequest), Equals, true)

	tx7 := NewObservedTx(GetRandomTx(), 1, "whatever")
	m7 := NewMsgObservedTxOut(ObservedTxs{tx7}, acc)
	err7 := m7.ValidateBasic()
	c.Assert(err7, NotNil)
	c.Assert(errors.Is(err4, se.ErrUnknownRequest), Equals, true)
}
