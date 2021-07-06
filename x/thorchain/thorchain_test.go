package thorchain

import (
	"sort"
	"testing"

	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/tss/go-tss/blame"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
)

func TestPackage(t *testing.T) { TestingT(t) }

var (
	bnbSingleTxFee = cosmos.NewUint(37500)
	bnbMultiTxFee  = cosmos.NewUint(30000)
)

// Gas Fees
var BNBGasFeeSingleton = common.Gas{
	{Asset: common.BNBAsset, Amount: bnbSingleTxFee},
}

var BNBGasFeeMulti = common.Gas{
	{Asset: common.BNBAsset, Amount: bnbMultiTxFee},
}

type ThorchainSuite struct{}

var _ = Suite(&ThorchainSuite{})

func (s *ThorchainSuite) TestStaking(c *C) {
	var err error
	ctx, keeper := setupKeeperForTest(c)
	user1 := GetRandomBNBAddress()
	user2 := GetRandomBNBAddress()
	txID := GetRandomTxHash()
	constAccessor := constants.GetConstantValues(constants.SWVersion)
	eventManager := NewDummyEventMgr()
	c.Assert(err, IsNil)

	// create bnb pool
	pool := NewPool()
	pool.Asset = common.BNBAsset
	c.Assert(keeper.SetPool(ctx, pool), IsNil)
	stakeHandler := NewStakeHandler(keeper, NewDummyMgr())
	// stake for user1
	err = stakeHandler.stake(ctx, common.BNBAsset, cosmos.NewUint(100*common.One), cosmos.NewUint(100*common.One), user1, user1, txID, constAccessor)
	c.Assert(err, IsNil)
	err = stakeHandler.stake(ctx, common.BNBAsset, cosmos.NewUint(100*common.One), cosmos.NewUint(100*common.One), user1, user1, txID, constAccessor)
	c.Assert(err, IsNil)
	staker1, err := keeper.GetStaker(ctx, common.BNBAsset, user1)
	c.Assert(err, IsNil)
	c.Check(staker1.Units.IsZero(), Equals, false)

	// stake for user2
	err = stakeHandler.stake(ctx, common.BNBAsset, cosmos.NewUint(75*common.One), cosmos.NewUint(75*common.One), user2, user2, txID, constAccessor)
	c.Assert(err, IsNil)
	err = stakeHandler.stake(ctx, common.BNBAsset, cosmos.NewUint(75*common.One), cosmos.NewUint(75*common.One), user2, user2, txID, constAccessor)
	c.Assert(err, IsNil)
	staker2, err := keeper.GetStaker(ctx, common.BNBAsset, user2)
	c.Assert(err, IsNil)
	c.Check(staker2.Units.IsZero(), Equals, false)

	version := constants.SWVersion
	// unstake for user1
	msg := NewMsgUnStake(GetRandomTx(), user1, cosmos.NewUint(10000), common.BNBAsset, GetRandomBech32Addr())
	_, _, _, _, err = unstake(ctx, version, keeper, msg, eventManager)
	c.Assert(err, IsNil)
	staker1, err = keeper.GetStaker(ctx, common.BNBAsset, user1)
	c.Assert(err, IsNil)
	c.Check(staker1.Units.IsZero(), Equals, true)

	// unstake for user2
	msg = NewMsgUnStake(GetRandomTx(), user2, cosmos.NewUint(10000), common.BNBAsset, GetRandomBech32Addr())
	_, _, _, _, err = unstake(ctx, version, keeper, msg, eventManager)
	c.Assert(err, IsNil)
	staker2, err = keeper.GetStaker(ctx, common.BNBAsset, user2)
	c.Assert(err, IsNil)
	c.Check(staker2.Units.IsZero(), Equals, true)

	// check pool is now empty
	pool, err = keeper.GetPool(ctx, common.BNBAsset)
	c.Assert(err, IsNil)
	c.Check(pool.BalanceRune.IsZero(), Equals, true)
	remainGas := uint64(75000)
	if common.RuneAsset().Chain.Equals(common.THORChain) {
		remainGas = 37500
	}
	c.Check(pool.BalanceAsset.Uint64(), Equals, uint64(remainGas)) // leave a little behind for gas
	c.Check(pool.PoolUnits.IsZero(), Equals, true)

	// stake for user1, again
	err = stakeHandler.stake(ctx, common.BNBAsset, cosmos.NewUint(100*common.One), cosmos.NewUint(100*common.One), user1, user1, txID, constAccessor)
	c.Assert(err, IsNil)
	err = stakeHandler.stake(ctx, common.BNBAsset, cosmos.NewUint(100*common.One), cosmos.NewUint(100*common.One), user1, user1, txID, constAccessor)
	c.Assert(err, IsNil)
	staker1, err = keeper.GetStaker(ctx, common.BNBAsset, user1)
	c.Assert(err, IsNil)
	c.Check(staker1.Units.IsZero(), Equals, false)

	// check pool is NOT empty
	pool, err = keeper.GetPool(ctx, common.BNBAsset)
	c.Assert(err, IsNil)
	c.Check(pool.BalanceRune.Equal(cosmos.NewUint(200*common.One)), Equals, true)
	c.Check(pool.BalanceAsset.Equal(cosmos.NewUint(20000000000+remainGas)), Equals, true, Commentf("%d", pool.BalanceAsset.Uint64()))
	c.Check(pool.PoolUnits.IsZero(), Equals, false)
}

