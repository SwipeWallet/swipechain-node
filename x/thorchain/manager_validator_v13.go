package thorchain

import (
	"errors"
	"fmt"
	"net"
	"sort"

	abci "github.com/tendermint/tendermint/abci/types"
	tmtypes "github.com/tendermint/tendermint/types"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
	"gitlab.com/thorchain/thornode/x/thorchain/keeper"
)

// validatorMgrV13 is to manage a list of validators , and rotate them
type validatorMgrV13 struct {
	k          keeper.Keeper
	vaultMgr   VaultManager
	txOutStore TxOutStore
	eventMgr   EventManager
}

// newValidatorMgrV13 create a new instance of validatorMgrV1
func newValidatorMgrV13(k keeper.Keeper, vaultMgr VaultManager, txOutStore TxOutStore, eventMgr EventManager) *validatorMgrV13 {
	return &validatorMgrV13{
		k:          k,
		vaultMgr:   vaultMgr,
		txOutStore: txOutStore,
		eventMgr:   eventMgr,
	}
}

// BeginBlock when block begin
func (vm *validatorMgrV13) BeginBlock(ctx cosmos.Context, constAccessor constants.ConstantValues) error {
	height := common.BlockHeight(ctx)
	if height == genesisBlockHeight {
		if err := vm.setupValidatorNodes(ctx, height, constAccessor); err != nil {
			ctx.Logger().Error("fail to setup validator nodes", "error", err)
		}
	}
	if vm.k.RagnarokInProgress(ctx) {
		// ragnarok is in progress, no point to check node rotation
		return nil
	}
	minimumNodesForBFT := constAccessor.GetInt64Value(constants.MinimumNodesForBFT)
	totalActiveNodes, err := vm.k.TotalActiveNodeAccount(ctx)
	if err != nil {
		return err
	}

	rotatePerBlockHeight, err := vm.k.GetMimir(ctx, constants.RotatePerBlockHeight.String())
	if rotatePerBlockHeight < 0 || err != nil {
		rotatePerBlockHeight = constAccessor.GetInt64Value(constants.RotatePerBlockHeight)
	}

	// when total active nodes is more than MinimumNodesForBFT + 2, start to churn node in and out
	if minimumNodesForBFT+2 < int64(totalActiveNodes) {
		badValidatorRate, err := vm.k.GetMimir(ctx, constants.BadValidatorRate.String())
		if badValidatorRate < 0 || err != nil {
			badValidatorRate = constAccessor.GetInt64Value(constants.BadValidatorRate)
		}
		if err := vm.markBadActor(ctx, badValidatorRate); err != nil {
			return err
		}
		oldValidatorRate, err := vm.k.GetMimir(ctx, constants.OldValidatorRate.String())
		if oldValidatorRate < 0 || err != nil {
			oldValidatorRate = constAccessor.GetInt64Value(constants.OldValidatorRate)
		}
		if err := vm.markOldActor(ctx, oldValidatorRate); err != nil {
			return err
		}
		// when the active nodes didn't upgrade , boot them out one at a time
		if err := vm.markLowerVersion(ctx, rotatePerBlockHeight); err != nil {
			return err
		}
	}

	vaults, err := vm.k.GetAsgardVaultsByStatus(ctx, ActiveVault)
	if err != nil {
		return err
	}
	// calculate last churn block height
	var lastHeight int64 // the last block height we had a successful churn
	for _, vault := range vaults {
		if vault.BlockHeight > lastHeight {
			lastHeight = vault.BlockHeight
		}
	}

	// get constants
	desireValidatorSet, err := vm.k.GetMimir(ctx, constants.DesireValidatorSet.String())
	if desireValidatorSet < 0 || err != nil {
		desireValidatorSet = constAccessor.GetInt64Value(constants.DesireValidatorSet)
	}
	rotateRetryBlocks := constAccessor.GetInt64Value(constants.RotateRetryBlocks)

	// calculate if we need to retry a churn because we are overdue for a
	// successful one
	retryChurn := common.BlockHeight(ctx)-lastHeight > rotatePerBlockHeight && (common.BlockHeight(ctx)-lastHeight-rotatePerBlockHeight)%rotateRetryBlocks == 0

	if lastHeight+rotatePerBlockHeight == common.BlockHeight(ctx) || retryChurn {
		if retryChurn {
			ctx.Logger().Info("Checking for node account rotation... (retry)")
		} else {
			ctx.Logger().Info("Checking for node account rotation...")
		}

		// don't churn if we have retiring asgard vaults that still have funds
		retiringVaults, err := vm.k.GetAsgardVaultsByStatus(ctx, RetiringVault)
		if err != nil {
			return err
		}
		for _, vault := range retiringVaults {
			if vault.HasFunds() {
				ctx.Logger().Info("Skipping rotation due to retiring vaults still have funds.")
				return nil
			}
		}

		next, ok, err := vm.nextVaultNodeAccounts(ctx, int(desireValidatorSet), constAccessor)
		if err != nil {
			return err
		}
		if ok {
			if err := vm.vaultMgr.TriggerKeygen(ctx, next); err != nil {
				return err
			}
		}
	}

	return nil
}

