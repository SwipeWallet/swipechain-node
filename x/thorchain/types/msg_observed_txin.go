package types

import (
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
)

// MsgObservedTxIn defines a MsgObservedTxIn message
type MsgObservedTxIn struct {
	Txs    ObservedTxs       `json:"txs"`
	Signer cosmos.AccAddress `json:"signer"`
}

// NewMsgObservedTxIn is a constructor function for MsgObservedTxIn
func NewMsgObservedTxIn(txs ObservedTxs, signer cosmos.AccAddress) MsgObservedTxIn {
	return MsgObservedTxIn{
		Txs:    txs,
		Signer: signer,
	}
}

// Route should return the route key of the module
func (msg MsgObservedTxIn) Route() string { return RouterKey }

// Type should return the action
func (msg MsgObservedTxIn) Type() string { return "set_observed_txin" }

// ValidateBasic runs stateless checks on the message
func (msg MsgObservedTxIn) ValidateBasic() error {
	if msg.Signer.Empty() {
		return cosmos.ErrInvalidAddress(msg.Signer.String())
	}
	if len(msg.Txs) == 0 {
		return cosmos.ErrUnknownRequest("Txs cannot be empty")
	}
	for _, tx := range msg.Txs {
		if err := tx.Valid(); err != nil {
			return cosmos.ErrUnknownRequest(err.Error())
		}
		obAddr, err := tx.ObservedPubKey.GetAddress(tx.Tx.Coins[0].Asset.Chain)
		if err != nil {
			return cosmos.ErrUnknownRequest(err.Error())
		}
		if !tx.Tx.ToAddress.Equals(obAddr) {
			return cosmos.ErrUnknownRequest("request is not an inbound observed transaction")
		}
		if len(tx.Signers) > 0 {
			return cosmos.ErrUnknownRequest("signers must be empty")
		}
		if len(tx.OutHashes) > 0 {
			return cosmos.ErrUnknownRequest("out hashes must be empty")
		}
		if tx.Status != Incomplete {
			return cosmos.ErrUnknownRequest("status must be incomplete")
		}
	}

	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgObservedTxIn) GetSignBytes() []byte {
	return cosmos.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgObservedTxIn) GetSigners() []cosmos.AccAddress {
	return []cosmos.AccAddress{msg.Signer}
}
