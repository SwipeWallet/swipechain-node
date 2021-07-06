package types

import (
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

type MsgSwapSuite struct{}

var _ = Suite(&MsgSwapSuite{})

func (MsgSwapSuite) TestMsgSwap(c *C) {
	addr := GetRandomBech32Addr()
	c.Check(addr.Empty(), Equals, false)
	bnbAddress := GetRandomBNBAddress()
	txID := GetRandomTxHash()
	c.Check(txID.IsEmpty(), Equals, false)

	tx := common.NewTx(
		txID,
		GetRandomBNBAddress(),
		GetRandomBNBAddress(),
		common.Coins{
			common.NewCoin(common.BTCAsset, cosmos.NewUint(1)),
		},
		BNBGasFeeSingleton,
		"SWAP:BNB.BNB",
	)

	m := NewMsgSwap(tx, common.BNBAsset, bnbAddress, cosmos.NewUint(200000000), addr)
	EnsureMsgBasicCorrect(m, c)
	c.Check(m.Type(), Equals, "swap")

	inputs := []struct {
		requestTxHash common.TxID
		source        common.Asset
		target        common.Asset
		amount        cosmos.Uint
		requester     common.Address
		destination   common.Address
		targetPrice   cosmos.Uint
		signer        cosmos.AccAddress
	}{
		{
			requestTxHash: common.TxID(""),
			source:        common.RuneAsset(),
			target:        common.BNBAsset,
			amount:        cosmos.NewUint(100000000),
			requester:     bnbAddress,
			destination:   bnbAddress,
			targetPrice:   cosmos.NewUint(200000000),
			signer:        addr,
		},
		{
			requestTxHash: txID,
			source:        common.Asset{},
			target:        common.BNBAsset,
			amount:        cosmos.NewUint(100000000),
			requester:     bnbAddress,
			destination:   bnbAddress,
			targetPrice:   cosmos.NewUint(200000000),
			signer:        addr,
		},
		{
			requestTxHash: txID,
			source:        common.BNBAsset,
			target:        common.BNBAsset,
			amount:        cosmos.NewUint(100000000),
			requester:     bnbAddress,
			destination:   bnbAddress,
			targetPrice:   cosmos.NewUint(200000000),
			signer:        addr,
		},
		{
			requestTxHash: txID,
			source:        common.RuneAsset(),
			target:        common.Asset{},
			amount:        cosmos.NewUint(100000000),
			requester:     bnbAddress,
			destination:   bnbAddress,
			targetPrice:   cosmos.NewUint(200000000),
			signer:        addr,
		},
		{
			requestTxHash: txID,
			source:        common.RuneAsset(),
			target:        common.BNBAsset,
			amount:        cosmos.ZeroUint(),
			requester:     bnbAddress,
			destination:   bnbAddress,
			targetPrice:   cosmos.NewUint(200000000),
			signer:        addr,
		},
		{
			requestTxHash: txID,
			source:        common.RuneAsset(),
			target:        common.BNBAsset,
			amount:        cosmos.NewUint(100000000),
			requester:     common.NoAddress,
			destination:   bnbAddress,
			targetPrice:   cosmos.NewUint(200000000),
			signer:        addr,
		},
		{
			requestTxHash: txID,
			source:        common.RuneAsset(),
			target:        common.BNBAsset,
			amount:        cosmos.NewUint(100000000),
			requester:     bnbAddress,
			destination:   common.NoAddress,
			targetPrice:   cosmos.NewUint(200000000),
			signer:        addr,
		},
		{
			requestTxHash: txID,
			source:        common.RuneAsset(),
			target:        common.BNBAsset,
			amount:        cosmos.NewUint(100000000),
			requester:     bnbAddress,
			destination:   bnbAddress,
			targetPrice:   cosmos.NewUint(200000000),
			signer:        cosmos.AccAddress{},
		},
	}
	for _, item := range inputs {
		tx := common.NewTx(
			item.requestTxHash,
			item.requester,
			GetRandomBNBAddress(),
			common.Coins{
				common.NewCoin(item.source, item.amount),
			},
			BNBGasFeeSingleton,
			"SWAP:BNB.BNB",
		)

		m := NewMsgSwap(tx, item.target, item.destination, item.targetPrice, item.signer)
		c.Assert(m.ValidateBasic(), NotNil)
	}
}
