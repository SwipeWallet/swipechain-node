package thorchain

import (
	"errors"

	"github.com/blang/semver"
	se "github.com/cosmos/cosmos-sdk/types/errors"
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
	keeper "gitlab.com/thorchain/thornode/x/thorchain/keeper"
)

type HandlerOutboundTxSuite struct{}

type TestOutboundTxKeeper struct {
	keeper.KVStoreDummy
	activeNodeAccount NodeAccount
	vault             Vault
}

func (k *TestOutboundTxKeeper) GetNodeAccount(_ cosmos.Context, addr cosmos.AccAddress) (NodeAccount, error) {
	if k.activeNodeAccount.NodeAddress.Equals(addr) {
		return k.activeNodeAccount, nil
	}
	return NodeAccount{}, nil
}

var _ = Suite(&HandlerOutboundTxSuite{})

func (s *HandlerOutboundTxSuite) SetUpSuite(c *C) {
	SetupConfigForTest()
}

func (s *HandlerOutboundTxSuite) TestValidate(c *C) {
	ctx, _ := setupKeeperForTest(c)

	keeper := &TestOutboundTxKeeper{
		activeNodeAccount: GetRandomNodeAccount(NodeActive),
		vault:             GetRandomVault(),
	}

	mgr := NewDummyMgr()
	mgr.slasher = NewSlasherV1(keeper)

	handler := NewOutboundTxHandler(keeper, mgr)

	addr, err := keeper.vault.PubKey.GetAddress(common.BNBChain)
	c.Assert(err, IsNil)

	ver := constants.SWVersion

	tx := NewObservedTx(common.Tx{
		ID:          GetRandomTxHash(),
		Chain:       common.BNBChain,
		Coins:       common.Coins{common.NewCoin(common.BNBAsset, cosmos.NewUint(1*common.One))},
		Memo:        "",
		FromAddress: GetRandomBNBAddress(),
		ToAddress:   addr,
		Gas:         BNBGasFeeSingleton,
	}, 12, GetRandomPubKey())

	msgOutboundTx := NewMsgOutboundTx(tx, tx.Tx.ID, keeper.activeNodeAccount.NodeAddress)
	sErr := handler.validate(ctx, msgOutboundTx, ver)
	c.Assert(sErr, IsNil)

	// invalid version
	sErr = handler.validate(ctx, msgOutboundTx, semver.Version{})
	c.Assert(sErr, Equals, errBadVersion)

	result, err := handler.handle(ctx, msgOutboundTx, semver.Version{}, constants.GetConstantValues(constants.SWVersion))
	c.Check(result, IsNil)
	c.Check(err, NotNil)

	// invalid msg
	msgOutboundTx = MsgOutboundTx{}
	sErr = handler.validate(ctx, msgOutboundTx, ver)
	c.Assert(sErr, NotNil)
}

type outboundTxHandlerTestHelper struct {
	ctx           cosmos.Context
	pool          Pool
	version       semver.Version
	keeper        *outboundTxHandlerKeeperHelper
	asgardVault   Vault
	yggVault      Vault
	constAccessor constants.ConstantValues
	nodeAccount   NodeAccount
	inboundTx     ObservedTx
	toi           *TxOutItem
	mgr           Manager
}

type outboundTxHandlerKeeperHelper struct {
	keeper.Keeper
	observeTxVoterErrHash common.TxID
	errGetTxOut           bool
	errGetNodeAccount     bool
	errGetPool            bool
	errSetPool            bool
	errSetNodeAccount     bool
	errGetVaultData       bool
	errSetVaultData       bool
	vault                 Vault
}

func newOutboundTxHandlerKeeperHelper(keeper keeper.Keeper) *outboundTxHandlerKeeperHelper {
	return &outboundTxHandlerKeeperHelper{
		Keeper:                keeper,
		observeTxVoterErrHash: GetRandomTxHash(),
	}
}

func (k *outboundTxHandlerKeeperHelper) GetObservedTxInVoter(ctx cosmos.Context, hash common.TxID) (ObservedTxVoter, error) {
	if hash.Equals(k.observeTxVoterErrHash) {
		return ObservedTxVoter{}, kaboom
	}
	return k.Keeper.GetObservedTxOutVoter(ctx, hash)
}

func (k *outboundTxHandlerKeeperHelper) GetTxOut(ctx cosmos.Context, height int64) (*TxOut, error) {
	if k.errGetTxOut {
		return nil, kaboom
	}
	return k.Keeper.GetTxOut(ctx, height)
}

