package keeperv1

import (
	"fmt"
	"strings"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

// SetLastSignedHeight save last signed height into kv store
func (k KVStore) SetLastSignedHeight(ctx cosmos.Context, height int64) error {
	lastHeight, _ := k.GetLastSignedHeight(ctx)
	if lastHeight > height {
		err := fmt.Errorf("last signed height %d is larger than %d, block height can't go backward ", lastHeight, height)
		return dbError(ctx, "", err)
	}
	k.set(ctx, k.GetKey(ctx, prefixLastSignedHeight, ""), height)
	return nil
}

// GetLastSignedHeight get last signed height from key value store
func (k KVStore) GetLastSignedHeight(ctx cosmos.Context) (int64, error) {
	var record int64
	_, err := k.get(ctx, k.GetKey(ctx, prefixLastSignedHeight, ""), &record)
	return record, err
}

// SetLastChainHeight save last chain height
func (k KVStore) SetLastChainHeight(ctx cosmos.Context, chain common.Chain, height int64) error {
	lastHeight, _ := k.GetLastChainHeight(ctx, chain)
	if lastHeight > height {
		err := fmt.Errorf("last block height %d is larger than %d, block height can't go backward ", lastHeight, height)
		return dbError(ctx, "", err)
	}
	k.set(ctx, k.GetKey(ctx, prefixLastChainHeight, chain.String()), height)
	return nil
}

// GetLastChainHeight get last chain height
func (k KVStore) GetLastChainHeight(ctx cosmos.Context, chain common.Chain) (int64, error) {
	var record int64
	_, err := k.get(ctx, k.GetKey(ctx, prefixLastChainHeight, chain.String()), &record)
	return record, err
}

// GetLastChainHeights get the iterator for last chain height
func (k KVStore) GetLastChainHeights(ctx cosmos.Context) (map[common.Chain]int64, error) {
	iter := k.getIterator(ctx, prefixLastChainHeight)
	result := make(map[common.Chain]int64)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		key := string(iter.Key())
		c := strings.TrimPrefix(key, string(prefixLastChainHeight+"/"))
		chain, err := common.NewChain(c)
		if err != nil {
			return nil, fmt.Errorf("fail to parse chain: %w", err)
		}
		var height int64
		k.cdc.MustUnmarshalBinaryBare(iter.Value(), &height)
		result[chain] = height
	}
	return result, nil
}