func (s *ThorchainSuite) TestChurn(c *C) {
	ctx, keeper := setupKeeperForTest(c)
	ver := constants.SWVersion
	consts := constants.GetConstantValues(ver)
	mgr := NewManagers(keeper)
	c.Assert(mgr.BeginBlock(ctx), IsNil)
	// create starting point, vault and four node active node accounts
	vault := GetRandomVault()
	vault.AddFunds(common.Coins{
		common.NewCoin(common.RuneAsset(), cosmos.NewUint(100*common.One)),
		common.NewCoin(common.BNBAsset, cosmos.NewUint(79*common.One)),
	})
	addresses := make([]cosmos.AccAddress, 4)
	for i := 0; i <= 3; i++ {
		na := GetRandomNodeAccount(NodeActive)
		addresses[i] = na.NodeAddress
		na.SignerMembership = common.PubKeys{vault.PubKey}
		if i == 0 { // give the first node account slash points
			na.RequestedToLeave = true
		}
		vault.Membership = append(vault.Membership, na.PubKeySet.Secp256k1)
		c.Assert(keeper.SetNodeAccount(ctx, na), IsNil)
	}
	c.Assert(keeper.SetVault(ctx, vault), IsNil)

	// create new node account to rotate in
	na := GetRandomNodeAccount(NodeReady)
	c.Assert(keeper.SetNodeAccount(ctx, na), IsNil)

	// trigger marking bad actors as well as a keygen
	rotateHeight := consts.GetInt64Value(constants.RotatePerBlockHeight)
	ctx = ctx.WithBlockHeight(rotateHeight)
	c.Assert(mgr.ValidatorMgr().BeginBlock(ctx, consts), IsNil)

	// check we've created a keygen, with the correct members
	keygenBlock, err := keeper.GetKeygenBlock(ctx, common.BlockHeight(ctx))
	c.Assert(err, IsNil)
	c.Assert(keygenBlock.IsEmpty(), Equals, false)
	expected := append(vault.Membership[1:], na.PubKeySet.Secp256k1)
	c.Assert(keygenBlock.Keygens, HasLen, 1)
	keygen := keygenBlock.Keygens[0]
	// sort our slices so they are in the same order
	sort.Slice(expected, func(i, j int) bool { return expected[i].String() < expected[j].String() })
	sort.Slice(keygen.Members, func(i, j int) bool { return keygen.Members[i].String() < keygen.Members[j].String() })
	c.Assert(expected, HasLen, len(keygen.Members))
	for i := range expected {
		c.Assert(expected[i].Equals(keygen.Members[i]), Equals, true, Commentf("%d: %s <==> %s", i, expected[i], keygen.Members[i]))
	}

	// generate a tss keygen handler event
	newVaultPk := GetRandomPubKey()
	signer, err := keygen.Members[0].GetThorAddress()
	c.Assert(err, IsNil)
	msg := NewMsgTssPool(keygen.Members, newVaultPk, AsgardKeygen, common.BlockHeight(ctx), blame.Blame{}, common.Chains{common.RuneAsset().Chain}, signer)
	tssHandler := NewTssHandler(keeper, mgr)

	voter := NewTssVoter(msg.ID, msg.PubKeys, msg.PoolPubKey)
	signers := make([]cosmos.AccAddress, len(msg.PubKeys)-1)
	for i, pk := range msg.PubKeys {
		if i == 0 {
			continue
		}
		var err error
		signers[i-1], err = pk.GetThorAddress()
		c.Assert(err, IsNil)
	}
	voter.Signers = signers // ensure we have consensus, so handler is properly executed
	keeper.SetTssVoter(ctx, voter)

	_, err = tssHandler.Run(ctx, msg, ver, consts)
	c.Assert(err, IsNil)

	// check that we've rotated our vaults
	vault1, err := keeper.GetVault(ctx, vault.PubKey)
	c.Assert(err, IsNil)
	c.Assert(vault1.Status, Equals, RetiringVault) // first vault should now be retiring
	vault2, err := keeper.GetVault(ctx, newVaultPk)
	c.Assert(err, IsNil)
	c.Assert(vault2.Status, Equals, ActiveVault) // new vault should now be active
	c.Assert(vault2.Membership, HasLen, 4)

	// check our validators get rotated appropriately
	validators := mgr.ValidatorMgr().EndBlock(ctx, mgr, consts)
	nas, err := keeper.ListActiveNodeAccounts(ctx)
	c.Assert(err, IsNil)
	c.Assert(nas, HasLen, 4)
	c.Assert(validators, HasLen, 2)
	// ensure that the first one is rotated out and the new one is rotated in
	standby, err := keeper.GetNodeAccount(ctx, addresses[0])
	c.Assert(err, IsNil)
	c.Check(standby.Status == NodeDisabled, Equals, true)
	na, err = keeper.GetNodeAccount(ctx, na.NodeAddress)
	c.Assert(err, IsNil)
	c.Check(na.Status == NodeActive, Equals, true)

	// check that the funds can be migrated from the retiring vault to the new
	// vault
	ctx = ctx.WithBlockHeight(vault1.StatusSince)
	mgr.VaultMgr().EndBlock(ctx, mgr, consts) // should attempt to send 20% of the coin values
	vault, err = keeper.GetVault(ctx, vault1.PubKey)
	c.Assert(err, IsNil)
	items, err := mgr.TxOutStore().GetOutboundItems(ctx)
	c.Assert(err, IsNil)
	if common.RuneAsset().Chain.Equals(common.THORChain) {
		c.Assert(items, HasLen, 1)
		item := items[0]
		c.Check(item.Coin.Amount.Uint64(), Equals, uint64(1579925000), Commentf("%d", item.Coin.Amount.Uint64()))
	} else {
		c.Assert(items, HasLen, 2)
		item := items[0]
		c.Check(item.Coin.Amount.Uint64(), Equals, uint64(2000000000), Commentf("%d", item.Coin.Amount.Uint64()))
		item = items[1]
		c.Check(item.Coin.Amount.Uint64(), Equals, uint64(1579925000), Commentf("%d", item.Coin.Amount.Uint64()))
	}
	// check we empty the rest at the last migration event
	migrateInterval := consts.GetInt64Value(constants.FundMigrationInterval)
	ctx = ctx.WithBlockHeight(vault.StatusSince + (migrateInterval * 7))
	vault, err = keeper.GetVault(ctx, vault.PubKey)
	c.Assert(err, IsNil)
	vault.PendingTxBlockHeights = nil
	c.Assert(keeper.SetVault(ctx, vault), IsNil)
	mgr.VaultMgr().EndBlock(ctx, mgr, consts) // should attempt to send 100% of the coin values
	items, err = mgr.TxOutStore().GetOutboundItems(ctx)
	c.Assert(err, IsNil)
	if common.RuneAsset().Chain.Equals(common.THORChain) {
		c.Assert(items, HasLen, 1, Commentf("%d", len(items)))
		item := items[0]
		c.Check(item.Coin.Amount.Uint64(), Equals, uint64(7899925000), Commentf("%d", item.Coin.Amount.Uint64()))

	} else {
		c.Assert(items, HasLen, 2, Commentf("%d", len(items)))
		item := items[0]
		c.Check(item.Coin.Amount.Uint64(), Equals, uint64(10000000000), Commentf("%d", item.Coin.Amount.Uint64()))
		item = items[1]
		c.Check(item.Coin.Amount.Uint64(), Equals, uint64(7899925000), Commentf("%d", item.Coin.Amount.Uint64()))
	}
}