func (k *outboundTxHandlerKeeperHelper) GetNodeAccountByPubKey(ctx cosmos.Context, pk common.PubKey) (NodeAccount, error) {
	if k.errGetNodeAccount {
		return NodeAccount{}, kaboom
	}
	return k.Keeper.GetNodeAccountByPubKey(ctx, pk)
}

func (k *outboundTxHandlerKeeperHelper) GetPool(ctx cosmos.Context, asset common.Asset) (Pool, error) {
	if k.errGetPool {
		return NewPool(), kaboom
	}
	return k.Keeper.GetPool(ctx, asset)
}

func (k *outboundTxHandlerKeeperHelper) SetPool(ctx cosmos.Context, pool Pool) error {
	if k.errSetPool {
		return kaboom
	}
	return k.Keeper.SetPool(ctx, pool)
}

func (k *outboundTxHandlerKeeperHelper) SetNodeAccount(ctx cosmos.Context, na NodeAccount) error {
	if k.errSetNodeAccount {
		return kaboom
	}
	return k.Keeper.SetNodeAccount(ctx, na)
}

func (k *outboundTxHandlerKeeperHelper) GetAsgardVaultsByStatus(ctx cosmos.Context, status VaultStatus) (Vaults, error) {
	return k.Keeper.GetAsgardVaultsByStatus(ctx, status)
}

func (k *outboundTxHandlerKeeperHelper) GetVault(ctx cosmos.Context, _ common.PubKey) (Vault, error) {
	return k.vault, nil
}

func (k *outboundTxHandlerKeeperHelper) SetVault(ctx cosmos.Context, v Vault) error {
	k.vault = v
	return nil
}

func (k *outboundTxHandlerKeeperHelper) GetVaultData(ctx cosmos.Context) (VaultData, error) {
	if k.errGetVaultData {
		return VaultData{}, kaboom
	}
	return k.Keeper.GetVaultData(ctx)
}

func (k *outboundTxHandlerKeeperHelper) SetVaultData(ctx cosmos.Context, data VaultData) error {
	if k.errSetVaultData {
		return kaboom
	}
	return k.Keeper.SetVaultData(ctx, data)
}

// newOutboundTxHandlerTestHelper setup all the basic condition to test OutboundTxHandler
func newOutboundTxHandlerTestHelper(c *C) outboundTxHandlerTestHelper {
	ctx, k := setupKeeperForTest(c)
	ctx = ctx.WithBlockHeight(1023)
	pool := NewPool()
	pool.Asset = common.BNBAsset
	pool.BalanceAsset = cosmos.NewUint(100 * common.One)
	pool.BalanceRune = cosmos.NewUint(100 * common.One)

	version := constants.SWVersion
	asgardVault := GetRandomVault()
	c.Assert(k.SetVault(ctx, asgardVault), IsNil)
	addr, err := asgardVault.PubKey.GetAddress(common.RuneAsset().Chain)
	yggVault := GetRandomVault()
	c.Assert(err, IsNil)

	tx := NewObservedTx(common.Tx{
		ID:          GetRandomTxHash(),
		Chain:       common.RuneAsset().Chain,
		Coins:       common.Coins{common.NewCoin(common.BNBAsset, cosmos.NewUint(1*common.One))},
		Memo:        "SWAP:" + common.RuneAsset().String(),
		FromAddress: GetRandomRUNEAddress(),
		ToAddress:   addr,
		Gas:         BNBGasFeeSingleton,
	}, 12, GetRandomPubKey())

	voter := NewObservedTxVoter(tx.Tx.ID, make(ObservedTxs, 0))
	keeper := newOutboundTxHandlerKeeperHelper(k)
	voter.Height = common.BlockHeight(ctx)
	keeper.SetObservedTxOutVoter(ctx, voter)

	nodeAccount := GetRandomNodeAccount(NodeActive)
	nodeAccount.NodeAddress, err = yggVault.PubKey.GetThorAddress()
	c.Assert(err, IsNil)
	nodeAccount.Bond = cosmos.NewUint(100 * common.One)
	nodeAccount.PubKeySet = common.NewPubKeySet(yggVault.PubKey, yggVault.PubKey)
	c.Assert(keeper.SetNodeAccount(ctx, nodeAccount), IsNil)

	c.Assert(keeper.SetPool(ctx, pool), IsNil)

	constAccessor := constants.GetConstantValues(version)
	txOutStorage := NewTxOutStorageV1(keeper, constAccessor, NewDummyEventMgr())
	toi := &TxOutItem{
		Chain:       common.BNBChain,
		ToAddress:   tx.Tx.FromAddress,
		VaultPubKey: yggVault.PubKey,
		Coin:        common.NewCoin(common.BNBAsset, cosmos.NewUint(2*common.One)),
		Memo:        NewOutboundMemo(tx.Tx.ID).String(),
		InHash:      tx.Tx.ID,
	}
	mgr := NewDummyMgr()
	mgr.slasher = NewSlasherV1(keeper)
	result, err := txOutStorage.TryAddTxOutItem(ctx, mgr, toi)
	c.Assert(err, IsNil)
	c.Check(result, Equals, true)

	return outboundTxHandlerTestHelper{
		ctx:           ctx,
		pool:          pool,
		version:       version,
		keeper:        keeper,
		asgardVault:   asgardVault,
		yggVault:      yggVault,
		nodeAccount:   nodeAccount,
		inboundTx:     tx,
		toi:           toi,
		constAccessor: constAccessor,
		mgr:           mgr,
	}
}

