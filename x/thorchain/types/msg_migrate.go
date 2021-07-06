package types

import (
	"gitlab.com/thorchain/thornode/common/cosmos"
)

// MsgMigrate defines a MsgMigrate message
type MsgMigrate struct {
	Tx          ObservedTx        `json:"tx"`
	BlockHeight int64             `json:"block_height"`
	Signer      cosmos.AccAddress `json:"signer"`
}

// NewMsgMigrate is a constructor function for MsgMigrate
func NewMsgMigrate(tx ObservedTx, blockHeight int64, signer cosmos.AccAddress) MsgMigrate {
	return MsgMigrate{
		Tx:          tx,
		BlockHeight: blockHeight,
		Signer:      signer,
	}
}

// Route should return the name of the module
func (msg MsgMigrate) Route() string { return RouterKey }

// Type should return the action
func (msg MsgMigrate) Type() string { return "migrate" }

// ValidateBasic runs stateless checks on the message
func (msg MsgMigrate) ValidateBasic() error {
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
func (msg MsgMigrate) GetSignBytes() []byte {
	return cosmos.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgMigrate) GetSigners() []cosmos.AccAddress {
	return []cosmos.AccAddress{msg.Signer}
}
