package types

import (
	"github.com/blang/semver"

	"gitlab.com/thorchain/thornode/common/cosmos"
)

// MsgSetVersion defines a MsgSetVersion message
type MsgSetVersion struct {
	Version semver.Version    `json:"version"`
	Signer  cosmos.AccAddress `json:"signer"`
}

// NewMsgSetVersion is a constructor function for NewMsgSetVersion
func NewMsgSetVersion(version semver.Version, signer cosmos.AccAddress) MsgSetVersion {
	return MsgSetVersion{
		Version: version,
		Signer:  signer,
	}
}

// Route should return the route key of the module
func (msg MsgSetVersion) Route() string { return RouterKey }

// Type should return the action
func (msg MsgSetVersion) Type() string { return "set_version" }

// ValidateBasic runs stateless checks on the message
func (msg MsgSetVersion) ValidateBasic() error {
	if msg.Signer.Empty() {
		return cosmos.ErrInvalidAddress(msg.Signer.String())
	}
	if err := msg.Version.Validate(); err != nil {
		return cosmos.ErrUnknownRequest(err.Error())
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgSetVersion) GetSignBytes() []byte {
	return cosmos.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgSetVersion) GetSigners() []cosmos.AccAddress {
	return []cosmos.AccAddress{msg.Signer}
}
