package keeperv1

import (
	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/x/thorchain/keeper/types"
)

// GetStakerIterator iterate stakers
func (k KVStore) GetStakerIterator(ctx cosmos.Context, asset common.Asset) cosmos.Iterator {
	key := k.GetKey(ctx, prefixStaker, Staker{Asset: asset}.Key())
	return k.getIterator(ctx, types.DbPrefix(key))
}

// GetStaker retrieve staker from the data store
func (k KVStore) GetStaker(ctx cosmos.Context, asset common.Asset, addr common.Address) (Staker, error) {
	record := Staker{
		Asset:       asset,
		RuneAddress: addr,
		Units:       cosmos.ZeroUint(),
		PendingRune: cosmos.ZeroUint(),
	}
	_, err := k.get(ctx, k.GetKey(ctx, prefixStaker, record.Key()), &record)
	return record, err
}

// SetStaker save the staker to kv store
func (k KVStore) SetStaker(ctx cosmos.Context, staker Staker) {
	k.set(ctx, k.GetKey(ctx, prefixStaker, staker.Key()), staker)
}

// RemoveStaker remove the staker to kv store
func (k KVStore) RemoveStaker(ctx cosmos.Context, staker Staker) {
	k.del(ctx, k.GetKey(ctx, prefixStaker, staker.Key()))
}
