package keeperv1

import (
	"errors"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

// GetPoolIterator iterate pools
func (k KVStore) GetPoolIterator(ctx cosmos.Context) cosmos.Iterator {
	return k.getIterator(ctx, prefixPool)
}

// GetPools return all pool in key value store regardless state
func (k KVStore) GetPools(ctx cosmos.Context) (Pools, error) {
	var pools Pools
	iterator := k.GetPoolIterator(ctx)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var pool Pool
		err := k.Cdc().UnmarshalBinaryBare(iterator.Value(), &pool)
		if err != nil {
			return nil, dbError(ctx, "Unmarsahl: pool", err)
		}
		pools = append(pools, pool)
	}
	return pools, nil
}

// GetPool get the entire Pool metadata struct based on given asset
func (k KVStore) GetPool(ctx cosmos.Context, asset common.Asset) (Pool, error) {
	record := NewPool()
	_, err := k.get(ctx, k.GetKey(ctx, prefixPool, asset.String()), &record)
	return record, err
}

// SetPool save the entire Pool metadata struct to key value store
func (k KVStore) SetPool(ctx cosmos.Context, pool Pool) error {
	if pool.Asset.IsEmpty() {
		return errors.New("cannot save a pool with an empty asset")
	}

	k.set(ctx, k.GetKey(ctx, prefixPool, pool.Asset.String()), pool)
	return nil
}

// PoolExist check whether the given pool exist in the data store
func (k KVStore) PoolExist(ctx cosmos.Context, asset common.Asset) bool {
	return k.has(ctx, k.GetKey(ctx, prefixPool, asset.String()))
}

func (k KVStore) RemovePool(ctx cosmos.Context, asset common.Asset) {
	k.del(ctx, k.GetKey(ctx, prefixPool, asset.String()))
}
