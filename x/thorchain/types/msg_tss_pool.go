package types

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"gitlab.com/thorchain/tss/go-tss/blame"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

// MsgTssPool defines a MsgTssPool message
type MsgTssPool struct {
	ID         string            `json:"id"`
	PoolPubKey common.PubKey     `json:"pool_pub_key"`
	KeygenType KeygenType        `json:"keygen_type"`
	PubKeys    common.PubKeys    `json:"pubkeys"`
	Height     int64             `json:"height"`
	Blame      blame.Blame       `json:"blame"`
	Chains     common.Chains     `json:"chains"`
	Signer     cosmos.AccAddress `json:"signer"`
}

// NewMsgTssPool is a constructor function for MsgTssPool
func NewMsgTssPool(pks common.PubKeys, poolpk common.PubKey, KeygenType KeygenType, height int64, bl blame.Blame, chains common.Chains, signer cosmos.AccAddress) MsgTssPool {
	return MsgTssPool{
		ID:         getTssID(pks, poolpk, height, bl),
		PubKeys:    pks,
		PoolPubKey: poolpk,
		Height:     height,
		KeygenType: KeygenType,
		Blame:      bl,
		Chains:     chains,
		Signer:     signer,
	}
}

// getTssID
func getTssID(members common.PubKeys, poolPk common.PubKey, height int64, bl blame.Blame) string {
	// ensure input pubkeys list is deterministically sorted
	sort.SliceStable(members, func(i, j int) bool {
		return members[i].String() < members[j].String()
	})

	pubkeys := make([]string, len(bl.BlameNodes))
	for i, node := range bl.BlameNodes {
		pubkeys[i] = node.Pubkey
	}
	sort.SliceStable(pubkeys, func(i, j int) bool {
		return pubkeys[i] < pubkeys[j]
	})

	sb := strings.Builder{}
	for _, item := range members {
		sb.WriteString("m:" + item.String())
	}
	for _, item := range pubkeys {
		sb.WriteString("p:" + item)
	}
	sb.WriteString(poolPk.String())
	sb.WriteString(fmt.Sprintf("%d", height))
	hash := sha256.New()
	return hex.EncodeToString(hash.Sum([]byte(sb.String())))
}

// Route should return the route key of the module
func (msg MsgTssPool) Route() string { return RouterKey }

// Type should return the action
func (msg MsgTssPool) Type() string { return "set_tss_pool" }

// ValidateBasic runs stateless checks on the message
func (msg MsgTssPool) ValidateBasic() error {
	if msg.Signer.Empty() {
		return cosmos.ErrInvalidAddress(msg.Signer.String())
	}
	if len(msg.ID) == 0 {
		return cosmos.ErrUnknownRequest("ID cannot be blank")
	}
	if len(msg.PubKeys) < 2 {
		return cosmos.ErrUnknownRequest("Must have at least 2 pub keys")
	}
	if len(msg.PubKeys) > 100 {
		return cosmos.ErrUnknownRequest("Must have no more then 100 pub keys")
	}
	for _, pk := range msg.PubKeys {
		if pk.IsEmpty() {
			return cosmos.ErrUnknownRequest("Pubkey cannot be empty")
		}
	}
	// PoolPubKey can't be empty only when keygen success
	if msg.IsSuccess() {
		if msg.PoolPubKey.IsEmpty() {
			return cosmos.ErrUnknownRequest("Pool pubkey cannot be empty")
		}
	}
	// ensure pool pubkey is a valid bech32 pubkey
	if _, err := common.NewPubKey(msg.PoolPubKey.String()); err != nil {
		return cosmos.ErrUnknownRequest(err.Error())
	}
	if !msg.Chains.Has(common.RuneAsset().Chain) {
		return cosmos.ErrUnknownRequest("must support rune asset chain")
	}
	if len(msg.Chains) != len(msg.Chains.Distinct()) {
		return cosmos.ErrUnknownRequest("cannot have duplicate chains")
	}
	return nil
}

// IsSuccess when blame is empty , then treat it as success
func (msg MsgTssPool) IsSuccess() bool {
	return msg.Blame.IsEmpty()
}

// GetSignBytes encodes the message for signing
func (msg MsgTssPool) GetSignBytes() []byte {
	return cosmos.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgTssPool) GetSigners() []cosmos.AccAddress {
	return []cosmos.AccAddress{msg.Signer}
}
