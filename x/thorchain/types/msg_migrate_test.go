package types

import (
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
)

type MsgMigrateSuite struct{}

var _ = Suite(&MsgMigrateSuite{})

func (MsgMigrateSuite) TestMsgMigrateSuite(c *C) {
	txID := GetRandomTxHash()
	bnb := GetRandomBNBAddress()
	acc1 := GetRandomBech32Addr()
	tx := NewObservedTx(common.NewTx(
		txID,
		bnb,
		GetRandomBNBAddress(),
		common.Coins{common.NewCoin(common.BNBAsset, cosmos.OneUint())},
		BNBGasFeeSingleton,
		"migrate:10",
	), 12, GetRandomPubKey())
	m := NewMsgMigrate(tx, 10, acc1)
	EnsureMsgBasicCorrect(m, c)
	c.Check(m.Type(), Equals, "migrate")

	inputs := []struct {
		txID        common.TxID
		blockHeight int64
		sender      common.Address
		signer      cosmos.AccAddress
	}{
		{
			txID:        common.TxID(""),
			blockHeight: 1,
			sender:      bnb,
			signer:      acc1,
		},
		{
			txID:        txID,
			blockHeight: 0,
			sender:      bnb,
			signer:      acc1,
		},
		{
			txID:        txID,
			blockHeight: 1,
			sender:      common.NoAddress,
			signer:      acc1,
		},
		{
			txID:        txID,
			blockHeight: 1,
			sender:      bnb,
			signer:      cosmos.AccAddress{},
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
		m := NewMsgMigrate(tx, item.blockHeight, item.signer)
		err := m.ValidateBasic()
		c.Assert(err, NotNil, Commentf("%s", err.Error()))
	}
}
