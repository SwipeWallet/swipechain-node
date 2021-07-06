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

type HandlerUnBondSuite struct{}

type TestUnBondKeeper struct {
	keeper.KVStoreDummy
	activeNodeAccount   NodeAccount
	failGetNodeAccount  NodeAccount
	notEmptyNodeAccount NodeAccount
	jailNodeAccount     NodeAccount
	vault               Vault
}

func (k *TestUnBondKeeper) GetNodeAccount(_ cosmos.Context, addr cosmos.AccAddress) (NodeAccount, error) {
	if k.activeNodeAccount.NodeAddress.Equals(addr) {
		return k.activeNodeAccount, nil
	}
	if k.failGetNodeAccount.NodeAddress.Equals(addr) {
		return NodeAccount{}, fmt.Errorf("you asked for this error")
	}
	if k.notEmptyNodeAccount.NodeAddress.Equals(addr) {
		return k.notEmptyNodeAccount, nil
	}
	if k.jailNodeAccount.NodeAddress.Equals(addr) {
		return k.jailNodeAccount, nil
	}
	return NodeAccount{}, nil
}

func (k *TestUnBondKeeper) GetVault(ctx cosmos.Context, pk common.PubKey) (Vault, error) {
	if k.vault.PubKey.Equals(pk) {
		return k.vault, nil
	}
	return k.KVStoreDummy.GetVault(ctx, pk)
}

func (k *TestUnBondKeeper) VaultExists(ctx cosmos.Context, pkey common.PubKey) bool {
	if k.vault.PubKey.Equals(pkey) {
		return true
	}
	return false
}

func (k *TestUnBondKeeper) GetNodeAccountJail(ctx cosmos.Context, addr cosmos.AccAddress) (Jail, error) {
	if k.jailNodeAccount.NodeAddress.Equals(addr) {
		return Jail{
			NodeAddress:   addr,
			ReleaseHeight: ctx.BlockHeight() + 100,
			Reason:        "bad boy",
		}, nil
	}
	return Jail{}, nil
}

var _ = Suite(&HandlerUnBondSuite{})

