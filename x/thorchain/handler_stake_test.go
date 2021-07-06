package thorchain

import (
	"errors"

	"github.com/blang/semver"
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
	keeper "gitlab.com/thorchain/thornode/x/thorchain/keeper"
)

type HandlerStakeSuite struct{}

var _ = Suite(&HandlerStakeSuite{})

type MockStakeKeeper struct {
	keeper.KVStoreDummy
	currentPool        Pool
	activeNodeAccount  NodeAccount
	failGetPool        bool
	failGetNextEventID bool
}

func (m *MockStakeKeeper) PoolExist(_ cosmos.Context, asset common.Asset) bool {
	return m.currentPool.Asset.Equals(asset)
}

func (m *MockStakeKeeper) GetPools(_ cosmos.Context) (Pools, error) {
	return Pools{m.currentPool}, nil
}

func (m *MockStakeKeeper) GetPool(_ cosmos.Context, _ common.Asset) (Pool, error) {
	if m.failGetPool {
		return Pool{}, errors.New("fail to get pool")
	}
	return m.currentPool, nil
}

func (m *MockStakeKeeper) SetPool(_ cosmos.Context, pool Pool) error {
	m.currentPool = pool
	return nil
}

func (m *MockStakeKeeper) ListNodeAccountsWithBond(_ cosmos.Context) (NodeAccounts, error) {
	return NodeAccounts{m.activeNodeAccount}, nil
}

func (m *MockStakeKeeper) GetNodeAccount(_ cosmos.Context, addr cosmos.AccAddress) (NodeAccount, error) {
	if m.activeNodeAccount.NodeAddress.Equals(addr) {
		return m.activeNodeAccount, nil
	}
	return NodeAccount{}, errors.New("not exist")
}

func (m *MockStakeKeeper) GetStaker(_ cosmos.Context, asset common.Asset, addr common.Address) (Staker, error) {
	return Staker{
		Asset:        asset,
		RuneAddress:  addr,
		AssetAddress: addr,
		Units:        cosmos.ZeroUint(),
		PendingRune:  cosmos.ZeroUint(),
	}, nil
}

type MockConstant struct {
	constants.DummyConstants
}

func (HandlerStakeSuite) TestStakeHandler(c *C) {
	ctx, _ := setupKeeperForTest(c)
	activeNodeAccount := GetRandomNodeAccount(NodeActive)
	k := &MockStakeKeeper{
		activeNodeAccount: activeNodeAccount,
		currentPool: Pool{
			BalanceRune:  cosmos.ZeroUint(),
			BalanceAsset: cosmos.ZeroUint(),
			Asset:        common.BNBAsset,
			PoolUnits:    cosmos.ZeroUint(),
			Status:       PoolEnabled,
		},
	}
	// happy path
	mgr := NewManagers(k)
	c.Assert(mgr.BeginBlock(ctx), IsNil)
	stakeHandler := NewStakeHandler(k, mgr)
	preStakePool, err := k.GetPool(ctx, common.BNBAsset)
	c.Assert(err, IsNil)
	bnbAddr := GetRandomBNBAddress()
	stakeTxHash := GetRandomTxHash()
	tx := common.NewTx(
		stakeTxHash,
		bnbAddr,
		GetRandomBNBAddress(),
		common.Coins{common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One*5))},
		BNBGasFeeSingleton,
		"stake:BNB",
	)
	ver := constants.SWVersion
	constAccessor := constants.GetConstantValues(ver)
	msgSetStake := NewMsgStake(
		tx,
		common.BNBAsset,
		cosmos.NewUint(100*common.One),
		cosmos.NewUint(100*common.One),
		bnbAddr,
		bnbAddr,
		activeNodeAccount.NodeAddress)
	_, err = stakeHandler.Run(ctx, msgSetStake, ver, constAccessor)
	c.Assert(err, IsNil)
	postStakePool, err := k.GetPool(ctx, common.BNBAsset)
	c.Assert(err, IsNil)
	c.Assert(postStakePool.BalanceAsset.String(), Equals, preStakePool.BalanceAsset.Add(msgSetStake.AssetAmount).String())
	c.Assert(postStakePool.BalanceRune.String(), Equals, preStakePool.BalanceRune.Add(msgSetStake.RuneAmount).String())
}

