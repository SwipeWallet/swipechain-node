package thorchain

import (
	"errors"
	"fmt"

	"github.com/blang/semver"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store"
	se "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/params"
	"github.com/cosmos/cosmos-sdk/x/supply"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	dbm "github.com/tendermint/tm-db"
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"

	"gitlab.com/thorchain/thornode/x/thorchain/keeper"
	kv1 "gitlab.com/thorchain/thornode/x/thorchain/keeper/v1"
	"gitlab.com/thorchain/thornode/x/thorchain/types"
)

var kaboom = errors.New("kaboom!!!!!")

type HandlerSuite struct{}

var _ = Suite(&HandlerSuite{})

func (s *HandlerSuite) SetUpSuite(*C) {
	SetupConfigForTest()
}

func FundModule(c *C, ctx cosmos.Context, k keeper.Keeper, name string, amt uint64) {
	coin, err := common.NewCoin(common.RuneNative, cosmos.NewUint(amt*common.One)).Native()
	c.Assert(err, IsNil)
	err = k.Supply().MintCoins(ctx, ModuleName, cosmos.NewCoins(coin))
	c.Assert(err, IsNil)
	err = k.Supply().SendCoinsFromModuleToModule(ctx, ModuleName, name, cosmos.NewCoins(coin))
	c.Assert(err, IsNil)
}

func FundAccount(c *C, ctx cosmos.Context, k keeper.Keeper, addr cosmos.AccAddress, amt uint64) {
	coin, err := common.NewCoin(common.RuneNative, cosmos.NewUint(amt*common.One)).Native()
	c.Assert(err, IsNil)
	err = k.Supply().MintCoins(ctx, ModuleName, cosmos.NewCoins(coin))
	c.Assert(err, IsNil)
	err = k.Supply().SendCoinsFromModuleToAccount(ctx, ModuleName, addr, cosmos.NewCoins(coin))
	c.Assert(err, IsNil)
}

// nolint: deadcode unused
// create a codec used only for testing
func makeTestCodec() *codec.Codec {
	cdc := codec.New()
	bank.RegisterCodec(cdc)
	auth.RegisterCodec(cdc)
	RegisterCodec(cdc)
	supply.RegisterCodec(cdc)
	cosmos.RegisterCodec(cdc)
	codec.RegisterCrypto(cdc)
	return cdc
}

var (
	multiPerm    = "multiple permissions account"
	randomPerm   = "random permission"
	holder       = "holder"
	keyThorchain = cosmos.NewKVStoreKey(StoreKey)
)

func setupKeeperForTest(c *C) (cosmos.Context, keeper.Keeper) {
	keyAcc := cosmos.NewKVStoreKey(auth.StoreKey)
	keyParams := cosmos.NewKVStoreKey(params.StoreKey)
	tkeyParams := cosmos.NewTransientStoreKey(params.TStoreKey)
	keySupply := cosmos.NewKVStoreKey(supply.StoreKey)

	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(keyAcc, cosmos.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keySupply, cosmos.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyParams, cosmos.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyThorchain, cosmos.StoreTypeIAVL, db)
	ms.MountStoreWithDB(tkeyParams, cosmos.StoreTypeTransient, db)
	err := ms.LoadLatestVersion()
	c.Assert(err, IsNil)

	// if you would like to see the log , you can replace log.NewNopLogger with log.NewTMLogger(os.Stdout)
	ctx := cosmos.NewContext(ms, abci.Header{ChainID: "thorchain"}, false, log.NewNopLogger())
	ctx = ctx.WithBlockHeight(18)
	cdc := makeTestCodec()

	pk := params.NewKeeper(cdc, keyParams, tkeyParams)
	ak := auth.NewAccountKeeper(cdc, keyAcc, pk.Subspace(auth.DefaultParamspace), auth.ProtoBaseAccount)
	bk := bank.NewBaseKeeper(ak, pk.Subspace(bank.DefaultParamspace), nil)

	maccPerms := map[string][]string{
		auth.FeeCollectorName: nil,
		holder:                nil,
		supply.Minter:         {supply.Minter},
		supply.Burner:         {supply.Burner},
		multiPerm:             {supply.Minter, supply.Burner, supply.Staking},
		randomPerm:            {"random"},
		ModuleName:            {supply.Minter},
		ReserveName:           {},
		AsgardName:            {},
		BondName:              {supply.Staking},
	}
	supplyKeeper := supply.NewKeeper(cdc, keySupply, ak, bk, maccPerms)
	totalSupply := cosmos.NewCoins(cosmos.NewCoin("bep", cosmos.NewInt(1000*common.One)))
	supplyKeeper.SetSupply(ctx, supply.NewSupply(totalSupply))
	k := kv1.NewKVStore(bk, supplyKeeper, keyThorchain, cdc)

	FundModule(c, ctx, k, AsgardName, 100000000)

	// set bnb gas
	k.SetGas(ctx, common.BNBAsset, []cosmos.Uint{
		cosmos.NewUint(37500),
		cosmos.NewUint(30000),
	})
	return ctx, k
}

