package thorchain

import (
	"errors"

	"github.com/blang/semver"
	se "github.com/cosmos/cosmos-sdk/types/errors"
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/tss/go-tss/blame"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
	"gitlab.com/thorchain/thornode/x/thorchain/keeper"
)

type HandlerTssKeysignSuite struct{}

var _ = Suite(&HandlerTssKeysignSuite{})

type tssKeysignFailHandlerTestHelper struct {
	ctx           cosmos.Context
	version       semver.Version
	keeper        *tssKeysignKeeperHelper
	constAccessor constants.ConstantValues
	nodeAccount   NodeAccount
	mgr           Manager
	members       common.PubKeys
	blame         blame.Blame
}

type tssKeysignKeeperHelper struct {
	keeper.Keeper
	errListActiveAccounts           bool
	errGetTssVoter                  bool
	errFailToGetNodeAccountByPubKey bool
	errFailSetNodeAccount           bool
}

func newTssKeysignFailKeeperHelper(keeper keeper.Keeper) *tssKeysignKeeperHelper {
	return &tssKeysignKeeperHelper{
		Keeper: keeper,
	}
}

func (k *tssKeysignKeeperHelper) GetNodeAccountByPubKey(ctx cosmos.Context, pk common.PubKey) (NodeAccount, error) {
	if k.errFailToGetNodeAccountByPubKey {
		return NodeAccount{}, kaboom
	}
	return k.Keeper.GetNodeAccountByPubKey(ctx, pk)
}

func (k *tssKeysignKeeperHelper) SetNodeAccount(ctx cosmos.Context, na NodeAccount) error {
	if k.errFailSetNodeAccount {
		return kaboom
	}
	return k.Keeper.SetNodeAccount(ctx, na)
}

func (k *tssKeysignKeeperHelper) GetTssKeysignFailVoter(ctx cosmos.Context, id string) (TssKeysignFailVoter, error) {
	if k.errGetTssVoter {
		return TssKeysignFailVoter{}, kaboom
	}
	return k.Keeper.GetTssKeysignFailVoter(ctx, id)
}

func (k *tssKeysignKeeperHelper) ListActiveNodeAccounts(ctx cosmos.Context) (NodeAccounts, error) {
	if k.errListActiveAccounts {
		return NodeAccounts{}, kaboom
	}
	return k.Keeper.ListActiveNodeAccounts(ctx)
}

func signVoter(ctx cosmos.Context, keeper keeper.Keeper, except cosmos.AccAddress) (result []cosmos.AccAddress) {
	active, _ := keeper.ListActiveNodeAccounts(ctx)
	for _, na := range active {
		if na.NodeAddress.Equals(except) {
			continue
		}
		result = append(result, na.NodeAddress)
	}
	return
}

func newTssKeysignHandlerTestHelper(c *C, ver semver.Version) tssKeysignFailHandlerTestHelper {
	ctx, k := setupKeeperForTest(c)
	ctx = ctx.WithBlockHeight(1023)
	keeper := newTssKeysignFailKeeperHelper(k)
	// active account
	nodeAccount := GetRandomNodeAccount(NodeActive)
	nodeAccount.Bond = cosmos.NewUint(100 * common.One)
	c.Assert(keeper.SetNodeAccount(ctx, nodeAccount), IsNil)
	constAccessor := constants.GetConstantValues(ver)
	mgr := NewDummyMgr()

	var members []blame.Node
	for i := 0; i < 8; i++ {
		na := GetRandomNodeAccount(NodeActive)
		members = append(members, blame.Node{Pubkey: na.PubKeySet.Secp256k1.String()})
		_ = keeper.SetNodeAccount(ctx, na)
	}
	blame := blame.Blame{
		FailReason: "whatever",
		BlameNodes: []blame.Node{members[0], members[1]},
	}
	asgardVault := NewVault(common.BlockHeight(ctx), ActiveVault, AsgardVault, GetRandomPubKey(), common.Chains{common.BNBChain})
	c.Assert(keeper.SetVault(ctx, asgardVault), IsNil)
	return tssKeysignFailHandlerTestHelper{
		ctx:           ctx,
		version:       ver,
		keeper:        keeper,
		constAccessor: constAccessor,
		nodeAccount:   nodeAccount,
		mgr:           mgr,
		blame:         blame,
	}
}

