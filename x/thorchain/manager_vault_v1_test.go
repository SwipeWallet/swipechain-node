package thorchain

import (
	"errors"

	"github.com/blang/semver"
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
	"gitlab.com/thorchain/thornode/x/thorchain/keeper"
)

type VaultManagerV1TestSuite struct{}

var _ = Suite(&VaultManagerV1TestSuite{})

func (s *VaultManagerV1TestSuite) SetUpSuite(c *C) {
	SetupConfigForTest()
}

type TestRagnarokChainKeeper struct {
	keeper.KVStoreDummy
	activeVault Vault
	retireVault Vault
	yggVault    Vault
	pools       Pools
	stakers     []Staker
	na          NodeAccount
	err         error
}

func (k *TestRagnarokChainKeeper) ListNodeAccountsWithBond(_ cosmos.Context) (NodeAccounts, error) {
	return NodeAccounts{k.na}, k.err
}

func (k *TestRagnarokChainKeeper) ListActiveNodeAccounts(_ cosmos.Context) (NodeAccounts, error) {
	return NodeAccounts{k.na}, k.err
}

func (k *TestRagnarokChainKeeper) GetNodeAccount(ctx cosmos.Context, signer cosmos.AccAddress) (NodeAccount, error) {
	if k.na.NodeAddress.Equals(signer) {
		return k.na, nil
	}
	return NodeAccount{}, nil
}

func (k *TestRagnarokChainKeeper) GetAsgardVaultsByStatus(_ cosmos.Context, vt VaultStatus) (Vaults, error) {
	if vt == ActiveVault {
		return Vaults{k.activeVault}, k.err
	}
	return Vaults{k.retireVault}, k.err
}

func (k *TestRagnarokChainKeeper) VaultExists(_ cosmos.Context, _ common.PubKey) bool {
	return true
}

func (k *TestRagnarokChainKeeper) GetVault(_ cosmos.Context, _ common.PubKey) (Vault, error) {
	return k.yggVault, k.err
}

func (k *TestRagnarokChainKeeper) GetPools(_ cosmos.Context) (Pools, error) {
	return k.pools, k.err
}

func (k *TestRagnarokChainKeeper) GetPool(_ cosmos.Context, asset common.Asset) (Pool, error) {
	for _, pool := range k.pools {
		if pool.Asset.Equals(asset) {
			return pool, nil
		}
	}
	return Pool{}, errors.New("pool not found")
}

func (k *TestRagnarokChainKeeper) SetPool(_ cosmos.Context, pool Pool) error {
	for i, p := range k.pools {
		if p.Asset.Equals(pool.Asset) {
			k.pools[i] = pool
		}
	}
	return k.err
}

func (k *TestRagnarokChainKeeper) PoolExist(_ cosmos.Context, _ common.Asset) bool {
	return true
}

func (k *TestRagnarokChainKeeper) GetStakerIterator(ctx cosmos.Context, _ common.Asset) cosmos.Iterator {
	cdc := makeTestCodec()
	iter := keeper.NewDummyIterator()
	for _, staker := range k.stakers {
		iter.AddItem([]byte("key"), cdc.MustMarshalBinaryBare(staker))
	}
	return iter
}

func (k *TestRagnarokChainKeeper) GetStaker(_ cosmos.Context, asset common.Asset, addr common.Address) (Staker, error) {
	if asset.Equals(common.BTCAsset) {
		for i, staker := range k.stakers {
			if addr.Equals(staker.RuneAddress) {
				return k.stakers[i], k.err
			}
		}
	}
	return Staker{}, k.err
}

func (k *TestRagnarokChainKeeper) SetStaker(_ cosmos.Context, staker Staker) {
	for i, skr := range k.stakers {
		if staker.RuneAddress.Equals(skr.RuneAddress) {
			k.stakers[i] = staker
		}
	}
}

func (k *TestRagnarokChainKeeper) RemoveStaker(_ cosmos.Context, staker Staker) {
	for i, skr := range k.stakers {
		if staker.RuneAddress.Equals(skr.RuneAddress) {
			k.stakers[i] = staker
		}
	}
}

