package keeperv1

import (
	"fmt"
	"strconv"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

// AddToLiquidityFees - measure of fees collected in each block
func (k KVStore) AddToLiquidityFees(ctx cosmos.Context, asset common.Asset, fee cosmos.Uint) error {
	currentHeight := uint64(common.BlockHeight(ctx))

	totalFees, err := k.GetTotalLiquidityFees(ctx, currentHeight)
	if err != nil {
		return err
	}
	poolFees, err := k.GetPoolLiquidityFees(ctx, currentHeight, asset)
	if err != nil {
		return err
	}

	totalFees = totalFees.Add(fee)
	poolFees = poolFees.Add(fee)

	// update total liquidity
	k.set(ctx, k.GetKey(ctx, prefixTotalLiquidityFee, strconv.FormatUint(currentHeight, 10)), totalFees)

	// update pool liquidity
	k.set(ctx, k.GetKey(ctx, prefixPoolLiquidityFee, fmt.Sprintf("%d-%s", currentHeight, asset.String())), poolFees)
	return nil
}

func (k KVStore) getLiquidityFees(ctx cosmos.Context, key string) (cosmos.Uint, error) {
	record := cosmos.ZeroUint()
	_, err := k.get(ctx, key, &record)
	return record, err
}

// GetTotalLiquidityFees - total of all fees collected in each block
func (k KVStore) GetTotalLiquidityFees(ctx cosmos.Context, height uint64) (cosmos.Uint, error) {
	key := k.GetKey(ctx, prefixTotalLiquidityFee, strconv.FormatUint(height, 10))
	return k.getLiquidityFees(ctx, key)
}

// GetPoolLiquidityFees - total of fees collected in each block per pool
func (k KVStore) GetPoolLiquidityFees(ctx cosmos.Context, height uint64, asset common.Asset) (cosmos.Uint, error) {
	key := k.GetKey(ctx, prefixPoolLiquidityFee, fmt.Sprintf("%d-%s", height, asset.String()))
	return k.getLiquidityFees(ctx, key)
}
