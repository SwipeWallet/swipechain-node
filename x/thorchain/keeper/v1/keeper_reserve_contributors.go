package keeperv1

import (
	"fmt"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

// AddFeeToReserve add fee to reserve, the fee is always in RUNE
func (k KVStore) AddFeeToReserve(ctx cosmos.Context, fee cosmos.Uint) error {
	vault, err := k.GetVaultData(ctx)
	if err != nil {
		return fmt.Errorf("fail to get vault: %w", err)
	}
	if common.RuneAsset().Chain.Equals(common.THORChain) {
		coin := common.NewCoin(common.RuneNative, fee)
		sdkErr := k.SendFromModuleToModule(ctx, AsgardName, ReserveName, coin)
		if sdkErr != nil {
			return dbError(ctx, "fail to send fee to reserve", sdkErr)
		}
	} else {
		vault.TotalReserve = vault.TotalReserve.Add(fee)
	}
	return k.SetVaultData(ctx, vault)
}

// GetReservesContributors return those address who contributed to the reserve
func (k KVStore) GetReservesContributors(ctx cosmos.Context) (ReserveContributors, error) {
	record := make(ReserveContributors, 0)
	_, err := k.get(ctx, k.GetKey(ctx, prefixReserves, ""), &record)
	return record, err
}

// SetReserveContributors save reserve contributors to key value store
func (k KVStore) SetReserveContributors(ctx cosmos.Context, contributors ReserveContributors) error {
	k.set(ctx, k.GetKey(ctx, prefixReserves, ""), contributors)
	return nil
}
