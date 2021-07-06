package types

import (
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

type MsgOutboundTxSuite struct{}

var _ = Suite(&MsgOutboundTxSuite{})

func (MsgOutboundTxSuite) TestMsgOutboundTx(c *C) {
	txID := GetRandomTxHash()
	inTxID := GetRandomTxHash()
	bnb := GetRandomBNBAddress()
	acc1 := GetRandomBech32Addr()
	tx := NewObservedTx(common.NewTx(
		txID,
		bnb,
		GetRandomBNBAddress(),
		common.Coins{common.NewCoin(common.BNBAsset, cosmos.OneUint())},
		BNBGasFeeSingleton,
		"",
	), 12, GetRandomPubKey())
	m := NewMsgOutboundTx(tx, inTxID, acc1)
	EnsureMsgBasicCorrect(m, c)
	c.Check(m.Type(), Equals, "set_tx_outbound")

	inputs := []struct {
		txID   common.TxID
		inTxID common.TxID
		sender common.Address
		signer cosmos.AccAddress
	}{
		{
			txID:   common.TxID(""),
			inTxID: inTxID,
			sender: bnb,
			signer: acc1,
		},
		{
			txID:   txID,
			inTxID: common.TxID(""),
			sender: bnb,
			signer: acc1,
		},
		{
			txID:   txID,
			inTxID: inTxID,
			sender: common.NoAddress,
			signer: acc1,
		},
		{
			txID:   txID,
			inTxID: inTxID,
			sender: bnb,
			signer: cosmos.AccAddress{},
		},
	}

	for _, item := range inputs {
		tx := NewObservedTx(common.NewTx(
			item.txID,
			item.sender,
			GetRandomBNBAddress(),
			common.Coins{common.NewCoin(common.BNBAsset, cosmos.OneUint())},
			BNBGasFeeSingleton,
			"",
		), 12, GetRandomPubKey())
		m := NewMsgOutboundTx(tx, item.inTxID, item.signer)
		err := m.ValidateBasic()
		c.Assert(err, NotNil, Commentf("%s", err.Error()))
	}
}
