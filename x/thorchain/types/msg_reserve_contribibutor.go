package types

import (
	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

// MsgReserveContributor defines a MsgReserveContributor message
type MsgReserveContributor struct {
	Tx          common.Tx          `json:"tx"`
	Contributor ReserveContributor `json:"contributor"`
	Signer      cosmos.AccAddress  `json:"signer"`
}

// NewMsgReserveContributor is a constructor function for MsgReserveContributor
func NewMsgReserveContributor(tx common.Tx, contrib ReserveContributor, signer cosmos.AccAddress) MsgReserveContributor {
	return MsgReserveContributor{
		Tx:          tx,
		Contributor: contrib,
		Signer:      signer,
	}
}

// Route return the route key of module
func (msg MsgReserveContributor) Route() string { return RouterKey }

// Type return a unique action
func (msg MsgReserveContributor) Type() string { return "set_reserve_contributor" }

// ValidateBasic runs stateless checks on the message
func (msg MsgReserveContributor) ValidateBasic() error {
	if err := msg.Tx.Valid(); err != nil {
		return cosmos.ErrUnknownRequest(err.Error())
	}
	if msg.Signer.Empty() {
		return cosmos.ErrInvalidAddress(msg.Signer.String())
	}
	if err := msg.Contributor.Valid(); err != nil {
		return cosmos.ErrUnknownRequest(err.Error())
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgReserveContributor) GetSignBytes() []byte {
	return cosmos.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgReserveContributor) GetSigners() []cosmos.AccAddress {
	return []cosmos.AccAddress{msg.Signer}
}
