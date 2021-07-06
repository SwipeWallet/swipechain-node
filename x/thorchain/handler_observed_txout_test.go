package thorchain

import (
	"errors"
	"fmt"

	"github.com/blang/semver"
	se "github.com/cosmos/cosmos-sdk/types/errors"
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
	"gitlab.com/thorchain/thornode/x/thorchain/keeper"
)

type HandlerObservedTxOutSuite struct{}

type TestObservedTxOutValidateKeeper struct {
	keeper.KVStoreDummy
	activeNodeAccount NodeAccount
}

func (k *TestObservedTxOutValidateKeeper) GetNodeAccount(ctx cosmos.Context, signer cosmos.AccAddress) (NodeAccount, error) {
	if k.activeNodeAccount.NodeAddress.Equals(signer) {
		return k.activeNodeAccount, nil
	}
	return NodeAccount{}, nil
}

var _ = Suite(&HandlerObservedTxOutSuite{})

func (s *HandlerObservedTxOutSuite) TestValidate(c *C) {
	s.testValidateWithVersion(c, constants.SWVersion)
	s.testValidateWithVersion(c, semver.MustParse("0.13.0"))
}

func (s *HandlerObservedTxOutSuite) testValidateWithVersion(c *C, ver semver.Version) {
	var err error
	ctx, _ := setupKeeperForTest(c)
	activeNodeAccount := GetRandomNodeAccount(NodeActive)

	keeper := &TestObservedTxOutValidateKeeper{
		activeNodeAccount: activeNodeAccount,
	}

	handler := NewObservedTxOutHandler(keeper, NewDummyMgr())

	// happy path
	pk := GetRandomPubKey()
	txs := ObservedTxs{NewObservedTx(GetRandomTx(), 12, pk)}
	txs[0].Tx.FromAddress, err = pk.GetAddress(txs[0].Tx.Coins[0].Asset.Chain)
	c.Assert(err, IsNil)
	msg := NewMsgObservedTxOut(txs, activeNodeAccount.NodeAddress)
	err = handler.validate(ctx, msg, ver)
	c.Assert(err, IsNil)

	// invalid version
	err = handler.validate(ctx, msg, semver.Version{})
	c.Assert(err, Equals, errInvalidVersion)

	// inactive node account
	msg = NewMsgObservedTxOut(txs, GetRandomBech32Addr())
	err = handler.validate(ctx, msg, ver)
	c.Assert(errors.Is(err, se.ErrUnauthorized), Equals, true)

	// invalid msg
	msg = MsgObservedTxOut{}
	err = handler.validate(ctx, msg, ver)
	c.Assert(err, NotNil)
}

type TestObservedTxOutFailureKeeper struct {
	keeper.KVStoreDummy
}

type TestObservedTxOutHandleKeeper struct {
	keeper.KVStoreDummy
	nas        NodeAccounts
	na         NodeAccount
	voter      ObservedTxVoter
	yggExists  bool
	ygg        Vault
	height     int64
	pool       Pool
	txOutStore TxOutStore
	observing  []cosmos.AccAddress
	gas        []cosmos.Uint
}

func (k *TestObservedTxOutHandleKeeper) ListActiveNodeAccounts(_ cosmos.Context) (NodeAccounts, error) {
	return k.nas, nil
}

func (k *TestObservedTxOutHandleKeeper) IsActiveObserver(_ cosmos.Context, _ cosmos.AccAddress) bool {
	return true
}

func (k *TestObservedTxOutHandleKeeper) GetNodeAccountByPubKey(_ cosmos.Context, _ common.PubKey) (NodeAccount, error) {
	return k.nas[0], nil
}

func (k *TestObservedTxOutHandleKeeper) SetNodeAccount(_ cosmos.Context, na NodeAccount) error {
	k.na = na
	return nil
}

func (k *TestObservedTxOutHandleKeeper) GetObservedTxOutVoter(_ cosmos.Context, _ common.TxID) (ObservedTxVoter, error) {
	return k.voter, nil
}