func (k *TestRagnarokChainKeeper) GetGas(_ cosmos.Context, _ common.Asset) ([]cosmos.Uint, error) {
	return []cosmos.Uint{cosmos.NewUint(10)}, k.err
}

func (k *TestRagnarokChainKeeper) GetLowestActiveVersion(_ cosmos.Context) semver.Version {
	return constants.SWVersion
}

func (k *TestRagnarokChainKeeper) AddFeeToReserve(_ cosmos.Context, _ cosmos.Uint) error {
	return k.err
}

func (k *TestRagnarokChainKeeper) IsActiveObserver(_ cosmos.Context, _ cosmos.AccAddress) bool {
	return true
}

func (s *VaultManagerV1TestSuite) TestRagnarokChain(c *C) {
	ctx, k := setupKeeperForTest(c)
	ctx = ctx.WithBlockHeight(100000)
	ver := constants.SWVersion
	constAccessor := constants.GetConstantValues(ver)

	activeVault := GetRandomVault()
	activeVault.Coins = common.Coins{
		common.NewCoin(common.BNBAsset, cosmos.NewUint(100*common.One)),
	}
	retireVault := GetRandomVault()
	retireVault.Chains = common.Chains{common.BNBChain, common.BTCChain}
	yggVault := GetRandomVault()
	yggVault.Type = YggdrasilVault
	yggVault.Coins = common.Coins{
		common.NewCoin(common.BTCAsset, cosmos.NewUint(3*common.One)),
		common.NewCoin(common.RuneAsset(), cosmos.NewUint(300*common.One)),
	}

	btcPool := NewPool()
	btcPool.Asset = common.BTCAsset
	btcPool.BalanceRune = cosmos.NewUint(1000 * common.One)
	btcPool.BalanceAsset = cosmos.NewUint(10 * common.One)
	btcPool.PoolUnits = cosmos.NewUint(1600)

	bnbPool := NewPool()
	bnbPool.Asset = common.BNBAsset
	bnbPool.BalanceRune = cosmos.NewUint(1000 * common.One)
	bnbPool.BalanceAsset = cosmos.NewUint(10 * common.One)
	bnbPool.PoolUnits = cosmos.NewUint(1600)

	addr := GetRandomRUNEAddress()
	stakers := []Staker{
		{
			RuneAddress:     addr,
			LastStakeHeight: 5,
			Units:           btcPool.PoolUnits.QuoUint64(2),
			PendingRune:     cosmos.ZeroUint(),
		},
		{
			RuneAddress:     GetRandomRUNEAddress(),
			LastStakeHeight: 10,
			Units:           btcPool.PoolUnits.QuoUint64(2),
			PendingRune:     cosmos.ZeroUint(),
		},
	}

	keeper := &TestRagnarokChainKeeper{
		na:          GetRandomNodeAccount(NodeActive),
		activeVault: activeVault,
		retireVault: retireVault,
		yggVault:    yggVault,
		pools:       Pools{bnbPool, btcPool},
		stakers:     stakers,
	}

	mgr := NewDummyMgr()

	vaultMgr := NewVaultMgrV1(keeper, mgr.TxOutStore(), mgr.EventMgr())

	err := vaultMgr.manageChains(ctx, mgr, constAccessor)
	c.Assert(err, IsNil)
	c.Check(keeper.pools[1].Asset.Equals(common.BTCAsset), Equals, true)
	c.Check(keeper.pools[1].PoolUnits.IsZero(), Equals, true, Commentf("%d\n", keeper.pools[1].PoolUnits.Uint64()))
	c.Check(keeper.pools[0].PoolUnits.Equal(cosmos.NewUint(1600)), Equals, true)
	for _, skr := range keeper.stakers {
		c.Check(skr.Units.IsZero(), Equals, true)
	}

	// ensure we have requested for ygg funds to be returned
	txOutStore := mgr.TxOutStore()
	c.Assert(err, IsNil)
	items, err := txOutStore.GetOutboundItems(ctx)
	c.Assert(err, IsNil)

	// 1 ygg return + 4 unstakes
	if common.RuneAsset().Chain.Equals(common.THORChain) {
		c.Check(items, HasLen, 3, Commentf("Len %d", items))
	} else {
		c.Check(items, HasLen, 5, Commentf("Len %d", items))
	}
	c.Check(items[0].Memo, Equals, NewYggdrasilReturn(common.BlockHeight(ctx)).String())
	c.Check(items[0].Chain.Equals(common.BTCChain), Equals, true)

	ctx, k = setupKeeperForTest(c)
	helper := NewVaultGenesisSetupTestHelper(k)
	mgr1 := NewManagers(helper)
	mgr1.BeginBlock(ctx)
	vaultMgr1 := NewVaultMgrV1(helper, mgr1.TxOutStore(), mgr1.EventMgr())
	// fail to get active nodes should error out
	helper.failToListActiveAccounts = true
	c.Assert(vaultMgr1.ragnarokChain(ctx, common.BNBChain, 1, mgr, constAccessor), NotNil)
	helper.failToListActiveAccounts = false

	// no active nodes , should error
	c.Assert(vaultMgr1.ragnarokChain(ctx, common.BNBChain, 1, mgr, constAccessor), NotNil)
	helper.Keeper.SetNodeAccount(ctx, GetRandomNodeAccount(NodeActive))
	helper.Keeper.SetNodeAccount(ctx, GetRandomNodeAccount(NodeActive))

	// fail to get pools should error out
	helper.failGetPools = true
	c.Assert(vaultMgr1.ragnarokChain(ctx, common.BNBChain, 1, mgr, constAccessor), NotNil)
	helper.failGetPools = false

	// fail to get active asgard vault should error out
	helper.failGetActiveAsgardVault = true
	c.Assert(vaultMgr1.ragnarokChain(ctx, common.BNBChain, 1, mgr, constAccessor), NotNil)
	helper.failGetActiveAsgardVault = false
}

