package types

import (
	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

// MsgSwitch defines a MsgSwitch message
type MsgSwitch struct {
	Tx          common.Tx         `json:"tx"`
	Destination common.Address    `json:"destination"`
	Signer      cosmos.AccAddress `json:"signer"`
}

// NewMsgSwitch is a constructor function for NewMsgSwitch
func NewMsgSwitch(tx common.Tx, addr common.Address, signer cosmos.AccAddress) MsgSwitch {
	return MsgSwitch{
		Tx:          tx,
		Destination: addr,
		Signer:      signer,
	}
}

// Route should return the route key of the module
func (msg MsgSwitch) Route() string { return RouterKey }

// Type should return the action
func (msg MsgSwitch) Type() string { return "switch" }

// ValidateBasic runs stateless checks on the message
func (msg MsgSwitch) ValidateBasic() error {
	if msg.Signer.Empty() {
		return cosmos.ErrInvalidAddress(msg.Signer.String())
	}
	if msg.Destination.IsEmpty() {
		return cosmos.ErrInvalidAddress(msg.Destination.String())
	}
	if err := msg.Tx.Valid(); err != nil {
		return cosmos.ErrUnknownRequest(err.Error())
	}
	// cannot be more or less than one coin
	if len(msg.Tx.Coins) != 1 {
		return cosmos.ErrInvalidCoins("must be only one coin (rune)")
	}
	if !msg.Tx.Coins[0].Asset.IsRune() {
		return cosmos.ErrInvalidCoins("must be rune")
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgSwitch) GetSignBytes() []byte {
	return cosmos.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgSwitch) GetSigners() []cosmos.AccAddress {
	return []cosmos.AccAddress{msg.Signer}
}
