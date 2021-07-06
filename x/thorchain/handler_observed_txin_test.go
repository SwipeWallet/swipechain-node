package thorchain

import (
	"errors"

	"github.com/blang/semver"
	se "github.com/cosmos/cosmos-sdk/types/errors"
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
	"gitlab.com/thorchain/thornode/x/thorchain/keeper"
)

type HandlerObservedTxInSuite struct{}

type TestObservedTxInValidateKeeper struct {
	keeper.KVStoreDummy
	activeNodeAccount NodeAccount
	standbyAccount    NodeAccount
}

func (k *TestObservedTxInValidateKeeper) GetNodeAccount(_ cosmos.Context, addr cosmos.AccAddress) (NodeAccount, error) {
	if addr.Equals(k.standbyAccount.NodeAddress) {
		return k.standbyAccount, nil
	}
	if addr.Equals(k.activeNodeAccount.NodeAddress) {
		return k.activeNodeAccount, nil
	}
	return NodeAccount{}, kaboom
}

func (k *TestObservedTxInValidateKeeper) SetNodeAccount(_ cosmos.Context, na NodeAccount) error {
	if na.NodeAddress.Equals(k.standbyAccount.NodeAddress) {
		k.standbyAccount = na
		return nil
	}
	return kaboom
}

var _ = Suite(&HandlerObservedTxInSuite{})

func (s *HandlerObservedTxInSuite) TestValidate(c *C) {
	var err error
	ctx, _ := setupKeeperForTest(c)
	activeNodeAccount := GetRandomNodeAccount(NodeActive)
	standbyAccount := GetRandomNodeAccount(NodeStandby)
	keeper := &TestObservedTxInValidateKeeper{
		activeNodeAccount: activeNodeAccount,
		standbyAccount:    standbyAccount,
	}

	handler := NewObservedTxInHandler(keeper, NewDummyMgr())

	// happy path
	ver := constants.SWVersion
	pk := GetRandomPubKey()
	txs := ObservedTxs{NewObservedTx(GetRandomTx(), 12, pk)}
	txs[0].Tx.ToAddress, err = pk.GetAddress(txs[0].Tx.Coins[0].Asset.Chain)
	c.Assert(err, IsNil)
	msg := NewMsgObservedTxIn(txs, activeNodeAccount.NodeAddress)
	err = handler.validate(ctx, msg, ver)
	c.Assert(err, IsNil)

	// invalid version
	err = handler.validate(ctx, msg, semver.Version{})
	c.Assert(err, Equals, errInvalidVersion)

	// inactive node account
	msg = NewMsgObservedTxIn(txs, GetRandomBech32Addr())
	err = handler.validate(ctx, msg, ver)
	c.Assert(errors.Is(err, se.ErrUnauthorized), Equals, true)

	// invalid msg
	msg = MsgObservedTxIn{}
	err = handler.validate(ctx, msg, ver)
	c.Assert(err, NotNil)
}

type TestObservedTxInFailureKeeper struct {
	keeper.KVStoreDummy
	pool Pool
}

func (k *TestObservedTxInFailureKeeper) GetPool(_ cosmos.Context, _ common.Asset) (Pool, error) {
	return k.pool, nil
}

func (s *HandlerObservedTxInSuite) TestFailure(c *C) {
	ctx, _ := setupKeeperForTest(c)
	// w := getHandlerTestWrapper(c, 1, true, false)

	keeper := &TestObservedTxInFailureKeeper{
		pool: Pool{
			Asset:        common.BNBAsset,
			BalanceRune:  cosmos.NewUint(200),
			BalanceAsset: cosmos.NewUint(300),
		},
	}
	mgr := NewDummyMgr()

	tx := NewObservedTx(GetRandomTx(), 12, GetRandomPubKey())
	ver := constants.SWVersion
	constAccessor := constants.GetConstantValues(ver)
	err := refundTx(ctx, tx, mgr, keeper, constAccessor, CodeInvalidMemo, "Invalid memo", "")
	c.Assert(err, IsNil)
	items, err := mgr.TxOutStore().GetOutboundItems(ctx)
	c.Assert(err, IsNil)
	c.Check(items, HasLen, 1)
}

