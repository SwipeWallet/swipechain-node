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

type HandlerRefundSuite struct{}

var _ = Suite(&HandlerRefundSuite{})

type refundTxHandlerTestHelper struct {
	ctx           cosmos.Context
	pool          Pool
	version       semver.Version
	keeper        *refundTxHandlerKeeperTestHelper
	asgardVault   Vault
	yggVault      Vault
	constAccessor constants.ConstantValues
	nodeAccount   NodeAccount
	inboundTx     ObservedTx
	toi           *TxOutItem
	mgr           Manager
}

type refundTxHandlerKeeperTestHelper struct {
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

func newRefundTxHandlerKeeperTestHelper(keeper keeper.Keeper) *refundTxHandlerKeeperTestHelper {
	return &refundTxHandlerKeeperTestHelper{
		Keeper:                keeper,
		observeTxVoterErrHash: GetRandomTxHash(),
	}
}

func (k *refundTxHandlerKeeperTestHelper) GetObservedTxInVoter(ctx cosmos.Context, hash common.TxID) (ObservedTxVoter, error) {
	if hash.Equals(k.observeTxVoterErrHash) {
		return ObservedTxVoter{}, kaboom
	}
	return k.Keeper.GetObservedTxOutVoter(ctx, hash)
}

func (k *refundTxHandlerKeeperTestHelper) GetTxOut(ctx cosmos.Context, height int64) (*TxOut, error) {
	if k.errGetTxOut {
		return nil, kaboom
	}
	return k.Keeper.GetTxOut(ctx, height)
}

func (k *refundTxHandlerKeeperTestHelper) GetNodeAccountByPubKey(ctx cosmos.Context, pk common.PubKey) (NodeAccount, error) {
	if k.errGetNodeAccount {
		return NodeAccount{}, kaboom
	}
	return k.Keeper.GetNodeAccountByPubKey(ctx, pk)
}

func (k *refundTxHandlerKeeperTestHelper) GetPool(ctx cosmos.Context, asset common.Asset) (Pool, error) {
	if k.errGetPool {
		return NewPool(), kaboom
	}
	return k.Keeper.GetPool(ctx, asset)
}

func (k *refundTxHandlerKeeperTestHelper) SetPool(ctx cosmos.Context, pool Pool) error {
	if k.errSetPool {
		return kaboom
	}
	return k.Keeper.SetPool(ctx, pool)
}

func (k *refundTxHandlerKeeperTestHelper) SetNodeAccount(ctx cosmos.Context, na NodeAccount) error {
	if k.errSetNodeAccount {
		return kaboom
	}
	return k.Keeper.SetNodeAccount(ctx, na)
}

func (k *refundTxHandlerKeeperTestHelper) GetVault(ctx cosmos.Context, _ common.PubKey) (Vault, error) {
	return k.vault, nil
}

func (k *refundTxHandlerKeeperTestHelper) SetVault(ctx cosmos.Context, v Vault) error {
	k.vault = v
	return nil
}

func (k *refundTxHandlerKeeperTestHelper) GetVaultData(ctx cosmos.Context) (VaultData, error) {
	if k.errGetVaultData {
		return VaultData{}, kaboom
	}
	return k.Keeper.GetVaultData(ctx)
}

func (k *refundTxHandlerKeeperTestHelper) SetVaultData(ctx cosmos.Context, data VaultData) error {
	if k.errSetVaultData {
		return kaboom
	}
	return k.Keeper.SetVaultData(ctx, data)
}

// newRefundTxHandlerTestHelper setup all the basic condition to test OutboundTxHandler
func newRefundTxHandlerTestHelper(c *C) refundTxHandlerTestHelper {
	ctx, k := setupKeeperForTest(c)
	ctx = ctx.WithBlockHeight(1023)
	pool := NewPool()
	pool.Asset = common.BNBAsset
	pool.BalanceAsset = cosmos.NewUint(100 * common.One)
	pool.BalanceRune = cosmos.NewUint(100 * common.One)

	version := constants.SWVersion
	asgardVault := GetRandomVault()
	addr, err := asgardVault.PubKey.GetAddress(common.BNBChain)
	yggVault := GetRandomVault()
	c.Assert(err, IsNil)

	tx := NewObservedTx(common.Tx{
		ID:          GetRandomTxHash(),
		Chain:       common.BNBChain,
		Coins:       common.Coins{common.NewCoin(common.BNBAsset, cosmos.NewUint(2*common.One))},
		Memo:        "swap:RUNE-67C",
		FromAddress: GetRandomBNBAddress(),
		ToAddress:   addr,
		Gas:         BNBGasFeeSingleton,
	}, 12, GetRandomPubKey())

	voter := NewObservedTxVoter(tx.Tx.ID, make(ObservedTxs, 0))
	keeper := newRefundTxHandlerKeeperTestHelper(k)
	voter.Height = common.BlockHeight(ctx)
	keeper.SetObservedTxOutVoter(ctx, voter)

	mgr := NewDummyMgr()
	mgr.slasher = NewSlasherV1(keeper)

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
		Memo:        NewRefundMemo(tx.Tx.ID).String(),
		InHash:      tx.Tx.ID,
	}
	result, err := txOutStorage.TryAddTxOutItem(ctx, mgr, toi)
	c.Assert(err, IsNil)
	c.Check(result, Equals, true)

	return refundTxHandlerTestHelper{
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

func (s *HandlerRefundSuite) TestRefundTxHandlerShouldUpdateTxOut(c *C) {
	testCases := []struct {
		name           string
		messageCreator func(helper refundTxHandlerTestHelper, tx ObservedTx) cosmos.Msg
		runner         func(handler RefundHandler, helper refundTxHandlerTestHelper, msg cosmos.Msg) (*cosmos.Result, error)
		expectedResult error
	}{
		{
			name: "invalid message should return an error",
			messageCreator: func(helper refundTxHandlerTestHelper, tx ObservedTx) cosmos.Msg {
				return NewMsgNoOp(GetRandomObservedTx(), helper.nodeAccount.NodeAddress)
			},
			runner: func(handler RefundHandler, helper refundTxHandlerTestHelper, msg cosmos.Msg) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, helper.version, helper.constAccessor)
			},
			expectedResult: errInvalidMessage,
		},
		{
			name: "if the version is lower than expected, it should return an error",
			messageCreator: func(helper refundTxHandlerTestHelper, tx ObservedTx) cosmos.Msg {
				return NewMsgRefundTx(tx, tx.Tx.ID, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler RefundHandler, helper refundTxHandlerTestHelper, msg cosmos.Msg) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, semver.MustParse("0.0.1"), helper.constAccessor)
			},
			expectedResult: errBadVersion,
		},
		{
			name: "fail to get observed TxVoter should result in an error",
			messageCreator: func(helper refundTxHandlerTestHelper, tx ObservedTx) cosmos.Msg {
				return NewMsgRefundTx(tx, helper.keeper.observeTxVoterErrHash, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler RefundHandler, helper refundTxHandlerTestHelper, msg cosmos.Msg) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, constants.SWVersion, helper.constAccessor)
			},
			expectedResult: errInternal,
		},
		{
			name: "fail to get txout should result in an error",
			messageCreator: func(helper refundTxHandlerTestHelper, tx ObservedTx) cosmos.Msg {
				return NewMsgRefundTx(tx, helper.inboundTx.Tx.ID, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler RefundHandler, helper refundTxHandlerTestHelper, msg cosmos.Msg) (*cosmos.Result, error) {
				helper.keeper.errGetTxOut = true
				return handler.Run(helper.ctx, msg, constants.SWVersion, helper.constAccessor)
			},
			expectedResult: se.ErrUnknownRequest,
		},
		{
			name: "fail to get node account should result in an error",
			messageCreator: func(helper refundTxHandlerTestHelper, tx ObservedTx) cosmos.Msg {
				tx.Tx.Coins = append(tx.Tx.Coins, common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One)))
				return NewMsgRefundTx(tx, helper.inboundTx.Tx.ID, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler RefundHandler, helper refundTxHandlerTestHelper, msg cosmos.Msg) (*cosmos.Result, error) {
				helper.keeper.errGetNodeAccount = true
				return handler.Run(helper.ctx, msg, constants.SWVersion, helper.constAccessor)
			},
			expectedResult: errInternal,
		},
		{
			name: "fail to get pool should result in an error",
			messageCreator: func(helper refundTxHandlerTestHelper, tx ObservedTx) cosmos.Msg {
				tx.Tx.Coins = append(tx.Tx.Coins, common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One)))
				return NewMsgRefundTx(tx, helper.inboundTx.Tx.ID, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler RefundHandler, helper refundTxHandlerTestHelper, msg cosmos.Msg) (*cosmos.Result, error) {
				helper.keeper.errGetPool = true
				return handler.Run(helper.ctx, msg, constants.SWVersion, helper.constAccessor)
			},
			expectedResult: errInternal,
		},
		{
			name: "fail to set pool should result in an error",
			messageCreator: func(helper refundTxHandlerTestHelper, tx ObservedTx) cosmos.Msg {
				tx.Tx.Coins = append(tx.Tx.Coins, common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One)))
				return NewMsgRefundTx(tx, helper.inboundTx.Tx.ID, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler RefundHandler, helper refundTxHandlerTestHelper, msg cosmos.Msg) (*cosmos.Result, error) {
				helper.keeper.errSetPool = true
				return handler.Run(helper.ctx, msg, constants.SWVersion, helper.constAccessor)
			},
			expectedResult: errInternal,
		},
		{
			name: "fail to set node account should result in an error",
			messageCreator: func(helper refundTxHandlerTestHelper, tx ObservedTx) cosmos.Msg {
				tx.Tx.Coins = append(tx.Tx.Coins, common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One)))
				return NewMsgRefundTx(tx, helper.inboundTx.Tx.ID, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler RefundHandler, helper refundTxHandlerTestHelper, msg cosmos.Msg) (*cosmos.Result, error) {
				helper.keeper.errSetNodeAccount = true
				return handler.Run(helper.ctx, msg, constants.SWVersion, helper.constAccessor)
			},
			expectedResult: errInternal,
		},
		{
			name: "fail to get vault data should result in an error",
			messageCreator: func(helper refundTxHandlerTestHelper, tx ObservedTx) cosmos.Msg {
				tx.Tx.Coins = append(tx.Tx.Coins, common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One)))
				tx.Tx.Coins = append(tx.Tx.Coins, common.NewCoin(common.RuneAsset(), cosmos.NewUint(common.One*2)))
				return NewMsgRefundTx(tx, helper.inboundTx.Tx.ID, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler RefundHandler, helper refundTxHandlerTestHelper, msg cosmos.Msg) (*cosmos.Result, error) {
				helper.keeper.errGetVaultData = true
				return handler.Run(helper.ctx, msg, constants.SWVersion, helper.constAccessor)
			},
			expectedResult: errInternal,
		},
		{
			name: "fail to set vault data should result in an error",
			messageCreator: func(helper refundTxHandlerTestHelper, tx ObservedTx) cosmos.Msg {
				tx.Tx.Coins = append(tx.Tx.Coins, common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One)))
				tx.Tx.Coins = append(tx.Tx.Coins, common.NewCoin(common.RuneAsset(), cosmos.NewUint(common.One*2)))
				return NewMsgRefundTx(tx, helper.inboundTx.Tx.ID, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler RefundHandler, helper refundTxHandlerTestHelper, msg cosmos.Msg) (*cosmos.Result, error) {
				helper.keeper.errSetVaultData = true
				return handler.Run(helper.ctx, msg, constants.SWVersion, helper.constAccessor)
			},
			expectedResult: errInternal,
		},
		{
			name: "valid outbound message, no event, no txout",
			messageCreator: func(helper refundTxHandlerTestHelper, tx ObservedTx) cosmos.Msg {
				return NewMsgRefundTx(tx, helper.inboundTx.Tx.ID, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler RefundHandler, helper refundTxHandlerTestHelper, msg cosmos.Msg) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, constants.SWVersion, helper.constAccessor)
			},
			expectedResult: nil,
		},
	}

	for _, tc := range testCases {
		helper := newRefundTxHandlerTestHelper(c)
		handler := NewRefundHandler(helper.keeper, helper.mgr)
		fromAddr, err := helper.yggVault.PubKey.GetAddress(common.BNBChain)
		c.Assert(err, IsNil)
		tx := NewObservedTx(common.Tx{
			ID:    GetRandomTxHash(),
			Chain: common.BNBChain,
			Coins: common.Coins{
				common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One)),
			},
			Memo:        NewRefundMemo(helper.inboundTx.Tx.ID).String(),
			FromAddress: fromAddr,
			ToAddress:   helper.inboundTx.Tx.FromAddress,
			Gas:         BNBGasFeeSingleton,
		}, common.BlockHeight(helper.ctx), helper.yggVault.PubKey)
		msg := tc.messageCreator(helper, tx)
		_, err = tc.runner(handler, helper, msg)
		if tc.expectedResult == nil {
			c.Check(err, IsNil)
		} else {
			c.Check(errors.Is(err, tc.expectedResult), Equals, true, Commentf("name:%s", tc.name))
		}
	}
}

