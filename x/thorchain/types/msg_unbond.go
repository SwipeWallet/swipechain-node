package types

import (
	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

// MsgUnBond when a user would like to remove some bond
type MsgUnBond struct {
	TxIn        common.Tx         `json:"tx_in"`
	NodeAddress cosmos.AccAddress `json:"node_address"`
	Amount      cosmos.Uint       `json:"amount"`
	BondAddress common.Address    `json:"bond_address"`
	Signer      cosmos.AccAddress `json:"signer"`
}

// NewMsgUnBond create new MsgUnBond message
func NewMsgUnBond(txin common.Tx, nodeAddr cosmos.AccAddress, amount cosmos.Uint, bondAddress common.Address, signer cosmos.AccAddress) MsgUnBond {
	return MsgUnBond{
		TxIn:        txin,
		NodeAddress: nodeAddr,
		Amount:      amount,
		BondAddress: bondAddress,
		Signer:      signer,
	}
}

// Route should return the router key of the module
func (msg MsgUnBond) Route() string { return RouterKey }

// Type should return the action
func (msg MsgUnBond) Type() string { return "unbond" }

// ValidateBasic runs stateless checks on the message
func (msg MsgUnBond) ValidateBasic() error {
	if msg.NodeAddress.Empty() {
		return cosmos.ErrInvalidAddress("node address cannot be empty")
	}
	if msg.Amount.IsZero() {
		return cosmos.ErrUnknownRequest("bond cannot be zero")
	}
	if msg.BondAddress.IsEmpty() {
		return cosmos.ErrInvalidAddress("bond address cannot be empty")
	}
	if err := msg.TxIn.Valid(); err != nil {
		return cosmos.ErrUnknownRequest(err.Error())
	}
	if msg.Signer.Empty() {
		return cosmos.ErrInvalidAddress("empty signer address")
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgUnBond) GetSignBytes() []byte {
	return cosmos.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgUnBond) GetSigners() []cosmos.AccAddress {
	return []cosmos.AccAddress{msg.Signer}
}