func (s *VaultManagerV1TestSuite) TestUpdateVaultData(c *C) {
	ctx, k := setupKeeperForTest(c)
	ver := constants.SWVersion
	constAccessor := constants.GetConstantValues(ver)
	helper := NewVaultGenesisSetupTestHelper(k)
	mgr := NewManagers(helper)
	mgr.BeginBlock(ctx)
	vaultMgr := NewVaultMgrV1(helper, mgr.TxOutStore(), mgr.EventMgr())

	// fail to get VaultData should return error
	helper.failGetVaultData = true
	c.Assert(vaultMgr.UpdateVaultData(ctx, constAccessor, mgr.gasMgr, mgr.eventMgr), NotNil)
	helper.failGetVaultData = false

	// TotalReserve is zero , should not doing anything
	vd := NewVaultData()
	err := k.SetVaultData(ctx, vd)
	c.Assert(err, IsNil)
	c.Assert(vaultMgr.UpdateVaultData(ctx, constAccessor, mgr.GasMgr(), mgr.EventMgr()), IsNil)

	// total staked is zero , should not doing anything
	vd.TotalReserve = cosmos.NewUint(common.One * 100)
	err = k.SetVaultData(ctx, vd)
	c.Assert(err, IsNil)
	c.Assert(vaultMgr.UpdateVaultData(ctx, constAccessor, mgr.GasMgr(), mgr.EventMgr()), IsNil)

	p := NewPool()
	p.Asset = common.BNBAsset
	p.BalanceRune = cosmos.NewUint(common.One * 100)
	p.BalanceAsset = cosmos.NewUint(common.One * 100)
	p.Status = PoolEnabled
	c.Assert(helper.SetPool(ctx, p), IsNil)
	// no active node , thus no bond
	c.Assert(vaultMgr.UpdateVaultData(ctx, constAccessor, mgr.GasMgr(), mgr.EventMgr()), IsNil)

	// with liquidity fee , and bonds
	helper.Keeper.AddToLiquidityFees(ctx, common.BNBAsset, cosmos.NewUint(50*common.One))

	c.Assert(vaultMgr.UpdateVaultData(ctx, constAccessor, mgr.GasMgr(), mgr.EventMgr()), IsNil)
	// add bond
	helper.Keeper.SetNodeAccount(ctx, GetRandomNodeAccount(NodeActive))
	helper.Keeper.SetNodeAccount(ctx, GetRandomNodeAccount(NodeActive))
	c.Assert(vaultMgr.UpdateVaultData(ctx, constAccessor, mgr.GasMgr(), mgr.EventMgr()), IsNil)

	// fail to get total liquidity fee should result an error
	helper.failGetTotalLiquidityFee = true
	if common.RuneAsset().Equals(common.RuneNative) {
		FundModule(c, ctx, helper, ReserveName, 100)
	}
	c.Assert(vaultMgr.UpdateVaultData(ctx, constAccessor, mgr.GasMgr(), mgr.EventMgr()), NotNil)
	helper.failGetTotalLiquidityFee = false

	helper.failToListActiveAccounts = true
	c.Assert(vaultMgr.UpdateVaultData(ctx, constAccessor, mgr.GasMgr(), mgr.EventMgr()), NotNil)
}

