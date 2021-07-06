package keeperv1

import (
	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

// TODO these will not be needed once dynamic network fee get merged

// GetGas get gas information from key value store
func (k KVStore) GetGas(ctx cosmos.Context, asset common.Asset) ([]cosmos.Uint, error) {
	var record []cosmos.Uint
	_, err := k.get(ctx, k.GetKey(ctx, prefixGas, asset.String()), &record)
	return record, err
}

// SetGas save gas information to key value store
func (k KVStore) SetGas(ctx cosmos.Context, asset common.Asset, units []cosmos.Uint) {
	k.set(ctx, k.GetKey(ctx, prefixGas, asset.String()), units)
}

// GetGasIterator iterate gas units
func (k KVStore) GetGasIterator(ctx cosmos.Context) cosmos.Iterator {
	return k.getIterator(ctx, prefixGas)
}