func (k *TestObservedTxOutHandleKeeper) SetObservedTxOutVoter(_ cosmos.Context, voter ObservedTxVoter) {
	k.voter = voter
}

func (k *TestObservedTxOutHandleKeeper) VaultExists(_ cosmos.Context, _ common.PubKey) bool {
	return k.yggExists
}

func (k *TestObservedTxOutHandleKeeper) GetVault(_ cosmos.Context, _ common.PubKey) (Vault, error) {
	return k.ygg, nil
}

func (k *TestObservedTxOutHandleKeeper) SetVault(_ cosmos.Context, ygg Vault) error {
	k.ygg = ygg
	return nil
}

func (k *TestObservedTxOutHandleKeeper) GetVaultData(_ cosmos.Context) (VaultData, error) {
	return NewVaultData(), nil
}

func (k *TestObservedTxOutHandleKeeper) SetVaultData(_ cosmos.Context, _ VaultData) error {
	return nil
}

func (k *TestObservedTxOutHandleKeeper) SetLastChainHeight(_ cosmos.Context, _ common.Chain, height int64) error {
	k.height = height
	return nil
}

func (k *TestObservedTxOutHandleKeeper) GetPool(_ cosmos.Context, _ common.Asset) (Pool, error) {
	return k.pool, nil
}

func (k *TestObservedTxOutHandleKeeper) GetTxOut(ctx cosmos.Context, _ int64) (*TxOut, error) {
	return k.txOutStore.GetBlockOut(ctx)
}

func (k *TestObservedTxOutHandleKeeper) FindPubKeyOfAddress(_ cosmos.Context, _ common.Address, _ common.Chain) (common.PubKey, error) {
	return k.ygg.PubKey, nil
}

func (k *TestObservedTxOutHandleKeeper) SetTxOut(_ cosmos.Context, _ *TxOut) error {
	return nil
}

func (k *TestObservedTxOutHandleKeeper) AddObservingAddresses(_ cosmos.Context, addrs []cosmos.AccAddress) error {
	k.observing = addrs
	return nil
}

func (k *TestObservedTxOutHandleKeeper) SetPool(ctx cosmos.Context, pool Pool) error {
	k.pool = pool
	return nil
}

func (k *TestObservedTxOutHandleKeeper) GetGas(ctx cosmos.Context, asset common.Asset) ([]cosmos.Uint, error) {
	return k.gas, nil
}

func (k *TestObservedTxOutHandleKeeper) SetGas(ctx cosmos.Context, asset common.Asset, units []cosmos.Uint) {
	k.gas = units
}

func (s *HandlerObservedTxOutSuite) TestHandle(c *C) {
	s.testHandleWithVersion(c, constants.SWVersion)
	s.testHandleWithVersion(c, semver.MustParse("0.13.0"))
}