type TestObservedTxInHandleKeeper struct {
	keeper.KVStoreDummy
	nas       NodeAccounts
	voter     ObservedTxVoter
	yggExists bool
	height    int64
	msg       MsgSwap
	pool      Pool
	observing []cosmos.AccAddress
	vault     Vault
	txOut     *TxOut
}

func (k *TestObservedTxInHandleKeeper) SetSwapQueueItem(_ cosmos.Context, msg MsgSwap) error {
	k.msg = msg
	return nil
}

func (k *TestObservedTxInHandleKeeper) ListActiveNodeAccounts(_ cosmos.Context) (NodeAccounts, error) {
	return k.nas, nil
}

func (k *TestObservedTxInHandleKeeper) GetObservedTxInVoter(_ cosmos.Context, _ common.TxID) (ObservedTxVoter, error) {
	return k.voter, nil
}

func (k *TestObservedTxInHandleKeeper) SetObservedTxInVoter(_ cosmos.Context, voter ObservedTxVoter) {
	k.voter = voter
}

func (k *TestObservedTxInHandleKeeper) VaultExists(_ cosmos.Context, _ common.PubKey) bool {
	return k.yggExists
}

func (k *TestObservedTxInHandleKeeper) SetLastChainHeight(_ cosmos.Context, _ common.Chain, height int64) error {
	k.height = height
	return nil
}

func (k *TestObservedTxInHandleKeeper) GetPool(_ cosmos.Context, _ common.Asset) (Pool, error) {
	return k.pool, nil
}

func (k *TestObservedTxInHandleKeeper) AddObservingAddresses(_ cosmos.Context, addrs []cosmos.AccAddress) error {
	k.observing = addrs
	return nil
}

func (k *TestObservedTxInHandleKeeper) GetVault(_ cosmos.Context, key common.PubKey) (Vault, error) {
	if k.vault.PubKey.Equals(key) {
		return k.vault, nil
	}
	return GetRandomVault(), kaboom
}

func (k *TestObservedTxInHandleKeeper) SetVault(_ cosmos.Context, vault Vault) error {
	if k.vault.PubKey.Equals(vault.PubKey) {
		k.vault = vault
		return nil
	}
	return kaboom
}

func (k *TestObservedTxInHandleKeeper) GetLowestActiveVersion(_ cosmos.Context) semver.Version {
	return constants.SWVersion
}

func (k *TestObservedTxInHandleKeeper) IsActiveObserver(_ cosmos.Context, addr cosmos.AccAddress) bool {
	if addr.Equals(k.nas[0].NodeAddress) {
		return true
	}
	return false
}

func (k *TestObservedTxInHandleKeeper) GetTxOut(ctx cosmos.Context, blockHeight int64) (*TxOut, error) {
	if k.txOut != nil && k.txOut.Height == blockHeight {
		return k.txOut, nil
	}
	return nil, kaboom
}

func (k *TestObservedTxInHandleKeeper) SetTxOut(ctx cosmos.Context, blockOut *TxOut) error {
	if k.txOut.Height == blockOut.Height {
		k.txOut = blockOut
		return nil
	}
	return kaboom
}

func (s *HandlerObservedTxInSuite) TestHandle(c *C) {
	s.testHandleWithVersion(c, constants.SWVersion)
	s.testHandleWithVersion(c, semver.MustParse("0.13.0"))
}

