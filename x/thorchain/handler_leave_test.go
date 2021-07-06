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

type HandlerLeaveSuite struct{}

var _ = Suite(&HandlerLeaveSuite{})

func (HandlerLeaveSuite) TestLeaveHandler_NotActiveNodeLeave(c *C) {
	w := getHandlerTestWrapper(c, 1, true, false)
	vault := GetRandomVault()
	w.keeper.SetVault(w.ctx, vault)
	leaveHandler := NewLeaveHandler(w.keeper, NewDummyMgr())
	acc2 := GetRandomNodeAccount(NodeStandby)
	acc2.Bond = cosmos.NewUint(100 * common.One)
	c.Assert(w.keeper.SetNodeAccount(w.ctx, acc2), IsNil)
	ygg := NewVault(common.BlockHeight(w.ctx), ActiveVault, YggdrasilVault, acc2.PubKeySet.Secp256k1, common.Chains{common.RuneAsset().Chain})
	c.Assert(w.keeper.SetVault(w.ctx, ygg), IsNil)

	FundModule(c, w.ctx, w.keeper, BondName, 100)

	txID := GetRandomTxHash()
	tx := common.NewTx(
		txID,
		acc2.BondAddress,
		GetRandomBNBAddress(),
		common.Coins{common.NewCoin(common.RuneAsset(), cosmos.OneUint())},
		BNBGasFeeSingleton,
		"LEAVE",
	)
	msgLeave := NewMsgLeave(tx, acc2.NodeAddress, w.activeNodeAccount.NodeAddress)
	ver := constants.SWVersion
	constAccessor := constants.GetConstantValues(ver)
	_, err := leaveHandler.Run(w.ctx, msgLeave, ver, constAccessor)
	c.Assert(err, IsNil)
	_, err = leaveHandler.Run(w.ctx, msgLeave, semver.Version{}, constAccessor)
	c.Assert(err, NotNil)
}

func (HandlerLeaveSuite) TestLeaveHandlerV5_NotActiveNodeLeave(c *C) {
	w := getHandlerTestWrapper(c, 1, true, false)
	vault := GetRandomVault()
	w.keeper.SetVault(w.ctx, vault)
	leaveHandler := NewLeaveHandler(w.keeper, NewDummyMgr())
	acc2 := GetRandomNodeAccount(NodeStandby)
	acc2.Bond = cosmos.NewUint(100 * common.One)
	c.Assert(w.keeper.SetNodeAccount(w.ctx, acc2), IsNil)
	ygg := NewVault(common.BlockHeight(w.ctx), ActiveVault, YggdrasilVault, acc2.PubKeySet.Secp256k1, common.Chains{common.RuneAsset().Chain})
	c.Assert(w.keeper.SetVault(w.ctx, ygg), IsNil)

	FundModule(c, w.ctx, w.keeper, BondName, 100)

	txID := GetRandomTxHash()
	tx := common.NewTx(
		txID,
		acc2.BondAddress,
		GetRandomBNBAddress(),
		common.Coins{common.NewCoin(common.RuneAsset(), cosmos.OneUint())},
		BNBGasFeeSingleton,
		"LEAVE",
	)
	msgLeave := NewMsgLeave(tx, acc2.NodeAddress, w.activeNodeAccount.NodeAddress)
	ver := semver.MustParse("0.5.0")
	constAccessor := constants.GetConstantValues(ver)
	_, err := leaveHandler.Run(w.ctx, msgLeave, ver, constAccessor)
	c.Assert(err, IsNil)
	accAfterLeave, err := w.keeper.GetNodeAccount(w.ctx, acc2.NodeAddress)
	c.Assert(err, IsNil)
	c.Assert(accAfterLeave.Status, Equals, NodeDisabled)
	_, err = leaveHandler.Run(w.ctx, msgLeave, semver.Version{}, constAccessor)
	c.Assert(err, NotNil)
}

