package keeperv1

import "gitlab.com/thorchain/thornode/common/cosmos"

// SetTssVoter - save a tss voter object
func (k KVStore) SetTssVoter(ctx cosmos.Context, tss TssVoter) {
	k.set(ctx, k.GetKey(ctx, prefixTss, tss.String()), tss)
}

// GetTssVoterIterator iterate tx in voters
func (k KVStore) GetTssVoterIterator(ctx cosmos.Context) cosmos.Iterator {
	return k.getIterator(ctx, prefixTss)
}

// GetTss - gets information of a tx hash
func (k KVStore) GetTssVoter(ctx cosmos.Context, id string) (TssVoter, error) {
	record := TssVoter{ID: id}
	_, err := k.get(ctx, k.GetKey(ctx, prefixTss, id), &record)
	return record, err
}
