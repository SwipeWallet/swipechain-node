package types

import (
	"errors"

	se "github.com/cosmos/cosmos-sdk/types/errors"
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

type MsgYggdrasilSuite struct{}

var _ = Suite(&MsgYggdrasilSuite{})

func (s *MsgYggdrasilSuite) TestMsgYggdrasil(c *C) {
	tx := GetRandomTx()
	pk := GetRandomPubKey()
	coins := common.Coins{
		common.NewCoin(common.BNBAsset, cosmos.NewUint(500*common.One)),
		common.NewCoin(common.BTCAsset, cosmos.NewUint(400*common.One)),
	}
	signer := GetRandomBech32Addr()
	msg := NewMsgYggdrasil(tx, pk, 12, true, coins, signer)
	EnsureMsgBasicCorrect(msg, c)
	c.Check(msg.PubKey.Equals(pk), Equals, true)
	c.Check(msg.AddFunds, Equals, true)
	c.Check(msg.Coins, HasLen, len(coins))
	c.Check(msg.Tx.Equals(tx), Equals, true)
	c.Check(msg.Signer.Equals(signer), Equals, true)
	c.Check(msg.BlockHeight, Equals, int64(12))
	c.Check(msg.Type(), Equals, "set_yggdrasil")

	msg1 := NewMsgYggdrasil(tx, pk, 12, true, coins, cosmos.AccAddress{})
	err1 := msg1.ValidateBasic()
	c.Check(err1, NotNil)
	c.Check(errors.Is(err1, se.ErrInvalidAddress), Equals, true)

	msg2 := NewMsgYggdrasil(tx, "", 12, true, coins, signer)
	err2 := msg2.ValidateBasic()
	c.Check(err2, NotNil)
	c.Check(errors.Is(err2, se.ErrUnknownRequest), Equals, true)

	msg3 := NewMsgYggdrasil(tx, pk, -12, true, coins, signer)
	err3 := msg3.ValidateBasic()
	c.Check(err3, NotNil)
	c.Check(errors.Is(err3, se.ErrUnknownRequest), Equals, true)

	msg4 := NewMsgYggdrasil(common.Tx{}, pk, 12, true, coins, signer)
	err4 := msg4.ValidateBasic()
	c.Check(err4, NotNil)
	c.Check(errors.Is(err4, se.ErrUnknownRequest), Equals, true)

	msg5 := NewMsgYggdrasil(GetRandomTx(), pk, 12, true, common.Coins{
		common.NewCoin(common.EmptyAsset, cosmos.ZeroUint()),
	}, signer)
	err5 := msg5.ValidateBasic()
	c.Check(err5, NotNil)
	c.Check(errors.Is(err5, se.ErrInvalidCoins), Equals, true)
}
