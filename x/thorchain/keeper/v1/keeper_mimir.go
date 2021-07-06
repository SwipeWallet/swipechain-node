package keeperv1

import "gitlab.com/thorchain/thornode/common/cosmos"

const KRAKEN string = "ReleaseTheKraken"

// GetMimir get a mimir value from key value store
func (k KVStore) GetMimir(ctx cosmos.Context, key string) (int64, error) {
	// if we have the kraken, mimir is no more, ignore him
	if k.haveKraken(ctx) {
		return -1, nil
	}

	record := int64(-1)
	_, err := k.get(ctx, k.GetKey(ctx, prefixMimir, key), &record)
	return record, err
}

// haveKraken - check to see if we have "released the kraken"
func (k KVStore) haveKraken(ctx cosmos.Context) bool {
	record := int64(-1)
	_, _ = k.get(ctx, k.GetKey(ctx, prefixMimir, KRAKEN), &record)
	return record >= 0
}

// SetMimir save a mimir value to key value store
func (k KVStore) SetMimir(ctx cosmos.Context, key string, value int64) {
	// if we have the kraken, mimir is no more, ignore him
	if k.haveKraken(ctx) {
		return
	}
	k.set(ctx, k.GetKey(ctx, prefixMimir, key), value)
}

// GetMimirIterator iterate gas units
func (k KVStore) GetMimirIterator(ctx cosmos.Context) cosmos.Iterator {
	return k.getIterator(ctx, prefixMimir)
}
