package common

import "gitlab.com/thorchain/thornode/common/cosmos"

// Fee represent fee
type Fee struct {
	Coins      Coins       `json:"coins"`
	PoolDeduct cosmos.Uint `json:"pool_deduct"`
}

// NewFee return a new instance of Fee
func NewFee(coins Coins, poolDeduct cosmos.Uint) Fee {
	return Fee{
		Coins:      coins,
		PoolDeduct: poolDeduct,
	}
}