func (s *HandlerRefundSuite) TestRefundTxNormalCase(c *C) {
	helper := newRefundTxHandlerTestHelper(c)
	handler := NewRefundHandler(helper.keeper, helper.mgr)

	fromAddr, err := helper.yggVault.PubKey.GetAddress(common.BNBChain)
	c.Assert(err, IsNil)
	tx := NewObservedTx(common.Tx{
		ID:    GetRandomTxHash(),
		Chain: common.BNBChain,
		Coins: common.Coins{
			common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One)),
		},
		Memo:        NewRefundMemo(helper.inboundTx.Tx.ID).String(),
		FromAddress: fromAddr,
		ToAddress:   helper.inboundTx.Tx.FromAddress,
		Gas:         BNBGasFeeSingleton,
	}, common.BlockHeight(helper.ctx), helper.yggVault.PubKey)
	// valid outbound message, with event, with txout
	outMsg := NewMsgRefundTx(tx, helper.inboundTx.Tx.ID, helper.nodeAccount.NodeAddress)
	_, err = handler.Run(helper.ctx, outMsg, constants.SWVersion, helper.constAccessor)
	c.Assert(err, IsNil)

	// txout should had been complete
	txOut, err := helper.keeper.GetTxOut(helper.ctx, common.BlockHeight(helper.ctx))
	c.Assert(err, IsNil)
	c.Assert(txOut.TxArray[0].OutHash.IsEmpty(), Equals, false)
}

