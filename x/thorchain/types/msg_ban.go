package types

import (
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
)

// MsgBan defines a MsgBan message
type MsgBan struct {
	NodeAddress cosmos.AccAddress `json:"node_address"`
	Signer      cosmos.AccAddress `json:"signer"`
}

// NewMsgBan is a constructor function for NewMsgBan
func NewMsgBan(addr, signer cosmos.AccAddress) MsgBan {
	return MsgBan{
		NodeAddress: addr,
		Signer:      signer,
	}
}

// Route should return the name of the module
func (msg MsgBan) Route() string { return RouterKey }

// Type should return the action
func (msg MsgBan) Type() string { return "ban" }

// ValidateBasic runs stateless checks on the message
func (msg MsgBan) ValidateBasic() error {
	if msg.Signer.Empty() {
		return cosmos.ErrInvalidAddress(msg.Signer.String())
	}
	if msg.NodeAddress.Empty() {
		return cosmos.ErrInvalidAddress(msg.NodeAddress.String())
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgBan) GetSignBytes() []byte {
	return cosmos.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgBan) GetSigners() []cosmos.AccAddress {
	return []cosmos.AccAddress{msg.Signer}
}
