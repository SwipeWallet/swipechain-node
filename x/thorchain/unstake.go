package thorchain

import (
	"errors"
	"fmt"

	"github.com/blang/semver"

	"gitlab.com/thorchain/thornode/common"
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
	keeper "gitlab.com/thorchain/thornode/x/thorchain/keeper"
)

func validateUnstake(ctx cosmos.Context, keeper keeper.Keeper, msg MsgUnStake) error {
	if msg.RuneAddress.IsEmpty() {
		return errors.New("empty rune address")
	}
	if msg.Tx.ID.IsEmpty() {
		return errors.New("request tx hash is empty")
	}
	if msg.Asset.IsEmpty() {
		return errors.New("empty asset")
	}
	withdrawBasisPoints := msg.UnstakeBasisPoints
	if !withdrawBasisPoints.GTE(cosmos.ZeroUint()) || withdrawBasisPoints.GT(cosmos.NewUint(MaxUnstakeBasisPoints)) {
		return fmt.Errorf("withdraw basis points %s is invalid", msg.UnstakeBasisPoints)
	}
	if !keeper.PoolExist(ctx, msg.Asset) {
		// pool doesn't exist
		return fmt.Errorf("pool-%s doesn't exist", msg.Asset)
	}
	return nil
}

// unstake withdraw all the asset
// it returns runeAmt,assetAmount,units, lastUnstake,err
func unstake(ctx cosmos.Context, version semver.Version, keeper keeper.Keeper, msg MsgUnStake, eventManager EventManager) (cosmos.Uint, cosmos.Uint, cosmos.Uint, cosmos.Uint, error) {
	if err := validateUnstake(ctx, keeper, msg); err != nil {
		ctx.Logger().Error("msg unstake fail validation", "error", err)
		return cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), err
	}

	pool, err := keeper.GetPool(ctx, msg.Asset)
	if err != nil {
		ctx.Logger().Error("fail to get pool", "error", err)
		return cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), err
	}

	stakerUnit, err := keeper.GetStaker(ctx, msg.Asset, msg.RuneAddress)
	if err != nil {
		ctx.Logger().Error("can't find staker", "error", err)
		return cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), err

	}

	poolUnits := pool.PoolUnits
	poolRune := pool.BalanceRune
	poolAsset := pool.BalanceAsset
	fStakerUnit := stakerUnit.Units
	if stakerUnit.Units.IsZero() || msg.UnstakeBasisPoints.IsZero() {
		return cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), errNoStakeUnitLeft
	}

	cv := constants.GetConstantValues(version)
	height := common.BlockHeight(ctx)
	if height < (stakerUnit.LastStakeHeight + cv.GetInt64Value(constants.StakeLockUpBlocks)) {
		return cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), errUnstakeWithin24Hours
	}

	ctx.Logger().Info("pool before unstake", "pool unit", poolUnits, "balance RUNE", poolRune, "balance asset", poolAsset)
	ctx.Logger().Info("staker before withdraw", "staker unit", fStakerUnit)
	withdrawRune, withDrawAsset, unitAfter, err := calculateUnstake(poolUnits, poolRune, poolAsset, fStakerUnit, msg.UnstakeBasisPoints)
	if err != nil {
		ctx.Logger().Error("fail to unstake", "error", err)
		return cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), errUnstakeFail
	}
	gasAsset := cosmos.ZeroUint()
	// If the pool is empty, and there is a gas asset, subtract required gas
	if common.SafeSub(poolUnits, fStakerUnit).Add(unitAfter).IsZero() {
		// minus gas costs for our transactions
		// TODO: chain specific logic should be in a single location
		if pool.Asset.IsBNB() {
			gasInfo, err := keeper.GetGas(ctx, pool.Asset)
			if err != nil {
				ctx.Logger().Error("fail to get gas for asset", "asset", pool.Asset, "error", err)
				return cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), errUnstakeFail
			}
			originalAsset := withDrawAsset
			multiplier := uint64(2)
			if common.RuneAsset().Chain.Equals(common.THORChain) {
				multiplier = 1
			}
			withDrawAsset = common.SafeSub(
				withDrawAsset,
				gasInfo[0].MulUint64(multiplier),
			)
			gasAsset = originalAsset.Sub(withDrawAsset)
		} else if pool.Asset.Chain.GetGasAsset().Equals(pool.Asset) {
			// leave half a RUNE as gas fee for BTC chain and ETH chain
			transactionFee := cv.GetInt64Value(constants.TransactionFee)
			gasAsset = pool.RuneValueInAsset(cosmos.NewUint(uint64(transactionFee / 2)))
			if gasAsset.GT(withDrawAsset) {
				gasAsset = withDrawAsset
			}
			withDrawAsset = common.SafeSub(withDrawAsset, gasAsset)
		}
	}

	withdrawRune = withdrawRune.Add(stakerUnit.PendingRune) // extract pending rune
	stakerUnit.PendingRune = cosmos.ZeroUint()              // reset pending to zero

	ctx.Logger().Info("client withdraw", "RUNE", withdrawRune, "asset", withDrawAsset, "units left", unitAfter)
	// update pool
	pool.PoolUnits = common.SafeSub(poolUnits, fStakerUnit).Add(unitAfter)
	pool.BalanceRune = common.SafeSub(poolRune, withdrawRune)
	pool.BalanceAsset = common.SafeSub(poolAsset, withDrawAsset)

	ctx.Logger().Info("pool after unstake", "pool unit", pool.PoolUnits, "balance RUNE", pool.BalanceRune, "balance asset", pool.BalanceAsset)
	// update staker
	stakerUnit.Units = unitAfter
	stakerUnit.LastUnStakeHeight = common.BlockHeight(ctx)

	// Create a pool event if THORNode have no rune or assets
	if pool.BalanceAsset.IsZero() || pool.BalanceRune.IsZero() {
		poolEvt := NewEventPool(pool.Asset, PoolBootstrap)
		if err := eventManager.EmitEvent(ctx, poolEvt); nil != err {
			ctx.Logger().Error("fail to emit pool event", "error", err)
		}
		pool.Status = PoolBootstrap
	}

	if err := keeper.SetPool(ctx, pool); err != nil {
		return cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), ErrInternal(err, "fail to save pool")
	}
	if keeper.RagnarokInProgress(ctx) {
		keeper.SetStaker(ctx, stakerUnit)
	} else {
		if !stakerUnit.Units.IsZero() {
			keeper.SetStaker(ctx, stakerUnit)
		} else {
			keeper.RemoveStaker(ctx, stakerUnit)
		}
	}
	return withdrawRune, withDrawAsset, common.SafeSub(fStakerUnit, unitAfter), gasAsset, nil
}

