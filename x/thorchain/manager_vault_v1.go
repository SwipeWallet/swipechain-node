package thorchain

import (
	"errors"
	"fmt"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
	"gitlab.com/thorchain/thornode/x/thorchain/keeper"
)

// const values used to emit events
const (
	EventTypeActiveVault   = "ActiveVault"
	EventTypeInactiveVault = "InactiveVault"
)

// VaultMgrV1 is going to manage the vaults
type VaultMgrV1 struct {
	k          keeper.Keeper
	txOutStore TxOutStore
	eventMgr   EventManager
}

// NewVaultMgrV1 create a new vault manager
func NewVaultMgrV1(k keeper.Keeper, txOutStore TxOutStore, eventMgr EventManager) *VaultMgrV1 {
	return &VaultMgrV1{
		k:          k,
		txOutStore: txOutStore,
		eventMgr:   eventMgr,
	}
}

func (vm *VaultMgrV1) processGenesisSetup(ctx cosmos.Context) error {
	if common.BlockHeight(ctx) != genesisBlockHeight {
		return nil
	}
	vaults, err := vm.k.GetAsgardVaults(ctx)
	if err != nil {
		return fmt.Errorf("fail to get vaults: %w", err)
	}
	if len(vaults) > 0 {
		ctx.Logger().Info("already have vault, no need to generate at genesis")
		return nil
	}
	active, err := vm.k.ListActiveNodeAccounts(ctx)
	if err != nil {
		return fmt.Errorf("fail to get all active node accounts")
	}
	if len(active) == 0 {
		return errors.New("no active accounts,cannot proceed")
	}
	if len(active) == 1 {
		vault := NewVault(0, ActiveVault, AsgardVault, active[0].PubKeySet.Secp256k1, common.Chains{common.RuneAsset().Chain})
		vault.Membership = common.PubKeys{active[0].PubKeySet.Secp256k1}
		if err := vm.k.SetVault(ctx, vault); err != nil {
			return fmt.Errorf("fail to save vault: %w", err)
		}
	} else {
		// Trigger a keygen ceremony
		if err := vm.TriggerKeygen(ctx, active); err != nil {
			return fmt.Errorf("fail to trigger a keygen: %w", err)
		}
	}
	return nil
}

