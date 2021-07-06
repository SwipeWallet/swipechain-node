package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"gitlab.com/thorchain/thornode/common"
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
)

// MsgNetworkFee is message bifrost will be used to send chain related network fee to THORNode
type MsgNetworkFee struct {
	BlockHeight        int64             `json:"block_height"`
	Chain              common.Chain      `json:"chain"`
	TransactionSize    int64             `json:"transaction_size"`
	TransactionFeeRate sdk.Uint          `json:"transaction_fee_rate"`
	Signer             cosmos.AccAddress `json:"signer"`
}

// NewMsgNetworkFee create a new instance of MsgNetworkFee
func NewMsgNetworkFee(blockHeight int64, chain common.Chain, transactionSize int64, transactionFeeRate sdk.Uint, signer cosmos.AccAddress) MsgNetworkFee {
	return MsgNetworkFee{
		BlockHeight:        blockHeight,
		Chain:              chain,
		TransactionSize:    transactionSize,
		TransactionFeeRate: transactionFeeRate,
		Signer:             signer,
	}
}

// Route should return the Route of the module
func (msg MsgNetworkFee) Route() string { return RouterKey }

// Type should return the action
func (msg MsgNetworkFee) Type() string { return "set_network_fee" }

// ValidateBasic runs stateless checks on the message
func (msg MsgNetworkFee) ValidateBasic() error {
	if msg.BlockHeight < 0 {
		return cosmos.ErrUnknownRequest("block height can't be negative")
	}
	if msg.Signer.Empty() {
		return cosmos.ErrInvalidAddress(msg.Signer.String())
	}
	if msg.Chain.IsEmpty() {
		return cosmos.ErrUnknownRequest("chain can't be empty")
	}
	if msg.TransactionSize <= 0 {
		return cosmos.ErrUnknownRequest("invalid transaction size")
	}
	if msg.TransactionFeeRate.IsZero() {
		return cosmos.ErrUnknownRequest("invalid transaction fee rate")
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgNetworkFee) GetSignBytes() []byte {
	return cosmos.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgNetworkFee) GetSigners() []cosmos.AccAddress {
	return []cosmos.AccAddress{msg.Signer}
}