func (s *HandlerOutboundTxSuite) TestOutboundTxHandlerShouldUpdateTxOut(c *C) {
	testCases := []struct {
		name           string
		messageCreator func(helper outboundTxHandlerTestHelper, tx ObservedTx) cosmos.Msg
		runner         func(handler OutboundTxHandler, helper outboundTxHandlerTestHelper, msg cosmos.Msg) (*cosmos.Result, error)
		expectedResult error
	}{
		{
			name: "invalid message should return an error",
			messageCreator: func(helper outboundTxHandlerTestHelper, tx ObservedTx) cosmos.Msg {
				return NewMsgNoOp(GetRandomObservedTx(), helper.nodeAccount.NodeAddress)
			},
			runner: func(handler OutboundTxHandler, helper outboundTxHandlerTestHelper, msg cosmos.Msg) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, helper.version, helper.constAccessor)
			},
			expectedResult: errInvalidMessage,
		},
		{
			name: "if the version is lower than expected, it should return an error",
			messageCreator: func(helper outboundTxHandlerTestHelper, tx ObservedTx) cosmos.Msg {
				return NewMsgOutboundTx(tx, tx.Tx.ID, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler OutboundTxHandler, helper outboundTxHandlerTestHelper, msg cosmos.Msg) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, semver.MustParse("0.0.1"), helper.constAccessor)
			},
			expectedResult: errBadVersion,
		},
		{
			name: "fail to get observed TxVoter should result in an error",
			messageCreator: func(helper outboundTxHandlerTestHelper, tx ObservedTx) cosmos.Msg {
				return NewMsgOutboundTx(tx, helper.keeper.observeTxVoterErrHash, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler OutboundTxHandler, helper outboundTxHandlerTestHelper, msg cosmos.Msg) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, constants.SWVersion, helper.constAccessor)
			},
			expectedResult: errInternal,
		},
		{
			name: "fail to get txout should result in an error",
			messageCreator: func(helper outboundTxHandlerTestHelper, tx ObservedTx) cosmos.Msg {
				return NewMsgOutboundTx(tx, helper.inboundTx.Tx.ID, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler OutboundTxHandler, helper outboundTxHandlerTestHelper, msg cosmos.Msg) (*cosmos.Result, error) {
				helper.keeper.errGetTxOut = true
				return handler.Run(helper.ctx, msg, constants.SWVersion, helper.constAccessor)
			},
			expectedResult: se.ErrUnknownRequest,
		},
		{
			name: "fail to get node account should result in an error",
			messageCreator: func(helper outboundTxHandlerTestHelper, tx ObservedTx) cosmos.Msg {
				tx.Tx.Coins = append(tx.Tx.Coins, common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One)))
				return NewMsgOutboundTx(tx, helper.inboundTx.Tx.ID, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler OutboundTxHandler, helper outboundTxHandlerTestHelper, msg cosmos.Msg) (*cosmos.Result, error) {
				helper.keeper.errGetNodeAccount = true
				return handler.Run(helper.ctx, msg, constants.SWVersion, helper.constAccessor)
			},
			expectedResult: errInternal,
		},
		{
			name: "fail to get pool should result in an error",
			messageCreator: func(helper outboundTxHandlerTestHelper, tx ObservedTx) cosmos.Msg {
				tx.Tx.Coins = append(tx.Tx.Coins, common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One)))
				return NewMsgOutboundTx(tx, helper.inboundTx.Tx.ID, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler OutboundTxHandler, helper outboundTxHandlerTestHelper, msg cosmos.Msg) (*cosmos.Result, error) {
				helper.keeper.errGetPool = true
				return handler.Run(helper.ctx, msg, constants.SWVersion, helper.constAccessor)
			},
			expectedResult: errInternal,
		},
		{
			name: "fail to set pool should result in an error",
			messageCreator: func(helper outboundTxHandlerTestHelper, tx ObservedTx) cosmos.Msg {
				tx.Tx.Coins = append(tx.Tx.Coins, common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One)))
				return NewMsgOutboundTx(tx, helper.inboundTx.Tx.ID, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler OutboundTxHandler, helper outboundTxHandlerTestHelper, msg cosmos.Msg) (*cosmos.Result, error) {
				helper.keeper.errSetPool = true
				return handler.Run(helper.ctx, msg, constants.SWVersion, helper.constAccessor)
			},
			expectedResult: errInternal,
		},
		{
			name: "fail to set node account should result in an error",
			messageCreator: func(helper outboundTxHandlerTestHelper, tx ObservedTx) cosmos.Msg {
				tx.Tx.Coins = append(tx.Tx.Coins, common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One)))
				return NewMsgOutboundTx(tx, helper.inboundTx.Tx.ID, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler OutboundTxHandler, helper outboundTxHandlerTestHelper, msg cosmos.Msg) (*cosmos.Result, error) {
				helper.keeper.errSetNodeAccount = true
				return handler.Run(helper.ctx, msg, constants.SWVersion, helper.constAccessor)
			},
			expectedResult: errInternal,
		},
		{
			name: "fail to get vault data should result in an error",
			messageCreator: func(helper outboundTxHandlerTestHelper, tx ObservedTx) cosmos.Msg {
				tx.Tx.Coins = append(tx.Tx.Coins, common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One)))
				tx.Tx.Coins = append(tx.Tx.Coins, common.NewCoin(common.RuneAsset(), cosmos.NewUint(common.One*2)))
				return NewMsgOutboundTx(tx, helper.inboundTx.Tx.ID, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler OutboundTxHandler, helper outboundTxHandlerTestHelper, msg cosmos.Msg) (*cosmos.Result, error) {
				helper.keeper.errGetVaultData = true
				return handler.Run(helper.ctx, msg, constants.SWVersion, helper.constAccessor)
			},
			expectedResult: errInternal,
		},
		{
			name: "fail to set vault data should result in an error",
			messageCreator: func(helper outboundTxHandlerTestHelper, tx ObservedTx) cosmos.Msg {
				tx.Tx.Coins = append(tx.Tx.Coins, common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One)))
				tx.Tx.Coins = append(tx.Tx.Coins, common.NewCoin(common.RuneAsset(), cosmos.NewUint(common.One*2)))
				return NewMsgOutboundTx(tx, helper.inboundTx.Tx.ID, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler OutboundTxHandler, helper outboundTxHandlerTestHelper, msg cosmos.Msg) (*cosmos.Result, error) {
				helper.keeper.errSetVaultData = true
				return handler.Run(helper.ctx, msg, constants.SWVersion, helper.constAccessor)
			},
			expectedResult: errInternal,
		},
		{
			name: "valid outbound message, no event, no txout",
			messageCreator: func(helper outboundTxHandlerTestHelper, tx ObservedTx) cosmos.Msg {
				return NewMsgOutboundTx(tx, helper.inboundTx.Tx.ID, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler OutboundTxHandler, helper outboundTxHandlerTestHelper, msg cosmos.Msg) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, constants.SWVersion, helper.constAccessor)
			},
			expectedResult: nil,
		},
	}

	for _, tc := range testCases {
		helper := newOutboundTxHandlerTestHelper(c)
		handler := NewOutboundTxHandler(helper.keeper, helper.mgr)
		fromAddr, err := helper.yggVault.PubKey.GetAddress(common.BNBChain)
		c.Assert(err, IsNil)
		tx := NewObservedTx(common.Tx{
			ID:    GetRandomTxHash(),
			Chain: common.BNBChain,
			Coins: common.Coins{
				common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One)),
			},
			Memo:        NewOutboundMemo(helper.inboundTx.Tx.ID).String(),
			FromAddress: fromAddr,
			ToAddress:   helper.inboundTx.Tx.FromAddress,
			Gas:         BNBGasFeeSingleton,
		}, common.BlockHeight(helper.ctx), helper.yggVault.PubKey)
		msg := tc.messageCreator(helper, tx)
		_, err = tc.runner(handler, helper, msg)
		if tc.expectedResult == nil {
			c.Check(err, IsNil)
		} else {
			c.Check(errors.Is(err, tc.expectedResult), Equals, true, Commentf("name: %s, Err: %s", tc.name, err))
		}
	}
}