func (s *HandlerObservedTxOutSuite) testHandleWithVersion(c *C, ver semver.Version) {
	var err error
	ctx, _ := setupKeeperForTest(c)

	constAccessor := constants.GetConstantValues(ver)
	tx := GetRandomTx()
	tx.Memo = fmt.Sprintf("OUTBOUND:%s", tx.ID)
	obTx := NewObservedTx(tx, 12, GetRandomPubKey())
	txs := ObservedTxs{obTx}
	pk := GetRandomPubKey()
	c.Assert(err, IsNil)

	ygg := NewVault(common.BlockHeight(ctx), ActiveVault, YggdrasilVault, pk, common.Chains{common.BNBChain})
	ygg.Coins = common.Coins{
		common.NewCoin(common.RuneAsset(), cosmos.NewUint(500)),
		common.NewCoin(common.BNBAsset, cosmos.NewUint(200*common.One)),
	}
	keeper := &TestObservedTxOutHandleKeeper{
		nas:   NodeAccounts{GetRandomNodeAccount(NodeActive)},
		voter: NewObservedTxVoter(tx.ID, make(ObservedTxs, 0)),
		pool: Pool{
			Asset:        common.BNBAsset,
			BalanceRune:  cosmos.NewUint(200),
			BalanceAsset: cosmos.NewUint(300),
		},
		yggExists: true,
		ygg:       ygg,
	}
	txOutStore := NewTxStoreDummy()
	keeper.txOutStore = txOutStore

	mgr := NewManagers(keeper)
	c.Assert(mgr.BeginBlock(ctx), IsNil)
	handler := NewObservedTxOutHandler(keeper, mgr)

	c.Assert(err, IsNil)
	msg := NewMsgObservedTxOut(txs, keeper.nas[0].NodeAddress)
	_, err = handler.handle(ctx, msg, ver, constAccessor)
	c.Assert(err, IsNil)
	c.Assert(err, IsNil)
	mgr.ObMgr().EndBlock(ctx, keeper)

	items, err := txOutStore.GetOutboundItems(ctx)
	c.Assert(err, IsNil)
	c.Assert(items, HasLen, 0)
	c.Check(keeper.observing, HasLen, 1)
	// make sure the coin has been subtract from the vault
	c.Check(ygg.Coins.GetCoin(common.BNBAsset).Amount.Equal(cosmos.NewUint(19999962499)), Equals, true, Commentf("%d", ygg.Coins.GetCoin(common.BNBAsset).Amount.Uint64()))
}

func (s *HandlerObservedTxOutSuite) TestGasUpdate(c *C) {
	s.testGasUpdateWithVersion(c, constants.SWVersion)
	s.testGasUpdateWithVersion(c, semver.MustParse("0.13.0"))
}

func (s *HandlerObservedTxOutSuite) testGasUpdateWithVersion(c *C, ver semver.Version) {
	var err error
	ctx, _ := setupKeeperForTest(c)

	constAccessor := constants.GetConstantValues(ver)
	tx := GetRandomTx()
	tx.Gas = common.Gas{
		{
			Asset:  common.BNBAsset,
			Amount: cosmos.NewUint(475000),
		},
	}
	tx.Memo = fmt.Sprintf("OUTBOUND:%s", tx.ID)
	obTx := NewObservedTx(tx, 12, GetRandomPubKey())
	txs := ObservedTxs{obTx}
	pk := GetRandomPubKey()
	c.Assert(err, IsNil)

	ygg := NewVault(common.BlockHeight(ctx), ActiveVault, YggdrasilVault, pk, common.Chains{common.BNBChain})
	ygg.Coins = common.Coins{
		common.NewCoin(common.RuneAsset(), cosmos.NewUint(500)),
		common.NewCoin(common.BNBAsset, cosmos.NewUint(200*common.One)),
	}
	keeper := &TestObservedTxOutHandleKeeper{
		nas:   NodeAccounts{GetRandomNodeAccount(NodeActive)},
		voter: NewObservedTxVoter(tx.ID, make(ObservedTxs, 0)),
		pool: Pool{
			Asset:        common.BNBAsset,
			BalanceRune:  cosmos.NewUint(200),
			BalanceAsset: cosmos.NewUint(300),
		},
		yggExists: true,
		ygg:       ygg,
	}
	txOutStore := NewTxStoreDummy()
	keeper.txOutStore = txOutStore

	handler := NewObservedTxOutHandler(keeper, NewDummyMgr())

	c.Assert(err, IsNil)
	msg := NewMsgObservedTxOut(txs, keeper.nas[0].NodeAddress)
	_, err = handler.handle(ctx, msg, ver, constAccessor)
	c.Assert(err, IsNil)
	gas := keeper.gas[0]
	c.Assert(gas.Equal(cosmos.NewUint(475000)), Equals, true, Commentf("%+v", gas))
	// revert the gas change , otherwise it messed up the other tests
	gasInfo := common.UpdateGasPrice(common.Tx{}, common.BNBAsset, []cosmos.Uint{cosmos.NewUint(37500), cosmos.NewUint(30000)})
	keeper.SetGas(ctx, common.BNBAsset, gasInfo)
}

