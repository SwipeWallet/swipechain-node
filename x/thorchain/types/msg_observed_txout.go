package types

import (
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
)

// MsgObservedTxOut defines a MsgObservedTxOut message
type MsgObservedTxOut struct {
	Txs    ObservedTxs       `json:"txs"`
	Signer cosmos.AccAddress `json:"signer"`
}

// NewMsgObservedTxOut is a constructor function for MsgObservedTxOut
func NewMsgObservedTxOut(txs ObservedTxs, signer cosmos.AccAddress) MsgObservedTxOut {
	return MsgObservedTxOut{
		Txs:    txs,
		Signer: signer,
	}
}

// Route should return the route key of the module
func (msg MsgObservedTxOut) Route() string { return RouterKey }

// Type should return the action
func (msg MsgObservedTxOut) Type() string { return "set_observed_txout" }

// ValidateBasic runs stateless checks on the message
func (msg MsgObservedTxOut) ValidateBasic() error {
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
		if !tx.Tx.FromAddress.Equals(obAddr) {
			return cosmos.ErrUnknownRequest("Request is not an outbound observed transaction")
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
func (msg MsgObservedTxOut) GetSignBytes() []byte {
	return cosmos.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgObservedTxOut) GetSigners() []cosmos.AccAddress {
	return []cosmos.AccAddress{msg.Signer}
}
