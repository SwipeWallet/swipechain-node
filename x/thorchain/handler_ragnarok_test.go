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

type HandlerRagnarokSuite struct{}

var _ = Suite(&HandlerRagnarokSuite{})

type TestRagnarokKeeper struct {
	keeper.KVStoreDummy
	activeNodeAccount NodeAccount
	vault             Vault
}

func (k *TestRagnarokKeeper) GetNodeAccount(_ cosmos.Context, addr cosmos.AccAddress) (NodeAccount, error) {
	if k.activeNodeAccount.NodeAddress.Equals(addr) {
		return k.activeNodeAccount, nil
	}
	return NodeAccount{}, nil
}

func (HandlerRagnarokSuite) TestRagnarok(c *C) {
	ctx, _ := setupKeeperForTest(c)

	keeper := &TestRagnarokKeeper{
		activeNodeAccount: GetRandomNodeAccount(NodeActive),
		vault:             GetRandomVault(),
	}

	handler := NewRagnarokHandler(keeper, NewDummyMgr())

	// invalid message should result errors
	msg := NewMsgNetworkFee(ctx.BlockHeight(), common.BNBChain, 1, bnbSingleTxFee, GetRandomBech32Addr())
	result, err := handler.Run(ctx, msg, constants.SWVersion, constants.GetConstantValues(constants.SWVersion))
	c.Check(result, IsNil, Commentf("invalid message should result an error"))
	c.Check(err, NotNil, Commentf("invalid message should result an error"))
	addr, err := keeper.vault.PubKey.GetAddress(common.BNBChain)
	c.Assert(err, IsNil)

	ver := constants.SWVersion
	constAccessor := constants.GetConstantValues(ver)

	tx := NewObservedTx(common.Tx{
		ID:          GetRandomTxHash(),
		Chain:       common.BNBChain,
		Coins:       common.Coins{common.NewCoin(common.BNBAsset, cosmos.NewUint(1*common.One))},
		Memo:        "",
		FromAddress: GetRandomBNBAddress(),
		ToAddress:   addr,
		Gas:         BNBGasFeeSingleton,
	}, 12, GetRandomPubKey())

	msgRagnarok := NewMsgRagnarok(tx, 1, keeper.activeNodeAccount.NodeAddress)
	err = handler.validate(ctx, msgRagnarok, ver)
	c.Assert(err, IsNil)

	// invalid version
	err = handler.validate(ctx, msgRagnarok, semver.Version{})
	c.Assert(err, Equals, errInvalidVersion)
	result, err = handler.Run(ctx, msgRagnarok, semver.Version{}, constants.GetConstantValues(constants.SWVersion))
	c.Check(result, IsNil, Commentf("invalid version should result an error"))
	c.Check(err, NotNil, Commentf("invalid version should result an error"))
	c.Check(errors.Is(err, errInvalidVersion), Equals, true)
	result, err = handler.handle(ctx, semver.Version{}, msgRagnarok, constAccessor)
	c.Check(result, IsNil, Commentf("invalid version should result an error"))
	c.Check(err, NotNil, Commentf("invalid version should result an error"))

	// invalid msg
	msgRagnarok = MsgRagnarok{}
	err = handler.validate(ctx, msgRagnarok, ver)
	c.Assert(err, NotNil)
	result, err = handler.Run(ctx, msgRagnarok, constants.SWVersion, constants.GetConstantValues(constants.SWVersion))
	c.Check(err, NotNil, Commentf("invalid message should fail validation"))
	c.Check(result, IsNil, Commentf("invalid message should fail validation"))
}

type TestRagnarokKeeperHappyPath struct {
	keeper.KVStoreDummy
	activeNodeAccount NodeAccount
	newVault          Vault
	retireVault       Vault
	txout             *TxOut
	pool              Pool
}

func (k *TestRagnarokKeeperHappyPath) GetTxOut(ctx cosmos.Context, blockHeight int64) (*TxOut, error) {
	if k.txout != nil && k.txout.Height == blockHeight {
		return k.txout, nil
	}
	return nil, kaboom
}

func (k *TestRagnarokKeeperHappyPath) SetTxOut(ctx cosmos.Context, blockOut *TxOut) error {
	if k.txout.Height == blockOut.Height {
		k.txout = blockOut
		return nil
	}
	return kaboom
}

func (k *TestRagnarokKeeperHappyPath) GetNodeAccountByPubKey(_ cosmos.Context, _ common.PubKey) (NodeAccount, error) {
	return k.activeNodeAccount, nil
}

func (k *TestRagnarokKeeperHappyPath) SetNodeAccount(_ cosmos.Context, na NodeAccount) error {
	k.activeNodeAccount = na
	return nil
}

func (k *TestRagnarokKeeperHappyPath) GetPool(_ cosmos.Context, _ common.Asset) (Pool, error) {
	return k.pool, nil
}