// EndBlock when block commit
func (vm *validatorMgrV13) EndBlock(ctx cosmos.Context, mgr Manager, constAccessor constants.ConstantValues) []abci.ValidatorUpdate {
	height := common.BlockHeight(ctx)
	activeNodes, err := vm.k.ListActiveNodeAccounts(ctx)
	if err != nil {
		ctx.Logger().Error("fail to get all active nodes", "error", err)
		return nil
	}

	// when ragnarok is in progress, just process ragnarok
	if vm.k.RagnarokInProgress(ctx) {
		// process ragnarok
		if err := vm.processRagnarok(ctx, mgr, constAccessor); err != nil {
			ctx.Logger().Error("fail to process ragnarok protocol", "error", err)
		}
		return nil
	}

	newNodes, removedNodes, err := vm.getChangedNodes(ctx, activeNodes)
	if err != nil {
		ctx.Logger().Error("fail to get node changes", "error", err)
		return nil
	}

	artificialRagnarokBlockHeight, err := vm.k.GetMimir(ctx, constants.ArtificialRagnarokBlockHeight.String())
	if artificialRagnarokBlockHeight < 0 || err != nil {
		artificialRagnarokBlockHeight = constAccessor.GetInt64Value(constants.ArtificialRagnarokBlockHeight)
	}
	if artificialRagnarokBlockHeight > 0 {
		ctx.Logger().Info("Artificial Ragnarok is planned", "height", artificialRagnarokBlockHeight)
	}
	minimumNodesForBFT := constAccessor.GetInt64Value(constants.MinimumNodesForBFT)
	nodesAfterChange := len(activeNodes) + len(newNodes) - len(removedNodes)
	if (len(activeNodes) >= int(minimumNodesForBFT) && nodesAfterChange < int(minimumNodesForBFT)) ||
		(artificialRagnarokBlockHeight > 0 && common.BlockHeight(ctx) >= artificialRagnarokBlockHeight) {
		// THORNode don't have enough validators for BFT

		// Check we're not migrating funds
		retiring, err := vm.k.GetAsgardVaultsByStatus(ctx, RetiringVault)
		if err != nil {
			ctx.Logger().Error("fail to get retiring vaults", "error", err)
		}

		if len(retiring) == 0 { // wait until all funds are migrated before starting ragnarok
			if err := vm.processRagnarok(ctx, mgr, constAccessor); err != nil {
				ctx.Logger().Error("fail to process ragnarok protocol", "error", err)
			}
		}
		// by return
		return nil
	}

	// no change
	if len(newNodes) == 0 && len(removedNodes) == 0 {
		return nil
	}

	validators := make([]abci.ValidatorUpdate, 0, len(newNodes)+len(removedNodes))
	for _, na := range newNodes {
		ctx.EventManager().EmitEvent(
			cosmos.NewEvent("UpdateNodeAccountStatus",
				cosmos.NewAttribute("Address", na.NodeAddress.String()),
				cosmos.NewAttribute("Former:", na.Status.String()),
				cosmos.NewAttribute("Current:", NodeActive.String())))
		na.UpdateStatus(NodeActive, height)
		na.LeaveHeight = 0
		na.RequestedToLeave = false
		vm.k.ResetNodeAccountSlashPoints(ctx, na.NodeAddress)
		if err := vm.k.SetNodeAccount(ctx, na); err != nil {
			ctx.Logger().Error("fail to save node account", "error", err)
		}
		pk, err := cosmos.GetPubKeyFromBech32(cosmos.Bech32PubKeyTypeConsPub, na.ValidatorConsPubKey)
		if err != nil {
			ctx.Logger().Error("fail to parse consensus public key", "key", na.ValidatorConsPubKey, "error", err)
			continue
		}
		validators = append(validators, abci.ValidatorUpdate{
			PubKey: tmtypes.TM2PB.PubKey(pk),
			Power:  100,
		})
	}
	for _, na := range removedNodes {
		status := NodeStandby
		if na.RequestedToLeave || na.ForcedToLeave {
			status = NodeDisabled
		}

		ctx.EventManager().EmitEvent(
			cosmos.NewEvent("UpdateNodeAccountStatus",
				cosmos.NewAttribute("Address", na.NodeAddress.String()),
				cosmos.NewAttribute("Former:", na.Status.String()),
				cosmos.NewAttribute("Current:", status.String())))
		na.UpdateStatus(status, height)
		if err := vm.k.SetNodeAccount(ctx, na); err != nil {
			ctx.Logger().Error("fail to save node account", "error", err)
		}

		if err := vm.payNodeAccountBondAward(ctx, na); err != nil {
			ctx.Logger().Error("fail to pay node account bond award", "error", err)
		}

		// return yggdrasil funds
		if err := vm.RequestYggReturn(ctx, na, mgr); err != nil {
			ctx.Logger().Error("fail to request yggdrasil funds return", "error", err)
		}

		pk, err := cosmos.GetPubKeyFromBech32(cosmos.Bech32PubKeyTypeConsPub, na.ValidatorConsPubKey)
		if err != nil {
			ctx.Logger().Error("fail to parse consensus public key", "key", na.ValidatorConsPubKey, "error", err)
			continue
		}
		validators = append(validators, abci.ValidatorUpdate{
			PubKey: tmtypes.TM2PB.PubKey(pk),
			Power:  0,
		})
	}

	// reset all nodes in ready status back to standby status
	ready, err := vm.k.ListNodeAccountsByStatus(ctx, NodeReady)
	if err != nil {
		ctx.Logger().Error("fail to get list of ready node accounts", "error", err)
	}
	for _, na := range ready {
		na.UpdateStatus(NodeStandby, common.BlockHeight(ctx))
		if err := vm.k.SetNodeAccount(ctx, na); err != nil {
			ctx.Logger().Error("fail to set node account", "error", err)
		}
	}

	return validators
}