func (s *VaultManagerV1TestSuite) TestCalcBlockRewards(c *C) {
	mgr := NewDummyMgr()
	vaultMgr := NewVaultMgrV1(keeper.KVStoreDummy{}, mgr.TxOutStore(), mgr.EventMgr())

	ver := constants.SWVersion
	constAccessor := constants.GetConstantValues(ver)
	emissionCurve := constAccessor.GetInt64Value(constants.EmissionCurve)
	blocksPerYear := constAccessor.GetInt64Value(constants.BlocksPerYear)
	bondR, poolR, stakerD := vaultMgr.calcBlockRewards(cosmos.NewUint(1000*common.One), cosmos.NewUint(2000*common.One), cosmos.NewUint(1000*common.One), cosmos.ZeroUint(), emissionCurve, blocksPerYear)
	c.Check(bondR.Uint64(), Equals, uint64(1761), Commentf("%d", bondR.Uint64()))
	c.Check(poolR.Uint64(), Equals, uint64(880), Commentf("%d", poolR.Uint64()))
	c.Check(stakerD.Uint64(), Equals, uint64(0), Commentf("%d", poolR.Uint64()))

	bondR, poolR, stakerD = vaultMgr.calcBlockRewards(cosmos.NewUint(1000*common.One), cosmos.NewUint(2000*common.One), cosmos.NewUint(1000*common.One), cosmos.NewUint(3000), emissionCurve, blocksPerYear)
	c.Check(bondR.Uint64(), Equals, uint64(3761), Commentf("%d", bondR.Uint64()))
	c.Check(poolR.Uint64(), Equals, uint64(0), Commentf("%d", poolR.Uint64()))
	c.Check(stakerD.Uint64(), Equals, uint64(1120), Commentf("%d", poolR.Uint64()))

	bondR, poolR, stakerD = vaultMgr.calcBlockRewards(cosmos.NewUint(1000*common.One), cosmos.NewUint(2000*common.One), cosmos.ZeroUint(), cosmos.ZeroUint(), emissionCurve, blocksPerYear)
	c.Check(bondR.Uint64(), Equals, uint64(0), Commentf("%d", bondR.Uint64()))
	c.Check(poolR.Uint64(), Equals, uint64(0), Commentf("%d", poolR.Uint64()))
	c.Check(stakerD.Uint64(), Equals, uint64(0), Commentf("%d", poolR.Uint64()))

	bondR, poolR, stakerD = vaultMgr.calcBlockRewards(cosmos.NewUint(1000*common.One), cosmos.NewUint(1000*common.One), cosmos.NewUint(1000*common.One), cosmos.ZeroUint(), emissionCurve, blocksPerYear)
	c.Check(bondR.Uint64(), Equals, uint64(2641), Commentf("%d", bondR.Uint64()))
	c.Check(poolR.Uint64(), Equals, uint64(0), Commentf("%d", poolR.Uint64()))
	c.Check(stakerD.Uint64(), Equals, uint64(0), Commentf("%d", poolR.Uint64()))

	bondR, poolR, stakerD = vaultMgr.calcBlockRewards(cosmos.ZeroUint(), cosmos.NewUint(1000*common.One), cosmos.NewUint(1000*common.One), cosmos.ZeroUint(), emissionCurve, blocksPerYear)
	c.Check(bondR.Uint64(), Equals, uint64(0), Commentf("%d", bondR.Uint64()))
	c.Check(poolR.Uint64(), Equals, uint64(2641), Commentf("%d", poolR.Uint64()))
	c.Check(stakerD.Uint64(), Equals, uint64(0), Commentf("%d", poolR.Uint64()))

	bondR, poolR, stakerD = vaultMgr.calcBlockRewards(cosmos.NewUint(2001*common.One), cosmos.NewUint(1000*common.One), cosmos.NewUint(1000*common.One), cosmos.ZeroUint(), emissionCurve, blocksPerYear)
	c.Check(bondR.Uint64(), Equals, uint64(2641), Commentf("%d", bondR.Uint64()))
	c.Check(poolR.Uint64(), Equals, uint64(0), Commentf("%d", poolR.Uint64()))
	c.Check(stakerD.Uint64(), Equals, uint64(0), Commentf("%d", poolR.Uint64()))
}