func (HandlerStakeSuite) TestStakeHandler_NoPool_ShouldCreateNewPool(c *C) {
	ctx, _ := setupKeeperForTest(c)
	activeNodeAccount := GetRandomNodeAccount(NodeActive)
	activeNodeAccount.Bond = cosmos.NewUint(1000000 * common.One)
	k := &MockStakeKeeper{
		activeNodeAccount: activeNodeAccount,
		currentPool: Pool{
			BalanceRune:  cosmos.ZeroUint(),
			BalanceAsset: cosmos.ZeroUint(),
			PoolUnits:    cosmos.ZeroUint(),
		},
	}
	// happy path
	mgr := NewManagers(k)
	c.Assert(mgr.BeginBlock(ctx), IsNil)
	stakeHandler := NewStakeHandler(k, mgr)
	preStakePool, err := k.GetPool(ctx, common.BNBAsset)
	c.Assert(err, IsNil)
	c.Assert(preStakePool.IsEmpty(), Equals, true)
	bnbAddr := GetRandomBNBAddress()
	stakeTxHash := GetRandomTxHash()
	tx := common.NewTx(
		stakeTxHash,
		bnbAddr,
		GetRandomBNBAddress(),
		common.Coins{common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One*5))},
		BNBGasFeeSingleton,
		"stake:BNB",
	)
	ver := constants.SWVersion
	constAccessor := constants.NewDummyConstants(map[constants.ConstantName]int64{
		constants.MaximumStakeRune: 600_000_00000000,
	}, map[constants.ConstantName]bool{
		constants.StrictBondStakeRatio: true,
	}, map[constants.ConstantName]string{})

	msgSetStake := NewMsgStake(
		tx,
		common.BNBAsset,
		cosmos.NewUint(100*common.One),
		cosmos.NewUint(100*common.One),
		bnbAddr,
		bnbAddr,
		activeNodeAccount.NodeAddress)
	_, err = stakeHandler.Run(ctx, msgSetStake, ver, constAccessor)
	c.Assert(err, IsNil)
	postStakePool, err := k.GetPool(ctx, common.BNBAsset)
	c.Assert(err, IsNil)
	c.Assert(postStakePool.BalanceAsset.String(), Equals, preStakePool.BalanceAsset.Add(msgSetStake.AssetAmount).String())
	c.Assert(postStakePool.BalanceRune.String(), Equals, preStakePool.BalanceRune.Add(msgSetStake.RuneAmount).String())

	// bad version
	_, err = stakeHandler.Run(ctx, msgSetStake, semver.Version{}, constAccessor)
	c.Assert(err, NotNil)
}