type handlerTestWrapper struct {
	ctx                  cosmos.Context
	keeper               keeper.Keeper
	mgr                  Manager
	activeNodeAccount    NodeAccount
	notActiveNodeAccount NodeAccount
}

func getHandlerTestWrapper(c *C, height int64, withActiveNode, withActieBNBPool bool) handlerTestWrapper {
	return getHandlerTestWrapperWithVersion(c, height, withActiveNode, withActieBNBPool, constants.SWVersion)
}

func getHandlerTestWrapperWithVersion(c *C, height int64, withActiveNode, withActieBNBPool bool, version semver.Version) handlerTestWrapper {
	ctx, k := setupKeeperForTest(c)
	ctx = ctx.WithBlockHeight(height)
	acc1 := GetRandomNodeAccount(NodeActive)
	acc1.Version = version
	if withActiveNode {
		c.Assert(k.SetNodeAccount(ctx, acc1), IsNil)
	}
	if withActieBNBPool {
		p, err := k.GetPool(ctx, common.BNBAsset)
		c.Assert(err, IsNil)
		p.Asset = common.BNBAsset
		p.Status = PoolEnabled
		p.BalanceRune = cosmos.NewUint(100 * common.One)
		p.BalanceAsset = cosmos.NewUint(100 * common.One)
		c.Assert(k.SetPool(ctx, p), IsNil)
	}
	constAccessor := constants.GetConstantValues(version)
	mgr := NewManagers(k)
	c.Assert(mgr.BeginBlock(ctx), IsNil)

	FundModule(c, ctx, k, AsgardName, 100000000)

	c.Assert(mgr.ValidatorMgr().BeginBlock(ctx, constAccessor), IsNil)

	return handlerTestWrapper{
		ctx:                  ctx,
		keeper:               k,
		mgr:                  mgr,
		activeNodeAccount:    acc1,
		notActiveNodeAccount: GetRandomNodeAccount(NodeDisabled),
	}
}

func (HandlerSuite) TestIsSignedByActiveNodeAccounts(c *C) {
	ctx, k := setupKeeperForTest(c)
	nodeAddr := GetRandomBech32Addr()
	c.Check(isSignedByActiveNodeAccounts(ctx, k, []cosmos.AccAddress{}), Equals, false)
	c.Check(isSignedByActiveNodeAccounts(ctx, k, []cosmos.AccAddress{nodeAddr}), Equals, false)
	nodeAccount1 := GetRandomNodeAccount(NodeWhiteListed)
	c.Assert(k.SetNodeAccount(ctx, nodeAccount1), IsNil)
	c.Check(isSignedByActiveNodeAccounts(ctx, k, []cosmos.AccAddress{nodeAccount1.NodeAddress}), Equals, false)
}