func (k *TestRagnarokKeeperHappyPath) SetPool(_ cosmos.Context, p Pool) error {
	k.pool = p
	return nil
}

func (HandlerRagnarokSuite) TestRagnarokHappyPath(c *C) {
	ctx, _ := setupKeeperForTest(c)
	retireVault := GetRandomVault()

	newVault := GetRandomVault()
	txout := NewTxOut(1)
	newVaultAddr, err := newVault.PubKey.GetAddress(common.BNBChain)
	c.Assert(err, IsNil)
	txout.TxArray = append(txout.TxArray, &TxOutItem{
		Chain:       common.BNBChain,
		InHash:      common.BlankTxID,
		ToAddress:   newVaultAddr,
		VaultPubKey: retireVault.PubKey,
		Coin:        common.NewCoin(common.BNBAsset, cosmos.NewUint(1024)),
		Memo:        NewRagnarokMemo(1).String(),
	})
	keeper := &TestRagnarokKeeperHappyPath{
		activeNodeAccount: GetRandomNodeAccount(NodeActive),
		newVault:          newVault,
		retireVault:       retireVault,
		txout:             txout,
	}
	addr, err := keeper.retireVault.PubKey.GetAddress(common.BNBChain)
	c.Assert(err, IsNil)
	handler := NewRagnarokHandler(keeper, NewDummyMgr())
	tx := NewObservedTx(common.Tx{
		ID:    GetRandomTxHash(),
		Chain: common.BNBChain,
		Coins: common.Coins{
			common.NewCoin(common.BNBAsset, cosmos.NewUint(1024)),
		},
		Memo:        NewRagnarokMemo(1).String(),
		FromAddress: addr,
		ToAddress:   newVaultAddr,
		Gas:         BNBGasFeeSingleton,
	}, 1, retireVault.PubKey)

	msgRagnarok := NewMsgRagnarok(tx, 1, keeper.activeNodeAccount.NodeAddress)
	ver := constants.SWVersion
	constAccessor := constants.GetConstantValues(ver)
	_, err = handler.handleV1(ctx, ver, msgRagnarok, constAccessor)
	c.Assert(err, IsNil)
	c.Assert(keeper.txout.TxArray[0].OutHash.Equals(tx.Tx.ID), Equals, true)

	// fail to get tx out
	msgRagnarok1 := NewMsgRagnarok(tx, 1024, keeper.activeNodeAccount.NodeAddress)
	result, err := handler.handleV1(ctx, ver, msgRagnarok1, constAccessor)
	c.Assert(err, NotNil)
	c.Assert(result, IsNil)
}

func (HandlerRagnarokSuite) TestSlash(c *C) {
	ctx, _ := setupKeeperForTest(c)
	retireVault := GetRandomVault()
	ver := constants.SWVersion
	constAccessor := constants.GetConstantValues(ver)

	newVault := GetRandomVault()
	txout := NewTxOut(1)
	newVaultAddr, err := newVault.PubKey.GetAddress(common.BNBChain)
	c.Assert(err, IsNil)

	pool := NewPool()
	pool.Asset = common.BNBAsset
	pool.BalanceAsset = cosmos.NewUint(100 * common.One)
	pool.BalanceRune = cosmos.NewUint(100 * common.One)
	na := GetRandomNodeAccount(NodeActive)
	na.Bond = cosmos.NewUint(100 * common.One)
	keeper := &TestRagnarokKeeperHappyPath{
		activeNodeAccount: na,
		newVault:          newVault,
		retireVault:       retireVault,
		txout:             txout,
		pool:              pool,
	}
	addr, err := keeper.retireVault.PubKey.GetAddress(common.BNBChain)
	c.Assert(err, IsNil)

	mgr := NewDummyMgr()
	mgr.slasher = NewSlasherV1(keeper)
	handler := NewRagnarokHandler(keeper, mgr)

	tx := NewObservedTx(common.Tx{
		ID:    GetRandomTxHash(),
		Chain: common.BNBChain,
		Coins: common.Coins{
			common.NewCoin(common.BNBAsset, cosmos.NewUint(1024)),
		},
		Memo:        NewRagnarokMemo(1).String(),
		FromAddress: addr,
		ToAddress:   newVaultAddr,
		Gas:         BNBGasFeeSingleton,
	}, 1, retireVault.PubKey)

	msgRagnarok := NewMsgRagnarok(tx, 1, keeper.activeNodeAccount.NodeAddress)
	_, err = handler.handleV1(ctx, constants.SWVersion, msgRagnarok, constAccessor)
	c.Assert(err, IsNil)
	c.Assert(keeper.activeNodeAccount.Bond.Equal(cosmos.NewUint(9999998464)), Equals, true, Commentf("%d", keeper.activeNodeAccount.Bond.Uint64()))
}
