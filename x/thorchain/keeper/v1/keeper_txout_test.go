package keeperv1

import (
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

type KeeperTxOutSuite struct{}

var _ = Suite(&KeeperTxOutSuite{})

func (KeeperTxOutSuite) TestKeeperTxOut(c *C) {
	ctx, k := setupKeeperForTest(c)
	txOut := NewTxOut(1)
	txOutItem := &TxOutItem{
		Chain:       common.BNBChain,
		ToAddress:   GetRandomBNBAddress(),
		VaultPubKey: GetRandomPubKey(),
		Coin:        common.NewCoin(common.BNBAsset, cosmos.NewUint(100*common.One)),
		Memo:        "hello",
	}
	txOut.TxArray = append(txOut.TxArray, txOutItem)
	c.Assert(k.SetTxOut(ctx, txOut), IsNil)
	txOut1, err := k.GetTxOut(ctx, 1)
	c.Assert(err, IsNil)
	c.Assert(txOut1, NotNil)
	c.Assert(txOut1.Height, Equals, int64(1))

	txOut2, err := k.GetTxOut(ctx, 100)
	c.Assert(err, IsNil)
	c.Assert(txOut2, NotNil)

	c.Check(k.AppendTxOut(ctx, 100, txOutItem), IsNil)

	iter := k.GetTxOutIterator(ctx)
	c.Check(iter, NotNil)
	defer iter.Close()
	c.Check(k.ClearTxOut(ctx, 100), IsNil)

	txOut3 := NewTxOut(1024)
	c.Check(k.SetTxOut(ctx, txOut3), IsNil)
}