// EndBlock move funds from retiring asgard vaults
func (vm *VaultMgrV1) EndBlock(ctx cosmos.Context, mgr Manager, constAccessor constants.ConstantValues) error {
	if common.BlockHeight(ctx) == genesisBlockHeight {
		return vm.processGenesisSetup(ctx)
	}

	migrateInterval, err := vm.k.GetMimir(ctx, constants.FundMigrationInterval.String())
	if migrateInterval < 0 || err != nil {
		migrateInterval = constAccessor.GetInt64Value(constants.FundMigrationInterval)
	}

	retiring, err := vm.k.GetAsgardVaultsByStatus(ctx, RetiringVault)
	if err != nil {
		return err
	}

	active, err := vm.k.GetAsgardVaultsByStatus(ctx, ActiveVault)
	if err != nil {
		return err
	}

	// if we have no active asgards to move funds to, don't move funds
	if len(active) == 0 {
		return nil
	}

	for _, vault := range retiring {
		if vault.LenPendingTxBlockHeights(common.BlockHeight(ctx), constAccessor) > 0 {
			ctx.Logger().Info("Skipping the migration of funds while transactions are still pending")
			return nil
		}
	}

	for _, vault := range retiring {
		if !vault.HasFunds() {
			vault.Status = InactiveVault
			if err := vm.k.SetVault(ctx, vault); err != nil {
				ctx.Logger().Error("fail to set vault to inactive", "error", err)
			}
			continue
		}

		// move partial funds every 30 minutes
		if (common.BlockHeight(ctx)-vault.StatusSince)%migrateInterval == 0 {
			for _, coin := range vault.Coins {
				if coin.IsNative() {
					continue
				}

				if coin.Amount.Equal(cosmos.ZeroUint()) {
					continue
				}

				// determine which active asgard vault is the best to send
				// these coins to. We target the vault with the least amount of
				// this particular coin
				cn := active[0].GetCoin(coin.Asset)
				pk := active[0].PubKey
				for _, asgard := range active {
					if cn.Amount.GT(asgard.GetCoin(coin.Asset).Amount) {
						cn = asgard.GetCoin(coin.Asset)
						pk = asgard.PubKey
					}
				}

				if pk.Equals(vault.PubKey) {
					continue
				}

				// get address of asgard pubkey
				addr, err := pk.GetAddress(coin.Asset.Chain)
				if err != nil {
					return err
				}

				// figure the nth time, we've sent migration txs from this vault
				nth := (common.BlockHeight(ctx)-vault.StatusSince)/migrateInterval + 1

				// Default amount set to total remaining amount. Relies on the
				// signer, to successfully send these funds while respecting
				// gas requirements (so it'll actually send slightly less)
				amt := coin.Amount
				if nth < 5 { // migrate partial funds 4 times
					// each round of migration, we are increasing the amount 20%.
					// Round 1 = 20%
					// Round 2 = 40%
					// Round 3 = 60%
					// Round 4 = 80%
					// Round 5 = 100%
					amt = amt.MulUint64(uint64(nth)).QuoUint64(5)
				}

				// TODO: make this not chain specific
				// minus gas costs for our transactions
				if coin.Asset.IsBNB() {
					gasInfo, err := vm.k.GetGas(ctx, coin.Asset)
					if err != nil {
						ctx.Logger().Error("fail to get gas for asset", "asset", coin.Asset, "error", err)
						return err
					}
					amt = common.SafeSub(
						amt,
						gasInfo[0].MulUint64(uint64(vault.CoinLength())),
					)
				}

				toi := &TxOutItem{
					Chain:       coin.Asset.Chain,
					InHash:      common.BlankTxID,
					ToAddress:   addr,
					VaultPubKey: vault.PubKey,
					Coin: common.Coin{
						Asset:  coin.Asset,
						Amount: amt,
					},
					Memo: NewMigrateMemo(common.BlockHeight(ctx)).String(),
				}
				ok, err := vm.txOutStore.TryAddTxOutItem(ctx, mgr, toi)
				if err != nil && !errors.Is(err, ErrNotEnoughToPayFee) {
					return err
				}
				if ok {
					vault.AppendPendingTxBlockHeights(common.BlockHeight(ctx), constAccessor)
					if err := vm.k.SetVault(ctx, vault); err != nil {
						return fmt.Errorf("fail to save vault: %w", err)
					}
				}
			}
		}
	}

	if common.BlockHeight(ctx)%migrateInterval == 0 {
		// checks to see if we need to ragnarok a chain, and ragnarok them (if needed)
		if err := vm.manageChains(ctx, mgr, constAccessor); err != nil {
			return err
		}
	}
	return nil
}

// TriggerKeygen generate a record to instruct signer kick off keygen process
func (vm *VaultMgrV1) TriggerKeygen(ctx cosmos.Context, nas NodeAccounts) error {
	halt, err := vm.k.GetMimir(ctx, "HaltChurning")
	if halt > 0 && halt <= common.BlockHeight(ctx) && err == nil {
		ctx.Logger().Info("churn event skipped due to mimir has halted churning")
		return nil
	}
	var members common.PubKeys
	for i := range nas {
		members = append(members, nas[i].PubKeySet.Secp256k1)
	}
	keygen, err := NewKeygen(common.BlockHeight(ctx), members, AsgardKeygen)
	if err != nil {
		return fmt.Errorf("fail to create a new keygen: %w", err)
	}
	keygenBlock, err := vm.k.GetKeygenBlock(ctx, common.BlockHeight(ctx))
	if err != nil {
		return fmt.Errorf("fail to get keygen block from data store: %w", err)
	}

	if !keygenBlock.Contains(keygen) {
		keygenBlock.Keygens = append(keygenBlock.Keygens, keygen)
	}
	vm.k.SetKeygenBlock(ctx, keygenBlock)
	return nil
}

