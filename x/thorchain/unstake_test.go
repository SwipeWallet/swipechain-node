package thorchain

import (
	"errors"
	"fmt"

	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
	keeper "gitlab.com/thorchain/thornode/x/thorchain/keeper"
	"gitlab.com/thorchain/thornode/x/thorchain/types"
)

type UnstakeSuite struct{}

var _ = Suite(&UnstakeSuite{})

func (s *UnstakeSuite) SetUpSuite(c *C) {
	SetupConfigForTest()
}

type UnstakeTestKeeper struct {
	keeper.KVStoreDummy
	store map[string]interface{}
}

func NewUnstakeTestKeeper() *UnstakeTestKeeper {
	return &UnstakeTestKeeper{
		store: make(map[string]interface{}),
	}
}

func (k *UnstakeTestKeeper) PoolExist(ctx cosmos.Context, asset common.Asset) bool {
	if asset.Equals(common.Asset{Chain: common.BNBChain, Symbol: "NOTEXIST", Ticker: "NOTEXIST"}) {
		return false
	}
	return true
}

func (k *UnstakeTestKeeper) GetPool(ctx cosmos.Context, asset common.Asset) (types.Pool, error) {
	if asset.Equals(common.Asset{Chain: common.BNBChain, Symbol: "NOTEXIST", Ticker: "NOTEXIST"}) {
		return types.Pool{}, nil
	} else {
		return types.Pool{
			BalanceRune:  cosmos.NewUint(100).MulUint64(common.One),
			BalanceAsset: cosmos.NewUint(100).MulUint64(common.One),
			PoolUnits:    cosmos.NewUint(100).MulUint64(common.One),
			Status:       types.Enabled,
			Asset:        asset,
		}, nil
	}
}

func (k *UnstakeTestKeeper) SetPool(ctx cosmos.Context, ps Pool) error {
	k.store[ps.Asset.String()] = ps
	return nil
}

func (k *UnstakeTestKeeper) GetGas(ctx cosmos.Context, asset common.Asset) ([]cosmos.Uint, error) {
	return []cosmos.Uint{cosmos.NewUint(37500), cosmos.NewUint(30000)}, nil
}