func (HandlerUnBondSuite) TestUnBondHandler_Run(c *C) {
	ctx, k1 := setupKeeperForTest(c)
	// happy path
	activeNodeAccount := GetRandomNodeAccount(NodeActive)
	standbyNodeAccount := GetRandomNodeAccount(NodeStandby)
	c.Assert(k1.SetNodeAccount(ctx, activeNodeAccount), IsNil)
	c.Assert(k1.SetNodeAccount(ctx, standbyNodeAccount), IsNil)
	vault := NewVault(12, ActiveVault, YggdrasilVault, standbyNodeAccount.PubKeySet.Secp256k1, nil)
	c.Assert(k1.SetVault(ctx, vault), IsNil)
	vault = NewVault(12, ActiveVault, AsgardVault, GetRandomPubKey(), nil)
	vault.Coins = common.Coins{
		common.NewCoin(common.RuneAsset(), cosmos.NewUint(10000*common.One)),
	}
	c.Assert(k1.SetVault(ctx, vault), IsNil)

	handler := NewUnBondHandler(k1, NewDummyMgr())
	ver := semver.MustParse("0.12.0")
	constAccessor := constants.GetConstantValues(ver)
	txIn := common.NewTx(
		GetRandomTxHash(),
		standbyNodeAccount.BondAddress,
		GetRandomBNBAddress(),
		common.Coins{
			common.NewCoin(common.RuneAsset(), cosmos.NewUint(uint64(1))),
		},
		BNBGasFeeSingleton,
		"unbond me please",
	)
	msg := NewMsgUnBond(txIn, standbyNodeAccount.NodeAddress, cosmos.NewUint(uint64(5*common.One)), standbyNodeAccount.BondAddress, activeNodeAccount.NodeAddress)
	_, err := handler.Run(ctx, msg, ver, constAccessor)
	c.Assert(err, IsNil)
	na, err := k1.GetNodeAccount(ctx, standbyNodeAccount.NodeAddress)
	c.Assert(err, IsNil)
	c.Check(na.Bond.Equal(cosmos.NewUint(95*common.One+1)), Equals, true, Commentf("%d", na.Bond.Uint64()))

	// test unbonding for 1 rune, should fail, and NOT increase bond with inbound rune
	mgrBad := NewDummyMgr()
	mgrBad.txOutStore = NewTxStoreFailDummy()
	handler.mgr = mgrBad
	msg = NewMsgUnBond(txIn, standbyNodeAccount.NodeAddress, cosmos.NewUint(uint64(1*common.One)), standbyNodeAccount.BondAddress, activeNodeAccount.NodeAddress)
	_, err = handler.Run(ctx, msg, ver, constAccessor)
	c.Assert(err, NotNil)
	na, err = k1.GetNodeAccount(ctx, standbyNodeAccount.NodeAddress)
	c.Assert(err, IsNil)
	c.Check(na.Bond.Equal(cosmos.NewUint(95*common.One+1)), Equals, true, Commentf("%d", na.Bond.Uint64()))
	handler.mgr = NewDummyMgr() // revert bad handler

	k := &TestUnBondKeeper{
		activeNodeAccount:   activeNodeAccount,
		failGetNodeAccount:  GetRandomNodeAccount(NodeActive),
		notEmptyNodeAccount: standbyNodeAccount,
		jailNodeAccount:     GetRandomNodeAccount(NodeStandby),
	}
	// invalid version
	handler = NewUnBondHandler(k, NewDummyMgr())
	ver = semver.Version{}
	_, err = handler.Run(ctx, msg, ver, constAccessor)
	c.Assert(errors.Is(err, errBadVersion), Equals, true)

	// simulate fail to get node account
	ver = constants.SWVersion
	msg = NewMsgUnBond(txIn, k.failGetNodeAccount.NodeAddress, cosmos.NewUint(uint64(1)), GetRandomBNBAddress(), activeNodeAccount.NodeAddress)
	_, err = handler.Run(ctx, msg, ver, constAccessor)
	c.Assert(errors.Is(err, errInternal), Equals, true)

	// simulate vault with funds
	k.vault = Vault{
		Type: YggdrasilVault,
		Coins: common.Coins{
			common.NewCoin(common.RuneAsset(), cosmos.NewUint(uint64(1))),
		},
		PubKey: standbyNodeAccount.PubKeySet.Secp256k1,
	}
	msg = NewMsgUnBond(txIn, standbyNodeAccount.NodeAddress, cosmos.NewUint(uint64(1)), GetRandomBNBAddress(), standbyNodeAccount.NodeAddress)
	_, err = handler.Run(ctx, msg, ver, constAccessor)
	c.Assert(errors.Is(err, se.ErrUnknownRequest), Equals, true)

	// simulate fail to get vault
	k.vault = GetRandomVault()
	msg = NewMsgUnBond(txIn, activeNodeAccount.NodeAddress, cosmos.NewUint(uint64(1)), GetRandomBNBAddress(), activeNodeAccount.NodeAddress)
	result, err := handler.Run(ctx, msg, ver, constAccessor)
	c.Assert(err, NotNil)
	c.Assert(result, IsNil)

	// simulate vault is not yggdrasil
	k.vault = Vault{
		Type:   AsgardVault,
		PubKey: standbyNodeAccount.PubKeySet.Secp256k1,
	}

	msg = NewMsgUnBond(txIn, standbyNodeAccount.NodeAddress, cosmos.NewUint(uint64(1)), GetRandomBNBAddress(), standbyNodeAccount.NodeAddress)
	result, err = handler.Run(ctx, msg, ver, constAccessor)
	c.Assert(err, NotNil)
	c.Assert(result, IsNil)

	// simulate jail nodeAccount can't unbound
	msg = NewMsgUnBond(txIn, k.jailNodeAccount.NodeAddress, cosmos.NewUint(uint64(1)), GetRandomBNBAddress(), k.jailNodeAccount.NodeAddress)
	result, err = handler.Run(ctx, msg, ver, constAccessor)
	c.Assert(err, NotNil)
	c.Assert(result, IsNil)

	// invalid message should cause error
	result, err = handler.Run(ctx, NewMsgMimir("whatever", 1, GetRandomBech32Addr()), ver, constAccessor)
	c.Assert(err, NotNil)
	c.Assert(result, IsNil)
}