func (HandlerLeaveSuite) TestLeaveHandler_ActiveNodeLeave(c *C) {
	var err error
	w := getHandlerTestWrapper(c, 1, true, false)
	leaveHandler := NewLeaveHandler(w.keeper, NewDummyMgr())
	acc2 := GetRandomNodeAccount(NodeActive)
	acc2.Bond = cosmos.NewUint(100 * common.One)
	c.Assert(w.keeper.SetNodeAccount(w.ctx, acc2), IsNil)
	txID := GetRandomTxHash()
	tx := common.NewTx(
		txID,
		acc2.BondAddress,
		GetRandomBNBAddress(),
		common.Coins{common.NewCoin(common.RuneAsset(), cosmos.OneUint())},
		BNBGasFeeSingleton,
		"",
	)
	msgLeave := NewMsgLeave(tx, acc2.NodeAddress, w.activeNodeAccount.NodeAddress)
	ver := constants.SWVersion
	constAccessor := constants.GetConstantValues(ver)
	_, err = leaveHandler.Run(w.ctx, msgLeave, ver, constAccessor)
	c.Assert(err, IsNil)

	acc2, err = w.keeper.GetNodeAccount(w.ctx, acc2.NodeAddress)
	c.Assert(err, IsNil)
	c.Check(acc2.Bond.Equal(cosmos.NewUint(10000000001)), Equals, true, Commentf("Bond:%d\n", acc2.Bond.Uint64()))
}

func (HandlerLeaveSuite) TestLeaveHandlerV5_ActiveNodeLeave(c *C) {
	var err error
	w := getHandlerTestWrapper(c, 1, true, false)
	leaveHandler := NewLeaveHandler(w.keeper, NewDummyMgr())
	acc2 := GetRandomNodeAccount(NodeActive)
	acc2.Bond = cosmos.NewUint(100 * common.One)
	c.Assert(w.keeper.SetNodeAccount(w.ctx, acc2), IsNil)
	txID := GetRandomTxHash()
	tx := common.NewTx(
		txID,
		acc2.BondAddress,
		GetRandomBNBAddress(),
		common.Coins{common.NewCoin(common.RuneAsset(), cosmos.OneUint())},
		BNBGasFeeSingleton,
		"",
	)
	msgLeave := NewMsgLeave(tx, acc2.NodeAddress, w.activeNodeAccount.NodeAddress)
	ver := semver.MustParse("0.5.0")
	constAccessor := constants.GetConstantValues(ver)
	_, err = leaveHandler.Run(w.ctx, msgLeave, ver, constAccessor)
	c.Assert(err, IsNil)

	acc2, err = w.keeper.GetNodeAccount(w.ctx, acc2.NodeAddress)
	c.Assert(err, IsNil)
	c.Check(acc2.Bond.Equal(cosmos.NewUint(10000000001)), Equals, true, Commentf("Bond:%d\n", acc2.Bond.Uint64()))
}

func (HandlerLeaveSuite) TestLeaveJail(c *C) {
	w := getHandlerTestWrapper(c, 1, true, false)
	vault := GetRandomVault()
	w.keeper.SetVault(w.ctx, vault)
	leaveHandler := NewLeaveHandler(w.keeper, NewDummyMgr())
	acc2 := GetRandomNodeAccount(NodeStandby)
	acc2.Bond = cosmos.NewUint(100 * common.One)
	c.Assert(w.keeper.SetNodeAccount(w.ctx, acc2), IsNil)

	w.keeper.SetNodeAccountJail(w.ctx, acc2.NodeAddress, common.BlockHeight(w.ctx)+100, "test it")

	ygg := NewVault(common.BlockHeight(w.ctx), ActiveVault, YggdrasilVault, acc2.PubKeySet.Secp256k1, common.Chains{common.RuneAsset().Chain})
	c.Assert(w.keeper.SetVault(w.ctx, ygg), IsNil)

	FundModule(c, w.ctx, w.keeper, BondName, 100)

	txID := GetRandomTxHash()
	tx := common.NewTx(
		txID,
		acc2.BondAddress,
		GetRandomBNBAddress(),
		common.Coins{common.NewCoin(common.RuneAsset(), cosmos.OneUint())},
		BNBGasFeeSingleton,
		"LEAVE",
	)
	msgLeave := NewMsgLeave(tx, acc2.NodeAddress, w.activeNodeAccount.NodeAddress)
	ver := constants.SWVersion
	constAccessor := constants.GetConstantValues(ver)
	_, err := leaveHandler.Run(w.ctx, msgLeave, ver, constAccessor)
	c.Assert(err, NotNil)
}

