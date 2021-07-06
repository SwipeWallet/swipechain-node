package keeperv1

import (
	"gitlab.com/thorchain/thornode/common/cosmos"
)

// GetVaultData retrieve vault data from key value store
func (k KVStore) GetVaultData(ctx cosmos.Context) (VaultData, error) {
	record := NewVaultData()
	_, err := k.get(ctx, k.GetKey(ctx, prefixVaultData, ""), &record)
	return record, err
}

// SetVaultData save the given vault data to key value store, it will overwrite existing vault
func (k KVStore) SetVaultData(ctx cosmos.Context, data VaultData) error {
	k.set(ctx, k.GetKey(ctx, prefixVaultData, ""), data)
	return nil
}