func (s *HandlerObservedTxInSuite) testHandleWithVersion(c *C, ver semver.Version) {
	var err error
	ctx, _ := setupKeeperForTest(c)
	constAccessor := constants.GetConstantValues(ver)

	tx := GetRandomTx()
	tx.Memo = "SWAP:BTC.BTC:" + GetRandomBTCAddress().String()
	obTx := NewObservedTx(tx, 12, GetRandomPubKey())
	txs := ObservedTxs{obTx}
	pk := GetRandomPubKey()
	txs[0].Tx.ToAddress, err = pk.GetAddress(txs[0].Tx.Coins[0].Asset.Chain)

	vault := GetRandomVault()
	vault.PubKey = obTx.ObservedPubKey

	keeper := &TestObservedTxInHandleKeeper{
		nas:   NodeAccounts{GetRandomNodeAccount(NodeActive)},
		voter: NewObservedTxVoter(tx.ID, make(ObservedTxs, 0)),
		vault: vault,
		pool: Pool{
			Asset:        common.BNBAsset,
			BalanceRune:  cosmos.NewUint(200),
			BalanceAsset: cosmos.NewUint(300),
		},
		yggExists: true,
	}

	mgr := NewManagers(keeper)
	c.Assert(mgr.BeginBlock(ctx), IsNil)
	handler := NewObservedTxInHandler(keeper, mgr)

	c.Assert(err, IsNil)
	msg := NewMsgObservedTxIn(txs, keeper.nas[0].NodeAddress)
	_, err = handler.handle(ctx, msg, ver, constAccessor)
	c.Assert(err, IsNil)
	mgr.ObMgr().EndBlock(ctx, keeper)

	c.Check(keeper.msg.Tx.ID.Equals(tx.ID), Equals, true)
	c.Check(keeper.observing, HasLen, 1)
	c.Check(keeper.height, Equals, int64(12))
	bnbCoin := keeper.vault.Coins.GetCoin(common.BNBAsset)
	c.Assert(bnbCoin.Amount.Equal(cosmos.OneUint()), Equals, true)
}

// Test migrate memo
func (s *HandlerObservedTxInSuite) TestMigrateMemo(c *C) {
	s.testMigrateMemoWithVersion(c, constants.SWVersion)
	s.testMigrateMemoWithVersion(c, semver.MustParse("0.10.0"))
}

// Test migrate memo
func (s *HandlerObservedTxInSuite) testMigrateMemoWithVersion(c *C, ver semver.Version) {
	var err error
	ctx, _ := setupKeeperForTest(c)

	constAccessor := constants.GetConstantValues(ver)

	vault := GetRandomVault()
	addr, err := vault.PubKey.GetAddress(common.BNBChain)
	c.Assert(err, IsNil)
	newVault := GetRandomVault()
	txout := NewTxOut(12)
	newVaultAddr, err := newVault.PubKey.GetAddress(common.BNBChain)
	c.Assert(err, IsNil)

	txout.TxArray = append(txout.TxArray, &TxOutItem{
		Chain:       common.BNBChain,
		InHash:      common.BlankTxID,
		ToAddress:   newVaultAddr,
		VaultPubKey: vault.PubKey,
		Coin:        common.NewCoin(common.BNBAsset, cosmos.NewUint(1024)),
		Memo:        NewMigrateMemo(1).String(),
	})
	tx := NewObservedTx(common.Tx{
		ID:    GetRandomTxHash(),
		Chain: common.BNBChain,
		Coins: common.Coins{
			common.NewCoin(common.BNBAsset, cosmos.NewUint(1024)),
		},
		Memo:        NewMigrateMemo(12).String(),
		FromAddress: addr,
		ToAddress:   newVaultAddr,
		Gas:         BNBGasFeeSingleton,
	}, 13, vault.PubKey)

	txs := ObservedTxs{tx}
	keeper := &TestObservedTxInHandleKeeper{
		nas:   NodeAccounts{GetRandomNodeAccount(NodeActive)},
		voter: NewObservedTxVoter(tx.Tx.ID, make(ObservedTxs, 0)),
		vault: vault,
		pool: Pool{
			Asset:        common.BNBAsset,
			BalanceRune:  cosmos.NewUint(200),
			BalanceAsset: cosmos.NewUint(300),
		},
		yggExists: true,
		txOut:     txout,
	}

	handler := NewObservedTxInHandler(keeper, NewDummyMgr())

	c.Assert(err, IsNil)
	msg := NewMsgObservedTxIn(txs, keeper.nas[0].NodeAddress)
	_, err = handler.handle(ctx, msg, ver, constAccessor)
	c.Assert(err, IsNil)
}