// RotateVault update vault to Retiring and new vault to active
func (vm *VaultMgrV1) RotateVault(ctx cosmos.Context, vault Vault) error {
	active, err := vm.k.GetAsgardVaultsByStatus(ctx, ActiveVault)
	if err != nil {
		return err
	}

	// find vaults the new vault conflicts with, mark them as inactive
	for _, asgard := range active {
		for _, member := range asgard.Membership {
			if vault.Contains(member) {
				asgard.UpdateStatus(RetiringVault, common.BlockHeight(ctx))
				if err := vm.k.SetVault(ctx, asgard); err != nil {
					return err
				}

				ctx.EventManager().EmitEvent(
					cosmos.NewEvent(EventTypeInactiveVault,
						cosmos.NewAttribute("set asgard vault to inactive", asgard.PubKey.String())))
				break
			}
		}
	}

	// Update Node account membership
	for _, member := range vault.Membership {
		na, err := vm.k.GetNodeAccountByPubKey(ctx, member)
		if err != nil {
			return err
		}
		na.TryAddSignerPubKey(vault.PubKey)
		if err := vm.k.SetNodeAccount(ctx, na); err != nil {
			return err
		}
	}

	if err := vm.k.SetVault(ctx, vault); err != nil {
		return err
	}

	ctx.EventManager().EmitEvent(
		cosmos.NewEvent(EventTypeActiveVault,
			cosmos.NewAttribute("add new asgard vault", vault.PubKey.String())))
	return nil
}

// manageChains - checks to see if we have any chains that we are ragnaroking,
// and ragnaroks them
func (vm *VaultMgrV1) manageChains(ctx cosmos.Context, mgr Manager, constAccessor constants.ConstantValues) error {
	chains, err := vm.findChainsToRetire(ctx)
	if err != nil {
		return err
	}

	active, err := vm.k.GetAsgardVaultsByStatus(ctx, ActiveVault)
	if err != nil {
		return err
	}
	vault := active.SelectByMinCoin(common.RuneAsset())
	if vault.IsEmpty() {
		return fmt.Errorf("unable to determine asgard vault")
	}

	migrateInterval, err := vm.k.GetMimir(ctx, constants.FundMigrationInterval.String())
	if migrateInterval < 0 || err != nil {
		migrateInterval = constAccessor.GetInt64Value(constants.FundMigrationInterval)
	}
	nth := (common.BlockHeight(ctx)-vault.StatusSince)/migrateInterval + 1
	if nth > 10 {
		nth = 10
	}

	for _, chain := range chains {
		if err := vm.recallChainFunds(ctx, chain, mgr); err != nil {
			return err
		}

		// only refund after the first nth. This gives yggs time to send funds
		// back to asgard
		if nth > 1 {
			if err := vm.ragnarokChain(ctx, chain, nth, mgr, constAccessor); err != nil {
				continue
			}
		}
	}
	return nil
}

// findChainsToRetire - evaluates the chains associated with active asgard
// vaults vs retiring asgard vaults to detemine if any chains need to be
// ragnarok'ed
func (vm *VaultMgrV1) findChainsToRetire(ctx cosmos.Context) (common.Chains, error) {
	chains := make(common.Chains, 0)

	active, err := vm.k.GetAsgardVaultsByStatus(ctx, ActiveVault)
	if err != nil {
		return chains, err
	}
	retiring, err := vm.k.GetAsgardVaultsByStatus(ctx, RetiringVault)
	if err != nil {
		return chains, err
	}

	// collect all chains for active vaults
	activeChains := make(common.Chains, 0)
	for _, v := range active {
		activeChains = append(activeChains, v.Chains...)
	}
	activeChains = activeChains.Distinct()

	// collect all chains for retiring vaults
	retiringChains := make(common.Chains, 0)
	for _, v := range retiring {
		retiringChains = append(retiringChains, v.Chains...)
	}
	retiringChains = retiringChains.Distinct()

	for _, chain := range retiringChains {
		// skip chain if its in active and retiring
		if activeChains.Has(chain) {
			continue
		}
		chains = append(chains, chain)
	}
	return chains, nil
}