// getChangedNodes to identify which node had been removed ,and which one had been added
// newNodes , removed nodes,err
func (vm *validatorMgrV13) getChangedNodes(ctx cosmos.Context, activeNodes NodeAccounts) (NodeAccounts, NodeAccounts, error) {
	var newActive NodeAccounts    // store the list of new active users
	var removedNodes NodeAccounts // nodes that had been removed

	readyNodes, err := vm.k.ListNodeAccountsByStatus(ctx, NodeReady)
	if err != nil {
		return newActive, removedNodes, fmt.Errorf("fail to list ready node accounts: %w", err)
	}

	activeVaults, err := vm.k.GetAsgardVaultsByStatus(ctx, ActiveVault)
	if err != nil {
		ctx.Logger().Error("fail to get active asgards", "error", err)
		return newActive, removedNodes, fmt.Errorf("fail to get active asgards: %w", err)
	}
	if len(activeVaults) == 0 {
		return newActive, removedNodes, errors.New("no active vault")
	}
	var membership common.PubKeys
	for _, vault := range activeVaults {
		membership = append(membership, vault.Membership...)
	}

	// find active node accounts that are no longer active
	for _, na := range activeNodes {
		found := false
		for _, vault := range activeVaults {
			if vault.Contains(na.PubKeySet.Secp256k1) {
				found = true
				break
			}
		}
		if na.ForcedToLeave {
			found = false
		}
		if !found && len(membership) > 0 {
			removedNodes = append(removedNodes, na)
		}
	}

	// find ready nodes that change to active
	for _, na := range readyNodes {
		for _, member := range membership {
			if na.PubKeySet.Contains(member) {
				newActive = append(newActive, na)
				break
			}
		}
	}

	return newActive, removedNodes, nil
}

// payNodeAccountBondAward pay
func (vm *validatorMgrV13) payNodeAccountBondAward(ctx cosmos.Context, na NodeAccount) error {
	if na.ActiveBlockHeight == 0 || na.Bond.IsZero() {
		return nil
	}
	// The node account seems to have become a non active node account.
	// Therefore, lets give them their bond rewards.
	vault, err := vm.k.GetVaultData(ctx)
	if err != nil {
		return fmt.Errorf("fail to get vault: %w", err)
	}

	slashPts, err := vm.k.GetNodeAccountSlashPoints(ctx, na.NodeAddress)
	if err != nil {
		return fmt.Errorf("fail to get node slash points: %w", err)
	}

	// Find number of blocks they have been an active node
	totalActiveBlocks := common.BlockHeight(ctx) - na.ActiveBlockHeight

	// find number of blocks they were well behaved (ie active - slash points)
	earnedBlocks := na.CalcBondUnits(common.BlockHeight(ctx), slashPts)

	// calc number of rune they are awarded
	reward := vault.CalcNodeRewards(earnedBlocks)

	// Add to their bond the amount rewarded
	na.Bond = na.Bond.Add(reward)

	// Minus the number of rune THORNode have awarded them
	vault.BondRewardRune = common.SafeSub(vault.BondRewardRune, reward)

	// Minus the number of units na has (do not include slash points)
	vault.TotalBondUnits = common.SafeSub(
		vault.TotalBondUnits,
		cosmos.NewUint(uint64(totalActiveBlocks)),
	)

	if err := vm.k.SetVaultData(ctx, vault); err != nil {
		return fmt.Errorf("fail to save vault data: %w", err)
	}
	na.ActiveBlockHeight = 0
	return vm.k.SetNodeAccount(ctx, na)
}

// determines when/if to run each part of the ragnarok process
func (vm *validatorMgrV13) processRagnarok(ctx cosmos.Context, mgr Manager, constAccessor constants.ConstantValues) error {
	// execute Ragnarok protocol, no going back
	// THORNode have to request the fund back now, because once it get to the rotate block height ,
	// THORNode won't have validators anymore
	ragnarokHeight, err := vm.k.GetRagnarokBlockHeight(ctx)
	if err != nil {
		return fmt.Errorf("fail to get ragnarok height: %w", err)
	}

	if ragnarokHeight == 0 {
		ragnarokHeight = common.BlockHeight(ctx)
		vm.k.SetRagnarokBlockHeight(ctx, ragnarokHeight)
		if err := vm.ragnarokProtocolStage1(ctx, mgr); err != nil {
			return fmt.Errorf("fail to execute ragnarok protocol step 1: %w", err)
		}
		if err := vm.ragnarokBondReward(ctx); err != nil {
			return fmt.Errorf("when ragnarok triggered ,fail to give all active node bond reward %w", err)
		}
		return nil
	}

	nth, err := vm.k.GetRagnarokNth(ctx)
	if err != nil {
		return fmt.Errorf("fail to get ragnarok nth: %w", err)
	}

	position, err := vm.k.GetRagnarokUnstakPosition(ctx)
	if err != nil {
		return fmt.Errorf("fail to get ragnarok position: %w", err)
	}
	if !position.IsEmpty() {
		if err := vm.ragnarokPools(ctx, nth, mgr, constAccessor); err != nil {
			ctx.Logger().Error("fail to ragnarok pools", "error", err)
		}
		return nil
	}

	// check if we have any pending ragnarok transactions
	pending, err := vm.k.GetRagnarokPending(ctx)
	if err != nil {
		return fmt.Errorf("fail to get ragnarok pending: %w", err)
	}
	if pending > 0 {
		txOutQueue, err := vm.getPendingTxOut(ctx, constAccessor)
		if err != nil {
			ctx.Logger().Error("fail to get pending tx out item", "error", err)
			return nil
		}
		if txOutQueue > 0 {
			ctx.Logger().Info("awaiting previous ragnarok transaction to clear before continuing", "nth", nth, "count", pending)
			return nil
		}
	}

	nth++ // increment by 1
	ctx.Logger().Info("starting next ragnarok iteration", "iteration", nth)
	err = vm.ragnarokProtocolStage2(ctx, nth, mgr, constAccessor)
	if err != nil {
		ctx.Logger().Error("fail to execute ragnarok protocol step 2", "error", err)
		return err
	}
	vm.k.SetRagnarokNth(ctx, nth)

	return nil
}

