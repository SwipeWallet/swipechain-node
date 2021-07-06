package thorchain

import (
	"errors"
	"sort"

	"github.com/blang/semver"
	se "github.com/cosmos/cosmos-sdk/types/errors"
	"gitlab.com/thorchain/tss/go-tss/blame"
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
	"gitlab.com/thorchain/thornode/x/thorchain/keeper"
)

type HandlerTssSuite struct{}

var _ = Suite(&HandlerTssSuite{})

type tssHandlerTestHelper struct {
	ctx           cosmos.Context
	version       semver.Version
	keeper        *tssKeeperHelper
	poolPk        common.PubKey
	constAccessor constants.ConstantValues
	nodeAccount   NodeAccount
	mgr           Manager
	members       common.PubKeys
	signer        cosmos.AccAddress
	keygenBlock   KeygenBlock
}

type tssKeeperHelper struct {
	keeper.Keeper
	errListActiveAccounts bool
	errGetTssVoter        bool
	errFailSaveVault      bool
	errFailGetNodeAccount bool
	errFailGetVaultData   bool
	errFailSetVaultData   bool
	errFailSetNodeAccount bool
}

func (k *tssKeeperHelper) GetNodeAccountByPubKey(ctx cosmos.Context, pk common.PubKey) (NodeAccount, error) {
	if k.errFailGetNodeAccount {
		return NodeAccount{}, kaboom
	}
	return k.Keeper.GetNodeAccountByPubKey(ctx, pk)
}

func (k *tssKeeperHelper) SetVault(ctx cosmos.Context, vault Vault) error {
	if k.errFailSaveVault {
		return kaboom
	}
	return k.Keeper.SetVault(ctx, vault)
}

func (k *tssKeeperHelper) GetTssVoter(ctx cosmos.Context, id string) (TssVoter, error) {
	if k.errGetTssVoter {
		return TssVoter{}, kaboom
	}
	return k.Keeper.GetTssVoter(ctx, id)
}

func (k *tssKeeperHelper) ListActiveNodeAccounts(ctx cosmos.Context) (NodeAccounts, error) {
	if k.errListActiveAccounts {
		return NodeAccounts{}, kaboom
	}
	return k.Keeper.ListActiveNodeAccounts(ctx)
}

func (k *tssKeeperHelper) GetVaultData(ctx cosmos.Context) (VaultData, error) {
	if k.errFailGetVaultData {
		return VaultData{}, kaboom
	}
	return k.Keeper.GetVaultData(ctx)
}

func (k *tssKeeperHelper) SetVaultData(ctx cosmos.Context, data VaultData) error {
	if k.errFailSetVaultData {
		return kaboom
	}
	return k.Keeper.SetVaultData(ctx, data)
}

func (k *tssKeeperHelper) SetNodeAccount(ctx cosmos.Context, na NodeAccount) error {
	if k.errFailSetNodeAccount {
		return kaboom
	}
	return k.Keeper.SetNodeAccount(ctx, na)
}

func newTssKeeperHelper(keeper keeper.Keeper) *tssKeeperHelper {
	return &tssKeeperHelper{
		Keeper: keeper,
	}
}