func (HandlerSuite) TestHandleTxInUnstakeMemo(c *C) {
	w := getHandlerTestWrapper(c, 1, true, false)

	vault := GetRandomVault()
	vault.Coins = common.Coins{
		common.NewCoin(common.BNBAsset, cosmos.NewUint(100*common.One)),
		common.NewCoin(common.RuneAsset(), cosmos.NewUint(100*common.One)),
	}
	w.keeper.SetVault(w.ctx, vault)
	vaultAddr, err := vault.PubKey.GetAddress(common.BNBChain)

	pool := NewPool()
	pool.Asset = common.BNBAsset
	pool.BalanceAsset = cosmos.NewUint(100 * common.One)
	pool.BalanceRune = cosmos.NewUint(100 * common.One)
	pool.PoolUnits = cosmos.NewUint(100)
	c.Assert(w.keeper.SetPool(w.ctx, pool), IsNil)

	runeAddr := GetRandomRUNEAddress()
	staker := Staker{
		Asset:        common.BNBAsset,
		RuneAddress:  runeAddr,
		AssetAddress: GetRandomBNBAddress(),
		PendingRune:  cosmos.ZeroUint(),
		Units:        cosmos.NewUint(100),
	}
	w.keeper.SetStaker(w.ctx, staker)

	tx := common.Tx{
		ID:    GetRandomTxHash(),
		Chain: common.BNBChain,
		Coins: common.Coins{
			common.NewCoin(common.RuneAsset(), cosmos.NewUint(1*common.One)),
		},
		Memo:        "withdraw:BNB.BNB",
		FromAddress: staker.RuneAddress,
		ToAddress:   vaultAddr,
		Gas:         BNBGasFeeSingleton,
	}

	msg := NewMsgUnStake(tx, staker.RuneAddress, cosmos.NewUint(uint64(MaxUnstakeBasisPoints)), common.BNBAsset, w.activeNodeAccount.NodeAddress)
	c.Assert(err, IsNil)

	handler := NewInternalHandler(w.keeper, w.mgr)

	FundModule(c, w.ctx, w.keeper, AsgardName, 500)

	_, err = handler(w.ctx, msg)
	c.Assert(err, IsNil)

	pool, err = w.keeper.GetPool(w.ctx, common.BNBAsset)
	c.Assert(err, IsNil)
	c.Assert(pool.IsEmpty(), Equals, false)
	c.Check(pool.Status, Equals, PoolBootstrap)
	c.Check(pool.PoolUnits.Uint64(), Equals, uint64(0), Commentf("%d", pool.PoolUnits.Uint64()))
	c.Check(pool.BalanceRune.Uint64(), Equals, uint64(0), Commentf("%d", pool.BalanceRune.Uint64()))
	remainGas := uint64(75000)
	if common.RuneAsset().Chain.Equals(common.THORChain) {
		remainGas = 37500
	}
	c.Check(pool.BalanceAsset.Uint64(), Equals, remainGas, Commentf("%d", pool.BalanceAsset.Uint64())) // leave a little behind for gas
}

func (HandlerSuite) TestRefund(c *C) {
	w := getHandlerTestWrapper(c, 1, true, false)

	pool := Pool{
		Asset:        common.BNBAsset,
		BalanceRune:  cosmos.NewUint(100 * common.One),
		BalanceAsset: cosmos.NewUint(100 * common.One),
	}
	c.Assert(w.keeper.SetPool(w.ctx, pool), IsNil)

	vault := GetRandomVault()
	c.Assert(w.keeper.SetVault(w.ctx, vault), IsNil)

	txin := NewObservedTx(
		common.Tx{
			ID:    GetRandomTxHash(),
			Chain: common.BNBChain,
			Coins: common.Coins{
				common.NewCoin(common.BNBAsset, cosmos.NewUint(100*common.One)),
			},
			Memo:        "withdraw:BNB.BNB",
			FromAddress: GetRandomBNBAddress(),
			ToAddress:   GetRandomBNBAddress(),
			Gas:         BNBGasFeeSingleton,
		},
		1024,
		vault.PubKey,
	)
	ver := constants.SWVersion
	constAccessor := constants.GetConstantValues(ver)
	txOutStore := w.mgr.TxOutStore()
	c.Assert(refundTx(w.ctx, txin, w.mgr, w.keeper, constAccessor, 0, "refund", ""), IsNil)
	items, err := txOutStore.GetOutboundItems(w.ctx)
	c.Assert(err, IsNil)
	c.Assert(items, HasLen, 1)

	// check THORNode DONT create a refund transaction when THORNode don't have a pool for
	// the asset sent.
	lokiAsset, _ := common.NewAsset(fmt.Sprintf("BNB.LOKI"))
	txin.Tx.Coins = common.Coins{
		common.NewCoin(lokiAsset, cosmos.NewUint(100*common.One)),
	}

	c.Assert(refundTx(w.ctx, txin, w.mgr, w.keeper, constAccessor, 0, "refund", ""), IsNil)
	items, err = txOutStore.GetOutboundItems(w.ctx)
	c.Assert(err, IsNil)
	c.Assert(items, HasLen, 1)

	pool, err = w.keeper.GetPool(w.ctx, lokiAsset)
	c.Assert(err, IsNil)
	// pool should be zero since we drop coins we don't recognize on the floor
	c.Assert(pool.BalanceAsset.Equal(cosmos.ZeroUint()), Equals, true, Commentf("%d", pool.BalanceAsset.Uint64()))

	// doing it a second time should keep it at zero
	c.Assert(refundTx(w.ctx, txin, w.mgr, w.keeper, constAccessor, 0, "refund", ""), IsNil)
	items, err = txOutStore.GetOutboundItems(w.ctx)
	c.Assert(err, IsNil)
	c.Assert(items, HasLen, 1)
	pool, err = w.keeper.GetPool(w.ctx, lokiAsset)
	c.Assert(err, IsNil)
	c.Assert(pool.BalanceAsset.Equal(cosmos.ZeroUint()), Equals, true)
}

