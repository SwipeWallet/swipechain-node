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

// MsgTssKeysignFail means TSS keysign failed
type MsgTssKeysignFail struct {
	ID     string            `json:"id"`
	Height int64             `json:"height"`
	Blame  blame.Blame       `json:"blame"`
	Memo   string            `json:"memo"`
	Coins  common.Coins      `json:"coins"`
	Signer cosmos.AccAddress `json:"signer"`
	PubKey common.PubKey     `json:"pub_key"`
}

// NewMsgTssKeysignFail create a new instance of MsgTssKeysignFail message
func NewMsgTssKeysignFail(height int64, blame blame.Blame, memo string, coins common.Coins, signer cosmos.AccAddress, pubKey common.PubKey) MsgTssKeysignFail {
	return MsgTssKeysignFail{
		ID:     getMsgTssKeysignFailID(blame.BlameNodes, height, memo, coins, pubKey),
		Height: height,
		Blame:  blame,
		Memo:   memo,
		Coins:  coins,
		Signer: signer,
		PubKey: pubKey,
	}
}

// getTssKeysignFailID this method will use all the members that caused the tss
// keysign failure , as well as the block height of the txout item to generate
// a hash, given that , if the same party keep failing the same txout item ,
// then we will only slash it once.
func getMsgTssKeysignFailID(members []blame.Node, height int64, memo string, coins common.Coins, pubKey common.PubKey) string {
	// ensure input pubkeys list is deterministically sorted
	sort.SliceStable(members, func(i, j int) bool {
		return members[i].Pubkey < members[j].Pubkey
	})
	sb := strings.Builder{}
	for _, item := range members {
		sb.WriteString(item.Pubkey)
	}
	sb.WriteString(fmt.Sprintf("%d", height))
	sb.WriteString(memo)
	sb.WriteString(pubKey.String())
	for _, c := range coins {
		sb.WriteString(c.String())
	}
	hash := sha256.New()
	return hex.EncodeToString(hash.Sum([]byte(sb.String())))
}

// Route should return the route key of the module
func (msg MsgTssKeysignFail) Route() string { return RouterKey }

// Type should return the action
func (msg MsgTssKeysignFail) Type() string { return "set_tss_keysign_fail" }

// ValidateBasic runs stateless checks on the message
func (msg MsgTssKeysignFail) ValidateBasic() error {
	if msg.Signer.Empty() {
		return cosmos.ErrInvalidAddress(msg.Signer.String())
	}
	if len(msg.ID) == 0 {
		return cosmos.ErrUnknownRequest("ID cannot be blank")
	}
	if len(msg.Coins) == 0 {
		return cosmos.ErrUnknownRequest("no coins")
	}
	if err := msg.Coins.Valid(); err != nil {
		return cosmos.ErrInvalidCoins(err.Error())
	}
	if msg.Blame.IsEmpty() {
		return cosmos.ErrUnknownRequest("tss blame is empty")
	}
	if msg.Height <= 0 {
		return cosmos.ErrUnknownRequest("block height cannot be equal to less than zero")
	}
	if len([]byte(msg.Memo)) > 150 {
		err := fmt.Errorf("memo must not exceed 150 bytes: %d", len([]byte(msg.Memo)))
		return cosmos.ErrUnknownRequest(err.Error())
	}
	if msg.PubKey.IsEmpty() {
		return cosmos.ErrUnknownRequest("vault pubkey cannot be empty")
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgTssKeysignFail) GetSignBytes() []byte {
	return cosmos.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgTssKeysignFail) GetSigners() []cosmos.AccAddress {
	return []cosmos.AccAddress{msg.Signer}
}