func (HandlerStakeSuite) TestStakeHandlerValidation(c *C) {
	ctx, _ := setupKeeperForTest(c)
	activeNodeAccount := GetRandomNodeAccount(NodeActive)
	k := &MockStakeKeeper{
		activeNodeAccount: activeNodeAccount,
		currentPool: Pool{
			BalanceRune:  cosmos.ZeroUint(),
			BalanceAsset: cosmos.ZeroUint(),
			Asset:        common.BNBAsset,
			PoolUnits:    cosmos.ZeroUint(),
			Status:       PoolEnabled,
		},
	}
	testCases := []struct {
		name           string
		msg            MsgStake
		expectedResult error
	}{
		{
			name:           "empty signer should fail",
			msg:            NewMsgStake(GetRandomTx(), common.BNBAsset, cosmos.NewUint(common.One*5), cosmos.NewUint(common.One*5), GetRandomBNBAddress(), GetRandomBNBAddress(), cosmos.AccAddress{}),
			expectedResult: errStakeFailValidation,
		},
		{
			name:           "empty asset should fail",
			msg:            NewMsgStake(GetRandomTx(), common.Asset{}, cosmos.NewUint(common.One*5), cosmos.NewUint(common.One*5), GetRandomBNBAddress(), GetRandomBNBAddress(), GetRandomNodeAccount(NodeActive).NodeAddress),
			expectedResult: errStakeFailValidation,
		},
		{
			name:           "empty RUNE address should fail",
			msg:            NewMsgStake(GetRandomTx(), common.BNBAsset, cosmos.NewUint(common.One*5), cosmos.NewUint(common.One*5), common.NoAddress, GetRandomBNBAddress(), GetRandomNodeAccount(NodeActive).NodeAddress),
			expectedResult: errStakeFailValidation,
		},
		{
			name:           "empty ASSET address should fail",
			msg:            NewMsgStake(GetRandomTx(), common.BTCAsset, cosmos.NewUint(common.One*5), cosmos.NewUint(common.One*5), GetRandomBNBAddress(), common.NoAddress, GetRandomNodeAccount(NodeActive).NodeAddress),
			expectedResult: errStakeFailValidation,
		},
		{
			name:           "total staker is more than total bond should fail",
			msg:            NewMsgStake(GetRandomTx(), common.BNBAsset, cosmos.NewUint(common.One*5000), cosmos.NewUint(common.One*5000), GetRandomBNBAddress(), GetRandomBNBAddress(), activeNodeAccount.NodeAddress),
			expectedResult: errStakeRUNEMoreThanBond,
		},
	}
	ver := constants.SWVersion
	constAccessor := constants.NewDummyConstants(map[constants.ConstantName]int64{
		constants.MaximumStakeRune: 600_000_00000000,
	}, map[constants.ConstantName]bool{
		constants.StrictBondStakeRatio: true,
	}, map[constants.ConstantName]string{})

	for _, item := range testCases {
		stakeHandler := NewStakeHandler(k, NewDummyMgr())
		_, err := stakeHandler.Run(ctx, item.msg, ver, constAccessor)
		c.Assert(errors.Is(err, item.expectedResult), Equals, true, Commentf("name:%s", item.name))
	}
}

func (HandlerStakeSuite) TestHandlerStakeFailScenario(c *C) {
	ctx, _ := setupKeeperForTest(c)
	activeNodeAccount := GetRandomNodeAccount(NodeActive)
	emptyPool := Pool{
		BalanceRune:  cosmos.ZeroUint(),
		BalanceAsset: cosmos.ZeroUint(),
		Asset:        common.BNBAsset,
		PoolUnits:    cosmos.ZeroUint(),
		Status:       PoolEnabled,
	}

	testCases := []struct {
		name           string
		k              keeper.Keeper
		expectedResult error
	}{
		{
			name: "fail to get pool should fail stake",
			k: &MockStakeKeeper{
				activeNodeAccount: activeNodeAccount,
				currentPool:       emptyPool,
				failGetPool:       true,
			},
			expectedResult: errInternal,
		},
		{
			name: "suspended pool should fail stake",
			k: &MockStakeKeeper{
				activeNodeAccount: activeNodeAccount,
				currentPool: Pool{
					BalanceRune:  cosmos.ZeroUint(),
					BalanceAsset: cosmos.ZeroUint(),
					Asset:        common.BNBAsset,
					PoolUnits:    cosmos.ZeroUint(),
					Status:       PoolSuspended,
				},
			},
			expectedResult: errInvalidPoolStatus,
		},
	}
	for _, tc := range testCases {
		bnbAddr := GetRandomBNBAddress()
		stakeTxHash := GetRandomTxHash()
		tx := common.NewTx(
			stakeTxHash,
			bnbAddr,
			GetRandomBNBAddress(),
			common.Coins{common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One*5))},
			BNBGasFeeSingleton,
			"stake:BNB",
		)
		ver := constants.SWVersion
		constAccessor := constants.GetConstantValues(ver)
		msgSetStake := NewMsgStake(
			tx,
			common.BNBAsset,
			cosmos.NewUint(100*common.One),
			cosmos.NewUint(100*common.One),
			bnbAddr,
			bnbAddr,
			activeNodeAccount.NodeAddress)
		mgr := NewManagers(tc.k)
		c.Assert(mgr.BeginBlock(ctx), IsNil)
		stakeHandler := NewStakeHandler(tc.k, mgr)
		_, err := stakeHandler.Run(ctx, msgSetStake, ver, constAccessor)
		c.Assert(errors.Is(err, tc.expectedResult), Equals, true, Commentf(tc.name))
	}
}

