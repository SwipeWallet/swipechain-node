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

type HandlerBondSuite struct{}

type TestBondKeeper struct {
	keeper.KVStoreDummy
	activeNodeAccount   NodeAccount
	failGetNodeAccount  NodeAccount
	notEmptyNodeAccount NodeAccount
}

func (k *TestBondKeeper) GetNodeAccount(_ cosmos.Context, addr cosmos.AccAddress) (NodeAccount, error) {
	if k.activeNodeAccount.NodeAddress.Equals(addr) {
		return k.activeNodeAccount, nil
	}
	if k.failGetNodeAccount.NodeAddress.Equals(addr) {
		return NodeAccount{}, fmt.Errorf("you asked for this error")
	}
	if k.notEmptyNodeAccount.NodeAddress.Equals(addr) {
		return k.notEmptyNodeAccount, nil
	}
	return NodeAccount{}, nil
}

var _ = Suite(&HandlerBondSuite{})

func (HandlerBondSuite) TestBondHandler_Run(c *C) {
	ctx, k1 := setupKeeperForTest(c)
	activeNodeAccount := GetRandomNodeAccount(NodeActive)
	k := &TestBondKeeper{
		activeNodeAccount:   activeNodeAccount,
		failGetNodeAccount:  GetRandomNodeAccount(NodeActive),
		notEmptyNodeAccount: GetRandomNodeAccount(NodeActive),
	}
	// happy path
	c.Assert(k1.SetNodeAccount(ctx, activeNodeAccount), IsNil)
	handler := NewBondHandler(k1, NewDummyMgr())
	ver := constants.SWVersion
	constAccessor := constants.GetConstantValues(ver)
	minimumBondInRune := constAccessor.GetInt64Value(constants.MinimumBondInRune)
	txIn := common.NewTx(
		GetRandomTxHash(),
		GetRandomBNBAddress(),
		GetRandomBNBAddress(),
		common.Coins{
			common.NewCoin(common.RuneAsset(), cosmos.NewUint(uint64(minimumBondInRune))),
		},
		BNBGasFeeSingleton,
		"bond",
	)
	msg := NewMsgBond(txIn, GetRandomNodeAccount(NodeStandby).NodeAddress, cosmos.NewUint(uint64(minimumBondInRune)), GetRandomBNBAddress(), activeNodeAccount.NodeAddress)
	_, err := handler.Run(ctx, msg, ver, constAccessor)
	c.Assert(err, IsNil)

	// invalid version
	handler = NewBondHandler(k, NewDummyMgr())
	ver = semver.Version{}
	_, err = handler.Run(ctx, msg, ver, constAccessor)
	c.Assert(errors.Is(err, errBadVersion), Equals, true)

	// simulate fail to get node account
	ver = constants.SWVersion
	msg = NewMsgBond(txIn, k.failGetNodeAccount.NodeAddress, cosmos.NewUint(uint64(minimumBondInRune)), GetRandomBNBAddress(), activeNodeAccount.NodeAddress)
	_, err = handler.Run(ctx, msg, ver, constAccessor)
	c.Assert(errors.Is(err, errInternal), Equals, true)

	msg = NewMsgBond(txIn, k.notEmptyNodeAccount.NodeAddress, cosmos.NewUint(uint64(minimumBondInRune)), GetRandomBNBAddress(), activeNodeAccount.NodeAddress)
	_, err = handler.Run(ctx, msg, ver, constAccessor)
	c.Assert(errors.Is(err, errInternal), Equals, true)
}

func (HandlerBondSuite) TestBondHandlerFailValidation(c *C) {
	ctx, k := setupKeeperForTest(c)
	activeNodeAccount := GetRandomNodeAccount(NodeActive)
	c.Assert(k.SetNodeAccount(ctx, activeNodeAccount), IsNil)
	handler := NewBondHandler(k, NewDummyMgr())
	ver := semver.MustParse("0.8.0")
	constAccessor := constants.GetConstantValues(ver)
	minimumBondInRune := constAccessor.GetInt64Value(constants.MinimumBondInRune)
	txIn := common.NewTx(
		GetRandomTxHash(),
		GetRandomBNBAddress(),
		GetRandomBNBAddress(),
		common.Coins{
			common.NewCoin(common.RuneAsset(), cosmos.NewUint(uint64(minimumBondInRune))),
		},
		BNBGasFeeSingleton,
		"apply",
	)
	txInNoTxID := txIn
	txInNoTxID.ID = ""
	testCases := []struct {
		name        string
		msg         MsgBond
		expectedErr error
	}{
		{
			name:        "empty node address",
			msg:         NewMsgBond(txIn, cosmos.AccAddress{}, cosmos.NewUint(uint64(minimumBondInRune)), GetRandomBNBAddress(), activeNodeAccount.NodeAddress),
			expectedErr: se.ErrInvalidAddress,
		},
		{
			name:        "zero bond",
			msg:         NewMsgBond(txIn, GetRandomNodeAccount(NodeStandby).NodeAddress, cosmos.ZeroUint(), GetRandomBNBAddress(), activeNodeAccount.NodeAddress),
			expectedErr: se.ErrUnknownRequest,
		},
		{
			name:        "empty bond address",
			msg:         NewMsgBond(txIn, GetRandomNodeAccount(NodeStandby).NodeAddress, cosmos.NewUint(uint64(minimumBondInRune)), common.Address(""), activeNodeAccount.NodeAddress),
			expectedErr: se.ErrInvalidAddress,
		},
		{
			name:        "empty request hash",
			msg:         NewMsgBond(txInNoTxID, GetRandomNodeAccount(NodeStandby).NodeAddress, cosmos.NewUint(uint64(minimumBondInRune)), GetRandomBNBAddress(), activeNodeAccount.NodeAddress),
			expectedErr: se.ErrUnknownRequest,
		},
		{
			name:        "empty signer",
			msg:         NewMsgBond(txIn, GetRandomNodeAccount(NodeStandby).NodeAddress, cosmos.NewUint(uint64(minimumBondInRune)), GetRandomBNBAddress(), cosmos.AccAddress{}),
			expectedErr: se.ErrInvalidAddress,
		},
		{
			name:        "msg not signed by active account",
			msg:         NewMsgBond(txIn, GetRandomNodeAccount(NodeStandby).NodeAddress, cosmos.NewUint(uint64(minimumBondInRune)), GetRandomBNBAddress(), GetRandomNodeAccount(NodeStandby).NodeAddress),
			expectedErr: se.ErrUnauthorized,
		},
	}
	for _, item := range testCases {
		c.Log(item.name)
		_, err := handler.Run(ctx, item.msg, ver, constAccessor)
		c.Check(errors.Is(err, item.expectedErr), Equals, true, Commentf("name: %s, %s != %s", item.name, item.expectedErr, err))
	}
}
