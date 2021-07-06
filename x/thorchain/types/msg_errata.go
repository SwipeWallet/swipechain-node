package types

import (
	"gitlab.com/thorchain/thornode/common"
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
)

// MsgErrataTx defines a MsgErrataTx message
type MsgErrataTx struct {
	TxID   common.TxID       `json:"txid"`
	Chain  common.Chain      `json:"chain"`
	Signer cosmos.AccAddress `json:"signer"`
}

// NewMsgErrataTx is a constructor function for NewMsgErrataTx
func NewMsgErrataTx(txID common.TxID, chain common.Chain, signer cosmos.AccAddress) MsgErrataTx {
	return MsgErrataTx{
		TxID:   txID,
		Chain:  chain,
		Signer: signer,
	}
}

// Route should return the name of the module
func (msg MsgErrataTx) Route() string { return RouterKey }

// Type should return the action
func (msg MsgErrataTx) Type() string { return "errata_tx" }

// ValidateBasic runs stateless checks on the message
func (msg MsgErrataTx) ValidateBasic() error {
	if msg.Signer.Empty() {
		return cosmos.ErrInvalidAddress(msg.Signer.String())
	}
	if msg.TxID.IsEmpty() {
		return cosmos.ErrUnknownRequest("Tx ID cannot be empty")
	}
	if msg.Chain.IsEmpty() {
		return cosmos.ErrUnknownRequest("chain cannot be empty")
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgErrataTx) GetSignBytes() []byte {
	return cosmos.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgErrataTx) GetSigners() []cosmos.AccAddress {
	return []cosmos.AccAddress{msg.Signer}
}