func (s *HandlerObservedTxOutSuite) TestHandleStolenFunds(c *C) {
	s.testHandleStolenFundsWithVersion(c, constants.SWVersion)
	s.testHandleStolenFundsWithVersion(c, semver.MustParse("0.13.0"))
}

func (s *HandlerObservedTxOutSuite) testHandleStolenFundsWithVersion(c *C, ver semver.Version) {
	var err error
	ctx, _ := setupKeeperForTest(c)

	constAccessor := constants.GetConstantValues(ver)
	tx := GetRandomTx()
	tx.Memo = "I AM A THIEF!" // bad memo
	obTx := NewObservedTx(tx, 12, GetRandomPubKey())
	obTx.Tx.Coins = common.Coins{
		common.NewCoin(common.RuneAsset(), cosmos.NewUint(300*common.One)),
		common.NewCoin(common.BNBAsset, cosmos.NewUint(100*common.One)),
	}
	txs := ObservedTxs{obTx}
	pk := GetRandomPubKey()
	c.Assert(err, IsNil)

	na := GetRandomNodeAccount(NodeActive)
	na.Bond = cosmos.NewUint(1000000 * common.One)
	na.PubKeySet.Secp256k1 = pk

	ygg := NewVault(common.BlockHeight(ctx), ActiveVault, YggdrasilVault, pk, common.Chains{common.BNBChain})
	ygg.Coins = common.Coins{
		common.NewCoin(common.RuneAsset(), cosmos.NewUint(500*common.One)),
		common.NewCoin(common.BNBAsset, cosmos.NewUint(200*common.One)),
	}
	keeper := &TestObservedTxOutHandleKeeper{
		nas:   NodeAccounts{na},
		voter: NewObservedTxVoter(tx.ID, make(ObservedTxs, 0)),
		pool: Pool{
			Asset:        common.BNBAsset,
			BalanceRune:  cosmos.NewUint(200 * common.One),
			BalanceAsset: cosmos.NewUint(300 * common.One),
		},
		yggExists: true,
		ygg:       ygg,
	}
	txOutStore := NewTxStoreDummy()
	keeper.txOutStore = txOutStore

	mgr := NewDummyMgr()
	mgr.slasher = NewSlasherV1(keeper)
	handler := NewObservedTxOutHandler(keeper, mgr)

	c.Assert(err, IsNil)
	msg := NewMsgObservedTxOut(txs, keeper.nas[0].NodeAddress)
	_, err = handler.handle(ctx, msg, ver, constAccessor)
	c.Assert(err, IsNil)
	// make sure the coin has been subtract from the vault
	c.Check(ygg.Coins.GetCoin(common.BNBAsset).Amount.Equal(cosmos.NewUint(9999962500)), Equals, true, Commentf("%d", ygg.Coins.GetCoin(common.BNBAsset).Amount.Uint64()))
	c.Assert(keeper.na.Bond.LT(cosmos.NewUint(1000000*common.One)), Equals, true, Commentf("%d", keeper.na.Bond.Uint64()))
}

type HandlerObservedTxOutTestHelper struct {
	keeper.Keeper
	failListActiveNodeAccounts bool
	failVaultExist             bool
	failGetObservedTxOutVote   bool
	failGetVault               bool
	failSetVault               bool
}

func NewHandlerObservedTxOutHelper(k keeper.Keeper) *HandlerObservedTxOutTestHelper {
	return &HandlerObservedTxOutTestHelper{
		Keeper: k,
	}
}

func (h *HandlerObservedTxOutTestHelper) ListActiveNodeAccounts(ctx cosmos.Context) (NodeAccounts, error) {
	if h.failListActiveNodeAccounts {
		return NodeAccounts{}, kaboom
	}
	return h.Keeper.ListActiveNodeAccounts(ctx)
}

func (h *HandlerObservedTxOutTestHelper) VaultExists(ctx cosmos.Context, pk common.PubKey) bool {
	if h.failVaultExist {
		return false
	}
	return h.Keeper.VaultExists(ctx, pk)
}