func (s *HandlerRefundSuite) TestRefundTxHandlerSendExtraFundShouldBeSlashed(c *C) {
	helper := newRefundTxHandlerTestHelper(c)
	handler := NewRefundHandler(helper.keeper, helper.mgr)
	fromAddr, err := helper.asgardVault.PubKey.GetAddress(common.BNBChain)
	c.Assert(err, IsNil)
	tx := NewObservedTx(common.Tx{
		ID:    GetRandomTxHash(),
		Chain: common.BNBChain,
		Coins: common.Coins{
			common.NewCoin(common.RuneAsset(), cosmos.NewUint(2*common.One)),
		},
		Memo:        NewRefundMemo(helper.inboundTx.Tx.ID).String(),
		FromAddress: fromAddr,
		ToAddress:   helper.inboundTx.Tx.FromAddress,
		Gas:         BNBGasFeeSingleton,
	}, common.BlockHeight(helper.ctx), helper.nodeAccount.PubKeySet.Secp256k1)
	expectedBond := helper.nodeAccount.Bond.Sub(cosmos.NewUint(common.One * 2).MulUint64(3).QuoUint64(2))
	vaultData, err := helper.keeper.GetVaultData(helper.ctx)
	c.Assert(err, IsNil)
	expectedVaultTotalReserve := vaultData.TotalReserve.Add(cosmos.NewUint(common.One * 2).QuoUint64(2))
	// valid outbound message, with event, with txout
	outMsg := NewMsgRefundTx(tx, helper.inboundTx.Tx.ID, helper.nodeAccount.NodeAddress)
	_, err = handler.Run(helper.ctx, outMsg, constants.SWVersion, helper.constAccessor)
	c.Assert(err, IsNil)
	na, err := helper.keeper.GetNodeAccount(helper.ctx, helper.nodeAccount.NodeAddress)
	c.Assert(na.Bond.Equal(expectedBond), Equals, true)
	vaultData, err = helper.keeper.GetVaultData(helper.ctx)
	c.Assert(err, IsNil)
	c.Assert(vaultData.TotalReserve.Equal(expectedVaultTotalReserve), Equals, true)
}

