package types

import (
	"errors"

	se "github.com/cosmos/cosmos-sdk/types/errors"
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/tss/go-tss/blame"

	"gitlab.com/thorchain/thornode/common"
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
)

type MsgTssKeysignFailSuite struct{}

var _ = Suite(&MsgTssKeysignFailSuite{})

func (s MsgTssKeysignFailSuite) TestMsgTssKeysignFail(c *C) {
	b := blame.Blame{
		FailReason: "fail to TSS sign",
		BlameNodes: []blame.Node{
			{Pubkey: GetRandomPubKey().String()},
			{Pubkey: GetRandomPubKey().String()},
		},
	}
	coins := common.Coins{
		common.NewCoin(common.RuneAsset(), cosmos.NewUint(100)),
	}
	msg := NewMsgTssKeysignFail(1, b, "hello", coins, GetRandomBech32Addr(), GetRandomPubKey())
	c.Check(msg.Type(), Equals, "set_tss_keysign_fail")
	EnsureMsgBasicCorrect(msg, c)
	c.Check(NewMsgTssKeysignFail(1, blame.Blame{}, "hello", coins, GetRandomBech32Addr(), GetRandomPubKey()), NotNil)
	c.Check(NewMsgTssKeysignFail(1, b, "", coins, GetRandomBech32Addr(), GetRandomPubKey()), NotNil)
	c.Check(NewMsgTssKeysignFail(1, b, "hello", common.Coins{}, GetRandomBech32Addr(), GetRandomPubKey()), NotNil)
	c.Check(NewMsgTssKeysignFail(1, b, "hello", common.Coins{
		common.NewCoin(common.BNBAsset, cosmos.NewUint(100)),
		common.NewCoin(common.EmptyAsset, cosmos.ZeroUint()),
	}, GetRandomBech32Addr(), GetRandomPubKey()), NotNil)
	c.Check(NewMsgTssKeysignFail(1, b, "hello", coins, cosmos.AccAddress{}, GetRandomPubKey()), NotNil)

	msg2 := NewMsgTssKeysignFail(1, b, "hello", coins, cosmos.AccAddress{}, GetRandomPubKey())
	err2 := msg2.ValidateBasic()
	c.Check(err2, NotNil)
	c.Check(errors.Is(err2, se.ErrInvalidAddress), Equals, true)

	msg3 := NewMsgTssKeysignFail(1, b, "hello", coins, GetRandomBech32Addr(), GetRandomPubKey())
	msg3.ID = ""
	err3 := msg3.ValidateBasic()
	c.Check(err3, NotNil)
	c.Check(errors.Is(err3, se.ErrUnknownRequest), Equals, true)

	msg4 := NewMsgTssKeysignFail(1, blame.Blame{}, "hello", coins, GetRandomBech32Addr(), GetRandomPubKey())
	err4 := msg4.ValidateBasic()
	c.Check(err4, NotNil)
	c.Check(errors.Is(err4, se.ErrUnknownRequest), Equals, true)

	msg4.Coins = append(msg4.Coins, common.NewCoin(common.EmptyAsset, cosmos.ZeroUint()))
	err4 = msg4.ValidateBasic()
	c.Check(err4, NotNil)
	c.Check(errors.Is(err4, se.ErrInvalidCoins), Equals, true)

	msg5 := NewMsgTssKeysignFail(1, b, "hello", common.Coins{}, GetRandomBech32Addr(), GetRandomPubKey())
	err5 := msg5.ValidateBasic()
	c.Check(err5, NotNil)
	c.Check(errors.Is(err5, se.ErrUnknownRequest), Equals, true)

	msg6 := NewMsgTssKeysignFail(1, b, "hello", coins, GetRandomBech32Addr(), common.EmptyPubKey)
	err6 := msg6.ValidateBasic()
	c.Check(err6, NotNil)
	c.Check(errors.Is(err6, se.ErrUnknownRequest), Equals, true)
}
