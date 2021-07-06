package types

import (
	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

// MsgOutboundTx defines a MsgOutboundTx message
type MsgOutboundTx struct {
	Tx     ObservedTx        `json:"tx"`
	InTxID common.TxID       `json:"tx_id"`
	Signer cosmos.AccAddress `json:"signer"`
}

// NewMsgOutboundTx is a constructor function for MsgOutboundTx
func NewMsgOutboundTx(tx ObservedTx, txID common.TxID, signer cosmos.AccAddress) MsgOutboundTx {
	return MsgOutboundTx{
		Tx:     tx,
		InTxID: txID,
		Signer: signer,
	}
}

// Route should return the route key of the module
func (msg MsgOutboundTx) Route() string { return RouterKey }

// Type should return the action
func (msg MsgOutboundTx) Type() string { return "set_tx_outbound" }

// ValidateBasic runs stateless checks on the message
func (msg MsgOutboundTx) ValidateBasic() error {
	if msg.Signer.Empty() {
		return cosmos.ErrInvalidAddress(msg.Signer.String())
	}
	if msg.InTxID.IsEmpty() {
		return cosmos.ErrUnknownRequest("In Tx ID cannot be empty")
	}
	if err := msg.Tx.Valid(); err != nil {
		return cosmos.ErrUnknownRequest(err.Error())
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgOutboundTx) GetSignBytes() []byte {
	return cosmos.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgOutboundTx) GetSigners() []cosmos.AccAddress {
	return []cosmos.AccAddress{msg.Signer}
}
