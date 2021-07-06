package keeperv1

import (
	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

// SetErrataTxVoter - save a errata voter object
func (k KVStore) SetErrataTxVoter(ctx cosmos.Context, errata ErrataTxVoter) {
	k.set(ctx, k.GetKey(ctx, prefixErrataTx, errata.String()), errata)
}

// GetErrataTxVoterIterator iterate errata tx voter
func (k KVStore) GetErrataTxVoterIterator(ctx cosmos.Context) cosmos.Iterator {
	return k.getIterator(ctx, prefixErrataTx)
}

// GetErrataTxVoter - gets information of errata tx voter
func (k KVStore) GetErrataTxVoter(ctx cosmos.Context, txID common.TxID, chain common.Chain) (ErrataTxVoter, error) {
	record := NewErrataTxVoter(txID, chain)
	_, err := k.get(ctx, k.GetKey(ctx, prefixErrataTx, record.String()), &record)
	return record, err
}
