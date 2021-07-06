package thorchain

import (
	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/x/thorchain/keeper"
)

// GasMgrV1 implement GasManager interface which will store the gas related events happened in thorchain to memory
// emit GasEvent per block if there are any
type GasMgrV1 struct {
	gasEvent *EventGas
	gas      common.Gas
	gasCount map[common.Asset]int64
}

// NewGasMgrV1 create a new instance of GasMgrV1
func NewGasMgrV1() *GasMgrV1 {
	return &GasMgrV1{
		gasEvent: NewEventGas(),
		gas:      common.Gas{},
		gasCount: make(map[common.Asset]int64, 0),
	}
}

func (gm *GasMgrV1) reset() {
	gm.gasEvent = NewEventGas()
	gm.gas = common.Gas{}
	gm.gasCount = make(map[common.Asset]int64, 0)
}

// BeginBlock need to be called when a new block get created , update the internal EventGas to new one
func (gm *GasMgrV1) BeginBlock() {
	gm.reset()
}

// AddGasAsset to the EventGas
func (gm *GasMgrV1) AddGasAsset(gas common.Gas) {
	gm.gas = gm.gas.Add(gas)
	for _, coin := range gas {
		gm.gasCount[coin.Asset]++
	}
}

// GetGas return the gas
func (gm *GasMgrV1) GetGas() common.Gas {
	return gm.gas
}

// EndBlock emit the events
func (gm *GasMgrV1) EndBlock(ctx cosmos.Context, keeper keeper.Keeper, eventManager EventManager) {
	gm.ProcessGas(ctx, keeper)

	if len(gm.gasEvent.Pools) == 0 {
		return
	}

	if err := eventManager.EmitGasEvent(ctx, gm.gasEvent); nil != err {
		ctx.Logger().Error("fail to emit gas event", "error", err)
	}
	gm.reset() // do not remove, will cause consensus failures
}

// ProcessGas to subsidise the pool with RUNE for the gas they have spent
func (gm *GasMgrV1) ProcessGas(ctx cosmos.Context, keeper keeper.Keeper) {
	vault, err := keeper.GetVaultData(ctx)
	if err != nil {
		ctx.Logger().Error("fail to get vault data", "error", err)
		return
	}
	for _, gas := range gm.gas {
		// if the coin is zero amount, don't need to do anything
		if gas.Amount.IsZero() {
			continue
		}

		pool, err := keeper.GetPool(ctx, gas.Asset)
		if err != nil {
			ctx.Logger().Error("fail to get pool", "pool", gas.Asset, "error", err)
			continue
		}
		if err := pool.Valid(); err != nil {
			ctx.Logger().Error("invalid pool", "pool", gas.Asset, "error", err)
			continue
		}
		runeGas := pool.AssetValueInRune(gas.Amount) // Convert to Rune (gas will never be RUNE)
		// If Rune owed now exceeds the Total Reserve, return it all
		if common.RuneAsset().Chain.Equals(common.THORChain) {
			if runeGas.LT(keeper.GetRuneBalanceOfModule(ctx, ReserveName)) {
				coin := common.NewCoin(common.RuneNative, runeGas)
				if err := keeper.SendFromModuleToModule(ctx, ReserveName, AsgardName, coin); err != nil {
					ctx.Logger().Error("fail to transfer funds from reserve to asgard", "pool", gas.Asset, "error", err)
					continue
				}
				pool.BalanceRune = pool.BalanceRune.Add(runeGas) // Add to the pool
			}
		} else {
			if runeGas.LTE(vault.TotalReserve) {
				vault.TotalReserve = common.SafeSub(vault.TotalReserve, runeGas) // Deduct from the Reserve.
				pool.BalanceRune = pool.BalanceRune.Add(runeGas)                 // Add to the pool
			} else {
				// since we didn't move any funds from reserve to the pool, set
				// the runeGas to zero so we emit the gas event to reflect the
				// appropriate amount
				runeGas = cosmos.ZeroUint()
			}
		}
		pool.BalanceAsset = common.SafeSub(pool.BalanceAsset, gas.Amount)

		if err := keeper.SetPool(ctx, pool); err != nil {
			ctx.Logger().Error("fail to set pool", "pool", gas.Asset, "error", err)
			continue
		}

		gasPool := GasPool{
			Asset:    gas.Asset,
			AssetAmt: gas.Amount,
			RuneAmt:  runeGas,
			Count:    gm.gasCount[gas.Asset],
		}
		gm.gasEvent.UpsertGasPool(gasPool)
	}

	if err := keeper.SetVaultData(ctx, vault); err != nil {
		ctx.Logger().Error("fail to set vault data", "error", err)
	}
}
