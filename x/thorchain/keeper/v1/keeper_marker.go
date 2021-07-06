package keeperv1

import (
	"errors"
	"fmt"
	"strings"

	"gitlab.com/thorchain/thornode/common/cosmos"
)

// ListTxMarker get all tx marker related to the given hash
func (k KVStore) ListTxMarker(ctx cosmos.Context, hash string) (TxMarkers, error) {
	record := make(TxMarkers, 0)
	_, err := k.get(ctx, k.GetKey(ctx, prefixSupportedTxMarker, hash), &record)
	return record, err
}

// SetTxMarkers save the given tx markers again the given hash
func (k KVStore) SetTxMarkers(ctx cosmos.Context, hash string, orig TxMarkers) error {
	marks := make(TxMarkers, 0)
	for _, mark := range orig {
		if !mark.IsEmpty() {
			marks = append(marks, mark)
		}
	}

	k.set(ctx, k.GetKey(ctx, prefixSupportedTxMarker, hash), marks)
	return nil
}

// AppendTxMarker append the given tx marker to store
func (k KVStore) AppendTxMarker(ctx cosmos.Context, hash string, mark TxMarker) error {
	if mark.IsEmpty() {
		return dbError(ctx, "unable to save tx marker:", errors.New("is empty"))
	}
	marks, err := k.ListTxMarker(ctx, hash)
	if err != nil {
		return err
	}
	marks = append(marks, mark)
	k.set(ctx, k.GetKey(ctx, prefixSupportedTxMarker, hash), marks)
	return nil
}

// GetAllTxMarkers get all tx markers from key value store
func (k KVStore) GetAllTxMarkers(ctx cosmos.Context) (map[string]TxMarkers, error) {
	result := make(map[string]TxMarkers)
	iter := k.getIterator(ctx, prefixSupportedTxMarker)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var marker TxMarkers
		if err := k.cdc.UnmarshalBinaryBare(iter.Value(), &marker); err != nil {
			return nil, fmt.Errorf("fail to unmarshal tx marker: %w", err)
		}

		strKey := string(iter.Key())
		k := strings.TrimPrefix(strKey, string(prefixSupportedTxMarker+"/"))
		result[k] = marker
	}
	return result, nil
}