func (vm *validatorMgrV13) getPendingTxOut(ctx cosmos.Context, constAccessor constants.ConstantValues) (int64, error) {
	signingTransactionPeriod := constAccessor.GetInt64Value(constants.SigningTransactionPeriod)
	startHeight := common.BlockHeight(ctx) - signingTransactionPeriod
	count := int64(0)
	for height := startHeight; height <= common.BlockHeight(ctx); height++ {
		txs, err := vm.k.GetTxOut(ctx, height)
		if err != nil {
			ctx.Logger().Error("fail to get tx out array from key value store", "error", err)
			return 0, fmt.Errorf("fail to get tx out array from key value store: %w", err)
		}
		for _, tx := range txs.TxArray {
			if tx.OutHash.IsEmpty() {
				count++
			}
		}
	}
	return count, nil
}

// ragnarokProtocolStage1 - request all yggdrasil pool to return the fund
// when THORNode observe the node return fund successfully, the node's bound will be refund.
func (vm *validatorMgrV13) ragnarokProtocolStage1(ctx cosmos.Context, mgr Manager) error {
	return vm.recallYggFunds(ctx, mgr)
}

func (vm *validatorMgrV13) ragnarokProtocolStage2(ctx cosmos.Context, nth int64, mgr Manager, constAccessor constants.ConstantValues) error {
	// Ragnarok Protocol
	// If THORNode can no longer be BFT, do a graceful shutdown of the entire network.
	// 1) THORNode will request all yggdrasil pool to return fund , if THORNode don't have yggdrasil pool THORNode will go to step 3 directly
	// 2) upon receiving the yggdrasil fund,  THORNode will refund the validator's bond
	// 3) once all yggdrasil fund get returned, return all fund to stakes

	// refund bonders
	if err := vm.ragnarokBond(ctx, nth, mgr); err != nil {
		ctx.Logger().Error("fail to ragnarok bond", "error", err)
	}

	// refund reserve contributors
	if err := vm.ragnarokReserve(ctx, nth, mgr); err != nil {
		ctx.Logger().Error("fail to ragnarok reserve", "error", err)
	}

	// refund stakers. This is last to ensure there is likely gas for the
	// returning bond and reserve
	if err := vm.ragnarokPools(ctx, nth, mgr, constAccessor); err != nil {
		ctx.Logger().Error("fail to ragnarok pools", "error", err)
	}

	return nil
}

func (vm *validatorMgrV13) ragnarokBondReward(ctx cosmos.Context) error {
	active, err := vm.k.ListActiveNodeAccounts(ctx)
	if err != nil {
		return fmt.Errorf("fail to get all active node account: %w", err)
	}
	for _, item := range active {
		if err := vm.payNodeAccountBondAward(ctx, item); err != nil {
			return fmt.Errorf("fail to pay node account(%s) bond award: %w", item.NodeAddress.String(), err)
		}
	}
	return nil
}

func (vm *validatorMgrV13) ragnarokReserve(ctx cosmos.Context, nth int64, mgr Manager) error {
	// don't ragnarok the reserve when rune is a native token
	if common.RuneAsset().Chain.Equals(common.THORChain) {
		return nil
	}

	contribs, err := vm.k.GetReservesContributors(ctx)
	if err != nil {
		ctx.Logger().Error("can't get reserve contributors", "error", err)
		return err
	}
	if len(contribs) == 0 {
		return nil
	}
	vaultData, err := vm.k.GetVaultData(ctx)
	if err != nil {
		ctx.Logger().Error("can't get vault data", "error", err)
		return err
	}

	if vaultData.TotalReserve.IsZero() {
		return nil
	}

	totalReserve := vaultData.TotalReserve
	totalContributions := cosmos.ZeroUint()
	for _, contrib := range contribs {
		totalContributions = totalContributions.Add(contrib.Amount)
	}

	// Since reserves are spent over time (via block rewards), reserve
	// contributors do not get back the full amounts they put in. Instead they
	// should get a percentage of the remaining amount, relative to the amount
	// they contributed. We'll be reducing the total reserve supply as we
	// refund reserves

	// nth * 10 == the amount of the bond we want to send
	for i, contrib := range contribs {
		share := common.GetShare(
			contrib.Amount,
			totalContributions,
			totalReserve,
		)
		if nth > 10 { // cap at 10
			nth = 10
		}
		amt := share.MulUint64(uint64(nth)).QuoUint64(10)
		vaultData.TotalReserve = common.SafeSub(vaultData.TotalReserve, amt)
		contribs[i].Amount = common.SafeSub(contrib.Amount, amt)

		// refund contribution
		txOutItem := &TxOutItem{
			Chain:     common.RuneAsset().Chain,
			ToAddress: contrib.Address,
			InHash:    common.BlankTxID,
			Coin:      common.NewCoin(common.RuneAsset(), amt),
			Memo:      NewRagnarokMemo(common.BlockHeight(ctx)).String(),
		}
		_, err = vm.txOutStore.TryAddTxOutItem(ctx, mgr, txOutItem)
		if err != nil && !errors.Is(err, ErrNotEnoughToPayFee) {
			return fmt.Errorf("fail to add outbound transaction")
		}

		// add a pending rangarok transaction
		pending, err := vm.k.GetRagnarokPending(ctx)
		if err != nil {
			return fmt.Errorf("fail to get ragnarok pending: %w", err)
		}
		vm.k.SetRagnarokPending(ctx, pending+1)

	}

	if err := vm.k.SetVaultData(ctx, vaultData); err != nil {
		return err
	}

	if err := vm.k.SetReserveContributors(ctx, contribs); err != nil {
		return err
	}

	return nil
}