func (HandlerSuite) TestGetMsgSwapFromMemo(c *C) {
	m, err := ParseMemo("swap:BNB.BNB")
	swapMemo, ok := m.(SwapMemo)
	c.Assert(ok, Equals, true)
	c.Assert(err, IsNil)

	txin := types.NewObservedTx(
		common.Tx{
			ID:    GetRandomTxHash(),
			Chain: common.BNBChain,
			Coins: common.Coins{
				common.NewCoin(
					common.RuneAsset(),
					cosmos.NewUint(100*common.One),
				),
			},
			Memo:        m.String(),
			FromAddress: GetRandomBNBAddress(),
			ToAddress:   GetRandomBNBAddress(),
			Gas:         BNBGasFeeSingleton,
		},
		1024,
		common.EmptyPubKey,
	)

	resultMsg1, err := getMsgSwapFromMemo(swapMemo, txin, GetRandomBech32Addr())
	c.Assert(resultMsg1, NotNil)
	c.Assert(err, IsNil)
}

func (HandlerSuite) TestGetMsgUnstakeFromMemo(c *C) {
	w := getHandlerTestWrapper(c, 1, true, false)
	tx := GetRandomTx()
	tx.Memo = "withdraw:10000"
	if common.RuneAsset().Equals(common.RuneNative) {
		tx.FromAddress = GetRandomTHORAddress()
	}
	obTx := NewObservedTx(tx, w.ctx.BlockHeight(), GetRandomPubKey())
	msg, err := processOneTxIn(w.ctx, w.keeper, obTx, w.activeNodeAccount.NodeAddress)
	c.Assert(err, IsNil)
	c.Assert(msg, NotNil)
	c.Assert(msg.Type(), Equals, MsgUnStake{}.Type())
}

func (HandlerSuite) TestGetMsgMigrationFromMemo(c *C) {
	w := getHandlerTestWrapper(c, 1, true, false)
	tx := GetRandomTx()
	tx.Memo = "migrate:10"
	obTx := NewObservedTx(tx, w.ctx.BlockHeight(), GetRandomPubKey())
	msg, err := processOneTxIn(w.ctx, w.keeper, obTx, w.activeNodeAccount.NodeAddress)
	c.Assert(err, IsNil)
	c.Assert(msg, NotNil)
	c.Assert(msg.Type(), Equals, MsgMigrate{}.Type())
}

func (HandlerSuite) TestGetMsgBondFromMemo(c *C) {
	w := getHandlerTestWrapper(c, 1, true, false)
	tx := GetRandomTx()
	tx.Coins = common.Coins{
		common.NewCoin(common.RuneAsset(), cosmos.NewUint(100*common.One)),
	}
	tx.Memo = "bond:" + GetRandomBech32Addr().String()
	obTx := NewObservedTx(tx, w.ctx.BlockHeight(), GetRandomPubKey())
	msg, err := processOneTxIn(w.ctx, w.keeper, obTx, w.activeNodeAccount.NodeAddress)
	c.Assert(err, IsNil)
	c.Assert(msg, NotNil)
	c.Assert(msg.Type(), Equals, MsgBond{}.Type())
}

func (HandlerSuite) TestGetMsgUnBondFromMemo(c *C) {
	w := getHandlerTestWrapper(c, 1, true, false)
	tx := GetRandomTx()
	tx.Coins = common.Coins{
		common.NewCoin(common.RuneAsset(), cosmos.NewUint(100*common.One)),
	}
	tx.Memo = "unbond:" + GetRandomTHORAddress().String() + ":1000"
	obTx := NewObservedTx(tx, w.ctx.BlockHeight(), GetRandomPubKey())
	msg, err := processOneTxIn(w.ctx, w.keeper, obTx, w.activeNodeAccount.NodeAddress)
	c.Assert(err, IsNil)
	c.Assert(msg, NotNil)
	c.Assert(msg.Type(), Equals, MsgUnBond{}.Type())
}

