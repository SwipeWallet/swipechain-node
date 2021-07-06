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

type HandlerAddSuite struct{}

var _ = Suite(&HandlerAddSuite{})

type HandlerAddTestHelper struct {
	keeper.Keeper
	failToGetPool  bool
	failToSavePool bool
}

func NewHandlerAddTestHelper(k keeper.Keeper) *HandlerAddTestHelper {
	return &HandlerAddTestHelper{
		Keeper: k,
	}
}

func (h *HandlerAddTestHelper) GetPool(ctx cosmos.Context, asset common.Asset) (Pool, error) {
	if h.failToGetPool {
		return NewPool(), kaboom
	}
	return h.Keeper.GetPool(ctx, asset)
}

func (h *HandlerAddTestHelper) SetPool(ctx cosmos.Context, p Pool) error {
	if h.failToSavePool {
		return kaboom
	}
	return h.Keeper.SetPool(ctx, p)
}

func (HandlerAddSuite) TestAdd(c *C) {
	w := getHandlerTestWrapper(c, 1, true, true)
	// happy path
	prePool, err := w.keeper.GetPool(w.ctx, common.BNBAsset)
	c.Assert(err, IsNil)
	mgr := NewDummyMgr()
	addHandler := NewAddHandler(w.keeper, mgr)
	msg := NewMsgAdd(GetRandomTx(), common.BNBAsset, cosmos.NewUint(common.One*5), cosmos.NewUint(common.One*5), w.activeNodeAccount.NodeAddress)
	ver := constants.SWVersion
	constAccessor := constants.GetConstantValues(ver)
	_, err = addHandler.Run(w.ctx, msg, ver, constAccessor)
	c.Assert(err, IsNil)
	afterPool, err := w.keeper.GetPool(w.ctx, common.BNBAsset)
	c.Assert(err, IsNil)
	c.Assert(afterPool.BalanceRune.String(), Equals, prePool.BalanceRune.Add(msg.RuneAmount).String())
	c.Assert(afterPool.BalanceAsset.String(), Equals, prePool.BalanceAsset.Add(msg.AssetAmount).String())

	// invalid version
	ver = semver.Version{}
	_, err = addHandler.Run(w.ctx, msg, ver, constAccessor)
	c.Check(errors.Is(err, errBadVersion), Equals, true)
	msgBan := NewMsgBan(GetRandomBech32Addr(), w.activeNodeAccount.NodeAddress)
	result, err := addHandler.Run(w.ctx, msgBan, semver.MustParse("0.1.0"), constAccessor)
	c.Check(err, NotNil)
	c.Check(errors.Is(err, errInvalidMessage), Equals, true)
	c.Check(result, IsNil)

	testKeeper := NewHandlerAddTestHelper(w.keeper)
	testKeeper.failToGetPool = true
	addHandler1 := NewAddHandler(testKeeper, mgr)
	result, err = addHandler1.Run(w.ctx, msg, semver.MustParse("0.1.0"), constAccessor)
	c.Check(err, NotNil)
	c.Check(errors.Is(err, errInternal), Equals, true)
	c.Check(result, IsNil)

	testKeeper = NewHandlerAddTestHelper(w.keeper)
	testKeeper.failToSavePool = true
	addHandler2 := NewAddHandler(testKeeper, mgr)
	result, err = addHandler2.Run(w.ctx, msg, semver.MustParse("0.1.0"), constAccessor)
	c.Check(err, NotNil)
	c.Check(errors.Is(err, errInternal), Equals, true)
	c.Check(result, IsNil)
}

func (HandlerAddSuite) TestHandleMsgAddValidation(c *C) {
	w := getHandlerTestWrapper(c, 1, true, false)
	testCases := []struct {
		name        string
		msg         MsgAdd
		expectedErr error
	}{
		{
			name:        "invalid signer address should fail",
			msg:         NewMsgAdd(GetRandomTx(), common.BNBAsset, cosmos.NewUint(common.One*5), cosmos.NewUint(common.One*5), cosmos.AccAddress{}),
			expectedErr: se.ErrInvalidAddress,
		},
		{
			name:        "empty asset should fail",
			msg:         NewMsgAdd(GetRandomTx(), common.Asset{}, cosmos.NewUint(common.One*5), cosmos.NewUint(common.One*5), w.activeNodeAccount.NodeAddress),
			expectedErr: se.ErrUnknownRequest,
		},
		{
			name:        "pool doesn't exist should fail",
			msg:         NewMsgAdd(GetRandomTx(), common.BNBAsset, cosmos.NewUint(common.One*5), cosmos.NewUint(common.One*5), w.activeNodeAccount.NodeAddress),
			expectedErr: se.ErrUnknownRequest,
		},
	}

	addHandler := NewAddHandler(w.keeper, NewDummyMgr())
	ver := constants.SWVersion
	cosntAccessor := constants.GetConstantValues(ver)
	for _, item := range testCases {
		_, err := addHandler.Run(w.ctx, item.msg, ver, cosntAccessor)
		c.Check(errors.Is(err, item.expectedErr), Equals, true, Commentf("name:%s", item.name))
	}
}