// recallChainFunds - sends a message to bifrost nodes to send back all funds
// associated with given chain
func (vm *VaultMgrV1) recallChainFunds(ctx cosmos.Context, chain common.Chain, mgr Manager) error {
	allNodes, err := vm.k.ListNodeAccountsWithBond(ctx)
	if err != nil {
		return fmt.Errorf("fail to list all node accounts: %w", err)
	}

	active, err := vm.k.GetAsgardVaultsByStatus(ctx, ActiveVault)
	if err != nil {
		return err
	}

	vault := active.SelectByMinCoin(common.RuneAsset())
	if vault.IsEmpty() {
		return fmt.Errorf("unable to determine asgard vault")
	}
	toAddr, err := vault.PubKey.GetAddress(chain)
	if err != nil {
		return err
	}

	// get yggdrasil to return funds back to asgard
	for _, node := range allNodes {
		if !vm.k.VaultExists(ctx, node.PubKeySet.Secp256k1) {
			continue
		}
		ygg, err := vm.k.GetVault(ctx, node.PubKeySet.Secp256k1)
		if err != nil {
			ctx.Logger().Error("fail to get ygg vault", "error", err)
			continue
		}
		if ygg.IsAsgard() {
			continue
		}

		if !ygg.HasFundsForChain(chain) {
			continue
		}

		if !toAddr.IsEmpty() {
			txOutItem := &TxOutItem{
				Chain:       chain,
				ToAddress:   toAddr,
				InHash:      common.BlankTxID,
				VaultPubKey: ygg.PubKey,
				Coin:        common.NewCoin(common.RuneAsset(), cosmos.ZeroUint()),
				Memo:        NewYggdrasilReturn(common.BlockHeight(ctx)).String(),
			}
			// yggdrasil- will not set coin field here, when signer see a
			// TxOutItem that has memo "yggdrasil-" it will query the chain
			// and find out all the remaining assets , and fill in the
			// field
			if err := vm.txOutStore.UnSafeAddTxOutItem(ctx, mgr, txOutItem); err != nil {
				return err
			}
		}
	}

	return nil
}

// ragnarokChain - ends a chain by unstaking all stakers of any pool that's
// asset is on the given chain
func (vm *VaultMgrV1) ragnarokChain(ctx cosmos.Context, chain common.Chain, nth int64, mgr Manager, constAccessor constants.ConstantValues) error {
	version := vm.k.GetLowestActiveVersion(ctx)
	nas, err := vm.k.ListActiveNodeAccounts(ctx)
	if err != nil {
		ctx.Logger().Error("can't get active nodes", "error", err)
		return err
	}
	if len(nas) == 0 {
		return fmt.Errorf("can't find any active nodes")
	}
	na := nas[0]

	pools, err := vm.k.GetPools(ctx)
	if err != nil {
		return err
	}
	unstakeHandler := NewUnstakeHandler(vm.k, mgr)

	active, err := vm.k.GetAsgardVaultsByStatus(ctx, ActiveVault)
	if err != nil {
		return err
	}
	vault := active.SelectByMinCoin(common.RuneAsset())
	if vault.IsEmpty() {
		return fmt.Errorf("unable to determine asgard vault")
	}

	// rangarok this chain
	for _, pool := range pools {
		if !pool.Asset.Chain.Equals(chain) || pool.PoolUnits.IsZero() {
			continue
		}
		iterator := vm.k.GetStakerIterator(ctx, pool.Asset)
		defer iterator.Close()
		for ; iterator.Valid(); iterator.Next() {
			var staker Staker
			vm.k.Cdc().MustUnmarshalBinaryBare(iterator.Value(), &staker)
			if staker.Units.IsZero() {
				continue
			}

			unstakeMsg := NewMsgUnStake(
				common.GetRagnarokTx(pool.Asset.Chain, staker.RuneAddress, staker.RuneAddress),
				staker.RuneAddress,
				cosmos.NewUint(uint64(MaxUnstakeBasisPoints/100*(nth*10))),
				pool.Asset,
				na.NodeAddress,
			)

			_, err := unstakeHandler.Run(ctx, unstakeMsg, version, constAccessor)
			if err != nil {
				ctx.Logger().Error("fail to unstake", "staker", staker.RuneAddress, "error", err)
			}
		}
	}

	return nil
}

