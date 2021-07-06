package types

import (
	"gitlab.com/thorchain/thornode/common"
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
)

// MsgLeave when an operator don't want to be a validator anymore
type MsgLeave struct {
	Tx          common.Tx         `json:"tx"`
	NodeAddress cosmos.AccAddress `json:"node_address"`
	Signer      cosmos.AccAddress `json:"signer"`
}

// NewMsgLeave create a new instance of MsgLeave
func NewMsgLeave(tx common.Tx, addr, signer cosmos.AccAddress) MsgLeave {
	return MsgLeave{
		Tx:          tx,
		NodeAddress: addr,
		Signer:      signer,
	}
}

// Route should return the router key of the module
func (msg MsgLeave) Route() string { return RouterKey }

// Type should return the action
func (msg MsgLeave) Type() string { return "leave" }

// ValidateBasic runs stateless checks on the message
func (msg MsgLeave) ValidateBasic() error {
	if msg.Tx.FromAddress.IsEmpty() {
		return cosmos.ErrInvalidAddress("from address cannot be empty")
	}
	if err := msg.Tx.Valid(); err != nil {
		return cosmos.ErrUnknownRequest(err.Error())
	}
	if msg.Signer.Empty() {
		return cosmos.ErrInvalidAddress("signer cannot be empty ")
	}
	if msg.NodeAddress.Empty() {
		return cosmos.ErrInvalidAddress("node address cannot be empty")
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgLeave) GetSignBytes() []byte {
	return cosmos.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgLeave) GetSigners() []cosmos.AccAddress {
	return []cosmos.AccAddress{msg.Signer}
}
