package thorchain

import (
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
)

type TxOutStoreSuite struct{}

var _ = Suite(&TxOutStoreSuite{})

func (s TxOutStoreSuite) TestAddGasFees(c *C) {
	ctx, k := setupKeeperForTest(c)
	tx := GetRandomObservedTx()

	gasMgr := NewGasMgrV1()
	err := AddGasFees(ctx, k, tx, gasMgr)
	c.Assert(err, IsNil)
	c.Assert(gasMgr.gas, HasLen, 1)
}

func (s TxOutStoreSuite) TestAddOutTxItem(c *C) {
	w := getHandlerTestWrapper(c, 1, true, true)
	vault := GetRandomVault()
	vault.Coins = common.Coins{
		common.NewCoin(common.RuneAsset(), cosmos.NewUint(10000*common.One)),
		common.NewCoin(common.BNBAsset, cosmos.NewUint(10000*common.One)),
	}
	w.keeper.SetVault(w.ctx, vault)

	acc1 := GetRandomNodeAccount(NodeActive)
	acc2 := GetRandomNodeAccount(NodeActive)
	acc3 := GetRandomNodeAccount(NodeActive)
	c.Assert(w.keeper.SetNodeAccount(w.ctx, acc1), IsNil)
	c.Assert(w.keeper.SetNodeAccount(w.ctx, acc2), IsNil)
	c.Assert(w.keeper.SetNodeAccount(w.ctx, acc3), IsNil)

	ygg := NewVault(common.BlockHeight(w.ctx), ActiveVault, YggdrasilVault, acc1.PubKeySet.Secp256k1, common.Chains{common.BNBChain})
	ygg.AddFunds(
		common.Coins{
			common.NewCoin(common.BNBAsset, cosmos.NewUint(40*common.One)),
		},
	)
	c.Assert(w.keeper.SetVault(w.ctx, ygg), IsNil)

	ygg = NewVault(common.BlockHeight(w.ctx), ActiveVault, YggdrasilVault, acc2.PubKeySet.Secp256k1, common.Chains{common.BNBChain})
	ygg.AddFunds(
		common.Coins{
			common.NewCoin(common.BNBAsset, cosmos.NewUint(50*common.One)),
		},
	)
	c.Assert(w.keeper.SetVault(w.ctx, ygg), IsNil)

	ygg = NewVault(common.BlockHeight(w.ctx), ActiveVault, YggdrasilVault, acc3.PubKeySet.Secp256k1, common.Chains{common.BNBChain})
	ygg.AddFunds(
		common.Coins{
			common.NewCoin(common.BNBAsset, cosmos.NewUint(100*common.One)),
		},
	)
	c.Assert(w.keeper.SetVault(w.ctx, ygg), IsNil)

	// Create voter
	inTxID := GetRandomTxHash()
	voter := NewObservedTxVoter(inTxID, ObservedTxs{
		ObservedTx{
			Signers: []cosmos.AccAddress{w.activeNodeAccount.NodeAddress, acc1.NodeAddress, acc2.NodeAddress},
		},
	})
	w.keeper.SetObservedTxInVoter(w.ctx, voter)

	// Should get acc2. Acc3 hasn't signed and acc2 is the highest value
	item := &TxOutItem{
		Chain:     common.BNBChain,
		ToAddress: GetRandomBNBAddress(),
		InHash:    inTxID,
		Coin:      common.NewCoin(common.BNBAsset, cosmos.NewUint(20*common.One)),
	}
	txOutStore := w.mgr.TxOutStore()
	txOutStore.TryAddTxOutItem(w.ctx, w.mgr, item)
	msgs, err := txOutStore.GetOutboundItems(w.ctx)
	c.Assert(err, IsNil)
	c.Assert(msgs, HasLen, 1)
	c.Assert(msgs[0].VaultPubKey.String(), Equals, acc2.PubKeySet.Secp256k1.String())
	c.Assert(msgs[0].Coin.Amount.Equal(cosmos.NewUint(19*common.One)), Equals, true)

	// Should get acc1. Acc3 hasn't signed and acc1 now has the highest amount
	// of coin.
	item = &TxOutItem{
		Chain:     common.BNBChain,
		ToAddress: GetRandomBNBAddress(),
		InHash:    inTxID,
		Coin:      common.NewCoin(common.BNBAsset, cosmos.NewUint(20*common.One)),
	}
	success, err := txOutStore.TryAddTxOutItem(w.ctx, w.mgr, item)
	c.Assert(success, Equals, true)
	c.Assert(err, IsNil)
	msgs, err = txOutStore.GetOutboundItems(w.ctx)
	c.Assert(err, IsNil)
	c.Assert(msgs, HasLen, 2)
	c.Assert(msgs[1].VaultPubKey.String(), Equals, acc1.PubKeySet.Secp256k1.String())

	item = &TxOutItem{
		Chain:     common.BNBChain,
		ToAddress: GetRandomBNBAddress(),
		InHash:    inTxID,
		Coin:      common.NewCoin(common.BNBAsset, cosmos.NewUint(1000*common.One)),
	}
	success, err = txOutStore.TryAddTxOutItem(w.ctx, w.mgr, item)
	c.Assert(err, IsNil)
	c.Assert(success, Equals, true)
	msgs, err = txOutStore.GetOutboundItems(w.ctx)
	c.Assert(err, IsNil)
	c.Assert(msgs, HasLen, 3)
	c.Assert(msgs[2].VaultPubKey.String(), Equals, vault.PubKey.String())
}

func (s TxOutStoreSuite) TestAddOutTxItemWithoutBFT(c *C) {
	w := getHandlerTestWrapper(c, 1, true, true)
	vault := GetRandomVault()
	vault.Coins = common.Coins{
		common.NewCoin(common.BNBAsset, cosmos.NewUint(100*common.One)),
	}
	w.keeper.SetVault(w.ctx, vault)

	inTxID := GetRandomTxHash()
	item := &TxOutItem{
		Chain:     common.BNBChain,
		ToAddress: GetRandomBNBAddress(),
		InHash:    inTxID,
		Coin:      common.NewCoin(common.BNBAsset, cosmos.NewUint(20*common.One)),
	}
	txOutStore := w.mgr.TxOutStore()
	success, err := txOutStore.TryAddTxOutItem(w.ctx, w.mgr, item)
	c.Assert(err, IsNil)
	c.Assert(success, Equals, true)
	msgs, err := txOutStore.GetOutboundItems(w.ctx)
	c.Assert(err, IsNil)
	c.Assert(msgs, HasLen, 1)
	c.Assert(msgs[0].Coin.Amount.Equal(cosmos.NewUint(19*common.One)), Equals, true)
}