func (s *ThorchainSuite) TestRagnarok(c *C) {
	var err error
	ctx, keeper := setupKeeperForTest(c)
	ctx = ctx.WithBlockHeight(10)
	ver := constants.SWVersion
	consts := constants.GetConstantValues(ver)

	mgr := NewManagers(keeper)
	c.Assert(mgr.BeginBlock(ctx), IsNil)

	// create active asgard vault
	asgard := GetRandomVault()
	c.Assert(keeper.SetVault(ctx, asgard), IsNil)

	// create pools
	pool := NewPool()
	pool.Asset = common.BNBAsset
	c.Assert(keeper.SetPool(ctx, pool), IsNil)
	boltAsset, err := common.NewAsset("BNB.BOLT-123")
	c.Assert(err, IsNil)
	pool.Asset = boltAsset
	c.Assert(keeper.SetPool(ctx, pool), IsNil)
	stakeHandler := NewStakeHandler(keeper, NewDummyMgr())
	// add stakers
	staker1 := GetRandomRUNEAddress() // Staker1
	staker1asset := GetRandomBNBAddress()
	err = stakeHandler.stake(ctx, common.BNBAsset, cosmos.NewUint(100*common.One), cosmos.NewUint(10*common.One), staker1, staker1asset, GetRandomTxHash(), consts)
	c.Assert(err, IsNil)
	err = stakeHandler.stake(ctx, boltAsset, cosmos.NewUint(50*common.One), cosmos.NewUint(11*common.One), staker1, staker1asset, GetRandomTxHash(), consts)
	c.Assert(err, IsNil)
	staker2 := GetRandomRUNEAddress() // staker2
	staker2asset := GetRandomBNBAddress()
	err = stakeHandler.stake(ctx, common.BNBAsset, cosmos.NewUint(155*common.One), cosmos.NewUint(15*common.One), staker2, staker2asset, GetRandomTxHash(), consts)
	c.Assert(err, IsNil)
	err = stakeHandler.stake(ctx, boltAsset, cosmos.NewUint(20*common.One), cosmos.NewUint(4*common.One), staker2, staker2asset, GetRandomTxHash(), consts)
	c.Assert(err, IsNil)
	staker3 := GetRandomRUNEAddress() // staker3
	staker3asset := GetRandomBNBAddress()
	err = stakeHandler.stake(ctx, common.BNBAsset, cosmos.NewUint(155*common.One), cosmos.NewUint(15*common.One), staker3, staker3asset, GetRandomTxHash(), consts)
	c.Assert(err, IsNil)
	stakers := []common.Address{
		staker1, staker2, staker3,
	}
	stakersAssets := []common.Address{
		staker1asset, staker2asset, staker3asset,
	}

	// get new pool data
	bnbPool, err := keeper.GetPool(ctx, common.BNBAsset)
	c.Assert(err, IsNil)
	boltPool, err := keeper.GetPool(ctx, boltAsset)
	c.Assert(err, IsNil)

	// Add bonders/validators
	bonderCount := 3
	bonders := make(NodeAccounts, bonderCount)
	for i := 1; i <= bonderCount; i++ {
		na := GetRandomNodeAccount(NodeActive)
		na.Bond = cosmos.NewUint(1_000_000 * uint64(i) * common.One)
		c.Assert(keeper.SetNodeAccount(ctx, na), IsNil)
		bonders[i-1] = na

		// Add bond to asgard
		asgard.AddFunds(common.Coins{
			common.NewCoin(common.RuneAsset(), na.Bond),
		})
		c.Assert(keeper.SetVault(ctx, asgard), IsNil)

		// create yggdrasil vault, with 1/3 of the staked funds
		ygg := GetRandomVault()
		ygg.PubKey = na.PubKeySet.Secp256k1
		ygg.Type = YggdrasilVault
		ygg.AddFunds(common.Coins{
			common.NewCoin(common.RuneAsset(), bnbPool.BalanceRune.QuoUint64(uint64(bonderCount))),
			common.NewCoin(common.BNBAsset, bnbPool.BalanceAsset.QuoUint64(uint64(bonderCount))),
			common.NewCoin(common.RuneAsset(), boltPool.BalanceRune.QuoUint64(uint64(bonderCount))),
			common.NewCoin(boltAsset, boltPool.BalanceAsset.QuoUint64(uint64(bonderCount))),
		})
		c.Assert(keeper.SetVault(ctx, ygg), IsNil)
	}

	// Add reserve contributors
	reserves := ReserveContributors{}
	if !common.RuneAsset().Chain.Equals(common.THORChain) {
		contrib1 := GetRandomBNBAddress()
		contrib2 := GetRandomBNBAddress()
		reserves = ReserveContributors{
			NewReserveContributor(contrib1, cosmos.NewUint(400_000_000*common.One)),
			NewReserveContributor(contrib2, cosmos.NewUint(100_000*common.One)),
		}
		resHandler := NewReserveContributorHandler(keeper, mgr)
		for _, res := range reserves {
			asgard.AddFunds(common.Coins{
				common.NewCoin(common.RuneAsset(), res.Amount),
			})
			msg := NewMsgReserveContributor(GetRandomTx(), res, bonders[0].NodeAddress)
			result, err := resHandler.handle(ctx, msg, ver)
			c.Assert(err, IsNil)
			c.Assert(result, NotNil)
		}
		c.Assert(keeper.SetVault(ctx, asgard), IsNil)
	}

	// ////////////////////////////////////////////////////////
	// ////////////// Start Ragnarok Protocol /////////////////
	// ////////////////////////////////////////////////////////
	vd := VaultData{
		BondRewardRune: cosmos.NewUint(1000_000 * common.One),
		TotalBondUnits: cosmos.NewUint(3 * 1014), // block height * node count
		TotalReserve:   cosmos.NewUint(400_100_000 * common.One),
	}
	c.Assert(keeper.SetVaultData(ctx, vd), IsNil)
	ctx = ctx.WithBlockHeight(1024)

	active, err := keeper.ListActiveNodeAccounts(ctx)
	c.Assert(err, IsNil)
	// this should trigger stage 1 of the ragnarok protocol. We should see a tx
	// out per node account
	mgr.ValidatorMgr().processRagnarok(ctx, mgr, consts)
	// after ragnarok get trigged , we pay bond reward immediately
	for idx, bonder := range bonders {
		na, err := keeper.GetNodeAccount(ctx, bonder.NodeAddress)
		c.Assert(err, IsNil)
		bonders[idx].Bond = na.Bond
	}
	// make sure we have enough yggdrasil returns
	items, err := mgr.TxOutStore().GetOutboundItems(ctx)
	c.Assert(err, IsNil)
	c.Assert(items, HasLen, bonderCount)
	for _, item := range items {
		c.Assert(item.Coin.Equals(common.NewCoin(common.RuneAsset(), cosmos.ZeroUint())), Equals, true)
	}

	// we'll assume the signer does it's job and sends the yggdrasil funds back
	// to asgard, and do it ourselves here manually
	for _, na := range active {
		ygg, err := keeper.GetVault(ctx, na.PubKeySet.Secp256k1)
		c.Assert(err, IsNil)
		asgard.AddFunds(ygg.Coins)
		c.Assert(keeper.SetVault(ctx, asgard), IsNil)
		ygg.SubFunds(ygg.Coins)
		c.Assert(keeper.SetVault(ctx, ygg), IsNil)
	}
	mgr.TxOutStore().ClearOutboundItems(ctx) // clear out txs

	// run stage 2 of ragnarok protocol, nth = 1
	for i := 1; i <= 20; i++ { // simulate each round of ragnarok (max of ten)
		c.Assert(mgr.ValidatorMgr().processRagnarok(ctx, mgr, consts), IsNil)
		items, err := mgr.TxOutStore().GetOutboundItems(ctx)
		c.Assert(err, IsNil)

		if common.RuneAsset().Chain.Equals(common.THORChain) {
			if i <= 10 {
				c.Assert(items, HasLen, 2, Commentf("%d", len(items)))
			} else {
				c.Assert(items, HasLen, 3, Commentf("%d", len(items)))
			}
		} else {
			if i <= 10 || i == 20 {
				c.Assert(items, HasLen, 6, Commentf("%d", len(items)))
			} else {
				c.Assert(items, HasLen, 9, Commentf("%d", len(items)))
			}
		}

		if !common.RuneAsset().Chain.Equals(common.THORChain) {
			// validate bonders have correct coin amounts being sent to them on each round of ragnarok
			if i > 10 && i < 20 {
				for _, bonder := range bonders {
					items := mgr.TxOutStore().GetOutboundItemByToAddress(ctx, bonder.BondAddress)
					c.Assert(items, HasLen, 1)
					outCoin := common.NewCoin(common.RuneAsset(), calcExpectedValue(bonder.Bond, i-10, 9))
					c.Assert(items[0].Coin.Equals(outCoin), Equals, true, Commentf("expect:%s, however:%s", outCoin.String(), items[0].Coin.String()))
				}
			}
		}

		// validate stakers get their returns
		for j, staker := range stakersAssets {
			items := mgr.TxOutStore().GetOutboundItemByToAddress(ctx, staker)
			// TODO: check item amounts
			if i <= 10 {
				if j == len(stakers)-1 {
					c.Assert(items, HasLen, 0, Commentf("%d", len(items)))
				} else {
					c.Assert(items, HasLen, 1, Commentf("%d", len(items)))
				}
			} else {
				c.Assert(items, HasLen, 1, Commentf("%d", len(items)))
			}
		}

		// validate reserve contributors get their returns
		if i <= 10 {
			for _, res := range reserves {
				items := mgr.TxOutStore().GetOutboundItemByToAddress(ctx, res.Address)
				c.Assert(items, HasLen, 1)
				outCoin := common.NewCoin(common.RuneAsset(), calcExpectedValue(res.Amount, i, 10))
				c.Assert(items[0].Coin.Equals(outCoin), Equals, true, Commentf("expect:%s, however:%s", outCoin, items[0].Coin))
			}
		}

		mgr.TxOutStore().ClearOutboundItems(ctx) // clear out txs
		keeper.SetRagnarokPending(ctx, 0)
		items, err = mgr.TxOutStore().GetOutboundItems(ctx)
		c.Assert(items, HasLen, 0)
		c.Assert(err, IsNil)
	}
}

