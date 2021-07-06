package keeperv1

import "gitlab.com/thorchain/thornode/common/cosmos"

// SetBanVoter - save a ban voter object
func (k KVStore) SetBanVoter(ctx cosmos.Context, ban BanVoter) {
	k.set(ctx, k.GetKey(ctx, prefixBanVoter, ban.String()), ban)
}

// GetBanVoter - gets information of ban voter
func (k KVStore) GetBanVoter(ctx cosmos.Context, addr cosmos.AccAddress) (BanVoter, error) {
	record := NewBanVoter(addr)
	_, err := k.get(ctx, k.GetKey(ctx, prefixBanVoter, record.String()), &record)
	return record, err
}

// GetBanVoterIterator - get an iterator for ban voter
func (k KVStore) GetBanVoterIterator(ctx cosmos.Context) cosmos.Iterator {
	return k.getIterator(ctx, prefixBanVoter)
}