func (h *HandlerObservedTxOutTestHelper) GetObservedTxOutVoter(ctx cosmos.Context, hash common.TxID) (ObservedTxVoter, error) {
	if h.failGetObservedTxOutVote {
		return ObservedTxVoter{}, kaboom
	}
	return h.Keeper.GetObservedTxOutVoter(ctx, hash)
}

func (h *HandlerObservedTxOutTestHelper) GetVault(ctx cosmos.Context, pk common.PubKey) (Vault, error) {
	if h.failGetVault {
		return Vault{}, kaboom
	}
	return h.Keeper.GetVault(ctx, pk)
}

func (h *HandlerObservedTxOutTestHelper) SetVault(ctx cosmos.Context, vault Vault) error {
	if h.failSetVault {
		return kaboom
	}
	return h.Keeper.SetVault(ctx, vault)
}

func setupAnObservedTxOut(ctx cosmos.Context, helper *HandlerObservedTxOutTestHelper, c *C) MsgObservedTxOut {
	activeNodeAccount := GetRandomNodeAccount(NodeActive)
	pk := GetRandomPubKey()
	tx := GetRandomTx()
	tx.Coins = common.Coins{
		common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One*3)),
	}
	tx.Memo = "OUTBOUND:" + GetRandomTxHash().String()
	addr, err := pk.GetAddress(tx.Coins[0].Asset.Chain)
	c.Assert(err, IsNil)
	tx.ToAddress = GetRandomBNBAddress()
	tx.FromAddress = addr
	obTx := NewObservedTx(tx, ctx.BlockHeight(), pk)
	txs := ObservedTxs{obTx}
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
	return NewMsgObservedTxOut(txs, activeNodeAccount.NodeAddress)
}

