package types

import (
	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

// MsgRefundTx defines a MsgRefundTx message
type MsgRefundTx struct {
	Tx     ObservedTx        `json:"tx"`
	InTxID common.TxID       `json:"tx_id"`
	Signer cosmos.AccAddress `json:"signer"`
}

// NewMsgRefundTx is a constructor function for MsgOutboundTx
func NewMsgRefundTx(tx ObservedTx, txID common.TxID, signer cosmos.AccAddress) MsgRefundTx {
	return MsgRefundTx{
		Tx:     tx,
		InTxID: txID,
		Signer: signer,
	}
}

// Route should return the route key of the module
func (msg MsgRefundTx) Route() string { return RouterKey }

// Type should return the action
func (msg MsgRefundTx) Type() string { return "set_tx_refund" }

// ValidateBasic runs stateless checks on the message
func (msg MsgRefundTx) ValidateBasic() error {
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
func (msg MsgRefundTx) GetSignBytes() []byte {
	return cosmos.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgRefundTx) GetSigners() []cosmos.AccAddress {
	return []cosmos.AccAddress{msg.Signer}
}