func (vm *validatorMgrV13) ragnarokBond(ctx cosmos.Context, nth int64, mgr Manager) error {
	// bond should be returned on the back 10, not the first 10
	nth -= 10
	if nth < 1 {
		return nil
	}

	nas, err := vm.k.ListNodeAccountsWithBond(ctx)
	if err != nil {
		ctx.Logger().Error("can't get nodes", "error", err)
		return err
	}
	// nth * 10 == the amount of the bond we want to send
	for _, na := range nas {
		if na.Bond.IsZero() {
			continue
		}
		if vm.k.VaultExists(ctx, na.PubKeySet.Secp256k1) {
			ygg, err := vm.k.GetVault(ctx, na.PubKeySet.Secp256k1)
			if err != nil {
				return err
			}
			if ygg.HasFunds() {
				ctx.Logger().Info(fmt.Sprintf("skip bond refund due to remaining funds: %s", na.NodeAddress))
				continue
			}
		}

		if nth >= 9 { // cap at 10
			nth = 10
		}
		amt := na.Bond.MulUint64(uint64(nth)).QuoUint64(10)

		// refund bond
		txOutItem := &TxOutItem{
			Chain:      common.RuneAsset().Chain,
			ToAddress:  na.BondAddress,
			InHash:     common.BlankTxID,
			Coin:       common.NewCoin(common.RuneAsset(), amt),
			Memo:       NewRagnarokMemo(common.BlockHeight(ctx)).String(),
			ModuleName: BondName,
		}
		ok, err := vm.txOutStore.TryAddTxOutItem(ctx, mgr, txOutItem)
		if err != nil {
			if !errors.Is(err, ErrNotEnoughToPayFee) {
				return err
			}
			ok = true
		}
		if !ok {
			continue
		}

		// add a pending rangarok transaction
		pending, err := vm.k.GetRagnarokPending(ctx)
		if err != nil {
			return fmt.Errorf("fail to get ragnarok pending: %w", err)
		}
		vm.k.SetRagnarokPending(ctx, pending+1)

		na.Bond = common.SafeSub(na.Bond, amt)
		if err := vm.k.SetNodeAccount(ctx, na); err != nil {
			return err
		}
	}

	return nil
}

func (vm *validatorMgrV13) ragnarokPools(ctx cosmos.Context, nth int64, mgr Manager, constAccessor constants.ConstantValues) error {
	nas, err := vm.k.ListActiveNodeAccounts(ctx)
	if err != nil {
		ctx.Logger().Error("can't get active nodes", "error", err)
		return err
	}
	if len(nas) == 0 {
		return fmt.Errorf("can't find any active nodes")
	}
	na := nas[0]

	position, err := vm.k.GetRagnarokUnstakPosition(ctx)
	if err != nil {
		return fmt.Errorf("fail to get ragnarok position: %w", err)
	}

	// each round of refund, we increase the percentage by 10%. This ensures
	// that we slowly refund each person, while not sending out too much too
	// fast. Also, we won't be running into any gas related issues until the
	// very last round, which, by my calculations, if someone staked 100 coins,
	// the last tx will send them 0.036288. So if we don't have enough gas to
	// send them, its only a very small portion that is not refunded.
	var basisPoints int64
	if nth > 20 || (nth%10) == 0 {
		basisPoints = MaxUnstakeBasisPoints
	} else {
		basisPoints = (nth % 10) * (MaxUnstakeBasisPoints / 10)
	}

	// go through all the pools
	pools, err := vm.k.GetPools(ctx)
	if err != nil {
		ctx.Logger().Error("can't get pools", "error", err)
		return err
	}
	// set all pools to bootstrap mode
	for _, pool := range pools {
		if pool.Status != PoolBootstrap {
			poolEvent := NewEventPool(pool.Asset, PoolBootstrap)
			if err := vm.eventMgr.EmitEvent(ctx, poolEvent); err != nil {
				ctx.Logger().Error("fail to emit pool event", "error", err)
			}

			pool.Status = PoolBootstrap
			if err := vm.k.SetPool(ctx, pool); err != nil {
				ctx.Logger().Error(err.Error())
				return err
			}
		}
	}

	version := vm.k.GetLowestActiveVersion(ctx)

	nextPool := false
	maxUnstakesPerBlock := 20
	count := 0

	for i := len(pools) - 1; i >= 0; i-- { // iterate backwards
		pool := pools[i]

		if nextPool { // we've iterated to the next pool after our position pool
			position.Pool = pool.Asset
		}

		if !position.Pool.IsEmpty() && !pool.Asset.Equals(position.Pool) {
			continue
		}

		nextPool = true
		position.Pool = pool.Asset

		// unstake gas asset pool on the back 10 nths
		if nth <= 10 && pool.Asset.Chain.GetGasAsset().Equals(pool.Asset) {
			continue
		}

		j := int64(-1)
		iterator := vm.k.GetStakerIterator(ctx, pool.Asset)
		for ; iterator.Valid(); iterator.Next() {
			j++
			if j == position.Number {
				position.Number++
				var staker Staker
				vm.k.Cdc().MustUnmarshalBinaryBare(iterator.Value(), &staker)
				if staker.Units.IsZero() {
					continue
				}

				unstakeMsg := NewMsgUnStake(
					common.GetRagnarokTx(pool.Asset.Chain, staker.RuneAddress, staker.RuneAddress),
					staker.RuneAddress,
					cosmos.NewUint(uint64(basisPoints)),
					pool.Asset,
					na.NodeAddress,
				)

				unstakeHandler := NewUnstakeHandler(vm.k, mgr)
				_, err := unstakeHandler.Run(ctx, unstakeMsg, version, constAccessor)
				if err != nil {
					ctx.Logger().Error("fail to unstake", "staker", staker.RuneAddress, "error", err)
				} else {
					count++
					pending, err := vm.k.GetRagnarokPending(ctx)
					if err != nil {
						return fmt.Errorf("fail to get ragnarok pending: %w", err)
					}
					vm.k.SetRagnarokPending(ctx, pending+2) // two outbound txs
					if count >= maxUnstakesPerBlock {
						break
					}
				}
			}
		}
		iterator.Close()
		if count >= maxUnstakesPerBlock {
			break
		}
		position.Number = 0
	}

	if count < maxUnstakesPerBlock { // we've completed all pools/stakers, reset the position
		position = RagnarokUnstakePosition{}
	}
	vm.k.SetRagnarokUnstakPosition(ctx, position)

	return nil
}