func (s *HandlerRefundSuite) TestOutboundTxHandlerSendAdditionalCoinsShouldBeSlashed(c *C) {
	helper := newRefundTxHandlerTestHelper(c)
	handler := NewRefundHandler(helper.keeper, helper.mgr)
	fromAddr, err := helper.asgardVault.PubKey.GetAddress(common.BNBChain)
	c.Assert(err, IsNil)
	tx := NewObservedTx(common.Tx{
		ID:    GetRandomTxHash(),
		Chain: common.BNBChain,
		Coins: common.Coins{
			common.NewCoin(common.RuneAsset(), cosmos.NewUint(1*common.One)),
			common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One)),
		},
		Memo:        NewRefundMemo(helper.inboundTx.Tx.ID).String(),
		FromAddress: fromAddr,
		ToAddress:   helper.inboundTx.Tx.FromAddress,
		Gas:         BNBGasFeeSingleton,
	}, common.BlockHeight(helper.ctx), helper.nodeAccount.PubKeySet.Secp256k1)
	expectedBond := cosmos.NewUint(9702970297)
	// slash one BNB and one rune
	outMsg := NewMsgRefundTx(tx, helper.inboundTx.Tx.ID, helper.nodeAccount.NodeAddress)
	_, err = handler.Run(helper.ctx, outMsg, constants.SWVersion, helper.constAccessor)
	c.Assert(err, IsNil)
	na, err := helper.keeper.GetNodeAccount(helper.ctx, helper.nodeAccount.NodeAddress)
	c.Assert(na.Bond.Equal(expectedBond), Equals, true, Commentf("Bond: %d != %d", na.Bond.Uint64(), expectedBond.Uint64()))
}