type StakeTestKeeper struct {
	keeper.KVStoreDummy
	store map[string]interface{}
}

// NewStakeTestKeeper
func NewStakeTestKeeper() *StakeTestKeeper {
	return &StakeTestKeeper{store: make(map[string]interface{})}
}

func (p *StakeTestKeeper) PoolExist(ctx cosmos.Context, asset common.Asset) bool {
	_, ok := p.store[asset.String()]
	return ok
}

var notExistStakerAsset, _ = common.NewAsset("BNB.NotExistStakerAsset")

func (p *StakeTestKeeper) GetPool(ctx cosmos.Context, asset common.Asset) (Pool, error) {
	if p, ok := p.store[asset.String()]; ok {
		return p.(Pool), nil
	}
	return NewPool(), nil
}

func (p *StakeTestKeeper) SetPool(ctx cosmos.Context, ps Pool) error {
	p.store[ps.Asset.String()] = ps
	return nil
}

func (p *StakeTestKeeper) GetStaker(ctx cosmos.Context, asset common.Asset, addr common.Address) (Staker, error) {
	if notExistStakerAsset.Equals(asset) {
		return Staker{}, errors.New("simulate error for test")
	}
	staker := Staker{
		Asset:       asset,
		RuneAddress: addr,
		Units:       cosmos.ZeroUint(),
		PendingRune: cosmos.ZeroUint(),
	}
	key := p.GetKey(ctx, "staker/", staker.Key())
	if res, ok := p.store[key]; ok {
		return res.(Staker), nil
	}
	return staker, nil
}

func (p *StakeTestKeeper) SetStaker(ctx cosmos.Context, staker Staker) {
	key := p.GetKey(ctx, "staker/", staker.Key())
	p.store[key] = staker
}

func (HandlerStakeSuite) TestCalculatePoolUnits(c *C) {
	inputs := []struct {
		name         string
		oldPoolUnits cosmos.Uint
		poolRune     cosmos.Uint
		poolAsset    cosmos.Uint
		stakeRune    cosmos.Uint
		stakeAsset   cosmos.Uint
		poolUnits    cosmos.Uint
		stakerUnits  cosmos.Uint
		expectedErr  error
	}{
		{
			name:         "first-stake-zero-rune",
			oldPoolUnits: cosmos.ZeroUint(),
			poolRune:     cosmos.ZeroUint(),
			poolAsset:    cosmos.ZeroUint(),
			stakeRune:    cosmos.ZeroUint(),
			stakeAsset:   cosmos.NewUint(100 * common.One),
			poolUnits:    cosmos.ZeroUint(),
			stakerUnits:  cosmos.ZeroUint(),
			expectedErr:  errors.New("total RUNE in the pool is zero"),
		},
		{
			name:         "first-stake-zero-asset",
			oldPoolUnits: cosmos.ZeroUint(),
			poolRune:     cosmos.ZeroUint(),
			poolAsset:    cosmos.ZeroUint(),
			stakeRune:    cosmos.NewUint(100 * common.One),
			stakeAsset:   cosmos.ZeroUint(),
			poolUnits:    cosmos.ZeroUint(),
			stakerUnits:  cosmos.ZeroUint(),
			expectedErr:  errors.New("total asset in the pool is zero"),
		},
		{
			name:         "first-stake",
			oldPoolUnits: cosmos.ZeroUint(),
			poolRune:     cosmos.ZeroUint(),
			poolAsset:    cosmos.ZeroUint(),
			stakeRune:    cosmos.NewUint(100 * common.One),
			stakeAsset:   cosmos.NewUint(100 * common.One),
			poolUnits:    cosmos.NewUint(100 * common.One),
			stakerUnits:  cosmos.NewUint(100 * common.One),
			expectedErr:  nil,
		},
		{
			name:         "second-stake",
			oldPoolUnits: cosmos.NewUint(500 * common.One),
			poolRune:     cosmos.NewUint(500 * common.One),
			poolAsset:    cosmos.NewUint(500 * common.One),
			stakeRune:    cosmos.NewUint(345 * common.One),
			stakeAsset:   cosmos.NewUint(234 * common.One),
			poolUnits:    cosmos.NewUint(78701684858),
			stakerUnits:  cosmos.NewUint(28701684858),
			expectedErr:  nil,
		},
	}

	for _, item := range inputs {
		poolUnits, stakerUnits, err := calculatePoolUnits(item.oldPoolUnits, item.poolRune, item.poolAsset, item.stakeRune, item.stakeAsset)
		if item.expectedErr == nil {
			c.Assert(err, IsNil)
		} else {
			c.Assert(err.Error(), Equals, item.expectedErr.Error())
		}

		c.Logf("poolUnits:%s,expectedUnits:%s", poolUnits, item.poolUnits)
		c.Check(item.poolUnits.Uint64(), Equals, poolUnits.Uint64())
		c.Logf("stakerUnits:%s,expectedStakerUnits:%s", stakerUnits, item.stakerUnits)
		c.Check(item.stakerUnits.Uint64(), Equals, stakerUnits.Uint64())
	}
}