type ObservedTxInHandlerTestHelper struct {
	keeper.Keeper
	failListActiveNodeAccounts bool
	failVaultExist             bool
	failGetObservedTxInVote    bool
	failGetVault               bool
	failSetVault               bool
}

func NewObservedTxInHandlerTestHelper(k keeper.Keeper) *ObservedTxInHandlerTestHelper {
	return &ObservedTxInHandlerTestHelper{
		Keeper: k,
	}
}

func (h *ObservedTxInHandlerTestHelper) ListActiveNodeAccounts(ctx cosmos.Context) (NodeAccounts, error) {
	if h.failListActiveNodeAccounts {
		return NodeAccounts{}, kaboom
	}
	return h.Keeper.ListActiveNodeAccounts(ctx)
}

func (h *ObservedTxInHandlerTestHelper) VaultExists(ctx cosmos.Context, pk common.PubKey) bool {
	if h.failVaultExist {
		return false
	}
	return h.Keeper.VaultExists(ctx, pk)
}

func (h *ObservedTxInHandlerTestHelper) GetObservedTxInVoter(ctx cosmos.Context, hash common.TxID) (ObservedTxVoter, error) {
	if h.failGetObservedTxInVote {
		return ObservedTxVoter{}, kaboom
	}
	return h.Keeper.GetObservedTxInVoter(ctx, hash)
}

func (h *ObservedTxInHandlerTestHelper) GetVault(ctx cosmos.Context, pk common.PubKey) (Vault, error) {
	if h.failGetVault {
		return Vault{}, kaboom
	}
	return h.Keeper.GetVault(ctx, pk)
}

func (h *ObservedTxInHandlerTestHelper) SetVault(ctx cosmos.Context, vault Vault) error {
	if h.failSetVault {
		return kaboom
	}
	return h.Keeper.SetVault(ctx, vault)
}

func setupAnLegitObservedTx(ctx cosmos.Context, helper *ObservedTxInHandlerTestHelper, c *C) MsgObservedTxIn {
	activeNodeAccount := GetRandomNodeAccount(NodeActive)
	pk := GetRandomPubKey()
	tx := GetRandomTx()
	tx.Coins = common.Coins{
		common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One*3)),
	}
	tx.Memo = "SWAP:RUNE"
	addr, err := pk.GetAddress(tx.Coins[0].Asset.Chain)
	c.Assert(err, IsNil)
	tx.ToAddress = addr
	obTx := NewObservedTx(tx, ctx.BlockHeight(), pk)
	txs := ObservedTxs{obTx}
	txs[0].Tx.ToAddress, err = pk.GetAddress(txs[0].Tx.Coins[0].Asset.Chain)
	c.Assert(err, IsNil)
	vault := GetRandomVault()
	vault.PubKey = obTx.ObservedPubKey
	helper.Keeper.SetNodeAccount(ctx, activeNodeAccount)
	c.Assert(helper.SetVault(ctx, vault), IsNil)
	p := NewPool()
	p.Asset = common.BNBAsset
	p.BalanceRune = cosmos.NewUint(100 * common.One)
	p.BalanceAsset = cosmos.NewUint(100 * common.One)
	p.Status = PoolEnabled
	helper.Keeper.SetPool(ctx, p)
	return NewMsgObservedTxIn(ObservedTxs{
		obTx,
	}, activeNodeAccount.NodeAddress)
}