func (HandlerUnBondSuite) TestUnBondHanlder_retiringvault(c *C) {
	ctx, k1 := setupKeeperForTest(c)
	// happy path
	activeNodeAccount := GetRandomNodeAccount(NodeActive)
	standbyNodeAccount := GetRandomNodeAccount(NodeStandby)
	c.Assert(k1.SetNodeAccount(ctx, activeNodeAccount), IsNil)
	c.Assert(k1.SetNodeAccount(ctx, standbyNodeAccount), IsNil)
	vault := NewVault(12, ActiveVault, YggdrasilVault, standbyNodeAccount.PubKeySet.Secp256k1, nil)
	c.Assert(k1.SetVault(ctx, vault), IsNil)
	vault = NewVault(12, ActiveVault, AsgardVault, GetRandomPubKey(), nil)
	vault.Coins = common.Coins{
		common.NewCoin(common.RuneAsset(), cosmos.NewUint(10000*common.One)),
	}
	c.Assert(k1.SetVault(ctx, vault), IsNil)
	retiringVault := NewVault(12, RetiringVault, AsgardVault, GetRandomPubKey(), common.Chains{
		common.BNBChain, common.BTCChain, common.ETHChain,
	})
	retiringVault.Membership = common.PubKeys{
		activeNodeAccount.PubKeySet.Secp256k1,
		standbyNodeAccount.PubKeySet.Secp256k1,
	}
	retiringVault.Coins = common.Coins{
		common.NewCoin(common.RuneAsset(), cosmos.NewUint(10000*common.One)),
	}
	c.Assert(k1.SetVault(ctx, retiringVault), IsNil)
	handler := NewUnBondHandler(k1, NewDummyMgr())
	ver := semver.MustParse("0.18.0")
	constAccessor := constants.GetConstantValues(ver)
	txIn := common.NewTx(
		GetRandomTxHash(),
		standbyNodeAccount.BondAddress,
		GetRandomBNBAddress(),
		common.Coins{
			common.NewCoin(common.RuneAsset(), cosmos.NewUint(uint64(1))),
		},
		BNBGasFeeSingleton,
		"unbond me please",
	)
	msg := NewMsgUnBond(txIn, standbyNodeAccount.NodeAddress, cosmos.NewUint(uint64(5*common.One)), standbyNodeAccount.BondAddress, activeNodeAccount.NodeAddress)
	_, err := handler.Run(ctx, msg, ver, constAccessor)
	c.Assert(err, NotNil)
}

func (HandlerUnBondSuite) TestUnBondHandlerFailValidation(c *C) {
	ctx, k := setupKeeperForTest(c)
	activeNodeAccount := GetRandomNodeAccount(NodeActive)
	c.Assert(k.SetNodeAccount(ctx, activeNodeAccount), IsNil)
	handler := NewUnBondHandler(k, NewDummyMgr())
	ver := constants.SWVersion
	constAccessor := constants.GetConstantValues(ver)
	txIn := common.NewTx(
		GetRandomTxHash(),
		activeNodeAccount.BondAddress,
		GetRandomBNBAddress(),
		common.Coins{
			common.NewCoin(common.RuneAsset(), cosmos.NewUint(uint64(1))),
		},
		BNBGasFeeSingleton,
		"unbond it",
	)
	txInNoTxID := txIn
	txInNoTxID.ID = ""
	testCases := []struct {
		name        string
		msg         MsgUnBond
		expectedErr error
	}{
		{
			name:        "empty node address",
			msg:         NewMsgUnBond(txIn, cosmos.AccAddress{}, cosmos.NewUint(uint64(1)), activeNodeAccount.BondAddress, activeNodeAccount.NodeAddress),
			expectedErr: se.ErrInvalidAddress,
		},
		{
			name:        "zero bond",
			msg:         NewMsgUnBond(txIn, GetRandomNodeAccount(NodeStandby).NodeAddress, cosmos.ZeroUint(), activeNodeAccount.BondAddress, activeNodeAccount.NodeAddress),
			expectedErr: se.ErrUnknownRequest,
		},
		{
			name:        "empty bond address",
			msg:         NewMsgUnBond(txIn, GetRandomNodeAccount(NodeStandby).NodeAddress, cosmos.NewUint(uint64(1)), common.Address(""), activeNodeAccount.NodeAddress),
			expectedErr: se.ErrInvalidAddress,
		},
		{
			name:        "empty request hash",
			msg:         NewMsgUnBond(txInNoTxID, GetRandomNodeAccount(NodeStandby).NodeAddress, cosmos.NewUint(uint64(1)), activeNodeAccount.BondAddress, activeNodeAccount.NodeAddress),
			expectedErr: se.ErrUnknownRequest,
		},
		{
			name:        "empty signer",
			msg:         NewMsgUnBond(txIn, GetRandomNodeAccount(NodeStandby).NodeAddress, cosmos.NewUint(uint64(1)), activeNodeAccount.BondAddress, cosmos.AccAddress{}),
			expectedErr: se.ErrInvalidAddress,
		},
		{
			name:        "account shouldn't be active",
			msg:         NewMsgUnBond(txIn, activeNodeAccount.NodeAddress, cosmos.NewUint(uint64(1)), activeNodeAccount.BondAddress, activeNodeAccount.NodeAddress),
			expectedErr: se.ErrUnknownRequest,
		},
		{
			name:        "request not from original bond address should not be accepted",
			msg:         NewMsgUnBond(GetRandomTx(), activeNodeAccount.NodeAddress, cosmos.NewUint(uint64(1)), activeNodeAccount.BondAddress, activeNodeAccount.NodeAddress),
			expectedErr: se.ErrUnauthorized,
		},
	}
	for _, item := range testCases {
		c.Log(item.name)
		_, err := handler.Run(ctx, item.msg, ver, constAccessor)

		c.Check(errors.Is(err, item.expectedErr), Equals, true, Commentf("name: %s, %s", item.name, err))
	}
}