func (s *HandlerRefundSuite) TestOutboundTxHandlerInvalidObservedTxVoterShouldSlash(c *C) {
	helper := newRefundTxHandlerTestHelper(c)
	handler := NewRefundHandler(helper.keeper, helper.mgr)
	fromAddr, err := helper.asgardVault.PubKey.GetAddress(common.BNBChain)
	c.Assert(err, IsNil)
	tx := NewObservedTx(common.Tx{
		ID:    GetRandomTxHash(),
		Chain: common.BNBChain,
		Coins: common.Coins{
			common.NewCoin(common.RuneAsset(), cosmos.NewUint(1*common.One)),
			common.NewCoin(common.BNBAsset, cosmos.NewUint(1*common.One)),
		},
		Memo:        NewRefundMemo(helper.inboundTx.Tx.ID).String(),
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

	// given the outbound tx doesn't have relevant OservedTxVoter in system , thus it should be slashed with 1.5 * the full amount of assets
	outMsg := NewMsgRefundTx(tx, tx.Tx.ID, helper.nodeAccount.NodeAddress)
	_, err = handler.Run(helper.ctx, outMsg, constants.SWVersion, helper.constAccessor)
	c.Assert(err, IsNil)
	na, err := helper.keeper.GetNodeAccount(helper.ctx, helper.nodeAccount.NodeAddress)
	c.Assert(na.Bond.Equal(expectedBond), Equals, true, Commentf("Bond: %d != %d", na.Bond.Uint64(), expectedBond.Uint64()))

	vaultData, err = helper.keeper.GetVaultData(helper.ctx)
	c.Assert(err, IsNil)
	c.Assert(vaultData.TotalReserve.Equal(expectedVaultTotalReserve), Equals, true)
	pool, err = helper.keeper.GetPool(helper.ctx, common.BNBAsset)
	c.Assert(err, IsNil)
	c.Assert(pool.BalanceRune.Equal(cosmos.NewUint(10047029703)), Equals, true, Commentf("%d/%d", pool.BalanceRune.Uint64(), cosmos.NewUint(10047029703).Uint64()))
	c.Assert(pool.BalanceAsset.Equal(poolBNB), Equals, true)
}