// UpdateVaultData Update the vault data to reflect changing in this block
func (vm *VaultMgrV1) UpdateVaultData(ctx cosmos.Context, constAccessor constants.ConstantValues, gasManager GasManager, eventMgr EventManager) error {
	vaultData, err := vm.k.GetVaultData(ctx)
	if err != nil {
		return fmt.Errorf("fail to get existing vault data: %w", err)
	}

	totalReserve := cosmos.ZeroUint()
	if common.RuneAsset().Chain.Equals(common.THORChain) {
		totalReserve = vm.k.GetRuneBalanceOfModule(ctx, ReserveName)
	} else {
		totalReserve = vaultData.TotalReserve
	}

	// when total reserve is zero , can't pay reward
	if totalReserve.IsZero() {
		return nil
	}
	currentHeight := uint64(common.BlockHeight(ctx))
	pools, totalStaked, err := vm.getTotalStakedRune(ctx)
	if err != nil {
		return fmt.Errorf("fail to get enabled pools and total staked rune: %w", err)
	}

	// If no Rune is staked, then don't give out block rewards.
	if totalStaked.IsZero() {
		return nil // If no Rune is staked, then don't give out block rewards.
	}

	// get total liquidity fees
	totalLiquidityFees, err := vm.k.GetTotalLiquidityFees(ctx, currentHeight)
	if err != nil {
		return fmt.Errorf("fail to get total liquidity fee: %w", err)
	}

	// NOTE: if we continue to have remaining gas to pay off (which is
	// extremely unlikely), ignore it for now (attempt to recover in the next
	// block). This should be OK as the asset amount in the pool has already
	// been deducted so the balances are correct. Just operating at a deficit.
	totalBonded, err := vm.getTotalActiveBond(ctx)
	if err != nil {
		return fmt.Errorf("fail to get total active bond: %w", err)
	}

	emissionCurve, err := vm.k.GetMimir(ctx, constants.EmissionCurve.String())
	if emissionCurve < 0 || err != nil {
		emissionCurve = constAccessor.GetInt64Value(constants.EmissionCurve)
	}
	blocksOerYear := constAccessor.GetInt64Value(constants.BlocksPerYear)
	bondReward, totalPoolRewards, stakerDeficit := vm.calcBlockRewards(totalStaked, totalBonded, totalReserve, totalLiquidityFees, emissionCurve, blocksOerYear)

	// given bondReward and toolPoolRewards are both calculated base on totalReserve, thus it should always have enough to pay the bond reward

	// Move Rune from the Reserve to the Bond and Pool Rewards
	totalRewards := bondReward.Add(totalPoolRewards)
	if totalRewards.GT(totalReserve) {
		totalRewards = totalReserve
	}
	totalReserve = common.SafeSub(totalReserve, totalRewards)
	if common.RuneAsset().Chain.Equals(common.THORChain) {
		coin := common.NewCoin(common.RuneNative, bondReward)
		if err := vm.k.SendFromModuleToModule(ctx, ReserveName, BondName, coin); err != nil {
			ctx.Logger().Error("fail to transfer funds from reserve to bond", "error", err)
			return fmt.Errorf("fail to transfer funds from reserve to bond: %w", err)
		}
	} else {
		vaultData.TotalReserve = totalReserve
	}
	vaultData.BondRewardRune = vaultData.BondRewardRune.Add(bondReward) // Add here for individual Node collection later

	var evtPools []PoolAmt

	if !totalPoolRewards.IsZero() { // If Pool Rewards to hand out

		var rewardAmts []cosmos.Uint
		var rewardPools []Pool
		// Pool Rewards are based on Fee Share
		for _, pool := range pools {
			if !pool.IsEnabled() {
				continue
			}
			amt := cosmos.ZeroUint()
			if totalLiquidityFees.IsZero() {
				amt = common.GetShare(pool.BalanceRune, totalStaked, totalPoolRewards)
			} else {
				fees, err := vm.k.GetPoolLiquidityFees(ctx, currentHeight, pool.Asset)
				if err != nil {
					ctx.Logger().Error("fail to get fees", "error", err)
					continue
				}
				amt = common.GetShare(fees, totalLiquidityFees, totalPoolRewards)
			}
			rewardAmts = append(rewardAmts, amt)
			evtPools = append(evtPools, PoolAmt{Asset: pool.Asset, Amount: int64(amt.Uint64())})
			rewardPools = append(rewardPools, pool)
		}
		// Pay out
		if err := vm.payPoolRewards(ctx, rewardAmts, rewardPools); err != nil {
			return err
		}

	} else { // Else deduct pool deficit

		for _, pool := range pools {
			if !pool.IsEnabled() {
				continue
			}
			poolFees, err := vm.k.GetPoolLiquidityFees(ctx, currentHeight, pool.Asset)
			if err != nil {
				return fmt.Errorf("fail to get liquidity fees for pool(%s): %w", pool.Asset, err)
			}
			if pool.BalanceRune.IsZero() || poolFees.IsZero() { // Safety checks
				continue
			}
			poolDeficit := vm.calcPoolDeficit(stakerDeficit, totalLiquidityFees, poolFees)
			if common.RuneAsset().Chain.Equals(common.THORChain) {
				coin := common.NewCoin(common.RuneNative, poolDeficit)
				if err := vm.k.SendFromModuleToModule(ctx, AsgardName, BondName, coin); err != nil {
					ctx.Logger().Error("fail to transfer funds from asgard to bond", "error", err)
					return fmt.Errorf("fail to transfer funds from asgard to bond: %w", err)
				}
			}
			if poolDeficit.GT(pool.BalanceRune) {
				poolDeficit = pool.BalanceRune
			}
			pool.BalanceRune = common.SafeSub(pool.BalanceRune, poolDeficit)
			vaultData.BondRewardRune = vaultData.BondRewardRune.Add(poolDeficit)
			if err := vm.k.SetPool(ctx, pool); err != nil {
				return fmt.Errorf("fail to set pool: %w", err)
			}
			evtPools = append(evtPools, PoolAmt{
				Asset:  pool.Asset,
				Amount: 0 - int64(poolDeficit.Uint64()),
			})
		}
	}

	rewardEvt := NewEventRewards(bondReward, evtPools)
	if err := eventMgr.EmitEvent(ctx, rewardEvt); err != nil {
		return fmt.Errorf("fail to emit reward event: %w", err)
	}
	i, err := getTotalActiveNodeWithBond(ctx, vm.k)
	if err != nil {
		return fmt.Errorf("fail to get total active node account: %w", err)
	}
	vaultData.TotalBondUnits = vaultData.TotalBondUnits.Add(cosmos.NewUint(uint64(i))) // Add 1 unit for each active Node

	return vm.k.SetVaultData(ctx, vaultData)
}