func (HandlerLeaveSuite) TestLeaveValidation(c *C) {
	w := getHandlerTestWrapper(c, 1, true, false)
	ver := constants.SWVersion
	constAccessor := constants.GetConstantValues(ver)
	testCases := []struct {
		name          string
		msgLeave      MsgLeave
		expectedError error
	}{
		{
			name: "empty from address should fail",
			msgLeave: NewMsgLeave(common.Tx{
				ID:          GetRandomTxHash(),
				Chain:       common.BNBChain,
				FromAddress: "",
				ToAddress:   GetRandomBNBAddress(),
				Coins: common.Coins{
					common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One)),
				},
				Gas: common.Gas{
					common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One)),
				},
				Memo: "",
			}, w.activeNodeAccount.NodeAddress, w.activeNodeAccount.NodeAddress),
			expectedError: se.ErrInvalidAddress,
		},
		{
			name: "non-matching from address should fail",
			msgLeave: NewMsgLeave(common.Tx{
				ID:          GetRandomTxHash(),
				Chain:       common.BNBChain,
				FromAddress: GetRandomBNBAddress(),
				ToAddress:   GetRandomBNBAddress(),
				Coins: common.Coins{
					common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One)),
				},
				Gas: common.Gas{
					common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One)),
				},
				Memo: "",
			}, w.activeNodeAccount.NodeAddress, w.activeNodeAccount.NodeAddress),
			expectedError: se.ErrUnauthorized,
		},
		{
			name: "empty tx id should fail",
			msgLeave: NewMsgLeave(common.Tx{
				ID:          common.TxID(""),
				Chain:       common.BNBChain,
				FromAddress: w.activeNodeAccount.BondAddress,
				ToAddress:   GetRandomBNBAddress(),
				Coins: common.Coins{
					common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One)),
				},
				Gas: common.Gas{
					common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One)),
				},
				Memo: "",
			}, w.activeNodeAccount.NodeAddress, w.activeNodeAccount.NodeAddress),
			expectedError: se.ErrUnknownRequest,
		},
		{
			name: "empty signer should fail",
			msgLeave: NewMsgLeave(common.Tx{
				ID:          GetRandomTxHash(),
				Chain:       common.BNBChain,
				FromAddress: w.activeNodeAccount.BondAddress,
				ToAddress:   GetRandomBNBAddress(),
				Coins: common.Coins{
					common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One)),
				},
				Gas: common.Gas{
					common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One)),
				},
				Memo: "",
			}, w.activeNodeAccount.NodeAddress, cosmos.AccAddress{}),
			expectedError: se.ErrInvalidAddress,
		},
	}
	for _, item := range testCases {
		c.Log(item.name)
		leaveHandler := NewLeaveHandler(w.keeper, NewDummyMgr())
		_, err := leaveHandler.Run(w.ctx, item.msgLeave, ver, constAccessor)
		c.Check(errors.Is(err, item.expectedError), Equals, true, Commentf("name:%s, %s", item.name, err))
	}
}

type LeaveHandlerTestHelper struct {
	keeper.Keeper
	failGetNodeAccount bool
	failGetVault       bool
	failSetNodeAccount bool
}

func NewLeaveHandlerTestHelper(k keeper.Keeper) *LeaveHandlerTestHelper {
	return &LeaveHandlerTestHelper{
		Keeper: k,
	}
}

func (h *LeaveHandlerTestHelper) GetNodeAccount(ctx cosmos.Context, addr cosmos.AccAddress) (NodeAccount, error) {
	if h.failGetNodeAccount {
		return NodeAccount{}, kaboom
	}
	return h.Keeper.GetNodeAccount(ctx, addr)
}

func (h *LeaveHandlerTestHelper) SetNodeAccount(ctx cosmos.Context, na NodeAccount) error {
	if h.failSetNodeAccount {
		return kaboom
	}
	return h.Keeper.SetNodeAccount(ctx, na)
}

func (h *LeaveHandlerTestHelper) GetVault(ctx cosmos.Context, pk common.PubKey) (Vault, error) {
	if h.failGetVault {
		return Vault{}, kaboom
	}
	return h.Keeper.GetVault(ctx, pk)
}

