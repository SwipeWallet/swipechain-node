package types

import (
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

type MsgStakeSuite struct{}

var _ = Suite(&MsgStakeSuite{})

func (MsgStakeSuite) TestMsgStake(c *C) {
	addr := GetRandomBech32Addr()
	c.Check(addr.Empty(), Equals, false)
	runeAddress := GetRandomRUNEAddress()
	assetAddress := GetRandomBNBAddress()
	txID := GetRandomTxHash()
	c.Check(txID.IsEmpty(), Equals, false)
	tx := common.NewTx(
		txID,
		runeAddress,
		GetRandomRUNEAddress(),
		common.Coins{
			common.NewCoin(common.BTCAsset, cosmos.NewUint(100000000)),
		},
		BNBGasFeeSingleton,
		"",
	)
	m := NewMsgStake(tx, common.BNBAsset, cosmos.NewUint(100000000), cosmos.NewUint(100000000), runeAddress, assetAddress, addr)
	EnsureMsgBasicCorrect(m, c)
	c.Check(m.Type(), Equals, "stake")

	inputs := []struct {
		asset     common.Asset
		r         cosmos.Uint
		amt       cosmos.Uint
		runeAddr  common.Address
		assetAddr common.Address
		txHash    common.TxID
		signer    cosmos.AccAddress
	}{
		{
			asset:     common.Asset{},
			r:         cosmos.NewUint(100000000),
			amt:       cosmos.NewUint(100000000),
			runeAddr:  runeAddress,
			assetAddr: assetAddress,
			txHash:    txID,
			signer:    addr,
		},
		{
			asset:     common.BNBAsset,
			r:         cosmos.NewUint(100000000),
			amt:       cosmos.NewUint(100000000),
			runeAddr:  common.NoAddress,
			assetAddr: common.NoAddress,
			txHash:    txID,
			signer:    addr,
		},
		{
			asset:     common.BNBAsset,
			r:         cosmos.NewUint(100000000),
			amt:       cosmos.NewUint(100000000),
			runeAddr:  runeAddress,
			assetAddr: assetAddress,
			txHash:    common.TxID(""),
			signer:    addr,
		},
		{
			asset:     common.BNBAsset,
			r:         cosmos.NewUint(100000000),
			amt:       cosmos.NewUint(100000000),
			runeAddr:  runeAddress,
			assetAddr: assetAddress,
			txHash:    txID,
			signer:    cosmos.AccAddress{},
		},
		{
			asset:     common.BNBAsset,
			r:         cosmos.NewUint(100000000),
			amt:       cosmos.NewUint(100000000),
			runeAddr:  "",
			assetAddr: assetAddress,
			txHash:    txID,
			signer:    addr,
		},
		{
			asset:     common.BTCAsset,
			r:         cosmos.NewUint(100000000),
			amt:       cosmos.NewUint(100000000),
			runeAddr:  runeAddress,
			assetAddr: "",
			txHash:    txID,
			signer:    addr,
		},
	}
	for i, item := range inputs {
		tx := common.NewTx(
			item.txHash,
			GetRandomRUNEAddress(),
			GetRandomBNBAddress(),
			common.Coins{
				common.NewCoin(item.asset, item.r),
			},
			BNBGasFeeSingleton,
			"",
		)
		m := NewMsgStake(tx, item.asset, item.r, item.amt, item.runeAddr, item.assetAddr, item.signer)
		c.Assert(m.ValidateBasic(), NotNil, Commentf("%d) %s\n", i, m))
	}
}
