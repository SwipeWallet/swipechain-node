package types

import (
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
)

// MsgNoOp defines a no op message
type MsgNoOp struct {
	ObservedTx ObservedTx        `json:"observed_tx"`
	Signer     cosmos.AccAddress `json:"signer"`
}

// NewMsgNoOp is a constructor function for MsgNoOp
func NewMsgNoOp(ObservedTx ObservedTx, signer cosmos.AccAddress) MsgNoOp {
	return MsgNoOp{
		ObservedTx: ObservedTx,
		Signer:     signer,
	}
}

// Route should return the pooldata of the module
func (msg MsgNoOp) Route() string { return RouterKey }

// Type should return the action
func (msg MsgNoOp) Type() string { return "set_noop" }

// ValidateBasic runs stateless checks on the message
func (msg MsgNoOp) ValidateBasic() error {
	if err := msg.ObservedTx.Valid(); err != nil {
		return cosmos.ErrInvalidCoins(err.Error())
	}
	if msg.Signer.Empty() {
		return cosmos.ErrInvalidAddress(msg.Signer.String())
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgNoOp) GetSignBytes() []byte {
	return cosmos.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgNoOp) GetSigners() []cosmos.AccAddress {
	return []cosmos.AccAddress{msg.Signer}
}
