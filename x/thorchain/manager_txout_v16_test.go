package thorchain

import (
	"github.com/blang/semver"
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

type TxOutStoreV16Suite struct{}

var _ = Suite(&TxOutStoreV16Suite{})

func (s TxOutStoreV16Suite) TestAddGasFees(c *C) {
	ctx, k := setupKeeperForTest(c)
	tx := GetRandomObservedTx()

	gasMgr := NewGasMgrV1()
	err := AddGasFees(ctx, k, tx, gasMgr)
	c.Assert(err, IsNil)
	c.Assert(gasMgr.gas, HasLen, 1)
}

func (s TxOutStoreV16Suite) TestAddOutTxItem(c *C) {
	version := semver.MustParse("0.16.0")
	w := getHandlerTestWrapperWithVersion(c, 1, true, true, version)
	vault := GetRandomVault()
	vault.Coins = common.Coins{
		common.NewCoin(common.RuneAsset(), cosmos.NewUint(10000*common.One)),
		common.NewCoin(common.BNBAsset, cosmos.NewUint(10000*common.One)),
	}
	w.keeper.SetVault(w.ctx, vault)

	acc1 := GetRandomNodeAccount(NodeActive)
	acc1.Version = version
	acc2 := GetRandomNodeAccount(NodeActive)
	acc2.Version = version
	acc3 := GetRandomNodeAccount(NodeActive)
	acc3.Version = version
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

func (s TxOutStoreV16Suite) TestAddOutTxItemWithoutBFT(c *C) {
	version := semver.MustParse("0.16.0")
	w := getHandlerTestWrapperWithVersion(c, 1, true, true, version)
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

func (s TxOutStoreV16Suite) TestAsgardDeductedOutstandingBalance(c *C) {
	version := semver.MustParse("0.16.0")
	w := getHandlerTestWrapperWithVersion(c, 1024, true, true, version)
	vault := GetRandomVault()
	vault.Coins = common.Coins{
		common.NewCoin(common.RuneAsset(), cosmos.NewUint(100*common.One)),
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
	newHeight := w.ctx.BlockHeight() + 1
	w.ctx = w.ctx.WithBlockHeight(newHeight)

	// asgard vault has an outstanding tx out item , it doesn't have enough to fulfil the outbound
	item1 := &TxOutItem{
		Chain:     common.BNBChain,
		ToAddress: GetRandomBNBAddress(),
		InHash:    GetRandomTxHash(),
		Coin:      common.NewCoin(common.BNBAsset, cosmos.NewUint(90*common.One)),
	}
	success, err = txOutStore.TryAddTxOutItem(w.ctx, w.mgr, item1)
	c.Assert(err, NotNil)
	c.Assert(success, Equals, false)
}
