package types

import (
	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

// MsgBond when a user would like to become a validator, and run a full set, they need send an `apply:bepaddress` with a bond to our pool address
type MsgBond struct {
	TxIn        common.Tx         `json:"tx_in"`
	NodeAddress cosmos.AccAddress `json:"node_address"`
	Bond        cosmos.Uint       `json:"bond"`
	BondAddress common.Address    `json:"bond_address"`
	Signer      cosmos.AccAddress `json:"signer"`
}

// NewMsgBond create new MsgBond message
func NewMsgBond(txin common.Tx, nodeAddr cosmos.AccAddress, bond cosmos.Uint, bondAddress common.Address, signer cosmos.AccAddress) MsgBond {
	return MsgBond{
		TxIn:        txin,
		NodeAddress: nodeAddr,
		Bond:        bond,
		BondAddress: bondAddress,
		Signer:      signer,
	}
}

// Route should return the router key of the module
func (msg MsgBond) Route() string { return RouterKey }

// Type should return the action
func (msg MsgBond) Type() string { return "bond" }

// ValidateBasic runs stateless checks on the message
func (msg MsgBond) ValidateBasic() error {
	if msg.NodeAddress.Empty() {
		return cosmos.ErrInvalidAddress("node address cannot be empty")
	}
	if msg.Bond.IsZero() {
		return cosmos.ErrUnknownRequest("bond cannot be zero")
	}
	if msg.BondAddress.IsEmpty() {
		return cosmos.ErrInvalidAddress("bond address cannot be empty")
	}
	if err := msg.TxIn.Valid(); err != nil {
		return cosmos.ErrUnknownRequest(err.Error())
	}
	if len(msg.TxIn.Coins) > 1 {
		return cosmos.ErrUnknownRequest("cannot bond more than one coin")
	}
	if !msg.TxIn.Coins[0].Asset.IsRune() {
		return cosmos.ErrUnknownRequest("cannot bond non-rune asset")
	}
	if msg.TxIn.Coins.IsEmpty() {
		return cosmos.ErrUnknownRequest("cannot bond with empty coins")
	}
	if msg.Signer.Empty() {
		return cosmos.ErrInvalidAddress("empty signer address")
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgBond) GetSignBytes() []byte {
	return cosmos.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgBond) GetSigners() []cosmos.AccAddress {
	return []cosmos.AccAddress{msg.Signer}
}