func calculateUnstake(poolUnits, poolRune, poolAsset, stakerUnits, withdrawBasisPoints cosmos.Uint) (cosmos.Uint, cosmos.Uint, cosmos.Uint, error) {
	if poolUnits.IsZero() {
		return cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), errors.New("poolUnits can't be zero")
	}
	if poolRune.IsZero() {
		return cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), errors.New("pool rune balance can't be zero")
	}
	if poolAsset.IsZero() {
		return cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), errors.New("pool asset balance can't be zero")
	}
	if stakerUnits.IsZero() {
		return cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), errors.New("staker unit can't be zero")
	}
	if withdrawBasisPoints.GT(cosmos.NewUint(MaxUnstakeBasisPoints)) {
		return cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), fmt.Errorf("withdraw basis point %s is not valid", withdrawBasisPoints.String())
	}

	unitsToClaim := common.GetShare(withdrawBasisPoints, cosmos.NewUint(10000), stakerUnits)
	withdrawRune := common.GetShare(unitsToClaim, poolUnits, poolRune)
	withdrawAsset := common.GetShare(unitsToClaim, poolUnits, poolAsset)
	unitAfter := common.SafeSub(stakerUnits, unitsToClaim)
	return withdrawRune, withdrawAsset, unitAfter, nil
}
