package thorchain

import (
	"errors"

	"github.com/blang/semver"
	se "github.com/cosmos/cosmos-sdk/types/errors"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
	"gitlab.com/thorchain/thornode/x/thorchain/keeper"

	. "gopkg.in/check.v1"
)

type HandlerYggdrasilSuite struct{}

var _ = Suite(&HandlerYggdrasilSuite{})

type yggdrasilTestKeeper struct {
	keeper.Keeper
	errGetVault        bool
	errGetAsgardVaults bool
	errGetNodeAccount  cosmos.AccAddress
	errGetPool         bool
	errGetTxOut        bool
}

func (k *yggdrasilTestKeeper) GetAsgardVaultsByStatus(ctx cosmos.Context, vs VaultStatus) (Vaults, error) {
	if k.errGetAsgardVaults {
		return Vaults{}, kaboom
	}
	return k.Keeper.GetAsgardVaultsByStatus(ctx, vs)
}

func (k *yggdrasilTestKeeper) GetTxOut(ctx cosmos.Context, height int64) (*TxOut, error) {
	if k.errGetTxOut {
		return nil, kaboom
	}
	return k.Keeper.GetTxOut(ctx, height)
}

func (k *yggdrasilTestKeeper) GetNodeAccountByPubKey(ctx cosmos.Context, pk common.PubKey) (NodeAccount, error) {
	addr, _ := pk.GetThorAddress()
	if k.errGetNodeAccount.Equals(addr) {
		return NodeAccount{}, kaboom
	}
	return k.Keeper.GetNodeAccountByPubKey(ctx, pk)
}

func (k *yggdrasilTestKeeper) SetNodeAccount(ctx cosmos.Context, na NodeAccount) error {
	return k.Keeper.SetNodeAccount(ctx, na)
}

func (k *yggdrasilTestKeeper) GetPool(ctx cosmos.Context, asset common.Asset) (Pool, error) {
	if k.errGetPool {
		return Pool{}, kaboom
	}
	return k.Keeper.GetPool(ctx, asset)
}

func (k *yggdrasilTestKeeper) SetPool(ctx cosmos.Context, p Pool) error {
	return k.Keeper.SetPool(ctx, p)
}

func (k *yggdrasilTestKeeper) GetVault(ctx cosmos.Context, pk common.PubKey) (Vault, error) {
	if k.errGetVault {
		return Vault{}, kaboom
	}
	return k.Keeper.GetVault(ctx, pk)
}

func (k *yggdrasilTestKeeper) GetAsgardVaults(ctx cosmos.Context) (Vaults, error) {
	if k.errGetAsgardVaults {
		return Vaults{}, kaboom
	}
	return k.Keeper.GetAsgardVaults(ctx)
}

type yggdrasilHandlerTestHelper struct {
	ctx           cosmos.Context
	pool          Pool
	version       semver.Version
	keeper        *yggdrasilTestKeeper
	asgardVault   Vault
	yggVault      Vault
	constAccessor constants.ConstantValues
	nodeAccount   NodeAccount
	mgr           Manager
}

func newYggdrasilTestKeeper(keeper keeper.Keeper) *yggdrasilTestKeeper {
	return &yggdrasilTestKeeper{
		Keeper: keeper,
	}
}