func (s *VaultManagerV1TestSuite) TestCalcPoolDeficit(c *C) {
	pool1Fees := cosmos.NewUint(1000)
	pool2Fees := cosmos.NewUint(3000)
	totalFees := cosmos.NewUint(4000)

	mgr := NewDummyMgr()
	vaultMgr := NewVaultMgrV1(keeper.KVStoreDummy{}, mgr.TxOutStore(), mgr.EventMgr())

	stakerDeficit := cosmos.NewUint(1120)
	amt1 := vaultMgr.calcPoolDeficit(stakerDeficit, totalFees, pool1Fees)
	amt2 := vaultMgr.calcPoolDeficit(stakerDeficit, totalFees, pool2Fees)

	c.Check(amt1.Equal(cosmos.NewUint(280)), Equals, true, Commentf("%d", amt1.Uint64()))
	c.Check(amt2.Equal(cosmos.NewUint(840)), Equals, true, Commentf("%d", amt2.Uint64()))
}

type VaultManagerTestHelpKeeper struct {
	keeper.Keeper
	failToGetAsgardVaults      bool
	failToListActiveAccounts   bool
	failToSetVault             bool
	failGetRetiringAsgardVault bool
	failGetActiveAsgardVault   bool
	failToSetPool              bool
	failGetVaultData           bool
	failGetTotalLiquidityFee   bool
	failGetPools               bool
}

func NewVaultGenesisSetupTestHelper(k keeper.Keeper) *VaultManagerTestHelpKeeper {
	return &VaultManagerTestHelpKeeper{
		Keeper: k,
	}
}

func (h *VaultManagerTestHelpKeeper) GetVaultData(ctx cosmos.Context) (VaultData, error) {
	if h.failGetVaultData {
		return VaultData{}, kaboom
	}
	return h.Keeper.GetVaultData(ctx)
}

func (h *VaultManagerTestHelpKeeper) GetAsgardVaults(ctx cosmos.Context) (Vaults, error) {
	if h.failToGetAsgardVaults {
		return Vaults{}, kaboom
	}
	return h.Keeper.GetAsgardVaults(ctx)
}

func (h *VaultManagerTestHelpKeeper) ListActiveNodeAccounts(ctx cosmos.Context) (NodeAccounts, error) {
	if h.failToListActiveAccounts {
		return NodeAccounts{}, kaboom
	}
	return h.Keeper.ListActiveNodeAccounts(ctx)
}

func (h *VaultManagerTestHelpKeeper) SetVault(ctx cosmos.Context, v Vault) error {
	if h.failToSetVault {
		return kaboom
	}
	return h.Keeper.SetVault(ctx, v)
}

func (h *VaultManagerTestHelpKeeper) GetAsgardVaultsByStatus(ctx cosmos.Context, vs VaultStatus) (Vaults, error) {
	if h.failGetRetiringAsgardVault && vs == RetiringVault {
		return Vaults{}, kaboom
	}
	if h.failGetActiveAsgardVault && vs == ActiveVault {
		return Vaults{}, kaboom
	}
	return h.Keeper.GetAsgardVaultsByStatus(ctx, vs)
}