func (vm *VaultMgrV1) getTotalStakedRune(ctx cosmos.Context) (Pools, cosmos.Uint, error) {
	// First get active pools and total staked Rune
	totalStaked := cosmos.ZeroUint()
	var pools Pools
	iterator := vm.k.GetPoolIterator(ctx)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var pool Pool
		if err := vm.k.Cdc().UnmarshalBinaryBare(iterator.Value(), &pool); err != nil {
			return nil, cosmos.ZeroUint(), fmt.Errorf("fail to unmarhsl pool: %w", err)
		}
		if !pool.BalanceRune.IsZero() {
			totalStaked = totalStaked.Add(pool.BalanceRune)
			pools = append(pools, pool)
		}
	}
	return pools, totalStaked, nil
}

func (vm *VaultMgrV1) getTotalActiveBond(ctx cosmos.Context) (cosmos.Uint, error) {
	totalBonded := cosmos.ZeroUint()
	nodes, err := vm.k.ListActiveNodeAccounts(ctx)
	if err != nil {
		return cosmos.ZeroUint(), fmt.Errorf("fail to get all active accounts: %w", err)
	}
	for _, node := range nodes {
		totalBonded = totalBonded.Add(node.Bond)
	}
	return totalBonded, nil
}

// Pays out Rewards
func (vm *VaultMgrV1) payPoolRewards(ctx cosmos.Context, poolRewards []cosmos.Uint, pools Pools) error {
	for i, reward := range poolRewards {
		pools[i].BalanceRune = pools[i].BalanceRune.Add(reward)
		if err := vm.k.SetPool(ctx, pools[i]); err != nil {
			return fmt.Errorf("fail to set pool: %w", err)
		}
		if common.RuneAsset().Chain.Equals(common.THORChain) {
			coin := common.NewCoin(common.RuneNative, reward)
			if err := vm.k.SendFromModuleToModule(ctx, ReserveName, AsgardName, coin); err != nil {
				return fmt.Errorf("fail to transfer funds from reserve to asgard: %w", err)
			}
		}
	}
	return nil
}