func (HandlerSuite) TestGetMsgStakeFromMemo(c *C) {
	w := getHandlerTestWrapper(c, 1, true, false)
	// Stake BNB, however THORNode send T-CAN as coin , which is incorrect, should result in an error
	m, err := ParseMemo(fmt.Sprintf("stake:BNB.BNB:%s", GetRandomRUNEAddress()))
	c.Assert(err, IsNil)
	stakeMemo, ok := m.(StakeMemo)
	c.Assert(ok, Equals, true)
	tcanAsset, err := common.NewAsset("BNB.TCAN-014")
	c.Assert(err, IsNil)
	runeAsset := common.RuneAsset()
	c.Assert(err, IsNil)

	txin := types.NewObservedTx(
		common.Tx{
			ID:    GetRandomTxHash(),
			Chain: common.BNBChain,
			Coins: common.Coins{
				common.NewCoin(tcanAsset,
					cosmos.NewUint(100*common.One)),
				common.NewCoin(runeAsset,
					cosmos.NewUint(100*common.One)),
			},
			Memo:        m.String(),
			FromAddress: GetRandomBNBAddress(),
			ToAddress:   GetRandomBNBAddress(),
			Gas:         BNBGasFeeSingleton,
		},
		1024,
		common.EmptyPubKey,
	)

	msg, err := getMsgStakeFromMemo(w.ctx, stakeMemo, txin, GetRandomBech32Addr())
	c.Assert(msg, NotNil)
	c.Assert(err, IsNil)

	// Asymentic stake should works fine, only RUNE
	txin.Tx.Coins = common.Coins{
		common.NewCoin(runeAsset,
			cosmos.NewUint(100*common.One)),
	}

	// stake only rune should be fine
	msg1, err1 := getMsgStakeFromMemo(w.ctx, stakeMemo, txin, GetRandomBech32Addr())
	c.Assert(msg1, NotNil)
	c.Assert(err1, IsNil)

	bnbAsset, err := common.NewAsset("BNB.BNB")
	c.Assert(err, IsNil)
	txin.Tx.Coins = common.Coins{
		common.NewCoin(bnbAsset,
			cosmos.NewUint(100*common.One)),
	}

	// stake only token(BNB) should be fine
	msg2, err2 := getMsgStakeFromMemo(w.ctx, stakeMemo, txin, GetRandomBech32Addr())
	c.Assert(msg2, NotNil)
	c.Assert(err2, IsNil)

	lokiAsset, _ := common.NewAsset(fmt.Sprintf("BNB.LOKI"))
	// Make sure the RUNE Address and Asset Address set correctly
	txin.Tx.Coins = common.Coins{
		common.NewCoin(runeAsset,
			cosmos.NewUint(100*common.One)),
		common.NewCoin(lokiAsset,
			cosmos.NewUint(100*common.One)),
	}

	runeAddr := txin.Tx.FromAddress
	if common.RuneAsset().Chain.Equals(common.THORChain) {
		runeAddr = GetRandomRUNEAddress()
	}
	lokiStakeMemo, err := ParseMemo(fmt.Sprintf("stake:BNB.LOKI:%s", runeAddr))
	c.Assert(err, IsNil)
	msg4, err4 := getMsgStakeFromMemo(w.ctx, lokiStakeMemo.(StakeMemo), txin, GetRandomBech32Addr())
	c.Assert(err4, IsNil)
	c.Assert(msg4, NotNil)
	msgStake := msg4.(MsgStake)
	c.Assert(msgStake, NotNil)
	c.Assert(msgStake.RuneAddress, Equals, runeAddr)
	c.Assert(msgStake.AssetAddress, Equals, txin.Tx.FromAddress)
}

func (HandlerSuite) TestMsgLeaveFromMemo(c *C) {
	w := getHandlerTestWrapper(c, 1, true, false)
	addr := types.GetRandomBech32Addr()
	txin := types.NewObservedTx(
		common.Tx{
			ID:          GetRandomTxHash(),
			Chain:       common.BNBChain,
			Coins:       common.Coins{common.NewCoin(common.RuneAsset(), cosmos.NewUint(1))},
			Memo:        fmt.Sprintf("LEAVE:%s", addr.String()),
			FromAddress: GetRandomBNBAddress(),
			ToAddress:   GetRandomBNBAddress(),
			Gas:         BNBGasFeeSingleton,
		},
		1024,
		common.EmptyPubKey,
	)

	msg, err := processOneTxIn(w.ctx, w.keeper, txin, addr)
	c.Assert(err, IsNil)
	c.Check(msg.ValidateBasic(), IsNil)
}