func (h *VaultManagerTestHelpKeeper) SetPool(ctx cosmos.Context, p Pool) error {
	if h.failToSetPool {
		return kaboom
	}
	return h.Keeper.SetPool(ctx, p)
}

func (h *VaultManagerTestHelpKeeper) GetTotalLiquidityFees(ctx cosmos.Context, height uint64) (cosmos.Uint, error) {
	if h.failGetTotalLiquidityFee {
		return cosmos.ZeroUint(), kaboom
	}
	return h.Keeper.GetTotalLiquidityFees(ctx, height)
}

func (h *VaultManagerTestHelpKeeper) GetPools(ctx cosmos.Context) (Pools, error) {
	if h.failGetPools {
		return Pools{}, kaboom
	}
	return h.Keeper.GetPools(ctx)
}

func (*VaultManagerV1TestSuite) TestProcessGenesisSetup(c *C) {
	ctx, k := setupKeeperForTest(c)
	ver := constants.SWVersion
	constAccessor := constants.GetConstantValues(ver)
	helper := NewVaultGenesisSetupTestHelper(k)
	ctx = ctx.WithBlockHeight(1)
	mgr := NewManagers(helper)
	mgr.BeginBlock(ctx)
	vaultMgr := NewVaultMgrV1(helper, mgr.TxOutStore(), mgr.EventMgr())
	// no active account
	c.Assert(vaultMgr.EndBlock(ctx, mgr, constAccessor), NotNil)

	nodeAccount := GetRandomNodeAccount(NodeActive)
	k.SetNodeAccount(ctx, nodeAccount)
	c.Assert(vaultMgr.EndBlock(ctx, mgr, constAccessor), IsNil)
	// make sure asgard vault get created
	vaults, err := k.GetAsgardVaults(ctx)
	c.Assert(err, IsNil)
	c.Assert(vaults, HasLen, 1)

	// fail to get asgard vaults should return an error
	helper.failToGetAsgardVaults = true
	c.Assert(vaultMgr.EndBlock(ctx, mgr, constAccessor), NotNil)
	helper.failToGetAsgardVaults = false

	// vault already exist , it should not do anything , and should not error
	c.Assert(vaultMgr.EndBlock(ctx, mgr, constAccessor), IsNil)

	ctx, k = setupKeeperForTest(c)
	helper = NewVaultGenesisSetupTestHelper(k)
	ctx = ctx.WithBlockHeight(1)
	mgr = NewManagers(helper)
	mgr.BeginBlock(ctx)
	vaultMgr = NewVaultMgrV1(helper, mgr.TxOutStore(), mgr.EventMgr())
	helper.failToListActiveAccounts = true
	c.Assert(vaultMgr.EndBlock(ctx, mgr, constAccessor), NotNil)
	helper.failToListActiveAccounts = false

	helper.failToSetVault = true
	c.Assert(vaultMgr.EndBlock(ctx, mgr, constAccessor), NotNil)
	helper.failToSetVault = false

	helper.failGetRetiringAsgardVault = true
	ctx = ctx.WithBlockHeight(1024)
	c.Assert(vaultMgr.EndBlock(ctx, mgr, constAccessor), NotNil)
	helper.failGetRetiringAsgardVault = false

	helper.failGetActiveAsgardVault = true
	c.Assert(vaultMgr.EndBlock(ctx, mgr, constAccessor), NotNil)
	helper.failGetActiveAsgardVault = false
}

func (*VaultManagerV1TestSuite) TestGetTotalActiveBond(c *C) {
	ctx, k := setupKeeperForTest(c)
	helper := NewVaultGenesisSetupTestHelper(k)
	mgr := NewManagers(helper)
	mgr.BeginBlock(ctx)
	vaultMgr := NewVaultMgrV1(helper, mgr.TxOutStore(), mgr.EventMgr())
	helper.failToListActiveAccounts = true
	bond, err := vaultMgr.getTotalActiveBond(ctx)
	c.Assert(err, NotNil)
	c.Assert(bond.Equal(cosmos.ZeroUint()), Equals, true)
	helper.failToListActiveAccounts = false
	helper.Keeper.SetNodeAccount(ctx, GetRandomNodeAccount(NodeActive))
	bond, err = vaultMgr.getTotalActiveBond(ctx)
	c.Assert(err, IsNil)
	c.Assert(bond.Uint64() > 0, Equals, true)
}

