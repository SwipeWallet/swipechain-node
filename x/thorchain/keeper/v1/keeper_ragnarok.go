package keeperv1

import (
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/x/thorchain/keeper/types"
)

// RagnarokInProgress return true only when Ragnarok is happening, when Ragnarok block height is not 0
func (k KVStore) RagnarokInProgress(ctx cosmos.Context) bool {
	height, err := k.GetRagnarokBlockHeight(ctx)
	if err != nil {
		ctx.Logger().Error("fail to get ragnarok block height", "error", err)
		return true
	}
	return height > 0
}

// getRagnarokValue - fetches the ragnarok value at given prefix
func (k KVStore) getRagnarokValue(ctx cosmos.Context, prefix types.DbPrefix) (int64, error) {
	record := int64(0)
	_, err := k.get(ctx, k.GetKey(ctx, prefix, ""), &record)
	return record, err
}

// GetRagnarokBlockHeight get ragnarok block height from key value store
func (k KVStore) GetRagnarokBlockHeight(ctx cosmos.Context) (int64, error) {
	return k.getRagnarokValue(ctx, prefixRagnarokHeight)
}

// SetRagnarokBlockHeight save ragnarok block height to key value store, once it get set , it means ragnarok started
func (k KVStore) SetRagnarokBlockHeight(ctx cosmos.Context, height int64) {
	k.set(ctx, k.GetKey(ctx, prefixRagnarokHeight, ""), height)
}

// GetRagnarokNth when ragnarok get triggered , THORNode will use a few rounds to refund all assets
// this method return which round it is in
func (k KVStore) GetRagnarokNth(ctx cosmos.Context) (int64, error) {
	return k.getRagnarokValue(ctx, prefixRagnarokNth)
}

// SetRagnarokNth save the round number into key value store
func (k KVStore) SetRagnarokNth(ctx cosmos.Context, nth int64) {
	k.set(ctx, k.GetKey(ctx, prefixRagnarokNth, ""), nth)
}

// GetRagnarokPending get ragnarok pending state from key value store
func (k KVStore) GetRagnarokPending(ctx cosmos.Context) (int64, error) {
	return k.getRagnarokValue(ctx, prefixRagnarokPending)
}

// SetRagnarokPending save ragnarok pending to key value store
func (k KVStore) SetRagnarokPending(ctx cosmos.Context, pending int64) {
	k.set(ctx, k.GetKey(ctx, prefixRagnarokPending, ""), pending)
}

// GetRagnarokUnstakPosition get ragnarok unstaking position
func (k KVStore) GetRagnarokUnstakPosition(ctx cosmos.Context) (RagnarokUnstakePosition, error) {
	record := RagnarokUnstakePosition{}
	_, err := k.get(ctx, k.GetKey(ctx, prefixRagnarokPosition, ""), &record)
	return record, err
}

// SetRagnarokUnstakPosition set ragnarok unstake position
func (k KVStore) SetRagnarokUnstakPosition(ctx cosmos.Context, position RagnarokUnstakePosition) {
	k.set(ctx, k.GetKey(ctx, prefixRagnarokPosition, ""), position)
}