func newTssHandlerTestHelper(c *C, version semver.Version) tssHandlerTestHelper {
	ctx, k := setupKeeperForTest(c)
	ctx = ctx.WithBlockHeight(1023)
	keeper := newTssKeeperHelper(k)
	FundModule(c, ctx, k, BondName, 500)
	// active account
	nodeAccount := GetRandomNodeAccount(NodeActive)
	nodeAccount.Bond = cosmos.NewUint(100 * common.One)
	c.Assert(keeper.SetNodeAccount(ctx, nodeAccount), IsNil)

	constAccessor := constants.GetConstantValues(version)
	mgr := NewDummyMgr()

	var members common.PubKeys
	for i := 0; i < 8; i++ {
		members = append(members, GetRandomPubKey())
	}
	sort.SliceStable(members, func(i, j int) bool {
		return members[i].String() < members[j].String()
	})
	signer, err := members[0].GetThorAddress()
	c.Assert(err, IsNil)

	keygenBlock := NewKeygenBlock(common.BlockHeight(ctx))
	keygenBlock.Keygens = []Keygen{
		{Members: members},
	}
	keeper.SetKeygenBlock(ctx, keygenBlock)

	poolPk := GetRandomPubKey()
	msg := NewMsgTssPool(members, poolPk, AsgardKeygen, common.BlockHeight(ctx), blame.Blame{}, common.Chains{common.RuneAsset().Chain}, signer)
	voter := NewTssVoter(msg.ID, members, poolPk)
	keeper.SetTssVoter(ctx, voter)

	asgardVault := NewVault(common.BlockHeight(ctx), ActiveVault, AsgardVault, GetRandomPubKey(), common.Chains{common.RuneAsset().Chain})
	c.Assert(keeper.SetVault(ctx, asgardVault), IsNil)
	return tssHandlerTestHelper{
		ctx:           ctx,
		version:       version,
		keeper:        keeper,
		poolPk:        poolPk,
		constAccessor: constAccessor,
		nodeAccount:   nodeAccount,
		mgr:           mgr,
		members:       members,
		signer:        signer,
	}
}

func (s *HandlerTssSuite) TestTssHandler(c *C) {
	s.testTssHandlerWithVersion(c, constants.SWVersion)
	s.testTssHandlerWithVersion(c, semver.MustParse("0.13.0"))
}