func (s *HandlerOutboundTxSuite) TestOutboundTxNormalCase(c *C) {
	helper := newOutboundTxHandlerTestHelper(c)
	handler := NewOutboundTxHandler(helper.keeper, helper.mgr)

	fromAddr, err := helper.yggVault.PubKey.GetAddress(common.BNBChain)
	c.Assert(err, IsNil)
	tx := NewObservedTx(common.Tx{
		ID:    GetRandomTxHash(),
		Chain: common.BNBChain,
		Coins: common.Coins{
			common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One)),
		},
		Memo:        NewOutboundMemo(helper.inboundTx.Tx.ID).String(),
		FromAddress: fromAddr,
		ToAddress:   helper.inboundTx.Tx.FromAddress,
		Gas:         BNBGasFeeSingleton,
	}, common.BlockHeight(helper.ctx), helper.yggVault.PubKey)
	// valid outbound message, with event, with txout
	outMsg := NewMsgOutboundTx(tx, helper.inboundTx.Tx.ID, helper.nodeAccount.NodeAddress)
	_, err = handler.Run(helper.ctx, outMsg, constants.SWVersion, helper.constAccessor)
	c.Assert(err, IsNil)
	// txout should had been complete

	txOut, err := helper.keeper.GetTxOut(helper.ctx, common.BlockHeight(helper.ctx))
	c.Assert(err, IsNil)
	c.Assert(txOut.TxArray[0].OutHash.IsEmpty(), Equals, false)
}