func (h HandlerTssKeysignSuite) TestTssKeysignFailHandler_accept_standby_node_messages(c *C) {
	ver := semver.MustParse("0.18.0")
	helper := newTssKeysignHandlerTestHelper(c, ver)
	handler := NewTssKeysignHandler(helper.keeper, NewDummyMgr())
	vault := NewVault(1024, RetiringVault, AsgardVault, GetRandomPubKey(), common.Chains{
		common.BNBChain,
	})
	accounts := NodeAccounts{}
	for i := 0; i < 8; i++ {
		na := GetRandomNodeAccount(NodeActive)
		_ = helper.keeper.SetNodeAccount(helper.ctx, na)
		vault.Membership = append(vault.Membership, na.PubKeySet.Secp256k1)
		accounts = append(accounts, na)
	}
	naStandby := GetRandomNodeAccount(NodeStandby)
	_ = helper.keeper.SetNodeAccount(helper.ctx, naStandby)
	vault.Membership = append(vault.Membership, naStandby.PubKeySet.Secp256k1)
	c.Assert(helper.keeper.SetVault(helper.ctx, vault), IsNil)
	for idx, item := range accounts {
		if idx >= 4 {
			break
		}
		msg := NewMsgTssKeysignFail(common.BlockHeight(helper.ctx), helper.blame, "hello", common.Coins{common.NewCoin(common.BNBAsset, cosmos.NewUint(100))}, item.NodeAddress, vault.PubKey)
		result, err := handler.Run(helper.ctx, msg, ver, constants.GetConstantValues(ver))
		c.Assert(result, NotNil)
		c.Assert(err, IsNil)
	}
	msg := NewMsgTssKeysignFail(common.BlockHeight(helper.ctx), helper.blame, "hello", common.Coins{common.NewCoin(common.BNBAsset, cosmos.NewUint(100))}, naStandby.NodeAddress, vault.PubKey)
	result, err := handler.Run(helper.ctx, msg, ver, constants.GetConstantValues(ver))
	c.Assert(result, NotNil)
	c.Assert(err, IsNil)
}

func (h HandlerTssKeysignSuite) TestTssKeysignFailHandler(c *C) {
	h.testTssKeysignFailHandlerWithVersion(c, constants.SWVersion)
	h.testTssKeysignFailHandlerWithVersion(c, semver.MustParse("0.13.0"))
}