// RequestYggReturn request the node that had been removed (yggdrasil) to return their fund
func (vm *validatorMgrV13) RequestYggReturn(ctx cosmos.Context, node NodeAccount, mgr Manager) error {
	if !vm.k.VaultExists(ctx, node.PubKeySet.Secp256k1) {
		return nil
	}
	ygg, err := vm.k.GetVault(ctx, node.PubKeySet.Secp256k1)
	if err != nil {
		return fmt.Errorf("fail to get yggdrasil: %w", err)
	}
	if ygg.IsAsgard() {
		return nil
	}
	if !ygg.HasFunds() {
		return nil
	}

	chains := make(common.Chains, 0)

	active, err := vm.k.GetAsgardVaultsByStatus(ctx, ActiveVault)
	if err != nil {
		return err
	}

	retiring, err := vm.k.GetAsgardVaultsByStatus(ctx, RetiringVault)
	if err != nil {
		return err
	}

	for _, v := range append(active, retiring...) {
		chains = append(chains, v.Chains...)
	}
	chains = chains.Distinct()

	vault := active.SelectByMaxCoin(common.RuneAsset())
	if vault.IsEmpty() {
		return fmt.Errorf("unable to determine asgard vault")
	}
	for _, chain := range chains {
		if chain.Equals(common.THORChain) {
			continue
		}

		toAddr, err := vault.PubKey.GetAddress(chain)
		if err != nil {
			return err
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
			// yggdrasil- will not set coin field here, when signer see a TxOutItem that has memo "yggdrasil-" it will query the chain
			// and find out all the remaining assets , and fill in the field
			if err := vm.txOutStore.UnSafeAddTxOutItem(ctx, mgr, txOutItem); err != nil {
				return err
			}
		}
	}

	return nil
}

func (vm *validatorMgrV13) recallYggFunds(ctx cosmos.Context, mgr Manager) error {
	iter := vm.k.GetVaultIterator(ctx)
	defer iter.Close()
	vaults := Vaults{}
	for ; iter.Valid(); iter.Next() {
		var vault Vault
		if err := vm.k.Cdc().UnmarshalBinaryBare(iter.Value(), &vault); err != nil {
			return fmt.Errorf("fail to unmarshal vault, %w", err)
		}
		if vault.IsYggdrasil() && vault.HasFunds() {
			vaults = append(vaults, vault)
		}
	}

	if len(vaults) == 0 {
		return nil
	}

	for _, vault := range vaults {
		na, err := vm.k.GetNodeAccountByPubKey(ctx, vault.PubKey)
		if err != nil {
			ctx.Logger().Error("fail to get node account", "error", err)
			continue
		}
		if err := vm.RequestYggReturn(ctx, na, mgr); err != nil {
			return fmt.Errorf("fail to request yggdrasil fund back: %w", err)
		}
	}
	return fmt.Errorf("some yggdrasil vaults (%d) still have funds", len(vaults))
}

// setupValidatorNodes it is one off it only get called when genesis
func (vm *validatorMgrV13) setupValidatorNodes(ctx cosmos.Context, height int64, constAccessor constants.ConstantValues) error {
	if height != genesisBlockHeight {
		ctx.Logger().Info("only need to setup validator node when start up", "height", height)
		return nil
	}

	iter := vm.k.GetNodeAccountIterator(ctx)
	defer iter.Close()
	readyNodes := NodeAccounts{}
	activeCandidateNodes := NodeAccounts{}
	for ; iter.Valid(); iter.Next() {
		var na NodeAccount
		if err := vm.k.Cdc().UnmarshalBinaryBare(iter.Value(), &na); err != nil {
			return fmt.Errorf("fail to unmarshal node account, %w", err)
		}
		// when THORNode first start , THORNode only care about these two status
		switch na.Status {
		case NodeReady:
			readyNodes = append(readyNodes, na)
		case NodeActive:
			activeCandidateNodes = append(activeCandidateNodes, na)
		}
	}
	totalActiveValidators := len(activeCandidateNodes)
	totalNominatedValidators := len(readyNodes)
	if totalActiveValidators == 0 && totalNominatedValidators == 0 {
		return errors.New("no validators available")
	}

	sort.Sort(activeCandidateNodes)
	sort.Sort(readyNodes)
	activeCandidateNodes = append(activeCandidateNodes, readyNodes...)
	desireValidatorSet, err := vm.k.GetMimir(ctx, constants.DesireValidatorSet.String())
	if desireValidatorSet < 0 || err != nil {
		desireValidatorSet = constAccessor.GetInt64Value(constants.DesireValidatorSet)
	}
	for idx, item := range activeCandidateNodes {
		if int64(idx) < desireValidatorSet {
			item.UpdateStatus(NodeActive, common.BlockHeight(ctx))
		} else {
			item.UpdateStatus(NodeStandby, common.BlockHeight(ctx))
		}
		if err := vm.k.SetNodeAccount(ctx, item); err != nil {
			return fmt.Errorf("fail to save node account: %w", err)
		}
	}
	return nil
}