func newYggdrasilHandlerTestHelper(c *C, version semver.Version) yggdrasilHandlerTestHelper {
	ctx, k := setupKeeperForTest(c)
	ctx = ctx.WithBlockHeight(1023)

	keeper := newYggdrasilTestKeeper(k)

	// test pool
	pool := NewPool()
	pool.Asset = common.BNBAsset
	pool.BalanceAsset = cosmos.NewUint(100 * common.One)
	pool.BalanceRune = cosmos.NewUint(100 * common.One)
	c.Assert(keeper.SetPool(ctx, pool), IsNil)

	// active account
	nodeAccount := GetRandomNodeAccount(NodeActive)
	nodeAccount.Bond = cosmos.NewUint(100 * common.One)
	c.Assert(keeper.SetNodeAccount(ctx, nodeAccount), IsNil)

	constAccessor := constants.GetConstantValues(version)

	mgr := NewDummyMgr()
	mgr.validatorMgr = newValidatorMgrV1(k, mgr.VaultMgr(), mgr.TxOutStore(), mgr.EventMgr())
	mgr.slasher = NewSlasherV1(keeper)
	c.Assert(mgr.ValidatorMgr().BeginBlock(ctx, constAccessor), IsNil)
	asgardVault := GetRandomVault()
	asgardVault.Type = AsgardVault
	asgardVault.Status = ActiveVault
	c.Assert(keeper.SetVault(ctx, asgardVault), IsNil)
	yggdrasilVault := GetRandomVault()
	yggdrasilVault.PubKey = nodeAccount.PubKeySet.Secp256k1
	yggdrasilVault.Type = YggdrasilVault
	yggdrasilVault.Status = ActiveVault
	c.Assert(keeper.SetVault(ctx, yggdrasilVault), IsNil)

	return yggdrasilHandlerTestHelper{
		ctx:           ctx,
		version:       version,
		keeper:        keeper,
		nodeAccount:   nodeAccount,
		constAccessor: constAccessor,
		mgr:           mgr,
		asgardVault:   asgardVault,
		yggVault:      yggdrasilVault,
	}
}

func (s *HandlerYggdrasilSuite) TestYggdrasilHandler(c *C) {
	versions := []semver.Version{
		constants.SWVersion,
		semver.MustParse("0.8.0"),
		semver.MustParse("0.17.0"),
	}
	for _, v := range versions {
		s.testYggdrasilHandlerWithVersion(c, v)
	}
}

