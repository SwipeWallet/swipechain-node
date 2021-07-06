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

type HandlerMigrateSuite struct{}

var _ = Suite(&HandlerMigrateSuite{})

type TestMigrateKeeper struct {
	keeper.KVStoreDummy
	activeNodeAccount NodeAccount
	vault             Vault
}

// GetNodeAccount
func (k *TestMigrateKeeper) GetNodeAccount(_ cosmos.Context, addr cosmos.AccAddress) (NodeAccount, error) {
	if k.activeNodeAccount.NodeAddress.Equals(addr) {
		return k.activeNodeAccount, nil
	}
	return NodeAccount{}, nil
}

func (HandlerMigrateSuite) TestMigrate(c *C) {
	ctx, _ := setupKeeperForTest(c)

	keeper := &TestMigrateKeeper{
		activeNodeAccount: GetRandomNodeAccount(NodeActive),
		vault:             GetRandomVault(),
	}

	handler := NewMigrateHandler(keeper, NewDummyMgr())

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

	msgMigrate := NewMsgMigrate(tx, 1, keeper.activeNodeAccount.NodeAddress)
	err = handler.validate(ctx, msgMigrate, ver)
	c.Assert(err, IsNil)

	// invalid version
	err = handler.validate(ctx, msgMigrate, semver.Version{})
	c.Assert(err, Equals, errInvalidVersion)

	// invalid msg
	msgMigrate = MsgMigrate{}
	err = handler.validate(ctx, msgMigrate, ver)
	c.Assert(err, NotNil)
}

type TestMigrateKeeperHappyPath struct {
	keeper.KVStoreDummy
	activeNodeAccount NodeAccount
	newVault          Vault
	retireVault       Vault
	txout             *TxOut
	pool              Pool
}

func (k *TestMigrateKeeperHappyPath) GetTxOut(ctx cosmos.Context, blockHeight int64) (*TxOut, error) {
	if k.txout != nil && k.txout.Height == blockHeight {
		return k.txout, nil
	}
	return nil, kaboom
}

func (k *TestMigrateKeeperHappyPath) SetTxOut(ctx cosmos.Context, blockOut *TxOut) error {
	if k.txout.Height == blockOut.Height {
		k.txout = blockOut
		return nil
	}
	return kaboom
}

func (k *TestMigrateKeeperHappyPath) GetNodeAccountByPubKey(_ cosmos.Context, _ common.PubKey) (NodeAccount, error) {
	return k.activeNodeAccount, nil
}

func (k *TestMigrateKeeperHappyPath) SetNodeAccount(_ cosmos.Context, na NodeAccount) error {
	k.activeNodeAccount = na
	return nil
}

func (k *TestMigrateKeeperHappyPath) GetPool(_ cosmos.Context, _ common.Asset) (Pool, error) {
	return k.pool, nil
}

func (k *TestMigrateKeeperHappyPath) SetPool(_ cosmos.Context, p Pool) error {
	k.pool = p
	return nil
}

func (HandlerMigrateSuite) TestMigrateHappyPath(c *C) {
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
		Memo:        NewMigrateMemo(1).String(),
	})
	keeper := &TestMigrateKeeperHappyPath{
		activeNodeAccount: GetRandomNodeAccount(NodeActive),
		newVault:          newVault,
		retireVault:       retireVault,
		txout:             txout,
	}
	addr, err := keeper.retireVault.PubKey.GetAddress(common.BNBChain)
	c.Assert(err, IsNil)
	handler := NewMigrateHandler(keeper, NewDummyMgr())
	tx := NewObservedTx(common.Tx{
		ID:    GetRandomTxHash(),
		Chain: common.BNBChain,
		Coins: common.Coins{
			common.NewCoin(common.BNBAsset, cosmos.NewUint(1024)),
		},
		Memo:        NewMigrateMemo(1).String(),
		FromAddress: addr,
		ToAddress:   newVaultAddr,
		Gas:         BNBGasFeeSingleton,
	}, 1, retireVault.PubKey)

	msgMigrate := NewMsgMigrate(tx, 1, keeper.activeNodeAccount.NodeAddress)
	constantAccessor := constants.GetConstantValues(constants.SWVersion)
	_, err = handler.Run(ctx, msgMigrate, constants.SWVersion, constantAccessor)
	c.Assert(err, IsNil)
	c.Assert(keeper.txout.TxArray[0].OutHash.Equals(tx.Tx.ID), Equals, true)
}

func (HandlerMigrateSuite) TestSlash(c *C) {
	ctx, _ := setupKeeperForTest(c)
	retireVault := GetRandomVault()

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
	keeper := &TestMigrateKeeperHappyPath{
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
	handler := NewMigrateHandler(keeper, mgr)
	tx := NewObservedTx(common.Tx{
		ID:    GetRandomTxHash(),
		Chain: common.BNBChain,
		Coins: common.Coins{
			common.NewCoin(common.BNBAsset, cosmos.NewUint(1024)),
		},
		Memo:        NewMigrateMemo(1).String(),
		FromAddress: addr,
		ToAddress:   newVaultAddr,
		Gas:         BNBGasFeeSingleton,
	}, 1, retireVault.PubKey)

	msgMigrate := NewMsgMigrate(tx, 1, keeper.activeNodeAccount.NodeAddress)
	_, err = handler.handleV1(ctx, constants.SWVersion, msgMigrate)
	c.Assert(err, IsNil)
	c.Assert(keeper.activeNodeAccount.Bond.Equal(cosmos.NewUint(9999998464)), Equals, true, Commentf("%d", keeper.activeNodeAccount.Bond.Uint64()))
}

func (HandlerMigrateSuite) TestHandlerMigrateValidation(c *C) {
	// invalid message should return an error
	ctx, k := setupKeeperForTest(c)
	mgr := NewManagers(k)
	mgr.BeginBlock(ctx)
	h := NewMigrateHandler(k, mgr)
	constantAccessor := constants.GetConstantValues(constants.SWVersion)
	result, err := h.Run(ctx, NewMsgNetworkFee(ctx.BlockHeight(), common.BNBChain, 1, bnbSingleTxFee, GetRandomBech32Addr()), semver.MustParse("0.1.0"), constantAccessor)
	c.Check(err, NotNil)
	c.Check(result, IsNil)
	c.Check(errors.Is(err, errInvalidMessage), Equals, true)
}