// Iterate over active node accounts, finding bad actors with high slash points
func (vm *validatorMgrV13) findBadActors(ctx cosmos.Context) (NodeAccounts, error) {
	badActors := make(NodeAccounts, 0)
	nas, err := vm.k.ListActiveNodeAccounts(ctx)
	if err != nil {
		return badActors, err
	}

	if len(nas) == 0 {
		return nil, nil
	}

	// NOTE: Our score gives a numerical representation of the behavior our a
	// node account. The lower the score, the worse behavior. The score is
	// determined by relative to how many slash points they have over how long
	// they have been an active node account.
	type badTracker struct {
		Score       cosmos.Uint
		NodeAccount NodeAccount
	}
	tracker := make([]badTracker, 0, len(nas))
	totalScore := cosmos.ZeroUint()

	// Find bad actor relative to age / slashpoints
	for _, na := range nas {
		slashPts, err := vm.k.GetNodeAccountSlashPoints(ctx, na.NodeAddress)
		if err != nil {
			ctx.Logger().Error("fail to get node slash points", "error", err)
		}
		if slashPts == 0 {
			continue
		}

		if common.BlockHeight(ctx)-na.StatusSince < 720 {
			// this node account is too new (1 hour) to be considered for removal
			continue
		}

		// get to the 8th decimal point, but keep numbers integers for safer math
		age := cosmos.NewUint(uint64((common.BlockHeight(ctx) - na.StatusSince) * common.One))
		score := age.QuoUint64(uint64(slashPts))
		totalScore = totalScore.Add(score)

		tracker = append(tracker, badTracker{
			Score:       score,
			NodeAccount: na,
		})
	}

	if len(tracker) == 0 {
		// no offenders, exit nicely
		return nil, nil
	}

	sort.SliceStable(tracker, func(i, j int) bool {
		return tracker[i].Score.LT(tracker[j].Score)
	})

	// score lower is worse
	avgScore := totalScore.QuoUint64(uint64(len(nas)))

	// NOTE: our redline is a hard line in the sand to determine if a node
	// account is sufficiently bad that it should just be removed now. This
	// ensures that if we have multiple "really bad" node accounts, they all
	// can get removed in the same churn. It is important to note we shouldn't
	// be able to churn out more than 1/3rd of our node accounts in a single
	// churn, as that could threaten the security of the funds. This logic to
	// protect against this is not inside this function.
	redline := avgScore.QuoUint64(3)

	// find any node accounts that have crossed the red line
	for _, track := range tracker {
		if redline.GTE(track.Score) {
			badActors = append(badActors, track.NodeAccount)
		}
	}

	// if no one crossed the redline, lets just grab the worse offender
	if len(badActors) == 0 {
		badActors = NodeAccounts{tracker[0].NodeAccount}
	}

	return badActors, nil
}

// Iterate over active node accounts, finding the one that has been active longest
func (vm *validatorMgrV13) findOldActor(ctx cosmos.Context) (NodeAccount, error) {
	na := NodeAccount{}
	nas, err := vm.k.ListActiveNodeAccounts(ctx)
	if err != nil {
		return na, err
	}

	na.StatusSince = common.BlockHeight(ctx) // set the start status age to "now"
	for _, n := range nas {
		if n.StatusSince < na.StatusSince {
			na = n
		}
	}

	return na, nil
}

// Mark an old to be churned out
func (vm *validatorMgrV13) markActor(ctx cosmos.Context, na NodeAccount, reason string) error {
	if !na.IsEmpty() && na.LeaveHeight == 0 {
		ctx.Logger().Info(fmt.Sprintf("Marked Validator to be churned out %s: %s", na.NodeAddress, reason))
		na.LeaveHeight = common.BlockHeight(ctx)
		return vm.k.SetNodeAccount(ctx, na)
	}
	return nil
}

// Mark an old actor to be churned out
func (vm *validatorMgrV13) markOldActor(ctx cosmos.Context, rate int64) error {
	if common.BlockHeight(ctx)%rate == 0 {
		na, err := vm.findOldActor(ctx)
		if err != nil {
			return err
		}
		if err := vm.markActor(ctx, na, "for age"); err != nil {
			return err
		}
	}
	return nil
}

// Mark a bad actor to be churned out
func (vm *validatorMgrV13) markBadActor(ctx cosmos.Context, rate int64) error {
	if common.BlockHeight(ctx)%rate == 0 {
		nas, err := vm.findBadActors(ctx)
		if err != nil {
			return err
		}
		for _, na := range nas {
			if err := vm.markActor(ctx, na, "for bad behavior"); err != nil {
				return err
			}
		}
	}
	return nil
}

func (vm *validatorMgrV13) markLowerVersion(ctx cosmos.Context, rate int64) error {
	if common.BlockHeight(ctx)%rate == 0 {
		na, err := vm.findLowerVersionActor(ctx)
		if err != nil {
			return err
		}
		if !na.IsEmpty() {
			if err := vm.markActor(ctx, na, "for version lower than minimum join version"); err != nil {
				return err
			}
		}
	}
	return nil
}

// findLowerVersionActor go through the active node account list , find the node account that has version
// that is lower than the minimum join version
func (vm *validatorMgrV13) findLowerVersionActor(ctx cosmos.Context) (NodeAccount, error) {
	minimumVersion := vm.k.GetMinJoinVersionV1(ctx)
	activeNodes, err := vm.k.ListNodeAccountsByStatus(ctx, NodeActive)
	if err != nil {
		return NodeAccount{}, err
	}
	for _, na := range activeNodes {
		if na.Version.LT(minimumVersion) {
			return na, nil
		}
	}
	return NodeAccount{}, nil
}

