package keeperv1

import (
	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

// SetObservedNetworkFeeVoter - save a observed network fee voter object
func (k KVStore) SetObservedNetworkFeeVoter(ctx cosmos.Context, networkFeeVoter ObservedNetworkFeeVoter) {
	k.set(ctx, k.GetKey(ctx, prefixNetworkFeeVoter, networkFeeVoter.String()), networkFeeVoter)
}

// GetObservedNetworkFeeVoterIterator iterate tx in voters
func (k KVStore) GetObservedNetworkFeeVoterIterator(ctx cosmos.Context) cosmos.Iterator {
	return k.getIterator(ctx, prefixNetworkFeeVoter)
}

// GetObservedNetworkFeeVoter - gets information of an observed network fee voter
func (k KVStore) GetObservedNetworkFeeVoter(ctx cosmos.Context, height int64, chain common.Chain) (ObservedNetworkFeeVoter, error) {
	record := NewObservedNetworkFeeVoter(height, chain)
	_, err := k.get(ctx, k.GetKey(ctx, prefixNetworkFeeVoter, record.String()), &record)
	return record, err
}
