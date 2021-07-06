package types

import (
	"gitlab.com/thorchain/thornode/common/cosmos"
)

// MsgRagnarok defines a MsgRagnarok message
type MsgRagnarok struct {
	Tx          ObservedTx        `json:"tx"`
	BlockHeight int64             `json:"block_height"`
	Signer      cosmos.AccAddress `json:"signer"`
}

// NewMsgRagnarok is a constructor function for MsgRagnarok
func NewMsgRagnarok(tx ObservedTx, blockHeight int64, signer cosmos.AccAddress) MsgRagnarok {
	return MsgRagnarok{
		Tx:          tx,
		BlockHeight: blockHeight,
		Signer:      signer,
	}
}

// Route should return the name of the module
func (msg MsgRagnarok) Route() string { return RouterKey }

// Type should return the action
func (msg MsgRagnarok) Type() string { return "ragnarok" }

// ValidateBasic runs stateless checks on the message
func (msg MsgRagnarok) ValidateBasic() error {
	if msg.Signer.Empty() {
		return cosmos.ErrInvalidAddress(msg.Signer.String())
	}
	if msg.BlockHeight <= 0 {
		return cosmos.ErrUnknownRequest("invalid block height")
	}
	if err := msg.Tx.Valid(); err != nil {
		return cosmos.ErrUnknownRequest(err.Error())
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgRagnarok) GetSignBytes() []byte {
	return cosmos.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgRagnarok) GetSigners() []cosmos.AccAddress {
	return []cosmos.AccAddress{msg.Signer}
}