func (HandlerStakeSuite) TestCalculatePoolUnitsV14(c *C) {
	inputs := []struct {
		name         string
		oldPoolUnits cosmos.Uint
		poolRune     cosmos.Uint
		poolAsset    cosmos.Uint
		stakeRune    cosmos.Uint
		stakeAsset   cosmos.Uint
		poolUnits    cosmos.Uint
		stakerUnits  cosmos.Uint
		expectedErr  error
	}{
		{
			name:         "first-stake-zero-rune",
			oldPoolUnits: cosmos.ZeroUint(),
			poolRune:     cosmos.ZeroUint(),
			poolAsset:    cosmos.ZeroUint(),
			stakeRune:    cosmos.ZeroUint(),
			stakeAsset:   cosmos.NewUint(100 * common.One),
			poolUnits:    cosmos.ZeroUint(),
			stakerUnits:  cosmos.ZeroUint(),
			expectedErr:  errors.New("total RUNE in the pool is zero"),
		},
		{
			name:         "first-stake-zero-asset",
			oldPoolUnits: cosmos.ZeroUint(),
			poolRune:     cosmos.ZeroUint(),
			poolAsset:    cosmos.ZeroUint(),
			stakeRune:    cosmos.NewUint(100 * common.One),
			stakeAsset:   cosmos.ZeroUint(),
			poolUnits:    cosmos.ZeroUint(),
			stakerUnits:  cosmos.ZeroUint(),
			expectedErr:  errors.New("total asset in the pool is zero"),
		},
		{
			name:         "first-stake",
			oldPoolUnits: cosmos.ZeroUint(),
			poolRune:     cosmos.ZeroUint(),
			poolAsset:    cosmos.ZeroUint(),
			stakeRune:    cosmos.NewUint(100 * common.One),
			stakeAsset:   cosmos.NewUint(100 * common.One),
			poolUnits:    cosmos.NewUint(100 * common.One),
			stakerUnits:  cosmos.NewUint(100 * common.One),
			expectedErr:  nil,
		},
		{
			name:         "second-stake",
			oldPoolUnits: cosmos.NewUint(500 * common.One),
			poolRune:     cosmos.NewUint(500 * common.One),
			poolAsset:    cosmos.NewUint(500 * common.One),
			stakeRune:    cosmos.NewUint(345 * common.One),
			stakeAsset:   cosmos.NewUint(234 * common.One),
			poolUnits:    cosmos.NewUint(77110505346),
			stakerUnits:  cosmos.NewUint(27110505346),
			expectedErr:  nil,
		},
	}

	for _, item := range inputs {
		c.Logf("Name: %s", item.name)
		poolUnits, stakerUnits, err := calculatePoolUnitsV14(item.oldPoolUnits, item.poolRune, item.poolAsset, item.stakeRune, item.stakeAsset)
		if item.expectedErr == nil {
			c.Assert(err, IsNil)
		} else {
			c.Assert(err.Error(), Equals, item.expectedErr.Error())
		}

		c.Check(item.poolUnits.Uint64(), Equals, poolUnits.Uint64(), Commentf("%d / %d", item.poolUnits.Uint64(), poolUnits.Uint64()))
		c.Check(item.stakerUnits.Uint64(), Equals, stakerUnits.Uint64(), Commentf("%d / %d", item.stakerUnits.Uint64(), stakerUnits.Uint64()))
	}
}

