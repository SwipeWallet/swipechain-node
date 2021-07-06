package types

import (
	"net"

	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
)

// MsgSetIPAddress defines a MsgSetIPAddress message
type MsgSetIPAddress struct {
	IPAddress string            `json:"ip_address"`
	Signer    cosmos.AccAddress `json:"signer"`
}

// NewMsgSetIPAddress is a constructor function for NewMsgSetIPAddress
func NewMsgSetIPAddress(ip string, signer cosmos.AccAddress) MsgSetIPAddress {
	return MsgSetIPAddress{
		IPAddress: ip,
		Signer:    signer,
	}
}

// Route should return the name of the module
func (msg MsgSetIPAddress) Route() string { return RouterKey }

// Type should return the action
func (msg MsgSetIPAddress) Type() string { return "set_ip_address" }

// ValidateBasic runs stateless checks on the message
func (msg MsgSetIPAddress) ValidateBasic() error {
	if msg.Signer.Empty() {
		return cosmos.ErrInvalidAddress(msg.Signer.String())
	}
	if net.ParseIP(msg.IPAddress) == nil {
		return cosmos.ErrUnknownRequest("invalid IP address")
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgSetIPAddress) GetSignBytes() []byte {
	return cosmos.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgSetIPAddress) GetSigners() []cosmos.AccAddress {
	return []cosmos.AccAddress{msg.Signer}
}