func (h HandlerTssKeysignSuite) testTssKeysignFailHandlerWithVersion(c *C, ver semver.Version) {
	testCases := []struct {
		name           string
		messageCreator func(helper tssKeysignFailHandlerTestHelper) cosmos.Msg
		runner         func(handler TssKeysignHandler, msg cosmos.Msg, helper tssKeysignFailHandlerTestHelper) (*cosmos.Result, error)
		validator      func(helper tssKeysignFailHandlerTestHelper, msg cosmos.Msg, result *cosmos.Result, c *C)
		expectedResult error
	}{
		{
			name: "invalid message should return an error",
			messageCreator: func(helper tssKeysignFailHandlerTestHelper) cosmos.Msg {
				return NewMsgNoOp(GetRandomObservedTx(), helper.nodeAccount.NodeAddress)
			},
			runner: func(handler TssKeysignHandler, msg cosmos.Msg, helper tssKeysignFailHandlerTestHelper) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, helper.version, helper.constAccessor)
			},
			expectedResult: errInvalidMessage,
		},
		{
			name: "bad version should return an error",
			messageCreator: func(helper tssKeysignFailHandlerTestHelper) cosmos.Msg {
				return NewMsgTssKeysignFail(common.BlockHeight(helper.ctx), helper.blame, "hello", common.Coins{common.NewCoin(common.BNBAsset, cosmos.NewUint(100))}, helper.nodeAccount.NodeAddress, GetRandomPubKey())
			},
			runner: func(handler TssKeysignHandler, msg cosmos.Msg, helper tssKeysignFailHandlerTestHelper) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, semver.MustParse("0.0.1"), helper.constAccessor)
			},
			expectedResult: errBadVersion,
		},
		{
			name: "Not signed by an active account should return an error",
			messageCreator: func(helper tssKeysignFailHandlerTestHelper) cosmos.Msg {
				return NewMsgTssKeysignFail(common.BlockHeight(helper.ctx), helper.blame, "hello", common.Coins{common.NewCoin(common.BNBAsset, cosmos.NewUint(100))}, GetRandomBech32Addr(), GetRandomPubKey())
			},
			runner: func(handler TssKeysignHandler, msg cosmos.Msg, helper tssKeysignFailHandlerTestHelper) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, ver, helper.constAccessor)
			},
			expectedResult: se.ErrUnauthorized,
		},
		{
			name: "empty signer should return an error",
			messageCreator: func(helper tssKeysignFailHandlerTestHelper) cosmos.Msg {
				return NewMsgTssKeysignFail(common.BlockHeight(helper.ctx), helper.blame, "hello", common.Coins{common.NewCoin(common.BNBAsset, cosmos.NewUint(100))}, cosmos.AccAddress{}, GetRandomPubKey())
			},
			runner: func(handler TssKeysignHandler, msg cosmos.Msg, helper tssKeysignFailHandlerTestHelper) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, ver, helper.constAccessor)
			},
			expectedResult: se.ErrInvalidAddress,
		},
		{
			name: "empty id should return an error",
			messageCreator: func(helper tssKeysignFailHandlerTestHelper) cosmos.Msg {
				tssMsg := NewMsgTssKeysignFail(common.BlockHeight(helper.ctx), helper.blame, "hello", common.Coins{common.NewCoin(common.BNBAsset, cosmos.NewUint(100))}, helper.nodeAccount.NodeAddress, GetRandomPubKey())
				tssMsg.ID = ""
				return tssMsg
			},
			runner: func(handler TssKeysignHandler, msg cosmos.Msg, helper tssKeysignFailHandlerTestHelper) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, ver, helper.constAccessor)
			},
			expectedResult: se.ErrUnknownRequest,
		},
		{
			name: "empty member pubkeys should return an error",
			messageCreator: func(helper tssKeysignFailHandlerTestHelper) cosmos.Msg {
				return NewMsgTssKeysignFail(common.BlockHeight(helper.ctx), blame.Blame{
					FailReason: "",
					BlameNodes: []blame.Node{},
				}, "hello", common.Coins{common.NewCoin(common.BNBAsset, cosmos.NewUint(100))}, helper.nodeAccount.NodeAddress, GetRandomPubKey())
			},
			runner: func(handler TssKeysignHandler, msg cosmos.Msg, helper tssKeysignFailHandlerTestHelper) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, ver, helper.constAccessor)
			},
			expectedResult: se.ErrUnknownRequest,
		},
		{
			name: "normal blame should works fine",
			messageCreator: func(helper tssKeysignFailHandlerTestHelper) cosmos.Msg {
				return NewMsgTssKeysignFail(common.BlockHeight(helper.ctx), helper.blame, "hello", common.Coins{common.NewCoin(common.BNBAsset, cosmos.NewUint(100))}, helper.nodeAccount.NodeAddress, GetRandomPubKey())
			},
			runner: func(handler TssKeysignHandler, msg cosmos.Msg, helper tssKeysignFailHandlerTestHelper) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, ver, helper.constAccessor)
			},
			expectedResult: nil,
		},
		{
			name: "when the same signer already sign the tss keysign failure , it should not do anything",
			messageCreator: func(helper tssKeysignFailHandlerTestHelper) cosmos.Msg {
				msg := NewMsgTssKeysignFail(common.BlockHeight(helper.ctx), helper.blame, "hello", common.Coins{common.NewCoin(common.BNBAsset, cosmos.NewUint(100))}, helper.nodeAccount.NodeAddress, GetRandomPubKey())
				voter, _ := helper.keeper.Keeper.GetTssKeysignFailVoter(helper.ctx, msg.ID)
				voter.Sign(msg.Signer)
				helper.keeper.Keeper.SetTssKeysignFailVoter(helper.ctx, voter)
				return msg
			},
			runner: func(handler TssKeysignHandler, msg cosmos.Msg, helper tssKeysignFailHandlerTestHelper) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, ver, helper.constAccessor)
			},
			expectedResult: nil,
		},
		{
			name: "fail to list active node accounts should return an error",
			messageCreator: func(helper tssKeysignFailHandlerTestHelper) cosmos.Msg {
				return NewMsgTssKeysignFail(common.BlockHeight(helper.ctx), helper.blame, "hello", common.Coins{common.NewCoin(common.BNBAsset, cosmos.NewUint(100))}, helper.nodeAccount.NodeAddress, GetRandomPubKey())
			},
			runner: func(handler TssKeysignHandler, msg cosmos.Msg, helper tssKeysignFailHandlerTestHelper) (*cosmos.Result, error) {
				helper.keeper.errListActiveAccounts = true
				return handler.Run(helper.ctx, msg, ver, helper.constAccessor)
			},
			expectedResult: kaboom,
		},
		{
			name: "fail to get Tss Keysign fail voter should return an error",
			messageCreator: func(helper tssKeysignFailHandlerTestHelper) cosmos.Msg {
				return NewMsgTssKeysignFail(common.BlockHeight(helper.ctx), helper.blame, "hello", common.Coins{common.NewCoin(common.BNBAsset, cosmos.NewUint(100))}, helper.nodeAccount.NodeAddress, GetRandomPubKey())
			},
			runner: func(handler TssKeysignHandler, msg cosmos.Msg, helper tssKeysignFailHandlerTestHelper) (*cosmos.Result, error) {
				helper.keeper.errGetTssVoter = true
				return handler.Run(helper.ctx, msg, ver, helper.constAccessor)
			},
			expectedResult: kaboom,
		},
		{
			name: "fail to get node account should return an error",
			messageCreator: func(helper tssKeysignFailHandlerTestHelper) cosmos.Msg {
				return NewMsgTssKeysignFail(common.BlockHeight(helper.ctx), helper.blame, "hello", common.Coins{common.NewCoin(common.BNBAsset, cosmos.NewUint(100))}, helper.nodeAccount.NodeAddress, GetRandomPubKey())
			},
			runner: func(handler TssKeysignHandler, msg cosmos.Msg, helper tssKeysignFailHandlerTestHelper) (*cosmos.Result, error) {
				mmsg := msg.(MsgTssKeysignFail)
				// prepopulate the voter with other signers
				voter, _ := helper.keeper.GetTssKeysignFailVoter(helper.ctx, mmsg.ID)
				voter.Signers = signVoter(helper.ctx, helper.keeper, mmsg.Signer)
				helper.keeper.SetTssKeysignFailVoter(helper.ctx, voter)
				helper.keeper.errFailToGetNodeAccountByPubKey = true
				return handler.Run(helper.ctx, msg, ver, helper.constAccessor)
			},
			expectedResult: errInternal,
		},
		{
			name: "without majority it should not take any actions",
			messageCreator: func(helper tssKeysignFailHandlerTestHelper) cosmos.Msg {
				return NewMsgTssKeysignFail(common.BlockHeight(helper.ctx), helper.blame, "hello", common.Coins{common.NewCoin(common.BNBAsset, cosmos.NewUint(100))}, helper.nodeAccount.NodeAddress, GetRandomPubKey())
			},
			runner: func(handler TssKeysignHandler, msg cosmos.Msg, helper tssKeysignFailHandlerTestHelper) (*cosmos.Result, error) {
				for i := 0; i < 3; i++ {
					na := GetRandomNodeAccount(NodeActive)
					if err := helper.keeper.SetNodeAccount(helper.ctx, na); err != nil {
						return nil, ErrInternal(err, "fail to set node account")
					}
				}
				return handler.Run(helper.ctx, msg, ver, helper.constAccessor)
			},
			expectedResult: nil,
		},
		{
			name: "with majority it should take actions",
			messageCreator: func(helper tssKeysignFailHandlerTestHelper) cosmos.Msg {
				return NewMsgTssKeysignFail(common.BlockHeight(helper.ctx), helper.blame, "hello", common.Coins{common.NewCoin(common.BNBAsset, cosmos.NewUint(100))}, helper.nodeAccount.NodeAddress, GetRandomPubKey())
			},
			runner: func(handler TssKeysignHandler, msg cosmos.Msg, helper tssKeysignFailHandlerTestHelper) (*cosmos.Result, error) {
				var na NodeAccount
				for i := 0; i < 3; i++ {
					na = GetRandomNodeAccount(NodeActive)
					if err := helper.keeper.SetNodeAccount(helper.ctx, na); err != nil {
						return nil, ErrInternal(err, "fail to set node account")
					}
				}
				_, err := handler.Run(helper.ctx, msg, ver, helper.constAccessor)
				if err != nil {
					return nil, err
				}
				msg = NewMsgTssKeysignFail(common.BlockHeight(helper.ctx), helper.blame, "hello", common.Coins{common.NewCoin(common.BNBAsset, cosmos.NewUint(100))}, na.NodeAddress, GetRandomPubKey())
				return handler.Run(helper.ctx, msg, ver, helper.constAccessor)
			},
			expectedResult: nil,
		},
	}
	for _, tc := range testCases {
		helper := newTssKeysignHandlerTestHelper(c, ver)
		handler := NewTssKeysignHandler(helper.keeper, NewDummyMgr())
		msg := tc.messageCreator(helper)

		c.Logf(">Name: %s\n", tc.name)
		result, err := tc.runner(handler, msg, helper)
		if tc.expectedResult == nil {
			c.Logf("Name: %s, %s\n", tc.name, err)
			c.Assert(err, IsNil)
		} else {
			c.Assert(errors.Is(err, tc.expectedResult), Equals, true, Commentf("name:%s, %w", tc.name, err))
		}
		if tc.validator != nil {
			tc.validator(helper, msg, result, c)
		}
	}
}