func (HandlerLeaveSuite) TestLeaveDifferentValidations(c *C) {
	testCases := []struct {
		name            string
		messageProvider func(ctx cosmos.Context, helper *LeaveHandlerTestHelper) cosmos.Msg
		validator       func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *LeaveHandlerTestHelper, name string, msg cosmos.Msg)
	}{
		{
			name: "invalid message type should return an error",
			messageProvider: func(ctx cosmos.Context, helper *LeaveHandlerTestHelper) cosmos.Msg {
				return NewMsgNetworkFee(1024, common.BTCChain, 1, bnbSingleTxFee, GetRandomBech32Addr())
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *LeaveHandlerTestHelper, name string, msg cosmos.Msg) {
				c.Check(err, NotNil, Commentf(name))
				c.Check(result, IsNil, Commentf(name))
			},
		},
		{
			name: "fail to get node account should return an error",
			messageProvider: func(ctx cosmos.Context, helper *LeaveHandlerTestHelper) cosmos.Msg {
				helper.failGetNodeAccount = true
				return NewMsgLeave(GetRandomTx(), GetRandomBech32Addr(), GetRandomBech32Addr())
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *LeaveHandlerTestHelper, name string, msg cosmos.Msg) {
				c.Check(err, NotNil, Commentf(name))
				c.Check(result, IsNil, Commentf(name))
			},
		},
		{
			name: "empty node account should return an error",
			messageProvider: func(ctx cosmos.Context, helper *LeaveHandlerTestHelper) cosmos.Msg {
				return NewMsgLeave(GetRandomTx(), GetRandomBech32Addr(), GetRandomBech32Addr())
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *LeaveHandlerTestHelper, name string, msg cosmos.Msg) {
				c.Check(err, NotNil, Commentf(name))
				c.Check(result, IsNil, Commentf(name))
			},
		},
		{
			name: "fail to refund bond should return an error",
			messageProvider: func(ctx cosmos.Context, helper *LeaveHandlerTestHelper) cosmos.Msg {
				nodeAccount := GetRandomNodeAccount(NodeStandby)
				activeNodeAccount := GetRandomNodeAccount(NodeActive)
				helper.Keeper.SetNodeAccount(ctx, activeNodeAccount)
				helper.Keeper.SetNodeAccount(ctx, nodeAccount)
				tx := GetRandomTx()
				tx.FromAddress = nodeAccount.BondAddress
				// when there is no asgard vault to refund, refund should fail
				return NewMsgLeave(tx, nodeAccount.NodeAddress, GetRandomBech32Addr())
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *LeaveHandlerTestHelper, name string, msg cosmos.Msg) {
				c.Check(err, NotNil, Commentf(name))
				c.Check(result, IsNil, Commentf(name))
			},
		},
		{
			name: "vault not exist should refund bond",
			messageProvider: func(ctx cosmos.Context, helper *LeaveHandlerTestHelper) cosmos.Msg {
				nodeAccount := GetRandomNodeAccount(NodeStandby)
				activeNodeAccount := GetRandomNodeAccount(NodeActive)
				helper.Keeper.SetNodeAccount(ctx, activeNodeAccount)
				helper.Keeper.SetNodeAccount(ctx, nodeAccount)
				tx := GetRandomTx()
				tx.FromAddress = nodeAccount.BondAddress
				// add an asgard vault , otherwise we won't be able to send out fund
				vault := GetRandomVault()
				helper.SetVault(ctx, vault)
				return NewMsgLeave(tx, nodeAccount.NodeAddress, GetRandomBech32Addr())
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *LeaveHandlerTestHelper, name string, msg cosmos.Msg) {
				c.Check(err, IsNil, Commentf(name))
				c.Check(result, NotNil, Commentf(name))
			},
		},
		{
			name: "fail to get vault should return an error",
			messageProvider: func(ctx cosmos.Context, helper *LeaveHandlerTestHelper) cosmos.Msg {
				nodeAccount := GetRandomNodeAccount(NodeStandby)
				activeNodeAccount := GetRandomNodeAccount(NodeActive)
				helper.Keeper.SetNodeAccount(ctx, activeNodeAccount)
				helper.Keeper.SetNodeAccount(ctx, nodeAccount)
				tx := GetRandomTx()
				tx.FromAddress = nodeAccount.BondAddress
				vault := NewVault(1024, ActiveVault, YggdrasilVault, nodeAccount.PubKeySet.Secp256k1, common.Chains{common.BNBChain, common.BTCChain})
				helper.Keeper.SetVault(ctx, vault)
				helper.failGetVault = true
				return NewMsgLeave(tx, nodeAccount.NodeAddress, GetRandomBech32Addr())
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *LeaveHandlerTestHelper, name string, msg cosmos.Msg) {
				c.Check(err, NotNil, Commentf(name))
				c.Check(result, IsNil, Commentf(name))
			},
		},
		{
			name: "when vault still has fund , it should request yggdrasil return",
			messageProvider: func(ctx cosmos.Context, helper *LeaveHandlerTestHelper) cosmos.Msg {
				nodeAccount := GetRandomNodeAccount(NodeStandby)
				activeNodeAccount := GetRandomNodeAccount(NodeActive)
				helper.Keeper.SetNodeAccount(ctx, activeNodeAccount)
				helper.Keeper.SetNodeAccount(ctx, nodeAccount)
				tx := GetRandomTx()
				tx.FromAddress = nodeAccount.BondAddress
				vault := NewVault(1024, ActiveVault, YggdrasilVault, nodeAccount.PubKeySet.Secp256k1, common.Chains{common.BNBChain, common.BTCChain})
				vault.AddFunds(common.Coins{
					common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One*100)),
				})
				helper.Keeper.SetVault(ctx, vault)
				asgardVault := NewVault(1024, ActiveVault, AsgardVault, GetRandomPubKey(), common.Chains{common.BNBChain, common.BTCChain})
				helper.Keeper.SetVault(ctx, asgardVault)
				return NewMsgLeave(tx, nodeAccount.NodeAddress, GetRandomBech32Addr())
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *LeaveHandlerTestHelper, name string, msg cosmos.Msg) {
				c.Check(err, IsNil, Commentf(name))
				c.Check(result, NotNil, Commentf(name))
			},
		},
		{
			name: "fail to save node account should return an error",
			messageProvider: func(ctx cosmos.Context, helper *LeaveHandlerTestHelper) cosmos.Msg {
				nodeAccount := GetRandomNodeAccount(NodeStandby)
				activeNodeAccount := GetRandomNodeAccount(NodeActive)
				helper.Keeper.SetNodeAccount(ctx, activeNodeAccount)
				helper.Keeper.SetNodeAccount(ctx, nodeAccount)
				tx := GetRandomTx()
				tx.FromAddress = nodeAccount.BondAddress
				asgardVault := NewVault(1024, ActiveVault, AsgardVault, GetRandomPubKey(), common.Chains{common.BNBChain, common.BTCChain})
				helper.Keeper.SetVault(ctx, asgardVault)
				helper.failSetNodeAccount = true
				return NewMsgLeave(tx, nodeAccount.NodeAddress, GetRandomBech32Addr())
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *LeaveHandlerTestHelper, name string, msg cosmos.Msg) {
				c.Check(err, NotNil, Commentf(name))
				c.Check(result, IsNil, Commentf(name))
			},
		},
		{
			name: "when node account is still belongs to a retiring vault , don't return bond",
			messageProvider: func(ctx cosmos.Context, helper *LeaveHandlerTestHelper) cosmos.Msg {
				nodeAccount := GetRandomNodeAccount(NodeDisabled)
				nodeAccount.Bond = cosmos.NewUint(100)
				activeNodeAccount := GetRandomNodeAccount(NodeActive)
				helper.Keeper.SetNodeAccount(ctx, activeNodeAccount)
				helper.Keeper.SetNodeAccount(ctx, nodeAccount)
				tx := GetRandomTx()
				tx.FromAddress = nodeAccount.BondAddress
				asgardVault := NewVault(1024, ActiveVault, AsgardVault, GetRandomPubKey(), common.Chains{common.BNBChain, common.BTCChain})
				helper.Keeper.SetVault(ctx, asgardVault)
				retiringVault := NewVault(1000, RetiringVault, AsgardVault, GetRandomPubKey(), common.Chains{common.BNBChain, common.BTCChain})
				retiringVault.Membership = common.PubKeys{
					nodeAccount.PubKeySet.Secp256k1,
					GetRandomPubKey(),
				}
				helper.Keeper.SetVault(ctx, retiringVault)
				return NewMsgLeave(tx, nodeAccount.NodeAddress, GetRandomBech32Addr())
			},
			validator: func(c *C, ctx cosmos.Context, result *cosmos.Result, err error, helper *LeaveHandlerTestHelper, name string, msg cosmos.Msg) {
				leaveMsg := msg.(MsgLeave)
				na, err := helper.GetNodeAccount(ctx, leaveMsg.NodeAddress)
				c.Assert(err, IsNil)
				c.Assert(na.Bond.Equal(cosmos.NewUint(100)), Equals, true)
				c.Check(err, IsNil, Commentf(name))
				c.Check(result, NotNil, Commentf(name))
			},
		},
	}

	for _, tc := range testCases {
		ctx, k := setupKeeperForTest(c)
		FundModule(c, ctx, k, BondName, 1000)
		helper := NewLeaveHandlerTestHelper(k)
		mgr := NewManagers(helper)
		mgr.BeginBlock(ctx)
		handler := NewLeaveHandler(helper, mgr)
		msg := tc.messageProvider(ctx, helper)
		ver := semver.MustParse("0.5.0")
		constantAccessor := constants.GetConstantValues(ver)
		result, err := handler.Run(ctx, msg, ver, constantAccessor)
		tc.validator(c, ctx, result, err, helper, tc.name, msg)
	}
}
