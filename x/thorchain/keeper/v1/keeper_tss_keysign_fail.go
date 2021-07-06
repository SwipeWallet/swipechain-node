package keeperv1

import "gitlab.com/thorchain/thornode/common/cosmos"

// SetTssKeysignFailVoter - save a tss keysign fail voter object
func (k KVStore) SetTssKeysignFailVoter(ctx cosmos.Context, tss TssKeysignFailVoter) {
	k.set(ctx, k.GetKey(ctx, prefixTssKeysignFailure, tss.String()), tss)
}

// GetTssKeysignFailVoterIterator iterate tx in voters
func (k KVStore) GetTssKeysignFailVoterIterator(ctx cosmos.Context) cosmos.Iterator {
	return k.getIterator(ctx, prefixTssKeysignFailure)
}

// GetTssKeysignFailVoter - gets information of a tss keysign failure voter object
func (k KVStore) GetTssKeysignFailVoter(ctx cosmos.Context, id string) (TssKeysignFailVoter, error) {
	record := TssKeysignFailVoter{ID: id}
	_, err := k.get(ctx, k.GetKey(ctx, prefixTssKeysignFailure, id), &record)
	return record, err
}