func (s *HandlerOutboundTxSuite) TestOuboundTxHandlerSendExtraFundShouldBeSlashed(c *C) {
	helper := newOutboundTxHandlerTestHelper(c)
	handler := NewOutboundTxHandler(helper.keeper, helper.mgr)
	fromAddr, err := helper.asgardVault.PubKey.GetAddress(common.BNBChain)
	c.Assert(err, IsNil)
	tx := NewObservedTx(common.Tx{
		ID:    GetRandomTxHash(),
		Chain: common.BNBChain,
		Coins: common.Coins{
			common.NewCoin(common.RuneAsset(), cosmos.NewUint(2*common.One)),
		},
		Memo:        NewOutboundMemo(helper.inboundTx.Tx.ID).String(),
		FromAddress: fromAddr,
		ToAddress:   helper.inboundTx.Tx.FromAddress,
		Gas:         BNBGasFeeSingleton,
	}, common.BlockHeight(helper.ctx), helper.nodeAccount.PubKeySet.Secp256k1)
	expectedBond := helper.nodeAccount.Bond.Sub(cosmos.NewUint(2 * common.One).MulUint64(3).QuoUint64(2))
	vaultData, err := helper.keeper.GetVaultData(helper.ctx)
	c.Assert(err, IsNil)
	expectedVaultTotalReserve := vaultData.TotalReserve.Add(cosmos.NewUint(common.One * 2).QuoUint64(2))
	// valid outbound message, with event, with txout
	outMsg := NewMsgOutboundTx(tx, helper.inboundTx.Tx.ID, helper.nodeAccount.NodeAddress)
	_, err = handler.Run(helper.ctx, outMsg, constants.SWVersion, helper.constAccessor)
	c.Assert(err, IsNil)
	na, err := helper.keeper.GetNodeAccount(helper.ctx, helper.nodeAccount.NodeAddress)
	c.Assert(na.Bond.Equal(expectedBond), Equals, true)
	vaultData, err = helper.keeper.GetVaultData(helper.ctx)
	c.Assert(err, IsNil)
	c.Assert(vaultData.TotalReserve.Equal(expectedVaultTotalReserve), Equals, true)
}

