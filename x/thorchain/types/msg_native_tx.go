package types

import (
	"fmt"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

// MsgNativeTx defines a MsgNativeTx message
type MsgNativeTx struct {
	Coins  common.Coins      `json:"coins"`
	Memo   string            `json:"memo"`
	Signer cosmos.AccAddress `json:"signer"`
}

// NewMsgNativeTx is a constructor function for NewMsgNativeTx
func NewMsgNativeTx(coins common.Coins, memo string, signer cosmos.AccAddress) MsgNativeTx {
	return MsgNativeTx{
		Coins:  coins,
		Memo:   memo,
		Signer: signer,
	}
}

// Route should return the route key of the module
func (msg MsgNativeTx) Route() string { return RouterKey }

// Type should return the action
func (msg MsgNativeTx) Type() string { return "native_tx" }

// ValidateBasic runs stateless checks on the message
func (msg MsgNativeTx) ValidateBasic() error {
	if msg.Signer.Empty() {
		return cosmos.ErrInvalidAddress(msg.Signer.String())
	}
	if err := msg.Coins.Valid(); err != nil {
		return cosmos.ErrUnknownRequest(err.Error())
	}
	for _, coin := range msg.Coins {
		if !coin.IsNative() {
			return cosmos.ErrUnknownRequest("all coins must be native to THORChain")
		}
	}
	if len([]byte(msg.Memo)) > 150 {
		err := fmt.Errorf("memo must not exceed 150 bytes: %d", len([]byte(msg.Memo)))
		return cosmos.ErrUnknownRequest(err.Error())
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgNativeTx) GetSignBytes() []byte {
	return cosmos.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgNativeTx) GetSigners() []cosmos.AccAddress {
	return []cosmos.AccAddress{msg.Signer}
}