func (HandlerObservedTxOutSuite) TestHandlerObservedTxOut_DifferentValidations(c *C) {
	testCases := []struct {
		name            string
		messageProvider func(c *C, ctx cosmos.Context, helper *HandlerObservedTxOutTestHelper) cosmos.Msg
		validator       func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *HandlerObservedTxOutTestHelper, name string)
	}{
		{
			name: "invalid message should return an error",
			messageProvider: func(c *C, ctx cosmos.Context, helper *HandlerObservedTxOutTestHelper) cosmos.Msg {
				return NewMsgNetworkFee(ctx.BlockHeight(), common.BNBChain, 1, bnbSingleTxFee, GetRandomBech32Addr())
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *HandlerObservedTxOutTestHelper, name string) {
				c.Check(err, NotNil, Commentf(name))
				c.Check(result, IsNil)
				c.Check(errors.Is(err, errInvalidMessage), Equals, true)
			},
		},
		{
			name: "message fail validation should return an error",
			messageProvider: func(c *C, ctx cosmos.Context, helper *HandlerObservedTxOutTestHelper) cosmos.Msg {
				return NewMsgObservedTxOut(ObservedTxs{
					NewObservedTx(GetRandomTx(), ctx.BlockHeight(), GetRandomPubKey()),
				}, GetRandomBech32Addr())
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *HandlerObservedTxOutTestHelper, name string) {
				c.Check(err, NotNil, Commentf(name))
				c.Check(result, IsNil)
			},
		},
		{
			name: "voter already vote for the tx should return without doing anything",
			messageProvider: func(c *C, ctx cosmos.Context, helper *HandlerObservedTxOutTestHelper) cosmos.Msg {
				m := setupAnObservedTxOut(ctx, helper, c)
				voter, err := helper.Keeper.GetObservedTxOutVoter(ctx, m.Txs[0].Tx.ID)
				c.Assert(err, IsNil)
				voter.Add(m.Txs[0], m.Signer)
				helper.Keeper.SetObservedTxOutVoter(ctx, voter)
				return m
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *HandlerObservedTxOutTestHelper, name string) {
				c.Check(err, IsNil, Commentf(name))
				c.Check(result, NotNil, Commentf(name))
			},
		},
		{
			name: "fail to list active node accounts should result in an error",
			messageProvider: func(c *C, ctx cosmos.Context, helper *HandlerObservedTxOutTestHelper) cosmos.Msg {
				m := setupAnObservedTxOut(ctx, helper, c)
				helper.failListActiveNodeAccounts = true
				return m
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *HandlerObservedTxOutTestHelper, name string) {
				c.Check(err, NotNil, Commentf(name))
				c.Check(result, IsNil, Commentf(name))
			},
		},
		{
			name: "vault not exist should not result in an error, it should continue",
			messageProvider: func(c *C, ctx cosmos.Context, helper *HandlerObservedTxOutTestHelper) cosmos.Msg {
				m := setupAnObservedTxOut(ctx, helper, c)
				helper.failVaultExist = true
				return m
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *HandlerObservedTxOutTestHelper, name string) {
				c.Check(err, IsNil, Commentf(name))
				c.Check(result, NotNil, Commentf(name))
			},
		},
		{
			name: "fail to get observedTxOutVoter should not result in an error, it should continue",
			messageProvider: func(c *C, ctx cosmos.Context, helper *HandlerObservedTxOutTestHelper) cosmos.Msg {
				m := setupAnObservedTxOut(ctx, helper, c)
				helper.failGetObservedTxOutVote = true
				return m
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *HandlerObservedTxOutTestHelper, name string) {
				c.Check(err, IsNil, Commentf(name))
				c.Check(result, NotNil, Commentf(name))
			},
		},
		{
			name: "empty memo should not result in an error, it should continue",
			messageProvider: func(c *C, ctx cosmos.Context, helper *HandlerObservedTxOutTestHelper) cosmos.Msg {
				m := setupAnObservedTxOut(ctx, helper, c)
				m.Txs[0].Tx.Memo = ""
				return m
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *HandlerObservedTxOutTestHelper, name string) {
				c.Check(err, IsNil, Commentf(name))
				c.Check(result, NotNil, Commentf(name))
				txOut, err := helper.GetTxOut(ctx, ctx.BlockHeight())
				c.Assert(err, IsNil, Commentf(name))
				c.Assert(txOut.IsEmpty(), Equals, true)
			},
		},
		{
			name: "fail to get vault, it should continue",
			messageProvider: func(c *C, ctx cosmos.Context, helper *HandlerObservedTxOutTestHelper) cosmos.Msg {
				m := setupAnObservedTxOut(ctx, helper, c)
				helper.failGetVault = true
				return m
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *HandlerObservedTxOutTestHelper, name string) {
				c.Check(err, IsNil, Commentf(name))
				c.Check(result, NotNil, Commentf(name))
			},
		},
		{
			name: "fail to set vault, it should continue",
			messageProvider: func(c *C, ctx cosmos.Context, helper *HandlerObservedTxOutTestHelper) cosmos.Msg {
				m := setupAnObservedTxOut(ctx, helper, c)
				helper.failSetVault = true
				return m
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *HandlerObservedTxOutTestHelper, name string) {
				c.Check(err, IsNil, Commentf(name))
				c.Check(result, NotNil, Commentf(name))
			},
		},
		{
			name: "ragnarok memo it should success",
			messageProvider: func(c *C, ctx cosmos.Context, helper *HandlerObservedTxOutTestHelper) cosmos.Msg {
				m := setupAnObservedTxOut(ctx, helper, c)
				m.Txs[0].Tx.Memo = "ragnarok:100"
				return m
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *HandlerObservedTxOutTestHelper, name string) {
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
			helper := NewHandlerObservedTxOutHelper(k)
			mgr := NewManagers(helper)
			mgr.BeginBlock(ctx)
			handler := NewObservedTxOutHandler(helper, mgr)
			msg := tc.messageProvider(c, ctx, helper)
			constantAccessor := constants.GetConstantValues(ver)
			result, err := handler.Run(ctx, msg, ver, constantAccessor)
			tc.validator(c, ctx, result, err, helper, tc.name)
		}
	}
}
