package keeperv1

import (
	"strconv"

	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
)

// SetKeygenBlock save the KeygenBlock to kv store
func (k KVStore) SetKeygenBlock(ctx cosmos.Context, keygen KeygenBlock) {
	k.set(ctx, k.GetKey(ctx, prefixKeygen, strconv.FormatInt(keygen.Height, 10)), keygen)
}

// GetKeygenBlockIterator return an iterator
func (k KVStore) GetKeygenBlockIterator(ctx cosmos.Context) cosmos.Iterator {
	return k.getIterator(ctx, prefixKeygen)
}

// GetKeygenBlock from a given height
func (k KVStore) GetKeygenBlock(ctx cosmos.Context, height int64) (KeygenBlock, error) {
	record := NewKeygenBlock(height)
	_, err := k.get(ctx, k.GetKey(ctx, prefixKeygen, strconv.FormatInt(height, 10)), &record)
	return record, err
}
