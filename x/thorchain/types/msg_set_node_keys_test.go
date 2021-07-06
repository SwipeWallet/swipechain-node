package types

import (
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

type MsgSetNodeKeysSuite struct{}

var _ = Suite(&MsgSetNodeKeysSuite{})

func (MsgSetNodeKeysSuite) TestMsgSetNodeKeys(c *C) {
	acc1 := GetRandomBech32Addr()
	c.Assert(acc1.Empty(), Equals, false)
	consensPubKey := GetRandomBech32ConsensusPubKey()
	pubKeys := common.PubKeySet{
		Secp256k1: GetRandomPubKey(),
		Ed25519:   GetRandomPubKey(),
	}
	msgSetNodeKeys := NewMsgSetNodeKeys(pubKeys, consensPubKey, acc1)
	c.Assert(msgSetNodeKeys.Route(), Equals, RouterKey)
	c.Assert(msgSetNodeKeys.Type(), Equals, "set_node_keys")
	c.Assert(msgSetNodeKeys.ValidateBasic(), IsNil)
	c.Assert(len(msgSetNodeKeys.GetSignBytes()) > 0, Equals, true)
	c.Assert(msgSetNodeKeys.GetSigners(), NotNil)
	c.Assert(msgSetNodeKeys.GetSigners()[0].String(), Equals, acc1.String())
	msgUpdateNodeAccount1 := NewMsgSetNodeKeys(pubKeys, "", acc1)
	c.Assert(msgUpdateNodeAccount1.ValidateBasic(), NotNil)

	msgUpdateNodeAccount2 := NewMsgSetNodeKeys(pubKeys, consensPubKey, cosmos.AccAddress{})
	c.Assert(msgUpdateNodeAccount2.ValidateBasic(), NotNil)

	emptyPubKeySet := NewMsgSetNodeKeys(common.PubKeySet{}, consensPubKey, acc1)
	c.Assert(emptyPubKeySet.ValidateBasic(), NotNil)
}