func (HandlerObservedTxInSuite) TestObservedTxHandler_validations(c *C) {
	testCases := []struct {
		name            string
		messageProvider func(c *C, ctx cosmos.Context, helper *ObservedTxInHandlerTestHelper) cosmos.Msg
		validator       func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *ObservedTxInHandlerTestHelper, name string)
	}{
		{
			name: "invalid message should return an error",
			messageProvider: func(c *C, ctx cosmos.Context, helper *ObservedTxInHandlerTestHelper) cosmos.Msg {
				return NewMsgNetworkFee(ctx.BlockHeight(), common.BNBChain, 1, bnbSingleTxFee, GetRandomBech32Addr())
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *ObservedTxInHandlerTestHelper, name string) {
				c.Check(err, NotNil, Commentf(name))
				c.Check(result, IsNil)
				c.Check(errors.Is(err, errInvalidMessage), Equals, true)
			},
		},
		{
			name: "message fail validation should return an error",
			messageProvider: func(c *C, ctx cosmos.Context, helper *ObservedTxInHandlerTestHelper) cosmos.Msg {
				return NewMsgObservedTxIn(ObservedTxs{
					NewObservedTx(GetRandomTx(), ctx.BlockHeight(), GetRandomPubKey()),
				}, GetRandomBech32Addr())
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *ObservedTxInHandlerTestHelper, name string) {
				c.Check(err, NotNil, Commentf(name))
				c.Check(result, IsNil)
			},
		},
		{
			name: "signer vote for the same tx should be slashed , and not doing anything else",
			messageProvider: func(c *C, ctx cosmos.Context, helper *ObservedTxInHandlerTestHelper) cosmos.Msg {
				m := setupAnLegitObservedTx(ctx, helper, c)
				voter, err := helper.Keeper.GetObservedTxInVoter(ctx, m.Txs[0].Tx.ID)
				c.Assert(err, IsNil)
				voter.Add(m.Txs[0], m.Signer)
				helper.Keeper.SetObservedTxInVoter(ctx, voter)
				return m
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *ObservedTxInHandlerTestHelper, name string) {
				c.Check(err, IsNil, Commentf(name))
				c.Check(result, NotNil, Commentf(name))
			},
		},
		{
			name: "fail to list active node accounts should result in an error",
			messageProvider: func(c *C, ctx cosmos.Context, helper *ObservedTxInHandlerTestHelper) cosmos.Msg {
				m := setupAnLegitObservedTx(ctx, helper, c)
				helper.failListActiveNodeAccounts = true
				return m
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *ObservedTxInHandlerTestHelper, name string) {
				c.Check(err, NotNil, Commentf(name))
				c.Check(result, IsNil, Commentf(name))
			},
		},
		{
			name: "vault not exist should not result in an error, it should continue",
			messageProvider: func(c *C, ctx cosmos.Context, helper *ObservedTxInHandlerTestHelper) cosmos.Msg {
				m := setupAnLegitObservedTx(ctx, helper, c)
				helper.failVaultExist = true
				return m
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *ObservedTxInHandlerTestHelper, name string) {
				c.Check(err, IsNil, Commentf(name))
				c.Check(result, NotNil, Commentf(name))
			},
		},
		{
			name: "fail to get observedTxInVoter should not result in an error, it should continue",
			messageProvider: func(c *C, ctx cosmos.Context, helper *ObservedTxInHandlerTestHelper) cosmos.Msg {
				m := setupAnLegitObservedTx(ctx, helper, c)
				helper.failGetObservedTxInVote = true
				return m
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *ObservedTxInHandlerTestHelper, name string) {
				c.Check(err, IsNil, Commentf(name))
				c.Check(result, NotNil, Commentf(name))
			},
		},
		{
			name: "empty memo should not result in an error, it should continue",
			messageProvider: func(c *C, ctx cosmos.Context, helper *ObservedTxInHandlerTestHelper) cosmos.Msg {
				m := setupAnLegitObservedTx(ctx, helper, c)
				m.Txs[0].Tx.Memo = ""
				return m
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *ObservedTxInHandlerTestHelper, name string) {
				c.Check(err, IsNil, Commentf(name))
				c.Check(result, NotNil, Commentf(name))
				txOut, err := helper.GetTxOut(ctx, ctx.BlockHeight())
				c.Assert(err, IsNil, Commentf(name))
				c.Assert(txOut.IsEmpty(), Equals, false)
			},
		},
		{
			name: "fail to get vault, it should continue",
			messageProvider: func(c *C, ctx cosmos.Context, helper *ObservedTxInHandlerTestHelper) cosmos.Msg {
				m := setupAnLegitObservedTx(ctx, helper, c)
				helper.failGetVault = true
				return m
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *ObservedTxInHandlerTestHelper, name string) {
				c.Check(err, IsNil, Commentf(name))
				c.Check(result, NotNil, Commentf(name))
			},
		},
		{
			name: "fail to set vault, it should continue",
			messageProvider: func(c *C, ctx cosmos.Context, helper *ObservedTxInHandlerTestHelper) cosmos.Msg {
				m := setupAnLegitObservedTx(ctx, helper, c)
				helper.failSetVault = true
				return m
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *ObservedTxInHandlerTestHelper, name string) {
				c.Check(err, IsNil, Commentf(name))
				c.Check(result, NotNil, Commentf(name))
			},
		},
		{
			name: "if the vault is not asgard, it should continue",
			messageProvider: func(c *C, ctx cosmos.Context, helper *ObservedTxInHandlerTestHelper) cosmos.Msg {
				m := setupAnLegitObservedTx(ctx, helper, c)
				vault, err := helper.Keeper.GetVault(ctx, m.Txs[0].ObservedPubKey)
				c.Assert(err, IsNil)
				vault.Type = YggdrasilVault
				helper.Keeper.SetVault(ctx, vault)
				return m
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *ObservedTxInHandlerTestHelper, name string) {
				c.Check(err, IsNil, Commentf(name))
				c.Check(result, NotNil, Commentf(name))
			},
		},
		{
			name: "inactive vault, it should continue",
			messageProvider: func(c *C, ctx cosmos.Context, helper *ObservedTxInHandlerTestHelper) cosmos.Msg {
				m := setupAnLegitObservedTx(ctx, helper, c)
				vault, err := helper.Keeper.GetVault(ctx, m.Txs[0].ObservedPubKey)
				c.Assert(err, IsNil)
				vault.Status = InactiveVault
				helper.Keeper.SetVault(ctx, vault)
				return m
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *ObservedTxInHandlerTestHelper, name string) {
				c.Check(err, IsNil, Commentf(name))
				c.Check(result, NotNil, Commentf(name))
			},
		},
		{
			name: "chain halt, it should refund",
			messageProvider: func(c *C, ctx cosmos.Context, helper *ObservedTxInHandlerTestHelper) cosmos.Msg {
				m := setupAnLegitObservedTx(ctx, helper, c)
				helper.Keeper.SetMimir(ctx, "HaltTrading", 1)
				return m
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *ObservedTxInHandlerTestHelper, name string) {
				c.Check(err, IsNil, Commentf(name))
				c.Check(result, NotNil, Commentf(name))
				txOut, err := helper.GetTxOut(ctx, ctx.BlockHeight())
				c.Assert(err, IsNil, Commentf(name))
				c.Assert(txOut.IsEmpty(), Equals, false)
			},
		},
		{
			name: "normal stake , it should success",
			messageProvider: func(c *C, ctx cosmos.Context, helper *ObservedTxInHandlerTestHelper) cosmos.Msg {
				m := setupAnLegitObservedTx(ctx, helper, c)
				m.Txs[0].Tx.Memo = "stake:BNB.BNB"
				return m
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *ObservedTxInHandlerTestHelper, name string) {
				c.Check(err, IsNil, Commentf(name))
				c.Check(result, NotNil, Commentf(name))
			},
		},
	}
	versions := []semver.Version{
		constants.SWVersion,
		semver.MustParse("0.13.0"),
	}
	for _, tc := range testCases {
		for _, ver := range versions {
			ctx, k := setupKeeperForTest(c)
			helper := NewObservedTxInHandlerTestHelper(k)
			mgr := NewManagers(helper)
			mgr.BeginBlock(ctx)
			handler := NewObservedTxInHandler(helper, mgr)
			msg := tc.messageProvider(c, ctx, helper)
			constantAccessor := constants.GetConstantValues(ver)
			result, err := handler.Run(ctx, msg, ver, constantAccessor)
			tc.validator(c, ctx, result, err, helper, tc.name)
		}
	}
}
