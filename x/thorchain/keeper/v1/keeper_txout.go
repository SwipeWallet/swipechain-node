package keeperv1

import (
	"strconv"

	"gitlab.com/thorchain/thornode/common/cosmos"
)

// AppendTxOut - append the given item to txOut
func (k KVStore) AppendTxOut(ctx cosmos.Context, height int64, item *TxOutItem) error {
	block, err := k.GetTxOut(ctx, height)
	if err != nil {
		return err
	}
	block.TxArray = append(block.TxArray, item)
	return k.SetTxOut(ctx, block)
}

// ClearTxOut - remove the txout of the given height from key value  store
func (k KVStore) ClearTxOut(ctx cosmos.Context, height int64) error {
	k.del(ctx, k.GetKey(ctx, prefixTxOut, strconv.FormatInt(height, 10)))
	return nil
}

// SetTxOut - write the given txout information to key value store
func (k KVStore) SetTxOut(ctx cosmos.Context, blockOut *TxOut) error {
	if blockOut == nil || blockOut.IsEmpty() {
		return nil
	}
	k.set(ctx, k.GetKey(ctx, prefixTxOut, strconv.FormatInt(blockOut.Height, 10)), blockOut)
	return nil
}

// GetTxOutIterator iterate tx out
func (k KVStore) GetTxOutIterator(ctx cosmos.Context) cosmos.Iterator {
	return k.getIterator(ctx, prefixTxOut)
}

// GetTxOut - write the given txout information to key values tore
func (k KVStore) GetTxOut(ctx cosmos.Context, height int64) (*TxOut, error) {
	record := NewTxOut(height)
	_, err := k.get(ctx, k.GetKey(ctx, prefixTxOut, strconv.FormatInt(height, 10)), &record)
	return record, err
}