func (s *HandlerTssSuite) testTssHandlerWithVersion(c *C, ver semver.Version) {
	testCases := []struct {
		name           string
		messageCreator func(helper tssHandlerTestHelper) cosmos.Msg
		runner         func(handler TssHandler, msg cosmos.Msg, helper tssHandlerTestHelper) (*cosmos.Result, error)
		validator      func(helper tssHandlerTestHelper, msg cosmos.Msg, result *cosmos.Result, c *C)
		expectedResult error
	}{
		{
			name: "invalid message should return an error",
			messageCreator: func(helper tssHandlerTestHelper) cosmos.Msg {
				return NewMsgNoOp(GetRandomObservedTx(), helper.signer)
			},
			runner: func(handler TssHandler, msg cosmos.Msg, helper tssHandlerTestHelper) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, helper.version, helper.constAccessor)
			},
			expectedResult: errInvalidMessage,
		},
		{
			name: "bad version should return an error",
			messageCreator: func(helper tssHandlerTestHelper) cosmos.Msg {
				return NewMsgTssPool(helper.members, GetRandomPubKey(), AsgardKeygen, common.BlockHeight(helper.ctx), blame.Blame{}, common.Chains{common.RuneAsset().Chain}, helper.signer)
			},
			runner: func(handler TssHandler, msg cosmos.Msg, helper tssHandlerTestHelper) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, semver.MustParse("0.0.1"), helper.constAccessor)
			},
			expectedResult: errBadVersion,
		},
		{
			name: "Not signed by an active account should return an error",
			messageCreator: func(helper tssHandlerTestHelper) cosmos.Msg {
				return NewMsgTssPool(helper.members, GetRandomPubKey(), AsgardKeygen, common.BlockHeight(helper.ctx), blame.Blame{}, common.Chains{common.RuneAsset().Chain}, GetRandomNodeAccount(NodeActive).NodeAddress)
			},
			runner: func(handler TssHandler, msg cosmos.Msg, helper tssHandlerTestHelper) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, ver, helper.constAccessor)
			},
			expectedResult: se.ErrUnauthorized,
		},
		{
			name: "empty signer should return an error",
			messageCreator: func(helper tssHandlerTestHelper) cosmos.Msg {
				return NewMsgTssPool(helper.members, GetRandomPubKey(), AsgardKeygen, common.BlockHeight(helper.ctx), blame.Blame{}, common.Chains{common.RuneAsset().Chain}, cosmos.AccAddress{})
			},
			runner: func(handler TssHandler, msg cosmos.Msg, helper tssHandlerTestHelper) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, ver, helper.constAccessor)
			},
			expectedResult: se.ErrInvalidAddress,
		},
		{
			name: "empty id should return an error",
			messageCreator: func(helper tssHandlerTestHelper) cosmos.Msg {
				tssMsg := NewMsgTssPool(helper.members, GetRandomPubKey(), AsgardKeygen, common.BlockHeight(helper.ctx), blame.Blame{}, common.Chains{common.RuneAsset().Chain}, helper.signer)
				tssMsg.ID = ""
				return tssMsg
			},
			runner: func(handler TssHandler, msg cosmos.Msg, helper tssHandlerTestHelper) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, ver, helper.constAccessor)
			},
			expectedResult: se.ErrUnknownRequest,
		},
		{
			name: "empty member pubkeys should return an error",
			messageCreator: func(helper tssHandlerTestHelper) cosmos.Msg {
				tssMsg := NewMsgTssPool(common.PubKeys{}, GetRandomPubKey(), AsgardKeygen, common.BlockHeight(helper.ctx), blame.Blame{}, common.Chains{common.RuneAsset().Chain}, helper.signer)
				return tssMsg
			},
			runner: func(handler TssHandler, msg cosmos.Msg, helper tssHandlerTestHelper) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, ver, helper.constAccessor)
			},
			expectedResult: se.ErrUnknownRequest,
		},
		{
			name: "less than two member pubkeys should return an error",
			messageCreator: func(helper tssHandlerTestHelper) cosmos.Msg {
				tssMsg := NewMsgTssPool(common.PubKeys{GetRandomPubKey()}, GetRandomPubKey(), AsgardKeygen, common.BlockHeight(helper.ctx), blame.Blame{}, common.Chains{common.RuneAsset().Chain}, helper.signer)
				return tssMsg
			},
			runner: func(handler TssHandler, msg cosmos.Msg, helper tssHandlerTestHelper) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, ver, helper.constAccessor)
			},
			expectedResult: se.ErrUnknownRequest,
		},
		{
			name: "there are empty pubkeys in member pubkey should return an error",
			messageCreator: func(helper tssHandlerTestHelper) cosmos.Msg {
				tssMsg := NewMsgTssPool(common.PubKeys{GetRandomPubKey(), GetRandomPubKey(), common.EmptyPubKey}, GetRandomPubKey(), AsgardKeygen, common.BlockHeight(helper.ctx), blame.Blame{}, common.Chains{common.RuneAsset().Chain}, helper.signer)
				return tssMsg
			},
			runner: func(handler TssHandler, msg cosmos.Msg, helper tssHandlerTestHelper) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, ver, helper.constAccessor)
			},
			expectedResult: se.ErrUnknownRequest,
		},
		{
			name: "success while pool pub key is empty should return error",
			messageCreator: func(helper tssHandlerTestHelper) cosmos.Msg {
				tssMsg := NewMsgTssPool(helper.members, common.EmptyPubKey, AsgardKeygen, common.BlockHeight(helper.ctx), blame.Blame{}, common.Chains{common.RuneAsset().Chain}, helper.signer)
				return tssMsg
			},
			runner: func(handler TssHandler, msg cosmos.Msg, helper tssHandlerTestHelper) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, ver, helper.constAccessor)
			},
			expectedResult: se.ErrUnknownRequest,
		},
		{
			name: "invalid pool pub key should return error",
			messageCreator: func(helper tssHandlerTestHelper) cosmos.Msg {
				tssMsg := NewMsgTssPool(helper.members, "whatever", AsgardKeygen, common.BlockHeight(helper.ctx), blame.Blame{}, common.Chains{common.RuneAsset().Chain}, helper.signer)
				return tssMsg
			},
			runner: func(handler TssHandler, msg cosmos.Msg, helper tssHandlerTestHelper) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, ver, helper.constAccessor)
			},
			expectedResult: se.ErrUnknownRequest,
		},
		{
			name: "fail to get tss voter should return an error",
			messageCreator: func(helper tssHandlerTestHelper) cosmos.Msg {
				tssMsg := NewMsgTssPool(helper.members, GetRandomPubKey(), AsgardKeygen, common.BlockHeight(helper.ctx), blame.Blame{}, common.Chains{common.RuneAsset().Chain}, helper.signer)
				return tssMsg
			},
			runner: func(handler TssHandler, msg cosmos.Msg, helper tssHandlerTestHelper) (*cosmos.Result, error) {
				helper.keeper.errGetTssVoter = true
				return handler.Run(helper.ctx, msg, ver, helper.constAccessor)
			},
			expectedResult: kaboom,
		},
		{
			name: "fail to save vault should return an error",
			messageCreator: func(helper tssHandlerTestHelper) cosmos.Msg {
				tssMsg := NewMsgTssPool(helper.members, helper.poolPk, AsgardKeygen, common.BlockHeight(helper.ctx), blame.Blame{}, common.Chains{common.RuneAsset().Chain}, helper.signer)
				voter, err := helper.keeper.GetTssVoter(helper.ctx, tssMsg.ID)
				c.Assert(err, IsNil)
				for _, pk := range helper.members {
					addr, err := pk.GetThorAddress()
					c.Assert(err, IsNil)
					if addr.Equals(helper.signer) {
						continue
					}
					voter.Signers = append(voter.Signers, addr)
				}
				helper.keeper.SetTssVoter(helper.ctx, voter)
				return tssMsg
			},
			runner: func(handler TssHandler, msg cosmos.Msg, helper tssHandlerTestHelper) (*cosmos.Result, error) {
				helper.keeper.errFailSaveVault = true
				return handler.Run(helper.ctx, msg, ver, helper.constAccessor)
			},
			expectedResult: kaboom,
		},
		{
			name: "not having consensus should not perform any actions",
			messageCreator: func(helper tssHandlerTestHelper) cosmos.Msg {
				tssMsg := NewMsgTssPool(helper.members, GetRandomPubKey(), AsgardKeygen, common.BlockHeight(helper.ctx), blame.Blame{}, common.Chains{common.RuneAsset().Chain}, helper.signer)
				return tssMsg
			},
			runner: func(handler TssHandler, msg cosmos.Msg, helper tssHandlerTestHelper) (*cosmos.Result, error) {
				for i := 0; i < 8; i++ {
					na := GetRandomNodeAccount(NodeActive)
					_ = helper.keeper.SetNodeAccount(helper.ctx, na)
				}
				return handler.Run(helper.ctx, msg, ver, helper.constAccessor)
			},
			expectedResult: nil,
		},
		{
			name: "if signer already sign the voter, it should just return",
			messageCreator: func(helper tssHandlerTestHelper) cosmos.Msg {
				tssMsg := NewMsgTssPool(helper.members, GetRandomPubKey(), AsgardKeygen, common.BlockHeight(helper.ctx), blame.Blame{}, common.Chains{common.RuneAsset().Chain}, helper.signer)
				voter, _ := helper.keeper.Keeper.GetTssVoter(helper.ctx, tssMsg.ID)
				if voter.PoolPubKey.IsEmpty() {
					voter.PoolPubKey = tssMsg.PoolPubKey
					voter.PubKeys = tssMsg.PubKeys
				}
				voter.Sign(tssMsg.Signer, tssMsg.Chains)
				helper.keeper.Keeper.SetTssVoter(helper.ctx, voter)
				return tssMsg
			},
			runner: func(handler TssHandler, msg cosmos.Msg, helper tssHandlerTestHelper) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, ver, helper.constAccessor)
			},
			expectedResult: nil,
		},
		{
			name: "normal success",
			messageCreator: func(helper tssHandlerTestHelper) cosmos.Msg {
				tssMsg := NewMsgTssPool(helper.members, GetRandomPubKey(), AsgardKeygen, common.BlockHeight(helper.ctx), blame.Blame{}, common.Chains{common.RuneAsset().Chain}, helper.signer)
				return tssMsg
			},
			runner: func(handler TssHandler, msg cosmos.Msg, helper tssHandlerTestHelper) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, ver, helper.constAccessor)
			},
			expectedResult: nil,
		},
		{
			name: "fail to keygen with invalid blame node account address should return an error",
			messageCreator: func(helper tssHandlerTestHelper) cosmos.Msg {
				sort.SliceStable(helper.members, func(i, j int) bool {
					return helper.members[i].String() < helper.members[j].String()
				})
				b := blame.Blame{
					FailReason: "who knows",
					BlameNodes: []blame.Node{
						{Pubkey: "whatever"},
					},
				}
				tssMsg := NewMsgTssPool(helper.members, GetRandomPubKey(), AsgardKeygen, common.BlockHeight(helper.ctx), b, common.Chains{common.RuneAsset().Chain}, helper.signer)
				voter, err := helper.keeper.GetTssVoter(helper.ctx, tssMsg.ID)
				c.Assert(err, IsNil)
				for _, pk := range helper.members {
					addr, err := pk.GetThorAddress()
					c.Assert(err, IsNil)
					if addr.Equals(helper.signer) {
						continue
					}
					voter.Signers = append(voter.Signers, addr)
				}
				helper.keeper.SetTssVoter(helper.ctx, voter)
				return tssMsg
			},
			runner: func(handler TssHandler, msg cosmos.Msg, helper tssHandlerTestHelper) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg, ver, helper.constAccessor)
			},
			expectedResult: errInternal,
		},
		{
			name: "fail to keygen retry should be slashed",
			messageCreator: func(helper tssHandlerTestHelper) cosmos.Msg {
				thorAddr, _ := helper.members[3].GetThorAddress()
				na, _ := helper.keeper.GetNodeAccount(helper.ctx, thorAddr)
				na.UpdateStatus(NodeActive, common.BlockHeight(helper.ctx))
				_ = helper.keeper.SetNodeAccount(helper.ctx, na)
				b := blame.Blame{
					FailReason: "who knows",
					BlameNodes: []blame.Node{
						{
							Pubkey: helper.members[3].String(),
						},
					},
				}
				tssMsg := NewMsgTssPool(helper.members, GetRandomPubKey(), AsgardKeygen, common.BlockHeight(helper.ctx), b, common.Chains{common.RuneAsset().Chain}, helper.signer)
				voter, err := helper.keeper.GetTssVoter(helper.ctx, tssMsg.ID)
				c.Assert(err, IsNil)
				constAccessor := constants.GetConstantValues(helper.version)
				observeSlashPoints := constAccessor.GetInt64Value(constants.ObserveSlashPoints)
				for _, pk := range helper.members {
					addr, err := pk.GetThorAddress()
					c.Assert(err, IsNil)
					if addr.Equals(helper.signer) {
						continue
					}
					voter.Signers = append(voter.Signers, addr)
				}
				helper.mgr.Slasher().IncSlashPoints(helper.ctx, observeSlashPoints, voter.Signers...)
				helper.keeper.SetTssVoter(helper.ctx, voter)
				return tssMsg
			},
			runner: func(handler TssHandler, msg cosmos.Msg, helper tssHandlerTestHelper) (*cosmos.Result, error) {
				m, _ := msg.(MsgTssPool)
				voter, _ := helper.keeper.GetTssVoter(helper.ctx, m.ID)
				if voter.PoolPubKey.IsEmpty() {
					voter.PoolPubKey = m.PoolPubKey
					voter.PubKeys = m.PubKeys
				}
				addr, _ := helper.members[3].GetThorAddress()
				voter.Sign(addr, common.Chains{common.BNBChain})
				helper.keeper.SetTssVoter(helper.ctx, voter)
				return handler.Run(helper.ctx, msg, ver, helper.constAccessor)
			},
			validator: func(helper tssHandlerTestHelper, msg cosmos.Msg, result *cosmos.Result, c *C) {
				// make sure node get slashed
				pubKey := helper.members[3]
				na, err := helper.keeper.GetNodeAccountByPubKey(helper.ctx, pubKey)
				c.Assert(err, IsNil)
				slashPts, err := helper.keeper.GetNodeAccountSlashPoints(helper.ctx, na.NodeAddress)
				c.Assert(err, IsNil)
				c.Assert(slashPts > 0, Equals, true)
			},
			expectedResult: nil,
		},
		{
			name: "fail to keygen but can't get vault data should return an error",
			messageCreator: func(helper tssHandlerTestHelper) cosmos.Msg {
				b := blame.Blame{
					FailReason: "who knows",
					BlameNodes: []blame.Node{
						{Pubkey: helper.members[3].String()},
					},
				}
				tssMsg := NewMsgTssPool(helper.members, GetRandomPubKey(), AsgardKeygen, common.BlockHeight(helper.ctx), b, common.Chains{common.RuneAsset().Chain}, helper.signer)
				voter, err := helper.keeper.GetTssVoter(helper.ctx, tssMsg.ID)
				c.Assert(err, IsNil)
				for _, pk := range helper.members {
					addr, err := pk.GetThorAddress()
					c.Assert(err, IsNil)
					if addr.Equals(helper.signer) {
						continue
					}
					voter.Signers = append(voter.Signers, addr)
				}
				helper.keeper.SetTssVoter(helper.ctx, voter)
				return tssMsg
			},
			runner: func(handler TssHandler, msg cosmos.Msg, helper tssHandlerTestHelper) (*cosmos.Result, error) {
				helper.keeper.errFailGetVaultData = true
				return handler.Run(helper.ctx, msg, ver, helper.constAccessor)
			},
			expectedResult: kaboom,
		},
		{
			name: "fail to keygen retry and none active account should be slashed with bond",
			messageCreator: func(helper tssHandlerTestHelper) cosmos.Msg {
				b := blame.Blame{
					FailReason: "who knows",
					BlameNodes: []blame.Node{
						{Pubkey: helper.members[3].String()},
					},
				}
				tssMsg := NewMsgTssPool(helper.members, GetRandomPubKey(), AsgardKeygen, common.BlockHeight(helper.ctx), b, common.Chains{common.RuneAsset().Chain}, helper.signer)
				voter, err := helper.keeper.GetTssVoter(helper.ctx, tssMsg.ID)
				c.Assert(err, IsNil)
				for _, pk := range helper.members {
					addr, err := pk.GetThorAddress()
					c.Assert(err, IsNil)
					if addr.Equals(helper.signer) {
						continue
					}
					voter.Signers = append(voter.Signers, addr)
				}
				helper.keeper.SetTssVoter(helper.ctx, voter)
				return tssMsg
			},
			runner: func(handler TssHandler, msg cosmos.Msg, helper tssHandlerTestHelper) (*cosmos.Result, error) {
				vd := VaultData{
					BondRewardRune: cosmos.NewUint(5000 * common.One),
					TotalBondUnits: cosmos.NewUint(10000),
				}
				_ = helper.keeper.SetVaultData(helper.ctx, vd)
				return handler.Run(helper.ctx, msg, ver, helper.constAccessor)
			},
			validator: func(helper tssHandlerTestHelper, msg cosmos.Msg, result *cosmos.Result, c *C) {
				// make sure node get slashed
				pubKey := helper.members[3]
				na, err := helper.keeper.GetNodeAccountByPubKey(helper.ctx, pubKey)
				c.Assert(err, IsNil)
				c.Assert(na.Bond.Equal(cosmos.ZeroUint()), Equals, true)
				jail, err := helper.keeper.GetNodeAccountJail(helper.ctx, na.NodeAddress)
				c.Assert(err, IsNil)
				c.Check(jail.ReleaseHeight > 0, Equals, true)
			},
			expectedResult: nil,
		},
		{
			name: "fail to keygen and none active account, fail to set vault data should return an error",
			messageCreator: func(helper tssHandlerTestHelper) cosmos.Msg {
				b := blame.Blame{
					FailReason: "who knows",
					BlameNodes: []blame.Node{
						{Pubkey: helper.members[3].String()},
					},
				}
				tssMsg := NewMsgTssPool(helper.members, GetRandomPubKey(), AsgardKeygen, common.BlockHeight(helper.ctx), b, common.Chains{common.RuneAsset().Chain}, helper.signer)
				voter, err := helper.keeper.GetTssVoter(helper.ctx, tssMsg.ID)
				c.Assert(err, IsNil)
				for _, pk := range helper.members {
					addr, err := pk.GetThorAddress()
					c.Assert(err, IsNil)
					if addr.Equals(helper.signer) {
						continue
					}
					voter.Signers = append(voter.Signers, addr)
				}
				helper.keeper.SetTssVoter(helper.ctx, voter)
				return tssMsg
			},
			runner: func(handler TssHandler, msg cosmos.Msg, helper tssHandlerTestHelper) (*cosmos.Result, error) {
				vd := VaultData{
					BondRewardRune: cosmos.NewUint(5000 * common.One),
					TotalBondUnits: cosmos.NewUint(10000),
				}
				_ = helper.keeper.SetVaultData(helper.ctx, vd)
				helper.keeper.errFailSetVaultData = true
				return handler.Run(helper.ctx, msg, ver, helper.constAccessor)
			},
			expectedResult: nil,
		},
		{
			name: "fail to keygen and fail to get node account should return an error",
			messageCreator: func(helper tssHandlerTestHelper) cosmos.Msg {
				b := blame.Blame{
					FailReason: "who knows",
					BlameNodes: []blame.Node{
						{Pubkey: helper.members[3].String()},
					},
				}
				tssMsg := NewMsgTssPool(helper.members, GetRandomPubKey(), AsgardKeygen, common.BlockHeight(helper.ctx), b, common.Chains{common.RuneAsset().Chain}, helper.signer)
				voter, err := helper.keeper.GetTssVoter(helper.ctx, tssMsg.ID)
				c.Assert(err, IsNil)
				for _, pk := range helper.members {
					addr, err := pk.GetThorAddress()
					c.Assert(err, IsNil)
					if addr.Equals(helper.signer) {
						continue
					}
					voter.Signers = append(voter.Signers, addr)
				}
				helper.keeper.SetTssVoter(helper.ctx, voter)
				return tssMsg
			},
			runner: func(handler TssHandler, msg cosmos.Msg, helper tssHandlerTestHelper) (*cosmos.Result, error) {
				helper.keeper.errFailGetNodeAccount = true
				return handler.Run(helper.ctx, msg, ver, helper.constAccessor)
			},
			expectedResult: kaboom,
		},
		{
			name: "fail to keygen and fail to set node account should return an error",
			messageCreator: func(helper tssHandlerTestHelper) cosmos.Msg {
				b := blame.Blame{
					FailReason: "who knows",
					BlameNodes: []blame.Node{
						{Pubkey: helper.members[3].String()},
					},
				}
				tssMsg := NewMsgTssPool(helper.members, GetRandomPubKey(), AsgardKeygen, common.BlockHeight(helper.ctx), b, common.Chains{common.RuneAsset().Chain}, helper.signer)
				voter, err := helper.keeper.GetTssVoter(helper.ctx, tssMsg.ID)
				c.Assert(err, IsNil)
				for _, pk := range helper.members {
					addr, err := pk.GetThorAddress()
					c.Assert(err, IsNil)
					if addr.Equals(helper.signer) {
						continue
					}
					voter.Signers = append(voter.Signers, addr)
				}
				helper.keeper.SetTssVoter(helper.ctx, voter)
				return tssMsg
			},
			runner: func(handler TssHandler, msg cosmos.Msg, helper tssHandlerTestHelper) (*cosmos.Result, error) {
				helper.keeper.errFailSetNodeAccount = true
				return handler.Run(helper.ctx, msg, ver, helper.constAccessor)
			},
			expectedResult: kaboom,
		},
	}

	for _, tc := range testCases {
		helper := newTssHandlerTestHelper(c, ver)
		handler := NewTssHandler(helper.keeper, NewDummyMgr())
		msg := tc.messageCreator(helper)
		result, err := tc.runner(handler, msg, helper)
		if tc.expectedResult == nil {
			c.Assert(err, IsNil)
		} else {
			c.Assert(errors.Is(err, tc.expectedResult), Equals, true, Commentf("name:%s, %s", tc.name, err))
		}
		if tc.validator != nil {
			tc.validator(helper, msg, result, c)
		}
	}
}