// Calculate pool deficit based on the pool's accrued fees compared with total fees.
func (vm *VaultMgrV1) calcPoolDeficit(stakerDeficit, totalFees, poolFees cosmos.Uint) cosmos.Uint {
	return common.GetShare(poolFees, totalFees, stakerDeficit)
}

// Calculate the block rewards that bonders and stakers should receive
func (vm *VaultMgrV1) calcBlockRewards(totalStaked, totalBonded, totalReserve, totalLiquidityFees cosmos.Uint, emissionCurve, blocksPerYear int64) (cosmos.Uint, cosmos.Uint, cosmos.Uint) {
	// Block Rewards will take the latest reserve, divide it by the emission
	// curve factor, then divide by blocks per year
	trD := cosmos.NewDec(int64(totalReserve.Uint64()))
	ecD := cosmos.NewDec(emissionCurve)
	bpyD := cosmos.NewDec(blocksPerYear)
	blockRewardD := trD.Quo(ecD).Quo(bpyD)
	blockReward := cosmos.NewUint(uint64((blockRewardD).RoundInt64()))

	systemIncome := blockReward.Add(totalLiquidityFees) // Get total system income for block

	stakerSplit := vm.getPoolShare(totalStaked, totalBonded, systemIncome) // Get staker share
	bonderSplit := common.SafeSub(systemIncome, stakerSplit)               // Remainder to Bonders

	stakerDeficit := cosmos.ZeroUint()
	poolReward := cosmos.ZeroUint()

	if stakerSplit.GTE(totalLiquidityFees) {
		// Stakers have not been paid enough already, pay more
		poolReward = common.SafeSub(stakerSplit, totalLiquidityFees) // Get how much to divert to add to staker split
	} else {
		// Stakers have been paid too much, calculate deficit
		stakerDeficit = common.SafeSub(totalLiquidityFees, stakerSplit) // Deduct existing income from split
	}

	return bonderSplit, poolReward, stakerDeficit
}

func (vm *VaultMgrV1) getPoolShare(totalStaked, totalBonded, totalRewards cosmos.Uint) cosmos.Uint {
	// Targets a linear change in rewards from 0% staked, 33% staked, 100% staked.
	// 0% staked: All rewards to stakers
	// 33% staked: 33% to stakers
	// 100% staked: All rewards to Bonders

	if totalStaked.GTE(totalBonded) { // Zero payments to stakers when staked == bonded
		return cosmos.ZeroUint()
	}
	factor := totalBonded.Add(totalStaked).Quo(common.SafeSub(totalBonded, totalStaked)) // (y + x) / (y - x)
	return totalRewards.Quo(factor)
}

func getTotalActiveNodeWithBond(ctx cosmos.Context, k keeper.Keeper) (int64, error) {
	nas, err := k.ListActiveNodeAccounts(ctx)
	if err != nil {
		return 0, fmt.Errorf("fail to get active node accounts: %w", err)
	}
	var total int64
	for _, item := range nas {
		if !item.Bond.IsZero() {
			total++
		}
	}
	return total, nil
}
