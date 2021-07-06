package types

import (
	"errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"gitlab.com/thorchain/thornode/common"
)

// NetworkFee represent the fee rate and typical transaction size outbound from THORNode
// This is to keep the information reported by bifrost
// For BTC chain, TransactionFeeRate should be sats/vbyte
// For Binance chain , given fee is fixed , thus for single coin , transaction size will be 1, and the rate should be 37500, for multiple coin , Transaction size should the number of coins
type NetworkFee struct {
	Chain              common.Chain `json:"chain"`
	TransactionSize    int64        `json:"transaction_size"`
	TransactionFeeRate sdk.Uint     `json:"transaction_fee_rate"`
}

// NewNetworkFee create a new instance of network fee
func NewNetworkFee(chain common.Chain, transactionSize int64, transactionFeeRate sdk.Uint) NetworkFee {
	return NetworkFee{
		Chain:              chain,
		TransactionSize:    transactionSize,
		TransactionFeeRate: transactionFeeRate,
	}
}

func (f NetworkFee) Valid() error {
	if f.Chain.IsEmpty() {
		return errors.New("chain can't be empty")
	}
	if f.TransactionSize <= 0 {
		return errors.New("transaction size can't be negative")
	}
	if f.TransactionFeeRate.Equal(sdk.ZeroUint()) {
		return errors.New("transaction fee rate can't be zero")
	}
	return nil
}