func (p *UnstakeTestKeeper) GetStaker(ctx cosmos.Context, asset common.Asset, addr common.Address) (Staker, error) {
	if asset.Equals(common.Asset{Chain: common.BNBChain, Symbol: "NOTEXISTSTICKER", Ticker: "NOTEXISTSTICKER"}) {
		return types.Staker{}, errors.New("you asked for it")
	}
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

func (p *UnstakeTestKeeper) SetStaker(ctx cosmos.Context, staker Staker) {
	key := p.GetKey(ctx, "staker/", staker.Key())
	p.store[key] = staker
}

func (s UnstakeSuite) TestCalculateUnsake(c *C) {
	inputs := []struct {
		name                 string
		poolUnit             cosmos.Uint
		poolRune             cosmos.Uint
		poolAsset            cosmos.Uint
		stakerUnit           cosmos.Uint
		percentage           cosmos.Uint
		expectedUnstakeRune  cosmos.Uint
		expectedUnstakeAsset cosmos.Uint
		expectedUnitLeft     cosmos.Uint
		expectedErr          error
	}{
		{
			name:                 "zero-poolunit",
			poolUnit:             cosmos.ZeroUint(),
			poolRune:             cosmos.ZeroUint(),
			poolAsset:            cosmos.ZeroUint(),
			stakerUnit:           cosmos.ZeroUint(),
			percentage:           cosmos.ZeroUint(),
			expectedUnstakeRune:  cosmos.ZeroUint(),
			expectedUnstakeAsset: cosmos.ZeroUint(),
			expectedUnitLeft:     cosmos.ZeroUint(),
			expectedErr:          errors.New("poolUnits can't be zero"),
		},

		{
			name:                 "zero-poolrune",
			poolUnit:             cosmos.NewUint(500 * common.One),
			poolRune:             cosmos.ZeroUint(),
			poolAsset:            cosmos.ZeroUint(),
			stakerUnit:           cosmos.ZeroUint(),
			percentage:           cosmos.ZeroUint(),
			expectedUnstakeRune:  cosmos.ZeroUint(),
			expectedUnstakeAsset: cosmos.ZeroUint(),
			expectedUnitLeft:     cosmos.ZeroUint(),
			expectedErr:          errors.New("pool rune balance can't be zero"),
		},

		{
			name:                 "zero-poolasset",
			poolUnit:             cosmos.NewUint(500 * common.One),
			poolRune:             cosmos.NewUint(500 * common.One),
			poolAsset:            cosmos.ZeroUint(),
			stakerUnit:           cosmos.ZeroUint(),
			percentage:           cosmos.ZeroUint(),
			expectedUnstakeRune:  cosmos.ZeroUint(),
			expectedUnstakeAsset: cosmos.ZeroUint(),
			expectedUnitLeft:     cosmos.ZeroUint(),
			expectedErr:          errors.New("pool asset balance can't be zero"),
		},
		{
			name:                 "negative-stakerUnit",
			poolUnit:             cosmos.NewUint(500 * common.One),
			poolRune:             cosmos.NewUint(500 * common.One),
			poolAsset:            cosmos.NewUint(5100 * common.One),
			stakerUnit:           cosmos.ZeroUint(),
			percentage:           cosmos.ZeroUint(),
			expectedUnstakeRune:  cosmos.ZeroUint(),
			expectedUnstakeAsset: cosmos.ZeroUint(),
			expectedUnitLeft:     cosmos.ZeroUint(),
			expectedErr:          errors.New("staker unit can't be zero"),
		},

		{
			name:                 "percentage-larger-than-100",
			poolUnit:             cosmos.NewUint(500 * common.One),
			poolRune:             cosmos.NewUint(500 * common.One),
			poolAsset:            cosmos.NewUint(500 * common.One),
			stakerUnit:           cosmos.NewUint(100 * common.One),
			percentage:           cosmos.NewUint(12000),
			expectedUnstakeRune:  cosmos.ZeroUint(),
			expectedUnstakeAsset: cosmos.ZeroUint(),
			expectedUnitLeft:     cosmos.ZeroUint(),
			expectedErr:          fmt.Errorf("withdraw basis point %s is not valid", cosmos.NewUint(12000)),
		},
		{
			name:                 "unstake-1",
			poolUnit:             cosmos.NewUint(700 * common.One),
			poolRune:             cosmos.NewUint(700 * common.One),
			poolAsset:            cosmos.NewUint(700 * common.One),
			stakerUnit:           cosmos.NewUint(200 * common.One),
			percentage:           cosmos.NewUint(10000),
			expectedUnitLeft:     cosmos.ZeroUint(),
			expectedUnstakeAsset: cosmos.NewUint(200 * common.One),
			expectedUnstakeRune:  cosmos.NewUint(200 * common.One),
			expectedErr:          nil,
		},
		{
			name:                 "unstake-2",
			poolUnit:             cosmos.NewUint(100),
			poolRune:             cosmos.NewUint(15 * common.One),
			poolAsset:            cosmos.NewUint(155 * common.One),
			stakerUnit:           cosmos.NewUint(100),
			percentage:           cosmos.NewUint(1000),
			expectedUnitLeft:     cosmos.NewUint(90),
			expectedUnstakeAsset: cosmos.NewUint(1550000000),
			expectedUnstakeRune:  cosmos.NewUint(150000000),
			expectedErr:          nil,
		},
	}

	for _, item := range inputs {
		c.Logf("name:%s", item.name)
		withDrawRune, withDrawAsset, unitAfter, err := calculateUnstake(item.poolUnit, item.poolRune, item.poolAsset, item.stakerUnit, item.percentage)
		if item.expectedErr == nil {
			c.Assert(err, IsNil)
		} else {
			c.Assert(err.Error(), Equals, item.expectedErr.Error())
		}
		c.Logf("expected rune:%s,rune:%s", item.expectedUnstakeRune, withDrawRune)
		c.Check(item.expectedUnstakeRune.Uint64(), Equals, withDrawRune.Uint64(), Commentf("Expected %d, got %d", item.expectedUnstakeRune.Uint64(), withDrawRune.Uint64()))
		c.Check(item.expectedUnstakeAsset.Uint64(), Equals, withDrawAsset.Uint64(), Commentf("Expected %d, got %d", item.expectedUnstakeAsset.Uint64(), withDrawAsset.Uint64()))
		c.Check(item.expectedUnitLeft.Uint64(), Equals, unitAfter.Uint64())
	}
}

// TestValidateUnstake is to test validateUnstake function
func (s UnstakeSuite) TestValidateUnstake(c *C) {
	accountAddr := GetRandomNodeAccount(NodeWhiteListed).NodeAddress
	runeAddress, err := common.NewAddress("bnb1g0xakzh03tpa54khxyvheeu92hwzypkdce77rm")
	if err != nil {
		c.Error("fail to create new BNB Address")
	}
	inputs := []struct {
		name          string
		msg           MsgUnStake
		expectedError error
	}{
		{
			name: "empty-rune-address",
			msg: MsgUnStake{
				RuneAddress:        "",
				UnstakeBasisPoints: cosmos.NewUint(10000),
				Asset:              common.BNBAsset,
				Tx:                 common.Tx{ID: "28B40BF105A112389A339A64BD1A042E6140DC9082C679586C6CF493A9FDE3FE"},
				Signer:             accountAddr,
			},
			expectedError: errors.New("empty rune address"),
		},
		{
			name: "empty-withdraw-basis-points",
			msg: MsgUnStake{
				RuneAddress:        runeAddress,
				UnstakeBasisPoints: cosmos.ZeroUint(),
				Asset:              common.BNBAsset,
				Tx:                 common.Tx{ID: "28B40BF105A112389A339A64BD1A042E6140DC9082C679586C6CF493A9FDE3FE"},
				Signer:             accountAddr,
			},
			expectedError: nil,
		},
		{
			name: "empty-request-txhash",
			msg: MsgUnStake{
				RuneAddress:        runeAddress,
				UnstakeBasisPoints: cosmos.NewUint(10000),
				Asset:              common.BNBAsset,
				Tx:                 common.Tx{},
				Signer:             accountAddr,
			},
			expectedError: errors.New("request tx hash is empty"),
		},
		{
			name: "empty-asset",
			msg: MsgUnStake{
				RuneAddress:        runeAddress,
				UnstakeBasisPoints: cosmos.NewUint(10000),
				Asset:              common.Asset{},
				Tx:                 common.Tx{ID: "28B40BF105A112389A339A64BD1A042E6140DC9082C679586C6CF493A9FDE3FE"},
				Signer:             accountAddr,
			},
			expectedError: errors.New("empty asset"),
		},
		{
			name: "invalid-basis-point",
			msg: MsgUnStake{
				RuneAddress:        runeAddress,
				UnstakeBasisPoints: cosmos.NewUint(10001),
				Asset:              common.BNBAsset,
				Tx:                 common.Tx{ID: "28B40BF105A112389A339A64BD1A042E6140DC9082C679586C6CF493A9FDE3FE"},
				Signer:             accountAddr,
			},
			expectedError: errors.New("withdraw basis points 10001 is invalid"),
		},
		{
			name: "invalid-pool-notexist",
			msg: MsgUnStake{
				RuneAddress:        runeAddress,
				UnstakeBasisPoints: cosmos.NewUint(10000),
				Asset:              common.Asset{Chain: common.BNBChain, Ticker: "NOTEXIST", Symbol: "NOTEXIST"},
				Tx:                 common.Tx{ID: "28B40BF105A112389A339A64BD1A042E6140DC9082C679586C6CF493A9FDE3FE"},
				Signer:             accountAddr,
			},
			expectedError: errors.New("pool-BNB.NOTEXIST doesn't exist"),
		},
		{
			name: "all-good",
			msg: MsgUnStake{
				RuneAddress:        runeAddress,
				UnstakeBasisPoints: cosmos.NewUint(10000),
				Asset:              common.BNBAsset,
				Tx:                 common.Tx{ID: "28B40BF105A112389A339A64BD1A042E6140DC9082C679586C6CF493A9FDE3FE"},
				Signer:             accountAddr,
			},
			expectedError: nil,
		},
	}

	for _, item := range inputs {
		ctx, _ := setupKeeperForTest(c)
		ps := &UnstakeTestKeeper{}
		c.Logf("name:%s", item.name)
		err := validateUnstake(ctx, ps, item.msg)
		if item.expectedError != nil {
			c.Assert(err, NotNil)
			c.Assert(err.Error(), Equals, item.expectedError.Error())
			continue
		}
		c.Assert(err, IsNil)
	}
}

func (UnstakeSuite) TestUnstake(c *C) {
	ps := &UnstakeTestKeeper{}
	remainGas := uint64(75000)
	if common.RuneAsset().Chain.Equals(common.THORChain) {
		remainGas = 37500
	}
	accountAddr := GetRandomNodeAccount(NodeWhiteListed).NodeAddress
	runeAddress, err := common.NewAddress("bnb1g0xakzh03tpa54khxyvheeu92hwzypkdce77rm")
	if err != nil {
		c.Error("fail to create new BNB Address")
	}
	testCases := []struct {
		name          string
		msg           MsgUnStake
		ps            keeper.Keeper
		runeAmount    cosmos.Uint
		assetAmount   cosmos.Uint
		expectedError error
	}{
		{
			name: "empty-rune-address",
			msg: MsgUnStake{
				RuneAddress:        "",
				UnstakeBasisPoints: cosmos.NewUint(10000),
				Asset:              common.BNBAsset,
				Tx:                 common.Tx{ID: "28B40BF105A112389A339A64BD1A042E6140DC9082C679586C6CF493A9FDE3FE"},
				Signer:             accountAddr,
			},
			ps:            ps,
			runeAmount:    cosmos.ZeroUint(),
			assetAmount:   cosmos.ZeroUint(),
			expectedError: errors.New("empty rune address"),
		},
		{
			name: "empty-withdraw-basis-points",
			msg: MsgUnStake{
				RuneAddress:        runeAddress,
				UnstakeBasisPoints: cosmos.ZeroUint(),
				Asset:              common.BNBAsset,
				Tx:                 common.Tx{ID: "28B40BF105A112389A339A64BD1A042E6140DC9082C679586C6CF493A9FDE3FE"},
				Signer:             accountAddr,
			},
			ps:            ps,
			runeAmount:    cosmos.ZeroUint(),
			assetAmount:   cosmos.ZeroUint(),
			expectedError: errors.New("nothing to withdraw"),
		},
		{
			name: "empty-request-txhash",
			msg: MsgUnStake{
				RuneAddress:        runeAddress,
				UnstakeBasisPoints: cosmos.NewUint(10000),
				Asset:              common.BNBAsset,
				Tx:                 common.Tx{},
				Signer:             accountAddr,
			},
			ps:            ps,
			runeAmount:    cosmos.ZeroUint(),
			assetAmount:   cosmos.ZeroUint(),
			expectedError: errors.New("request tx hash is empty"),
		},
		{
			name: "empty-asset",
			msg: MsgUnStake{
				RuneAddress:        runeAddress,
				UnstakeBasisPoints: cosmos.NewUint(10000),
				Asset:              common.Asset{},
				Tx:                 common.Tx{ID: "28B40BF105A112389A339A64BD1A042E6140DC9082C679586C6CF493A9FDE3FE"},
				Signer:             accountAddr,
			},
			ps:            ps,
			runeAmount:    cosmos.ZeroUint(),
			assetAmount:   cosmos.ZeroUint(),
			expectedError: errors.New("empty asset"),
		},

		{
			name: "invalid-basis-point",
			msg: MsgUnStake{
				RuneAddress:        runeAddress,
				UnstakeBasisPoints: cosmos.NewUint(10001),
				Asset:              common.BNBAsset,
				Tx:                 common.Tx{ID: "28B40BF105A112389A339A64BD1A042E6140DC9082C679586C6CF493A9FDE3FE"},
				Signer:             accountAddr,
			},
			ps:            ps,
			runeAmount:    cosmos.ZeroUint(),
			assetAmount:   cosmos.ZeroUint(),
			expectedError: errors.New("withdraw basis points 10001 is invalid"),
		},
		{
			name: "invalid-pool-notexist",
			msg: MsgUnStake{
				RuneAddress:        runeAddress,
				UnstakeBasisPoints: cosmos.NewUint(10000),
				Asset:              common.Asset{Chain: common.BNBChain, Ticker: "NOTEXIST", Symbol: "NOTEXIST"},
				Tx:                 common.Tx{ID: "28B40BF105A112389A339A64BD1A042E6140DC9082C679586C6CF493A9FDE3FE"},
				Signer:             accountAddr,
			},
			ps:            ps,
			runeAmount:    cosmos.ZeroUint(),
			assetAmount:   cosmos.ZeroUint(),
			expectedError: errors.New("pool-BNB.NOTEXIST doesn't exist"),
		},
		{
			name: "invalid-pool-staker-notexist",
			msg: MsgUnStake{
				RuneAddress:        runeAddress,
				UnstakeBasisPoints: cosmos.NewUint(10000),
				Asset:              common.Asset{Chain: common.BNBChain, Ticker: "NOTEXISTSTICKER", Symbol: "NOTEXISTSTICKER"},
				Tx:                 common.Tx{ID: "28B40BF105A112389A339A64BD1A042E6140DC9082C679586C6CF493A9FDE3FE"},
				Signer:             accountAddr,
			},
			ps:            ps,
			runeAmount:    cosmos.ZeroUint(),
			assetAmount:   cosmos.ZeroUint(),
			expectedError: errors.New("you asked for it"),
		},
		{
			name: "nothing-to-withdraw",
			msg: MsgUnStake{
				RuneAddress:        runeAddress,
				UnstakeBasisPoints: cosmos.NewUint(0),
				Asset:              common.BNBAsset,
				Tx:                 common.Tx{ID: "28B40BF105A112389A339A64BD1A042E6140DC9082C679586C6CF493A9FDE3FE"},
				Signer:             accountAddr,
			},
			ps:            ps,
			runeAmount:    cosmos.ZeroUint(),
			assetAmount:   cosmos.ZeroUint(),
			expectedError: errors.New("nothing to withdraw"),
		},
		{
			name: "all-good",
			msg: MsgUnStake{
				RuneAddress:        runeAddress,
				UnstakeBasisPoints: cosmos.NewUint(10000),
				Asset:              common.BNBAsset,
				Tx:                 common.Tx{ID: "28B40BF105A112389A339A64BD1A042E6140DC9082C679586C6CF493A9FDE3FE"},
				Signer:             accountAddr,
			},
			ps:            getUnstakeTestKeeper(c),
			runeAmount:    cosmos.NewUint(100 * common.One),
			assetAmount:   cosmos.NewUint(100 * common.One).Sub(cosmos.NewUint(remainGas)),
			expectedError: nil,
		},
		{
			name: "all-good-half",
			msg: MsgUnStake{
				RuneAddress:        runeAddress,
				UnstakeBasisPoints: cosmos.NewUint(5000),
				Asset:              common.BNBAsset,
				Tx:                 common.Tx{ID: "28B40BF105A112389A339A64BD1A042E6140DC9082C679586C6CF493A9FDE3FE"},
				Signer:             accountAddr,
			},
			ps:            getUnstakeTestKeeper(c),
			runeAmount:    cosmos.NewUint(50 * common.One),
			assetAmount:   cosmos.NewUint(50 * common.One),
			expectedError: nil,
		},
	}
	for _, tc := range testCases {
		ctx, _ := setupKeeperForTest(c)
		c.Logf("name:%s", tc.name)
		version := constants.SWVersion
		eventManager := NewDummyEventMgr()
		c.Assert(err, IsNil)
		r, asset, _, _, err := unstake(ctx, version, tc.ps, tc.msg, eventManager)
		if tc.expectedError != nil {
			c.Assert(err, NotNil)
			c.Check(err.Error(), Equals, tc.expectedError.Error())
			c.Check(r.Uint64(), Equals, tc.runeAmount.Uint64())
			c.Check(asset.Uint64(), Equals, tc.assetAmount.Uint64())
			continue
		}
		c.Assert(err, IsNil)
		c.Check(r.Uint64(), Equals, tc.runeAmount.Uint64())
		c.Check(asset.Uint64(), Equals, tc.assetAmount.Uint64())
	}
}

func getUnstakeTestKeeper(c *C) keeper.Keeper {
	runeAddress, err := common.NewAddress("bnb1g0xakzh03tpa54khxyvheeu92hwzypkdce77rm")
	if err != nil {
		c.Error("fail to create new BNB Address")
	}

	ctx, _ := setupKeeperForTest(c)

	store := NewUnstakeTestKeeper()
	pool := Pool{
		BalanceRune:  cosmos.NewUint(100 * common.One),
		BalanceAsset: cosmos.NewUint(100 * common.One),
		Asset:        common.BNBAsset,
		PoolUnits:    cosmos.NewUint(100 * common.One),
		Status:       PoolEnabled,
	}
	c.Assert(store.SetPool(ctx, pool), IsNil)
	staker := Staker{
		Asset:        pool.Asset,
		RuneAddress:  runeAddress,
		AssetAddress: runeAddress,
		Units:        cosmos.NewUint(100 * common.One),
		PendingRune:  cosmos.ZeroUint(),
	}
	store.SetStaker(ctx, staker)
	return store
}