func (HandlerSuite) TestYggdrasilMemo(c *C) {
	w := getHandlerTestWrapper(c, 1, true, false)
	addr := types.GetRandomBech32Addr()
	txin := types.NewObservedTx(
		common.Tx{
			ID:          GetRandomTxHash(),
			Chain:       common.BNBChain,
			Coins:       common.Coins{common.NewCoin(common.RuneAsset(), cosmos.NewUint(1))},
			Memo:        "yggdrasil+:1024",
			FromAddress: GetRandomBNBAddress(),
			ToAddress:   GetRandomBNBAddress(),
			Gas:         BNBGasFeeSingleton,
		},
		1024,
		GetRandomPubKey(),
	)

	msg, err := processOneTxIn(w.ctx, w.keeper, txin, addr)
	c.Assert(err, IsNil)
	c.Check(msg.ValidateBasic(), IsNil)

	txin.Tx.Memo = "yggdrasil-:1024"
	msg, err = processOneTxIn(w.ctx, w.keeper, txin, addr)
	c.Assert(err, IsNil)
	c.Check(msg.ValidateBasic(), IsNil)
}

func (s *HandlerSuite) TestReserveContributor(c *C) {
	w := getHandlerTestWrapper(c, 1, true, false)
	addr := types.GetRandomBech32Addr()
	txin := types.NewObservedTx(
		common.Tx{
			ID:          GetRandomTxHash(),
			Chain:       common.BNBChain,
			Coins:       common.Coins{common.NewCoin(common.RuneAsset(), cosmos.NewUint(1))},
			Memo:        "reserve",
			FromAddress: GetRandomBNBAddress(),
			ToAddress:   GetRandomBNBAddress(),
			Gas:         BNBGasFeeSingleton,
		},
		1024,
		GetRandomPubKey(),
	)

	msg, err := processOneTxIn(w.ctx, w.keeper, txin, addr)
	c.Assert(err, IsNil)
	c.Check(msg.ValidateBasic(), IsNil)
	c.Check(msg.Type(), Equals, MsgReserveContributor{}.Type())
}

func (s *HandlerSuite) TestSwitch(c *C) {
	w := getHandlerTestWrapper(c, 1, true, false)
	addr := types.GetRandomBech32Addr()
	txin := types.NewObservedTx(
		common.Tx{
			ID:          GetRandomTxHash(),
			Chain:       common.BNBChain,
			Coins:       common.Coins{common.NewCoin(common.RuneAsset(), cosmos.NewUint(1))},
			Memo:        "switch:" + GetRandomBech32Addr().String(),
			FromAddress: GetRandomBNBAddress(),
			ToAddress:   GetRandomBNBAddress(),
			Gas:         BNBGasFeeSingleton,
		},
		1024,
		GetRandomPubKey(),
	)

	msg, err := processOneTxIn(w.ctx, w.keeper, txin, addr)
	c.Assert(err, IsNil)
	c.Check(msg.ValidateBasic(), IsNil)
	c.Check(msg.Type(), Equals, MsgSwitch{}.Type())
}

func (s *HandlerSuite) TestExternalHandler(c *C) {
	ctx, k := setupKeeperForTest(c)
	mgr := NewManagers(k)
	handler := NewExternalHandler(k, mgr)
	ctx = ctx.WithBlockHeight(1024)
	msg := NewMsgNetworkFee(1024, common.BNBChain, 1, bnbSingleTxFee, GetRandomBech32Addr())
	result, err := handler(ctx, msg)
	c.Check(err, NotNil)
	c.Check(errors.Is(err, se.ErrUnauthorized), Equals, true)
	c.Check(result, IsNil)
	na := GetRandomNodeAccount(NodeActive)
	k.SetNodeAccount(ctx, na)
	result, err = handler(ctx, NewMsgSetVersion(semver.MustParse("0.1.0"), na.NodeAddress))
	c.Assert(result, NotNil)
	c.Assert(err, IsNil)
}

func (s *HandlerSuite) TestFetchMemo(c *C) {
	w := getHandlerTestWrapper(c, 1, true, true)
	tx := GetRandomTx()
	tx.Memo = ""
	w.keeper.SetTxMarkers(w.ctx, tx.Hash(), TxMarkers{
		NewTxMarker(w.ctx.BlockHeight(), "HelloWorld"),
	})
	constantAccessor := constants.GetConstantValues(constants.SWVersion)
	memo := fetchMemo(w.ctx, constantAccessor, w.keeper, tx)
	c.Assert(memo, Equals, "HelloWorld")
}