func (*VaultManagerV1TestSuite) TestGetTotalStakedRune(c *C) {
	ctx, k := setupKeeperForTest(c)
	helper := NewVaultGenesisSetupTestHelper(k)
	mgr := NewManagers(helper)
	mgr.BeginBlock(ctx)
	vaultMgr := NewVaultMgrV1(helper, mgr.TxOutStore(), mgr.EventMgr())
	p := NewPool()
	p.Asset = common.BNBAsset
	p.BalanceRune = cosmos.NewUint(common.One * 100)
	p.BalanceAsset = cosmos.NewUint(common.One * 100)
	p.Status = PoolEnabled
	c.Assert(helper.SetPool(ctx, p), IsNil)
	pools, totalStaked, err := vaultMgr.getTotalStakedRune(ctx)
	c.Assert(err, IsNil)
	c.Assert(pools, HasLen, 1)
	c.Assert(totalStaked.Equal(p.BalanceRune), Equals, true)
}

func (*VaultManagerV1TestSuite) TestPayPoolRewards(c *C) {
	ctx, k := setupKeeperForTest(c)
	helper := NewVaultGenesisSetupTestHelper(k)
	mgr := NewManagers(helper)
	mgr.BeginBlock(ctx)
	vaultMgr := NewVaultMgrV1(helper, mgr.TxOutStore(), mgr.EventMgr())
	p := NewPool()
	p.Asset = common.BNBAsset
	p.BalanceRune = cosmos.NewUint(common.One * 100)
	p.BalanceAsset = cosmos.NewUint(common.One * 100)
	p.Status = PoolEnabled
	c.Assert(helper.SetPool(ctx, p), IsNil)
	vaultMgr.payPoolRewards(ctx, []cosmos.Uint{cosmos.NewUint(100 * common.One)}, Pools{p})
	helper.failToSetPool = true
	c.Assert(vaultMgr.payPoolRewards(ctx, []cosmos.Uint{cosmos.NewUint(100 * common.One)}, Pools{p}), NotNil)
}

func (*VaultManagerV1TestSuite) TestFindChainsToRetire(c *C) {
	ctx, k := setupKeeperForTest(c)
	helper := NewVaultGenesisSetupTestHelper(k)
	mgr := NewManagers(helper)
	mgr.BeginBlock(ctx)
	vaultMgr := NewVaultMgrV1(helper, mgr.TxOutStore(), mgr.EventMgr())
	// fail to get active asgard vault
	helper.failGetActiveAsgardVault = true
	chains, err := vaultMgr.findChainsToRetire(ctx)
	c.Assert(err, NotNil)
	c.Assert(chains, HasLen, 0)
	helper.failGetActiveAsgardVault = false

	// fail to get retire asgard vault
	helper.failGetRetiringAsgardVault = true
	chains, err = vaultMgr.findChainsToRetire(ctx)
	c.Assert(err, NotNil)
	c.Assert(chains, HasLen, 0)
	helper.failGetRetiringAsgardVault = false
}

func (*VaultManagerV1TestSuite) TestRecallChainFunds(c *C) {
	ctx, k := setupKeeperForTest(c)
	helper := NewVaultGenesisSetupTestHelper(k)
	mgr := NewManagers(helper)
	mgr.BeginBlock(ctx)
	vaultMgr := NewVaultMgrV1(helper, mgr.TxOutStore(), mgr.EventMgr())
	helper.failToListActiveAccounts = true
	c.Assert(vaultMgr.recallChainFunds(ctx, common.BNBChain, mgr), NotNil)
	helper.failToListActiveAccounts = false

	helper.failGetActiveAsgardVault = true
	c.Assert(vaultMgr.recallChainFunds(ctx, common.BNBChain, mgr), NotNil)
	helper.failGetActiveAsgardVault = false
}