func (s *HandlerOutboundTxSuite) TestOutboundTxHandlerSendAdditionalCoinsShouldBeSlashed(c *C) {
	helper := newOutboundTxHandlerTestHelper(c)
	handler := NewOutboundTxHandler(helper.keeper, helper.mgr)
	fromAddr, err := helper.asgardVault.PubKey.GetAddress(common.BNBChain)
	c.Assert(err, IsNil)
	tx := NewObservedTx(common.Tx{
		ID:    GetRandomTxHash(),
		Chain: common.BNBChain,
		Coins: common.Coins{
			common.NewCoin(common.RuneAsset(), cosmos.NewUint(1*common.One)),
			common.NewCoin(common.BNBAsset, cosmos.NewUint(1*common.One)),
		},
		Memo:        NewOutboundMemo(helper.inboundTx.Tx.ID).String(),
		FromAddress: fromAddr,
		ToAddress:   helper.inboundTx.Tx.FromAddress,
		Gas:         BNBGasFeeSingleton,
	}, common.BlockHeight(helper.ctx), helper.nodeAccount.PubKeySet.Secp256k1)
	expectedBond := cosmos.NewUint(9702970297)
	// slash one BNB, and one rune
	outMsg := NewMsgOutboundTx(tx, helper.inboundTx.Tx.ID, helper.nodeAccount.NodeAddress)
	_, err = handler.Run(helper.ctx, outMsg, constants.SWVersion, helper.constAccessor)
	c.Assert(err, IsNil)
	na, err := helper.keeper.GetNodeAccount(helper.ctx, helper.nodeAccount.NodeAddress)
	c.Assert(na.Bond.Equal(expectedBond), Equals, true, Commentf("%d/%d", na.Bond.Uint64(), expectedBond.Uint64()))
}

func (s *HandlerOutboundTxSuite) TestOutboundTxHandlerInvalidObservedTxVoterShouldSlash(c *C) {
	helper := newOutboundTxHandlerTestHelper(c)
	handler := NewOutboundTxHandler(helper.keeper, helper.mgr)
	fromAddr, err := helper.asgardVault.PubKey.GetAddress(common.BNBChain)
	c.Assert(err, IsNil)
	tx := NewObservedTx(common.Tx{
		ID:    GetRandomTxHash(),
		Chain: common.BNBChain,
		Coins: common.Coins{
			common.NewCoin(common.RuneAsset(), cosmos.NewUint(1*common.One)),
			common.NewCoin(common.BNBAsset, cosmos.NewUint(1*common.One)),
		},
		Memo:        NewOutboundMemo(helper.inboundTx.Tx.ID).String(),
		FromAddress: fromAddr,
		ToAddress:   helper.inboundTx.Tx.FromAddress,
		Gas:         BNBGasFeeSingleton,
	}, common.BlockHeight(helper.ctx), helper.nodeAccount.PubKeySet.Secp256k1)

	expectedBond := cosmos.NewUint(9702970297)
	vaultData, err := helper.keeper.GetVaultData(helper.ctx)
	c.Assert(err, IsNil)
	// expected 0.5 slashed RUNE be added to reserve
	expectedVaultTotalReserve := vaultData.TotalReserve.Add(cosmos.NewUint(common.One).QuoUint64(2))
	pool, err := helper.keeper.GetPool(helper.ctx, common.BNBAsset)
	c.Assert(err, IsNil)
	poolBNB := common.SafeSub(pool.BalanceAsset, cosmos.NewUint(common.One))

	// given the outbound tx doesn't have relevant OservedTxVoter in system ,
	// thus it should be slashed with 1.5 * the full amount of assets
	outMsg := NewMsgOutboundTx(tx, tx.Tx.ID, helper.nodeAccount.NodeAddress)
	na, _ := helper.keeper.GetNodeAccount(helper.ctx, helper.nodeAccount.NodeAddress)
	_, err = handler.Run(helper.ctx, outMsg, constants.SWVersion, helper.constAccessor)
	c.Assert(err, IsNil)
	na, err = helper.keeper.GetNodeAccount(helper.ctx, helper.nodeAccount.NodeAddress)
	c.Assert(na.Bond.Equal(expectedBond), Equals, true, Commentf("%d/%d", na.Bond.Uint64(), expectedBond.Uint64()))

	vaultData, err = helper.keeper.GetVaultData(helper.ctx)
	c.Assert(err, IsNil)
	c.Assert(vaultData.TotalReserve.Equal(expectedVaultTotalReserve), Equals, true)
	pool, err = helper.keeper.GetPool(helper.ctx, common.BNBAsset)
	c.Assert(err, IsNil)
	c.Assert(pool.BalanceRune.Equal(cosmos.NewUint(10047029703)), Equals, true, Commentf("%d/%d", pool.BalanceRune.Uint64(), cosmos.NewUint(10047029703)))
	c.Assert(pool.BalanceAsset.Equal(poolBNB), Equals, true, Commentf("%d/%d", pool.BalanceAsset.Uint64(), poolBNB.Uint64()))
}