func (s *HandlerYggdrasilSuite) testYggdrasilHandlerWithVersion(c *C, version semver.Version) {
	testCases := []struct {
		name           string
		messageCreator func(helper yggdrasilHandlerTestHelper) cosmos.Msg
		runner         func(handler YggdrasilHandler, msg cosmos.Msg, helper yggdrasilHandlerTestHelper) (*cosmos.Result, error)
		validator      func(helper yggdrasilHandlerTestHelper, msg cosmos.Msg, result *cosmos.Result, c *C)
		expectedResult error
	}{
		{
			name: "invalid message should return error",
			messageCreator: func(helper yggdrasilHandlerTestHelper) cosmos.Msg {
				return NewMsgNoOp(GetRandomObservedTx(), helper.nodeAccount.NodeAddress)
			},
			runner: func(handler YggdrasilHandler, msg cosmos.Msg, helper yggdrasilHandlerTestHelper) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, helper.version, helper.constAccessor)
			},
			expectedResult: errInvalidMessage,
		},
		{
			name: "bad version should return an error",
			messageCreator: func(helper yggdrasilHandlerTestHelper) cosmos.Msg {
				return NewMsgYggdrasil(GetRandomTx(), GetRandomPubKey(), 12, false, common.Coins{common.NewCoin(common.BNBAsset, cosmos.OneUint())}, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler YggdrasilHandler, msg cosmos.Msg, helper yggdrasilHandlerTestHelper) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, semver.MustParse("0.0.1"), helper.constAccessor)
			},
			expectedResult: errBadVersion,
		},
		{
			name: "empty pubkey should return an error",
			messageCreator: func(helper yggdrasilHandlerTestHelper) cosmos.Msg {
				return NewMsgYggdrasil(GetRandomTx(), "", 12, false, common.Coins{common.NewCoin(common.BNBAsset, cosmos.OneUint())}, GetRandomBech32Addr())
			},
			runner: func(handler YggdrasilHandler, msg cosmos.Msg, helper yggdrasilHandlerTestHelper) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, helper.version, helper.constAccessor)
			},
			expectedResult: se.ErrUnknownRequest,
		},
		{
			name: "empty tx should return an error",
			messageCreator: func(helper yggdrasilHandlerTestHelper) cosmos.Msg {
				return NewMsgYggdrasil(common.Tx{}, GetRandomPubKey(), 12, false, common.Coins{common.NewCoin(common.BNBAsset, cosmos.OneUint())}, GetRandomBech32Addr())
			},
			runner: func(handler YggdrasilHandler, msg cosmos.Msg, helper yggdrasilHandlerTestHelper) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, helper.version, helper.constAccessor)
			},
			expectedResult: se.ErrUnknownRequest,
		},
		{
			name: "invalid coin should return an error",
			messageCreator: func(helper yggdrasilHandlerTestHelper) cosmos.Msg {
				return NewMsgYggdrasil(GetRandomTx(), GetRandomPubKey(), 12, false, common.Coins{common.NewCoin(common.EmptyAsset, cosmos.OneUint())}, GetRandomBech32Addr())
			},
			runner: func(handler YggdrasilHandler, msg cosmos.Msg, helper yggdrasilHandlerTestHelper) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, helper.version, helper.constAccessor)
			},
			expectedResult: se.ErrInvalidCoins,
		},
		{
			name: "empty signer should return an error",
			messageCreator: func(helper yggdrasilHandlerTestHelper) cosmos.Msg {
				return NewMsgYggdrasil(GetRandomTx(), GetRandomPubKey(), 12, false, common.Coins{common.NewCoin(common.BNBAsset, cosmos.OneUint())}, cosmos.AccAddress{})
			},
			runner: func(handler YggdrasilHandler, msg cosmos.Msg, helper yggdrasilHandlerTestHelper) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, helper.version, helper.constAccessor)
			},
			expectedResult: se.ErrInvalidAddress,
		},
		{
			name: "fail to get yggdrasil vault should return an error",
			messageCreator: func(helper yggdrasilHandlerTestHelper) cosmos.Msg {
				return NewMsgYggdrasil(GetRandomTx(), GetRandomPubKey(), 12, false, common.Coins{common.NewCoin(common.BNBAsset, cosmos.OneUint())}, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler YggdrasilHandler, msg cosmos.Msg, helper yggdrasilHandlerTestHelper) (*cosmos.Result, error) {
				helper.keeper.errGetVault = true
				return handler.Run(helper.ctx, msg, helper.version, helper.constAccessor)
			},
			expectedResult: kaboom,
		},
		{
			name: "asgard fund yggdrasil should return success",
			messageCreator: func(helper yggdrasilHandlerTestHelper) cosmos.Msg {
				return NewMsgYggdrasil(GetRandomTx(), helper.asgardVault.PubKey, 13, true, common.Coins{common.NewCoin(common.BNBAsset, cosmos.OneUint())}, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler YggdrasilHandler, msg cosmos.Msg, helper yggdrasilHandlerTestHelper) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, helper.version, helper.constAccessor)
			},
			expectedResult: nil,
		},
		{
			name: "yggdrasil received fund from asgard should return success",
			messageCreator: func(helper yggdrasilHandlerTestHelper) cosmos.Msg {
				return NewMsgYggdrasil(GetRandomTx(), helper.yggVault.PubKey, 13, true, common.Coins{common.NewCoin(common.BNBAsset, cosmos.OneUint())}, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler YggdrasilHandler, msg cosmos.Msg, helper yggdrasilHandlerTestHelper) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, helper.version, helper.constAccessor)
			},
			expectedResult: nil,
		},
		{
			name: "yggdrasil return fund to asgard but to address is not asgard should be slashed",
			messageCreator: func(helper yggdrasilHandlerTestHelper) cosmos.Msg {
				fromAddr, _ := helper.yggVault.PubKey.GetAddress(common.BNBChain)
				coins := common.Coins{
					common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One)),
					common.NewCoin(common.RuneAsset(), cosmos.NewUint(common.One)),
				}
				tx := common.Tx{
					ID:          GetRandomTxHash(),
					Chain:       common.BNBChain,
					FromAddress: fromAddr,
					ToAddress:   GetRandomBNBAddress(),
					Coins:       coins,
					Gas:         BNBGasFeeSingleton,
					Memo:        "yggdrasil-:30",
				}
				return NewMsgYggdrasil(tx, helper.yggVault.PubKey, 12, false, coins, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler YggdrasilHandler, msg cosmos.Msg, helper yggdrasilHandlerTestHelper) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, constants.SWVersion, helper.constAccessor)
			},
			expectedResult: nil,
			validator: func(helper yggdrasilHandlerTestHelper, msg cosmos.Msg, result *cosmos.Result, c *C) {
				expectedBond := helper.nodeAccount.Bond.Sub(cosmos.NewUint(603787879))
				na, err := helper.keeper.GetNodeAccount(helper.ctx, helper.nodeAccount.NodeAddress)
				c.Assert(err, IsNil)
				c.Assert(na.Bond.Equal(expectedBond), Equals, true, Commentf("%d/%d", na.Bond.Uint64(), expectedBond.Uint64()))
			},
		},
		{
			name: "fail to get asgard vaults should return an error",
			messageCreator: func(helper yggdrasilHandlerTestHelper) cosmos.Msg {
				return NewMsgYggdrasil(GetRandomTx(), helper.yggVault.PubKey, 12, false, common.Coins{common.NewCoin(common.BNBAsset, cosmos.OneUint())}, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler YggdrasilHandler, msg cosmos.Msg, helper yggdrasilHandlerTestHelper) (*cosmos.Result, error) {
				helper.keeper.errGetAsgardVaults = true
				return handler.Run(helper.ctx, msg, helper.version, helper.constAccessor)
			},
			expectedResult: errInternal,
		},
		{
			name: "fail to get node accounts should return an error",
			messageCreator: func(helper yggdrasilHandlerTestHelper) cosmos.Msg {
				fromAddr, _ := helper.yggVault.PubKey.GetAddress(common.BNBChain)
				coins := common.Coins{
					common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One)),
					common.NewCoin(common.RuneAsset(), cosmos.NewUint(common.One)),
				}
				tx := common.Tx{
					ID:          GetRandomTxHash(),
					Chain:       common.BNBChain,
					FromAddress: fromAddr,
					ToAddress:   GetRandomBNBAddress(),
					Coins:       coins,
					Gas:         BNBGasFeeSingleton,
					Memo:        "yggdrasil-:30",
				}
				return NewMsgYggdrasil(tx, GetRandomPubKey(), 12, false, coins, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler YggdrasilHandler, msg cosmos.Msg, helper yggdrasilHandlerTestHelper) (*cosmos.Result, error) {
				ygg := msg.(MsgYggdrasil)
				addr, _ := ygg.PubKey.GetThorAddress()
				helper.keeper.errGetNodeAccount = addr
				return handler.Run(helper.ctx, msg, helper.version, helper.constAccessor)
			},
			expectedResult: errInternal,
		},
		{
			name: "yggdrasil return fund to asgard but fail to get pool should return an error",
			messageCreator: func(helper yggdrasilHandlerTestHelper) cosmos.Msg {
				fromAddr, _ := helper.yggVault.PubKey.GetAddress(common.BNBChain)
				coins := common.Coins{
					common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One)),
					common.NewCoin(common.RuneAsset(), cosmos.NewUint(common.One)),
				}
				tx := common.Tx{
					ID:          GetRandomTxHash(),
					Chain:       common.BNBChain,
					FromAddress: fromAddr,
					ToAddress:   GetRandomBNBAddress(),
					Coins:       coins,
					Gas:         BNBGasFeeSingleton,
					Memo:        "yggdrasil-:30",
				}
				return NewMsgYggdrasil(tx, helper.yggVault.PubKey, 12, false, coins, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler YggdrasilHandler, msg cosmos.Msg, helper yggdrasilHandlerTestHelper) (*cosmos.Result, error) {
				helper.keeper.errGetPool = true
				return handler.Run(helper.ctx, msg, helper.version, helper.constAccessor)
			},
			expectedResult: errInternal,
		},
		{
			name: "yggdrasil return fund to asgard should be slashed",
			messageCreator: func(helper yggdrasilHandlerTestHelper) cosmos.Msg {
				fromAddr, _ := helper.yggVault.PubKey.GetAddress(common.BNBChain)
				coins := common.Coins{
					common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One)),
					common.NewCoin(common.RuneAsset(), cosmos.NewUint(common.One)),
				}
				tx := common.Tx{
					ID:          GetRandomTxHash(),
					Chain:       common.BNBChain,
					FromAddress: fromAddr,
					ToAddress:   GetRandomBNBAddress(),
					Coins:       coins,
					Gas:         BNBGasFeeSingleton,
					Memo:        "yggdrasil-:30",
				}
				return NewMsgYggdrasil(tx, helper.asgardVault.PubKey, 12, false, coins, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler YggdrasilHandler, msg cosmos.Msg, helper yggdrasilHandlerTestHelper) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, helper.version, helper.constAccessor)
			},
			expectedResult: nil,
		},
		{
			name: "yggdrasil return fail to get txout item should return an error",
			messageCreator: func(helper yggdrasilHandlerTestHelper) cosmos.Msg {
				fromAddr, _ := helper.yggVault.PubKey.GetAddress(common.BNBChain)
				coins := common.Coins{
					common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One)),
					common.NewCoin(common.RuneAsset(), cosmos.NewUint(common.One)),
				}

				tx := common.Tx{
					ID:          GetRandomTxHash(),
					Chain:       common.BNBChain,
					FromAddress: fromAddr,
					ToAddress:   GetRandomBNBAddress(),
					Coins:       coins,
					Gas:         BNBGasFeeSingleton,
					Memo:        "yggdrasil-:30",
				}
				return NewMsgYggdrasil(tx, helper.asgardVault.PubKey, 12, false, coins, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler YggdrasilHandler, msg cosmos.Msg, helper yggdrasilHandlerTestHelper) (*cosmos.Result, error) {
				helper.keeper.errGetTxOut = true
				return handler.Run(helper.ctx, msg, helper.version, helper.constAccessor)
			},
			expectedResult: errInternal,
		},
		{
			name: "yggdrasil return fund to asgard should works fine",
			messageCreator: func(helper yggdrasilHandlerTestHelper) cosmos.Msg {
				fromAddr, _ := helper.yggVault.PubKey.GetAddress(common.BNBChain)
				coins := common.Coins{
					common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One)),
					common.NewCoin(common.RuneAsset(), cosmos.NewUint(common.One)),
				}
				tx := common.Tx{
					ID:          GetRandomTxHash(),
					Chain:       common.BNBChain,
					FromAddress: fromAddr,
					ToAddress:   GetRandomBNBAddress(),
					Coins:       coins,
					Gas:         BNBGasFeeSingleton,
					Memo:        "yggdrasil-:30",
				}
				txOut := NewTxOut(30)
				txOut.TxArray = append(txOut.TxArray, &TxOutItem{
					Chain:       common.BNBChain,
					ToAddress:   tx.ToAddress,
					VaultPubKey: helper.yggVault.PubKey,
					InHash:      common.BlankTxID,
				})
				helper.keeper.SetTxOut(helper.ctx, txOut)
				return NewMsgYggdrasil(tx, helper.asgardVault.PubKey, 30, false, coins, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler YggdrasilHandler, msg cosmos.Msg, helper yggdrasilHandlerTestHelper) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, helper.version, helper.constAccessor)
			},
			expectedResult: nil,
		},
		{
			name: "yggdrasil return fund to retiring asgard should works fine",
			messageCreator: func(helper yggdrasilHandlerTestHelper) cosmos.Msg {
				fromAddr, _ := helper.yggVault.PubKey.GetAddress(common.BNBChain)
				coins := common.Coins{
					common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One)),
					common.NewCoin(common.RuneAsset(), cosmos.NewUint(common.One)),
				}
				asgardVault := GetRandomVault()
				asgardVault.Status = RetiringVault
				asgardVault.Type = AsgardVault
				helper.keeper.SetVault(helper.ctx, asgardVault)
				addr, _ := asgardVault.PubKey.GetAddress(common.BNBChain)
				tx := common.Tx{
					ID:          GetRandomTxHash(),
					Chain:       common.BNBChain,
					FromAddress: fromAddr,
					ToAddress:   addr,
					Coins:       coins,
					Gas:         BNBGasFeeSingleton,
					Memo:        "yggdrasil-:30",
				}
				txOut := NewTxOut(30)
				txOut.TxArray = append(txOut.TxArray, &TxOutItem{
					Chain:       common.BNBChain,
					ToAddress:   tx.ToAddress,
					VaultPubKey: helper.yggVault.PubKey,
					InHash:      common.BlankTxID,
				})
				helper.keeper.SetTxOut(helper.ctx, txOut)
				return NewMsgYggdrasil(tx, helper.yggVault.PubKey, 30, false, coins, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler YggdrasilHandler, msg cosmos.Msg, helper yggdrasilHandlerTestHelper) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, helper.version, helper.constAccessor)
			},
			expectedResult: nil,
		},
		{
			name: "yggdrasil return fund to asgard should not be slashed",
			messageCreator: func(helper yggdrasilHandlerTestHelper) cosmos.Msg {
				fromAddr, _ := helper.yggVault.PubKey.GetAddress(common.BNBChain)
				coins := common.Coins{
					common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One)),
					common.NewCoin(common.RuneAsset(), cosmos.NewUint(common.One)),
				}
				// only when node account in NoneActive status can return yggdrasil fund
				helper.nodeAccount.Status = NodeStandby
				helper.keeper.SetNodeAccount(helper.ctx, helper.nodeAccount)
				asgardVault := GetRandomVault()
				asgardVault.Status = ActiveVault
				asgardVault.Type = AsgardVault
				helper.keeper.SetVault(helper.ctx, asgardVault)
				addr, _ := asgardVault.PubKey.GetAddress(common.BNBChain)
				tx := common.Tx{
					ID:          GetRandomTxHash(),
					Chain:       common.BNBChain,
					FromAddress: fromAddr,
					ToAddress:   addr,
					Coins:       coins,
					Gas:         BNBGasFeeSingleton,
					Memo:        "yggdrasil-:30",
				}
				return NewMsgYggdrasil(tx, helper.yggVault.PubKey, 30, false, coins, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler YggdrasilHandler, msg cosmos.Msg, helper yggdrasilHandlerTestHelper) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, helper.version, helper.constAccessor)
			},
			validator: func(helper yggdrasilHandlerTestHelper, msg cosmos.Msg, result *cosmos.Result, c *C) {
				if helper.version.EQ(semver.MustParse("0.1.0")) {
					// should be slashed
					na, err := helper.keeper.GetNodeAccount(helper.ctx, helper.nodeAccount.NodeAddress)
					c.Assert(err, IsNil)
					c.Assert(na.Bond.LT(cosmos.NewUint(100*common.One)), Equals, true)
				} else if helper.version.EQ(semver.MustParse("0.8.0")) {
					na, err := helper.keeper.GetNodeAccount(helper.ctx, helper.nodeAccount.NodeAddress)
					c.Assert(err, IsNil)
					c.Assert(na.Bond.LT(cosmos.NewUint(100*common.One)), Equals, true)
				} else if helper.version.EQ(semver.MustParse("0.17.0")) {
					na, err := helper.keeper.GetNodeAccount(helper.ctx, helper.nodeAccount.NodeAddress)
					c.Assert(err, IsNil)
					c.Assert(na.Bond.Equal(cosmos.NewUint(100*common.One)), Equals, true)
				}
			},
			expectedResult: nil,
		},
	}
	for _, tc := range testCases {
		helper := newYggdrasilHandlerTestHelper(c, version)
		handler := NewYggdrasilHandler(helper.keeper, helper.mgr)
		msg := tc.messageCreator(helper)
		result, err := tc.runner(handler, msg, helper)
		if tc.expectedResult == nil {
			c.Assert(err, IsNil)
		} else {
			c.Assert(errors.Is(err, tc.expectedResult), Equals, true, Commentf("case:%s,version:%s", tc.name, version.String()))
		}
		if tc.validator != nil {
			tc.validator(helper, msg, result, c)
		}
	}
}