func (HandlerStakeSuite) TestValidateStakeMessage(c *C) {
	ps := NewStakeTestKeeper()
	ctx, k := setupKeeperForTest(c)
	txID := GetRandomTxHash()
	bnbAddress := GetRandomBNBAddress()
	assetAddress := GetRandomBNBAddress()
	h := NewStakeHandler(ps, NewManagers(k))
	c.Assert(h.validateStakeMessage(ctx, ps, common.Asset{}, txID, bnbAddress, assetAddress), NotNil)
	c.Assert(h.validateStakeMessage(ctx, ps, common.BNBAsset, txID, bnbAddress, assetAddress), NotNil)
	c.Assert(h.validateStakeMessage(ctx, ps, common.BNBAsset, txID, bnbAddress, assetAddress), NotNil)
	c.Assert(h.validateStakeMessage(ctx, ps, common.BNBAsset, common.TxID(""), bnbAddress, assetAddress), NotNil)
	c.Assert(h.validateStakeMessage(ctx, ps, common.BNBAsset, txID, common.NoAddress, common.NoAddress), NotNil)
	c.Assert(h.validateStakeMessage(ctx, ps, common.BNBAsset, txID, bnbAddress, assetAddress), NotNil)
	c.Assert(h.validateStakeMessage(ctx, ps, common.BNBAsset, txID, common.NoAddress, assetAddress), NotNil)
	c.Assert(h.validateStakeMessage(ctx, ps, common.BTCAsset, txID, bnbAddress, common.NoAddress), NotNil)
	c.Assert(ps.SetPool(ctx, Pool{
		BalanceRune:  cosmos.NewUint(100 * common.One),
		BalanceAsset: cosmos.NewUint(100 * common.One),
		Asset:        common.BNBAsset,
		PoolUnits:    cosmos.NewUint(100 * common.One),
		Status:       PoolEnabled,
	}), IsNil)
	c.Assert(h.validateStakeMessage(ctx, ps, common.BNBAsset, txID, bnbAddress, assetAddress), Equals, nil)
}