// find any actor that are ready to become "ready" status
func (vm *validatorMgrV13) markReadyActors(ctx cosmos.Context, constAccessor constants.ConstantValues) error {
	standby, err := vm.k.ListNodeAccountsByStatus(ctx, NodeStandby)
	if err != nil {
		return err
	}
	ready, err := vm.k.ListNodeAccountsByStatus(ctx, NodeReady)
	if err != nil {
		return err
	}

	// check all ready and standby nodes are in "ready" state (upgrade/downgrade as needed)
	for _, na := range append(standby, ready...) {
		status, _ := vm.NodeAccountPreflightCheck(ctx, na, constAccessor)
		na.UpdateStatus(status, common.BlockHeight(ctx))

		if err := vm.k.SetNodeAccount(ctx, na); err != nil {
			return err
		}
	}

	return nil
}

func (vm *validatorMgrV13) NodeAccountPreflightCheck(ctx cosmos.Context, na NodeAccount, constAccessor constants.ConstantValues) (NodeStatus, error) {
	// ensure banned nodes can't get churned in again
	if na.ForcedToLeave {
		return NodeDisabled, fmt.Errorf("node account has been banned")
	}

	// Check if they've requested to leave
	if na.RequestedToLeave {
		return NodeStandby, fmt.Errorf("node account has requested to leave")
	}

	// Check that the node account has an IP address
	if net.ParseIP(na.IPAddress) == nil {
		return NodeStandby, fmt.Errorf("node account has invalid registered IP address")
	}

	// Check that the node account has an pubkey set
	if na.PubKeySet.IsEmpty() {
		return NodeWhiteListed, fmt.Errorf("node account has registered their pubkey set")
	}

	// ensure we have enough rune
	minBond, err := vm.k.GetMimir(ctx, constants.MinimumBondInRune.String())
	if minBond < 0 || err != nil {
		minBond = constAccessor.GetInt64Value(constants.MinimumBondInRune)
	}
	if na.Bond.LT(cosmos.NewUint(uint64(minBond))) {
		return NodeStandby, fmt.Errorf("node account does not have minimum bond requirement: %d/%d", na.Bond.Uint64(), minBond)
	}

	minVersion := vm.k.GetMinJoinVersion(ctx)
	// Check version number is still supported
	if na.Version.LT(minVersion) {
		return NodeStandby, fmt.Errorf("node account does not meet min version requirement: %s vs %s", na.Version, minVersion)
	}

	jail, err := vm.k.GetNodeAccountJail(ctx, na.NodeAddress)
	if err != nil {
		ctx.Logger().Error("fail to get node account jail", "error", err)
		return NodeStandby, fmt.Errorf("cannot fetch jail status: %w", err)
	}
	if jail.IsJailed(ctx) {
		return NodeStandby, fmt.Errorf("node account is jailed until block %d: %s", jail.ReleaseHeight, jail.Reason)
	}

	if vm.k.RagnarokInProgress(ctx) {
		return NodeStandby, fmt.Errorf("ragnarok is currently in progress: no churning")
	}

	return NodeReady, nil
}

// Returns a list of nodes to include in the next pool
func (vm *validatorMgrV13) nextVaultNodeAccounts(ctx cosmos.Context, targetCount int, constAccessor constants.ConstantValues) (NodeAccounts, bool, error) {
	rotation := false // track if are making any changes to the current active node accounts

	// update list of ready actors
	if err := vm.markReadyActors(ctx, constAccessor); err != nil {
		return nil, false, err
	}

	ready, err := vm.k.ListNodeAccountsByStatus(ctx, NodeReady)
	if err != nil {
		return nil, false, err
	}

	// sort by bond size, descending
	sort.SliceStable(ready, func(i, j int) bool {
		return ready[i].Bond.GT(ready[j].Bond)
	})

	active, err := vm.k.ListActiveNodeAccounts(ctx)
	if err != nil {
		return nil, false, err
	}
	// sort by LeaveHeight ascending
	// giving preferential treatment to people who are forced to leave
	//  and then requested to leave
	sort.SliceStable(active, func(i, j int) bool {
		if active[i].ForcedToLeave != active[j].ForcedToLeave {
			return active[i].ForcedToLeave
		}
		if active[i].RequestedToLeave != active[j].RequestedToLeave {
			return active[i].RequestedToLeave
		}
		// sort by LeaveHeight ascending , but exclude LeaveHeight == 0 , because that's the default value
		if active[i].LeaveHeight == 0 && active[j].LeaveHeight > 0 {
			return false
		}
		if active[i].LeaveHeight > 0 && active[j].LeaveHeight == 0 {
			return true
		}
		return active[i].LeaveHeight < active[j].LeaveHeight
	})

	toRemove := findCountToRemove(common.BlockHeight(ctx), active)
	if toRemove > 0 {
		rotation = true
		active = active[toRemove:]
	}

	// add ready nodes to become active
	limit := toRemove + 1 // Max limit of ready nodes to churn in
	minimumNodesForBFT := constAccessor.GetInt64Value(constants.MinimumNodesForBFT)
	if len(active)+limit < int(minimumNodesForBFT) {
		limit = int(minimumNodesForBFT) - len(active)
	}
	for i := 1; targetCount >= len(active); i++ {
		if len(ready) >= i {
			rotation = true
			active = append(active, ready[i-1])
		}
		if i == limit { // limit adding ready accounts
			break
		}
	}

	return active, rotation, nil
}