// calculate the expected coin amount taken from a original amount at nth round
// of ragnarok
func calcExpectedValue(total cosmos.Uint, nth, limit int) cosmos.Uint {
	var amt cosmos.Uint
	for i := uint64(1); i <= uint64(nth); i++ {
		if i == uint64(limit) {
			i = 10
		}
		amt = total.MulUint64(i).QuoUint64(10)
		total = total.Sub(amt)
	}
	return amt
}

func (s *ThorchainSuite) TestRagnarokNoOneLeave(c *C) {
	var err error
	ctx, keeper := setupKeeperForTest(c)
	ctx = ctx.WithBlockHeight(10)
	ver := constants.SWVersion
	consts := constants.GetConstantValues(ver)

	mgr := NewManagers(keeper)
	c.Assert(mgr.BeginBlock(ctx), IsNil)

	// create active asgard vault
	asgard := GetRandomVault()
	c.Assert(keeper.SetVault(ctx, asgard), IsNil)

	// create pools
	pool := NewPool()
	pool.Asset = common.BNBAsset
	c.Assert(keeper.SetPool(ctx, pool), IsNil)
	boltAsset, err := common.NewAsset("BNB.BOLT-123")
	c.Assert(err, IsNil)
	pool.Asset = boltAsset
	c.Assert(keeper.SetPool(ctx, pool), IsNil)
	stakeHandler := NewStakeHandler(keeper, NewDummyMgr())
	// add stakers
	staker1 := GetRandomRUNEAddress() // Staker1
	err = stakeHandler.stake(ctx, common.BNBAsset, cosmos.NewUint(100*common.One), cosmos.NewUint(10*common.One), staker1, staker1, GetRandomTxHash(), consts)
	c.Assert(err, IsNil)
	err = stakeHandler.stake(ctx, boltAsset, cosmos.NewUint(50*common.One), cosmos.NewUint(11*common.One), staker1, staker1, GetRandomTxHash(), consts)
	c.Assert(err, IsNil)
	staker2 := GetRandomRUNEAddress() // staker2
	err = stakeHandler.stake(ctx, common.BNBAsset, cosmos.NewUint(155*common.One), cosmos.NewUint(15*common.One), staker2, staker2, GetRandomTxHash(), consts)
	c.Assert(err, IsNil)
	err = stakeHandler.stake(ctx, boltAsset, cosmos.NewUint(20*common.One), cosmos.NewUint(4*common.One), staker2, staker2, GetRandomTxHash(), consts)
	c.Assert(err, IsNil)
	staker3 := GetRandomRUNEAddress() // staker3
	err = stakeHandler.stake(ctx, common.BNBAsset, cosmos.NewUint(155*common.One), cosmos.NewUint(15*common.One), staker3, staker3, GetRandomTxHash(), consts)
	c.Assert(err, IsNil)
	stakers := []common.Address{
		staker1, staker2, staker3,
	}
	_ = stakers

	// get new pool data
	bnbPool, err := keeper.GetPool(ctx, common.BNBAsset)
	c.Assert(err, IsNil)
	boltPool, err := keeper.GetPool(ctx, boltAsset)
	c.Assert(err, IsNil)

	// Add bonders/validators
	bonderCount := 4
	bonders := make(NodeAccounts, bonderCount)
	for i := 1; i <= bonderCount; i++ {
		na := GetRandomNodeAccount(NodeActive)
		na.Bond = cosmos.NewUint(1_000_000 * uint64(i) * common.One)
		c.Assert(keeper.SetNodeAccount(ctx, na), IsNil)
		bonders[i-1] = na

		// Add bond to asgard
		asgard.AddFunds(common.Coins{
			common.NewCoin(common.RuneAsset(), na.Bond),
		})
		asgard.Membership = append(asgard.Membership, na.PubKeySet.Secp256k1)
		c.Assert(keeper.SetVault(ctx, asgard), IsNil)

		// create yggdrasil vault, with 1/3 of the staked funds
		ygg := GetRandomVault()
		ygg.PubKey = na.PubKeySet.Secp256k1
		ygg.Type = YggdrasilVault
		ygg.AddFunds(common.Coins{
			common.NewCoin(common.RuneAsset(), bnbPool.BalanceRune.QuoUint64(uint64(bonderCount))),
			common.NewCoin(common.BNBAsset, bnbPool.BalanceAsset.QuoUint64(uint64(bonderCount))),
			common.NewCoin(common.RuneAsset(), boltPool.BalanceRune.QuoUint64(uint64(bonderCount))),
			common.NewCoin(boltAsset, boltPool.BalanceAsset.QuoUint64(uint64(bonderCount))),
		})
		c.Assert(keeper.SetVault(ctx, ygg), IsNil)

	}

	// Add reserve contributors
	contrib1 := GetRandomBNBAddress()
	contrib2 := GetRandomBNBAddress()
	reserves := ReserveContributors{
		NewReserveContributor(contrib1, cosmos.NewUint(400_000_000*common.One)),
		NewReserveContributor(contrib2, cosmos.NewUint(100_000*common.One)),
	}
	resHandler := NewReserveContributorHandler(keeper, mgr)
	for _, res := range reserves {
		asgard.AddFunds(common.Coins{
			common.NewCoin(common.RuneAsset(), res.Amount),
		})
		msg := NewMsgReserveContributor(GetRandomTx(), res, bonders[0].NodeAddress)
		_, err := resHandler.handle(ctx, msg, ver)
		c.Assert(err, IsNil)
	}
	c.Assert(keeper.SetVault(ctx, asgard), IsNil)
	asgard.Membership = asgard.Membership[:len(asgard.Membership)-1]
	c.Assert(keeper.SetVault(ctx, asgard), IsNil)
	// no validator should leave, because it trigger ragnarok
	updates := mgr.ValidatorMgr().EndBlock(ctx, mgr, consts)
	c.Assert(updates, IsNil)
	ragnarokHeight, err := keeper.GetRagnarokBlockHeight(ctx)
	c.Assert(err, IsNil)
	c.Assert(ragnarokHeight, Equals, common.BlockHeight(ctx))
	currentHeight := common.BlockHeight(ctx)
	migrateInterval := consts.GetInt64Value(constants.FundMigrationInterval)
	ctx = ctx.WithBlockHeight(currentHeight + migrateInterval)
	c.Assert(mgr.ValidatorMgr().BeginBlock(ctx, consts), IsNil)
	mgr.TxOutStore().ClearOutboundItems(ctx)
	c.Assert(mgr.ValidatorMgr().EndBlock(ctx, mgr, consts), IsNil)
}