func (HandlerStakeSuite) TestStake(c *C) {
	ps := NewStakeTestKeeper()
	ctx, _ := setupKeeperForTest(c)
	txID := GetRandomTxHash()

	bnbAddress := GetRandomBNBAddress()
	assetAddress := GetRandomBNBAddress()
	btcAddress, err := common.NewAddress("bc1qwqdg6squsna38e46795at95yu9atm8azzmyvckulcc7kytlcckxswvvzej")
	c.Assert(err, IsNil)
	constAccessor := constants.GetConstantValues(constants.SWVersion)
	h := NewStakeHandler(ps, NewDummyMgr())
	err = h.stake(ctx, common.Asset{}, cosmos.NewUint(100*common.One), cosmos.NewUint(100*common.One), bnbAddress, assetAddress, txID, constAccessor)
	c.Assert(err, NotNil)
	c.Assert(ps.SetPool(ctx, Pool{
		BalanceRune:  cosmos.ZeroUint(),
		BalanceAsset: cosmos.NewUint(100 * common.One),
		Asset:        common.BNBAsset,
		PoolUnits:    cosmos.NewUint(100 * common.One),
		Status:       PoolEnabled,
	}), IsNil)
	err = h.stake(ctx, common.BNBAsset, cosmos.NewUint(100*common.One), cosmos.NewUint(100*common.One), bnbAddress, assetAddress, txID, constAccessor)
	c.Assert(err, IsNil)
	s, err := ps.GetStaker(ctx, common.BNBAsset, bnbAddress)
	c.Assert(err, IsNil)
	c.Assert(s.Units.Equal(cosmos.NewUint(11250000000)), Equals, true)

	c.Assert(ps.SetPool(ctx, Pool{
		BalanceRune:  cosmos.NewUint(100 * common.One),
		BalanceAsset: cosmos.NewUint(100 * common.One),
		Asset:        notExistStakerAsset,
		PoolUnits:    cosmos.NewUint(100 * common.One),
		Status:       PoolEnabled,
	}), IsNil)
	// stake asymmetically
	err = h.stake(ctx, common.BNBAsset, cosmos.NewUint(100*common.One), cosmos.ZeroUint(), bnbAddress, assetAddress, txID, constAccessor)
	c.Assert(err, IsNil)
	err = h.stake(ctx, common.BNBAsset, cosmos.ZeroUint(), cosmos.NewUint(100*common.One), bnbAddress, assetAddress, txID, constAccessor)
	c.Assert(err, IsNil)

	err = h.stake(ctx, notExistStakerAsset, cosmos.NewUint(100*common.One), cosmos.NewUint(100*common.One), bnbAddress, assetAddress, txID, constAccessor)
	c.Assert(err, NotNil)
	c.Assert(ps.SetPool(ctx, Pool{
		BalanceRune:  cosmos.NewUint(100 * common.One),
		BalanceAsset: cosmos.NewUint(100 * common.One),
		Asset:        common.BNBAsset,
		PoolUnits:    cosmos.NewUint(100 * common.One),
		Status:       PoolEnabled,
	}), IsNil)

	for i := 1; i <= 150; i++ {
		staker := Staker{Units: cosmos.NewUint(common.One / 5000)}
		ps.SetStaker(ctx, staker)
	}
	err = h.stake(ctx, common.BNBAsset, cosmos.NewUint(common.One), cosmos.NewUint(common.One), bnbAddress, assetAddress, txID, constAccessor)
	c.Assert(err, IsNil)

	err = h.stake(ctx, common.BNBAsset, cosmos.NewUint(100*common.One), cosmos.NewUint(100*common.One), bnbAddress, assetAddress, txID, constAccessor)
	c.Assert(err, IsNil)
	p, err := ps.GetPool(ctx, common.BNBAsset)
	c.Assert(err, IsNil)
	c.Check(p.PoolUnits.Equal(cosmos.NewUint(201*common.One)), Equals, true, Commentf("%d", p.PoolUnits.Uint64()))

	// Test atomic cross chain staking
	// create BTC pool
	c.Assert(ps.SetPool(ctx, Pool{
		BalanceRune:  cosmos.ZeroUint(),
		BalanceAsset: cosmos.ZeroUint(),
		Asset:        common.BTCAsset,
		PoolUnits:    cosmos.ZeroUint(),
		Status:       PoolEnabled,
	}), IsNil)

	// stake rune
	err = h.stake(ctx, common.BTCAsset, cosmos.NewUint(100*common.One), cosmos.ZeroUint(), bnbAddress, btcAddress, txID, constAccessor)
	c.Assert(err, IsNil)
	s, err = ps.GetStaker(ctx, common.BTCAsset, bnbAddress)
	c.Assert(err, IsNil)
	c.Check(s.Units.IsZero(), Equals, true)
	// stake btc
	err = h.stake(ctx, common.BTCAsset, cosmos.ZeroUint(), cosmos.NewUint(100*common.One), bnbAddress, btcAddress, txID, constAccessor)
	c.Assert(err, IsNil)
	s, err = ps.GetStaker(ctx, common.BTCAsset, bnbAddress)
	c.Assert(err, IsNil)
	c.Check(s.Units.IsZero(), Equals, false)
	p, err = ps.GetPool(ctx, common.BTCAsset)
	c.Assert(err, IsNil)
	c.Check(p.BalanceAsset.Equal(cosmos.NewUint(100*common.One)), Equals, true, Commentf("%d", p.BalanceAsset.Uint64()))
	c.Check(p.BalanceRune.Equal(cosmos.NewUint(100*common.One)), Equals, true, Commentf("%d", p.BalanceRune.Uint64()))
	c.Check(p.PoolUnits.Equal(cosmos.NewUint(100*common.One)), Equals, true, Commentf("%d", p.PoolUnits.Uint64()))
}
