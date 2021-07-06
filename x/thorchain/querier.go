package thorchain

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/codec"
	abci "github.com/tendermint/tendermint/abci/types"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
	"gitlab.com/thorchain/thornode/x/thorchain/keeper"
	q "gitlab.com/thorchain/thornode/x/thorchain/query"
)

// NewQuerier is the module level router for state queries
func NewQuerier(keeper keeper.Keeper, kbs KeybaseStore) cosmos.Querier {
	return func(ctx cosmos.Context, path []string, req abci.RequestQuery) (res []byte, err error) {
		switch path[0] {
		case q.QueryPool.Key:
			return queryPool(ctx, path[1:], req, keeper)
		case q.QueryPools.Key:
			return queryPools(ctx, req, keeper)
		case q.QueryStakers.Key:
			return queryStakers(ctx, path[1:], req, keeper)
		case q.QueryTxInVoter.Key:
			return queryTxInVoter(ctx, path[1:], req, keeper)
		case q.QueryTxIn.Key:
			return queryTxIn(ctx, path[1:], req, keeper)
		case q.QueryKeysignArray.Key:
			return queryKeysign(ctx, kbs, path[1:], req, keeper)
		case q.QueryKeysignArrayPubkey.Key:
			return queryKeysign(ctx, kbs, path[1:], req, keeper)
		case q.QueryKeygensPubkey.Key:
			return queryKeygen(ctx, kbs, path[1:], req, keeper)
		case q.QueryQueue.Key:
			return queryQueue(ctx, path[1:], req, keeper)
		case q.QueryHeights.Key:
			return queryHeights(ctx, path[1:], req, keeper)
		case q.QueryChainHeights.Key:
			return queryHeights(ctx, path[1:], req, keeper)
		case q.QueryObservers.Key:
			return queryObservers(ctx, path[1:], req, keeper)
		case q.QueryObserver.Key:
			return queryObserver(ctx, path[1:], req, keeper)
		case q.QueryNodeAccount.Key:
			return queryNodeAccount(ctx, path[1:], req, keeper)
		case q.QueryNodeAccountCheck.Key:
			return queryNodeAccountCheck(ctx, path[1:], req, keeper)
		case q.QueryNodeAccounts.Key:
			return queryNodeAccounts(ctx, path[1:], req, keeper)
		case q.QueryPoolAddresses.Key:
			return queryPoolAddresses(ctx, path[1:], req, keeper)
		case q.QueryVaultData.Key:
			return queryVaultData(ctx, keeper)
		case q.QueryBalanceModule.Key:
			return queryBalanceModule(ctx, path[1:], keeper)
		case q.QueryVaultsAsgard.Key:
			return queryAsgardVaults(ctx, keeper)
		case q.QueryVaultsYggdrasil.Key:
			return queryYggdrasilVaults(ctx, keeper)
		case q.QueryVault.Key:
			return queryVault(ctx, path[1:], keeper)
		case q.QueryVaultPubkeys.Key:
			return queryVaultsPubkeys(ctx, keeper)
		case q.QueryTSSSigners.Key:
			return queryTSSSigners(ctx, path[1:], req, keeper)
		case q.QueryConstantValues.Key:
			return queryConstantValues(ctx, path[1:], req, keeper)
		case q.QueryVersion.Key:
			return queryVersion(ctx, path[1:], req, keeper)
		case q.QueryMimirValues.Key:
			return queryMimirValues(ctx, path[1:], req, keeper)
		case q.QueryBan.Key:
			return queryBan(ctx, path[1:], req, keeper)
		case q.QueryRagnarok.Key:
			return queryRagnarok(ctx, keeper)
		case q.QueryPendingOutbound.Key:
			return queryPendingOutbound(ctx, keeper)
		default:
			return nil, cosmos.ErrUnknownRequest(
				fmt.Sprintf("unknown thorchain query endpoint: %s", path[0]),
			)
		}
	}
}

func queryRagnarok(ctx cosmos.Context, keeper keeper.Keeper) ([]byte, error) {
	ragnarokInProgress := keeper.RagnarokInProgress(ctx)
	res, err := codec.MarshalJSONIndent(keeper.Cdc(), ragnarokInProgress)
	if err != nil {
		return nil, ErrInternal(err, "fail to marshal response to json")
	}
	return res, nil
}

func queryBalanceModule(ctx cosmos.Context, path []string, keeper keeper.Keeper) ([]byte, error) {
	supplier := keeper.Supply()
	mod := supplier.GetModuleAccount(ctx, AsgardName)

	res, err := codec.MarshalJSONIndent(keeper.Cdc(), mod.GetCoins())
	if err != nil {
		return nil, ErrInternal(err, "fail to marshal response to json")
	}

	return res, nil
}

func queryVault(ctx cosmos.Context, path []string, keeper keeper.Keeper) ([]byte, error) {
	if len(path) < 2 {
		return nil, errors.New("not enough parameters")
	}
	chain, err := common.NewChain(path[0])
	if err != nil {
		return nil, fmt.Errorf("%s is invalid chain,%w", path[0], err)
	}
	addr, err := common.NewAddress(path[1])
	if err != nil {
		return nil, fmt.Errorf("%s is invalid address,%w", path[1], err)
	}
	iter := keeper.GetVaultIterator(ctx)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var v Vault
		if err := keeper.Cdc().UnmarshalBinaryBare(iter.Value(), &v); err != nil {
			ctx.Logger().Error("fail to unmarshal vault", "error", err)
			continue
		}
		vaultAddr, err := v.PubKey.GetAddress(chain)
		if err != nil {
			ctx.Logger().Error("fail to get vault address", "error", err, "chain", chain.String())
			continue
		}
		if vaultAddr.Equals(addr) {

			res, err := codec.MarshalJSONIndent(keeper.Cdc(), v)
			if err != nil {
				ctx.Logger().Error("fail to marshal vaults response to json", "error", err)
				return nil, fmt.Errorf("fail to marshal response to json: %w", err)
			}
			return res, nil
		}
	}
	return nil, errors.New("vault not found")
}

func queryAsgardVaults(ctx cosmos.Context, keeper keeper.Keeper) ([]byte, error) {
	vaults, err := keeper.GetAsgardVaults(ctx)
	if err != nil {
		return nil, fmt.Errorf("fail to get asgard vaults: %w", err)
	}

	var vaultsWithFunds Vaults
	for _, vault := range vaults {
		if vault.Status == InactiveVault {
			continue
		}
		if !vault.IsAsgard() {
			continue
		}
		if vault.HasFunds() || vault.Status == ActiveVault {
			vaultsWithFunds = append(vaultsWithFunds, vault)
		}
	}

	res, err := codec.MarshalJSONIndent(keeper.Cdc(), vaultsWithFunds)
	if err != nil {
		ctx.Logger().Error("fail to marshal vaults response to json", "error", err)
		return nil, fmt.Errorf("fail to marshal response to json: %w", err)
	}

	return res, nil
}

func getVaultChainAddress(ctx cosmos.Context, vault Vault) []QueryChainAddress {
	var result []QueryChainAddress
	allChains := append(vault.Chains, common.THORChain)
	for _, c := range allChains.Distinct() {
		addr, err := vault.PubKey.GetAddress(c)
		if err != nil {
			ctx.Logger().Error("fail to get address for %s:%w", c.String(), err)
			continue
		}
		result = append(result,
			QueryChainAddress{
				Chain:   c,
				Address: addr,
			})
	}
	return result
}

func queryYggdrasilVaults(ctx cosmos.Context, keeper keeper.Keeper) ([]byte, error) {
	vaults := make(Vaults, 0)
	iter := keeper.GetVaultIterator(ctx)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var vault Vault
		if err := keeper.Cdc().UnmarshalBinaryBare(iter.Value(), &vault); err != nil {
			ctx.Logger().Error("fail to unmarshal yggdrasil", "error", err)
			return nil, fmt.Errorf("fail to unmarshal yggdrasil: %w", err)
		}
		if vault.IsYggdrasil() && vault.HasFunds() {
			vaults = append(vaults, vault)
		}
	}

	respVaults := make([]QueryYggdrasilVaults, len(vaults))
	for i, vault := range vaults {
		totalValue := cosmos.ZeroUint()

		// find the bond of this node account
		na, err := keeper.GetNodeAccountByPubKey(ctx, vault.PubKey)
		if err != nil {
			ctx.Logger().Error("fail to get node account by pubkey", "error", err)
			continue
		}

		// calculate the total value of this yggdrasil vault
		for _, coin := range vault.Coins {
			if coin.Asset.IsRune() {
				totalValue = totalValue.Add(coin.Amount)
			} else {
				pool, err := keeper.GetPool(ctx, coin.Asset)
				if err != nil {
					ctx.Logger().Error("fail to get pool", "error", err)
					continue
				}
				totalValue = totalValue.Add(pool.AssetValueInRune(coin.Amount))
			}
		}

		respVaults[i] = QueryYggdrasilVaults{
			Vault:      vault,
			Status:     na.Status,
			Bond:       na.Bond,
			TotalValue: totalValue,
			Addresses:  getVaultChainAddress(ctx, vault),
		}
	}

	res, err := codec.MarshalJSONIndent(keeper.Cdc(), respVaults)
	if err != nil {
		ctx.Logger().Error("fail to marshal vaults response to json", "error", err)
		return nil, fmt.Errorf("fail to marshal response to json: %w", err)
	}

	return res, nil
}

func queryVaultsPubkeys(ctx cosmos.Context, keeper keeper.Keeper) ([]byte, error) {
	var resp struct {
		Asgard    common.PubKeys `json:"asgard"`
		Yggdrasil common.PubKeys `json:"yggdrasil"`
	}
	resp.Asgard = make(common.PubKeys, 0)
	resp.Yggdrasil = make(common.PubKeys, 0)
	iter := keeper.GetVaultIterator(ctx)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var vault Vault
		if err := keeper.Cdc().UnmarshalBinaryBare(iter.Value(), &vault); err != nil {
			ctx.Logger().Error("fail to unmarshal vault", "error", err)
			return nil, fmt.Errorf("fail to unmarshal vault: %w", err)
		}
		if vault.IsYggdrasil() {
			na, err := keeper.GetNodeAccountByPubKey(ctx, vault.PubKey)
			if err != nil {
				ctx.Logger().Error("fail to unmarshal vault", "error", err)
				return nil, fmt.Errorf("fail to unmarshal vault: %w", err)
			}
			if !na.Bond.IsZero() {
				resp.Yggdrasil = append(resp.Yggdrasil, vault.PubKey)
			}
		} else if vault.IsAsgard() {
			if vault.Status == ActiveVault || vault.Status == RetiringVault {
				resp.Asgard = append(resp.Asgard, vault.PubKey)
			}
		}
	}
	res, err := codec.MarshalJSONIndent(keeper.Cdc(), resp)
	if err != nil {
		ctx.Logger().Error("fail to marshal pubkeys response to json", "error", err)
		return nil, fmt.Errorf("fail to marshal response to json: %w", err)
	}
	return res, nil
}

func queryVaultData(ctx cosmos.Context, keeper keeper.Keeper) ([]byte, error) {
	data, err := keeper.GetVaultData(ctx)
	if err != nil {
		ctx.Logger().Error("fail to get vault", "error", err)
		return nil, fmt.Errorf("fail to get vault: %w", err)
	}

	if common.RuneAsset().Chain.Equals(common.THORChain) {
		data.TotalReserve = keeper.GetRuneBalanceOfModule(ctx, ReserveName)
	}

	res, err := codec.MarshalJSONIndent(keeper.Cdc(), data)
	if err != nil {
		ctx.Logger().Error("fail to marshal vault data to json", "error", err)
		return nil, fmt.Errorf("fail to marshal response to json: %w", err)
	}
	return res, nil
}

// TODO: select vault by bond/funds ratio
// TODO: if asgard vaults hold more non-rune funds than bond, do not give address, and error
func queryPoolAddresses(ctx cosmos.Context, path []string, req abci.RequestQuery, keeper keeper.Keeper) ([]byte, error) {
	haltTrading, err := keeper.GetMimir(ctx, "HaltTrading")
	if err != nil {
		ctx.Logger().Error("fail to get HaltTrading mimir", "error", err)
	}
	// when trading is halt , do not return any pool addresses
	halted := (haltTrading > 0 && haltTrading < common.BlockHeight(ctx) && err == nil) || keeper.RagnarokInProgress(ctx)
	active, err := keeper.GetAsgardVaultsByStatus(ctx, ActiveVault)
	if err != nil {
		ctx.Logger().Error("fail to get active vaults", "error", err)
		return nil, fmt.Errorf("fail to get active vaults: %w", err)
	}

	// TODO: halted trading should be enabled per chain. This will be used to
	// decom a chain and not accept new trades/stakes

	type address struct {
		Chain   common.Chain   `json:"chain"`
		PubKey  common.PubKey  `json:"pub_key"`
		Address common.Address `json:"address"`
		Halted  bool           `json:"halted"`
	}

	var resp struct {
		Current []address `json:"current"`
	}
	// select vault with lowest amount of rune
	vault := active.SelectByMinCoin(common.RuneAsset())

	chains := vault.Chains

	if len(chains) == 0 {
		chains = common.Chains{common.RuneAsset().Chain}
	}

	for _, chain := range chains {
		vaultAddress, err := vault.PubKey.GetAddress(chain)
		if err != nil {
			ctx.Logger().Error("fail to get address for chain", "error", err)
			return nil, fmt.Errorf("fail to get address for chain: %w", err)
		}

		addr := address{
			Chain:   chain,
			PubKey:  vault.PubKey,
			Address: vaultAddress,
			Halted:  halted,
		}

		resp.Current = append(resp.Current, addr)
	}

	res, err := codec.MarshalJSONIndent(keeper.Cdc(), resp)
	if err != nil {
		ctx.Logger().Error("fail to marshal current pool address to json", "error", err)
		return nil, fmt.Errorf("fail to marshal current pool address to json: %w", err)
	}

	return res, nil
}

func queryNodeAccount(ctx cosmos.Context, path []string, req abci.RequestQuery, keeper keeper.Keeper) ([]byte, error) {
	if len(path) == 0 {
		return nil, errors.New("node address not provided")
	}
	nodeAddress := path[0]
	addr, err := cosmos.AccAddressFromBech32(nodeAddress)
	if err != nil {
		return nil, cosmos.ErrUnknownRequest("invalid account address")
	}

	nodeAcc, err := keeper.GetNodeAccount(ctx, addr)
	if err != nil {
		return nil, fmt.Errorf("fail to get node accounts: %w", err)
	}

	slashPts, err := keeper.GetNodeAccountSlashPoints(ctx, addr)
	if err != nil {
		return nil, fmt.Errorf("fail to get node slash points: %w", err)
	}
	jail, err := keeper.GetNodeAccountJail(ctx, nodeAcc.NodeAddress)
	if err != nil {
		return nil, fmt.Errorf("fail to get node jail: %w", err)
	}

	result := NewQueryNodeAccount(nodeAcc)
	result.SlashPoints = slashPts
	result.Jail = jail
	// CurrentAward is an estimation of reward for node in active status
	// Node in other status should not have current reward
	if nodeAcc.Status == NodeActive {
		vaultData, err := keeper.GetVaultData(ctx)
		if err != nil {
			return nil, fmt.Errorf("fail to get vaultData: %w", err)
		}

		// find number of blocks they were well behaved (ie active - slash points)
		earnedBlocks := nodeAcc.CalcBondUnits(common.BlockHeight(ctx), slashPts)
		result.CurrentAward = vaultData.CalcNodeRewards(earnedBlocks)
	}

	res, err := codec.MarshalJSONIndent(keeper.Cdc(), result)
	if err != nil {
		return nil, fmt.Errorf("fail to marshal node account to json: %w", err)
	}

	return res, nil
}

func queryNodeAccountCheck(ctx cosmos.Context, path []string, req abci.RequestQuery, keeper keeper.Keeper) ([]byte, error) {
	if len(path) == 0 {
		return nil, errors.New("node address not provided")
	}
	nodeAddress := path[0]
	addr, err := cosmos.AccAddressFromBech32(nodeAddress)
	if err != nil {
		return nil, cosmos.ErrUnknownRequest("invalid account address")
	}

	nodeAcc, err := keeper.GetNodeAccount(ctx, addr)
	if err != nil {
		return nil, fmt.Errorf("fail to get node accounts: %w", err)
	}
	if nodeAcc.Status == NodeUnknown {
		return nil, fmt.Errorf("node (%s) doesn't exist", nodeAddress)
	}

	version := keeper.GetLowestActiveVersion(ctx)
	constAccessor := constants.GetConstantValues(version)
	if constAccessor == nil {
		return nil, fmt.Errorf("constants for version(%s) is not available", version)
	}

	mgr := NewManagers(keeper)
	err = mgr.BeginBlock(ctx)
	if err != nil {
		return nil, fmt.Errorf("fail to build manager: %w", err)
	}
	result := QueryNodeAccountPreflightCheck{}
	status, err := mgr.ValidatorMgr().NodeAccountPreflightCheck(ctx, nodeAcc, constAccessor)
	result.Status = status
	if err != nil {
		result.Description = err.Error()
		result.Code = 1
	} else {
		result.Description = "OK"
		result.Code = 0
	}

	res, err := codec.MarshalJSONIndent(keeper.Cdc(), result)
	if err != nil {
		return nil, fmt.Errorf("fail to marshal node account to json: %w", err)
	}

	return res, nil
}

func queryNodeAccounts(ctx cosmos.Context, path []string, req abci.RequestQuery, keeper keeper.Keeper) ([]byte, error) {
	nodeAccounts, err := keeper.ListNodeAccountsWithBond(ctx)
	if err != nil {
		return nil, fmt.Errorf("fail to get node accounts: %w", err)
	}

	vaultData, err := keeper.GetVaultData(ctx)
	if err != nil {
		return nil, fmt.Errorf("fail to get vaultData: %w", err)
	}

	result := make([]QueryNodeAccount, len(nodeAccounts))
	for i, na := range nodeAccounts {
		slashPts, err := keeper.GetNodeAccountSlashPoints(ctx, na.NodeAddress)
		if err != nil {
			return nil, fmt.Errorf("fail to get node slash points: %w", err)
		}
		// find number of blocks they were well behaved (ie active - slash points)
		earnedBlocks := na.CalcBondUnits(common.BlockHeight(ctx), slashPts)

		result[i] = NewQueryNodeAccount(na)
		result[i].SlashPoints = slashPts
		if na.Status == NodeActive {
			result[i].CurrentAward = vaultData.CalcNodeRewards(earnedBlocks)
		}

		jail, err := keeper.GetNodeAccountJail(ctx, na.NodeAddress)
		if err != nil {
			return nil, fmt.Errorf("fail to get node jail: %w", err)
		}
		result[i].Jail = jail
	}

	res, err := codec.MarshalJSONIndent(keeper.Cdc(), result)
	if err != nil {
		ctx.Logger().Error("fail to marshal observers to json", "error", err)
		return nil, fmt.Errorf("fail to marshal observers to json: %w", err)
	}

	return res, nil
}

// queryObservers will only return all the active accounts
func queryObservers(ctx cosmos.Context, path []string, req abci.RequestQuery, keeper keeper.Keeper) ([]byte, error) {
	activeAccounts, err := keeper.ListActiveNodeAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("fail to get node account iterator: %w", err)
	}
	result := make([]string, 0, len(activeAccounts))
	for _, item := range activeAccounts {
		result = append(result, item.NodeAddress.String())
	}
	res, err := codec.MarshalJSONIndent(keeper.Cdc(), result)
	if err != nil {
		ctx.Logger().Error("fail to marshal observers to json", "error", err)
		return nil, fmt.Errorf("fail to marshal observers to json: %w", err)
	}

	return res, nil
}

func queryObserver(ctx cosmos.Context, path []string, req abci.RequestQuery, keeper keeper.Keeper) ([]byte, error) {
	if len(path) == 0 {
		return nil, errors.New("observer address not provided")
	}
	observerAddr := path[0]
	addr, err := cosmos.AccAddressFromBech32(observerAddr)
	if err != nil {
		return nil, fmt.Errorf("invalid account address: %w", err)
	}

	nodeAcc, err := keeper.GetNodeAccount(ctx, addr)
	if err != nil {
		return nil, fmt.Errorf("fail to get node account: %w", err)
	}
	if nodeAcc.Status == NodeUnknown {
		return nil, fmt.Errorf("node account(%s) doesn't exist", observerAddr)
	}
	res, err := codec.MarshalJSONIndent(keeper.Cdc(), nodeAcc)
	if err != nil {
		ctx.Logger().Error("fail to marshal node account to json", "error", err)
		return nil, fmt.Errorf("fail to marshal node account to json: %w", err)
	}

	return res, nil
}

// queryStakers
func queryStakers(ctx cosmos.Context, path []string, req abci.RequestQuery, keeper keeper.Keeper) ([]byte, error) {
	if len(path) == 0 {
		return nil, errors.New("asset not provided")
	}
	asset, err := common.NewAsset(path[0])
	if err != nil {
		ctx.Logger().Error("fail to get parse asset", "error", err)
		return nil, fmt.Errorf("fail to parse asset: %w", err)
	}
	var stakers []Staker
	iterator := keeper.GetStakerIterator(ctx, asset)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var staker Staker
		keeper.Cdc().MustUnmarshalBinaryBare(iterator.Value(), &staker)
		stakers = append(stakers, staker)
	}
	res, err := codec.MarshalJSONIndent(keeper.Cdc(), stakers)
	if err != nil {
		ctx.Logger().Error("fail to marshal stakers to json", "error", err)
		return nil, fmt.Errorf("fail to marshal stakers to json: %w", err)
	}
	return res, nil
}

// nolint: unparam
func queryPool(ctx cosmos.Context, path []string, req abci.RequestQuery, keeper keeper.Keeper) ([]byte, error) {
	if len(path) == 0 {
		return nil, errors.New("asset not provided")
	}
	asset, err := common.NewAsset(path[0])
	if err != nil {
		ctx.Logger().Error("fail to parse asset", "error", err)
		return nil, fmt.Errorf("could not parse asset: %w", err)
	}

	pool, err := keeper.GetPool(ctx, asset)
	if err != nil {
		ctx.Logger().Error("fail to get pool", "error", err)
		return nil, fmt.Errorf("could not get pool: %w", err)
	}
	if pool.IsEmpty() {
		return nil, fmt.Errorf("pool: %s doesn't exist", path[0])
	}

	res, err := codec.MarshalJSONIndent(keeper.Cdc(), pool)
	if err != nil {
		return nil, fmt.Errorf("could not marshal result to JSON: %w", err)
	}
	return res, nil
}

func queryPools(ctx cosmos.Context, req abci.RequestQuery, keeper keeper.Keeper) ([]byte, error) {
	pools := Pools{}
	iterator := keeper.GetPoolIterator(ctx)
	for ; iterator.Valid(); iterator.Next() {
		var pool Pool
		if err := keeper.Cdc().UnmarshalBinaryBare(iterator.Value(), &pool); err != nil {
			return nil, fmt.Errorf("fail to unmarshal pool: %w", err)
		}
		// ignore pool if no stake units
		if pool.PoolUnits.IsZero() {
			continue
		}
		pools = append(pools, pool)
	}
	res, err := codec.MarshalJSONIndent(keeper.Cdc(), pools)
	if err != nil {
		return nil, fmt.Errorf("could not marshal pools result to json: %w", err)
	}
	return res, nil
}

func queryTxInVoter(ctx cosmos.Context, path []string, req abci.RequestQuery, keeper keeper.Keeper) ([]byte, error) {
	if len(path) == 0 {
		return nil, errors.New("tx id not provided")
	}
	hash, err := common.NewTxID(path[0])
	if err != nil {
		ctx.Logger().Error("fail to parse tx id", "error", err)
		return nil, fmt.Errorf("fail to parse tx id: %w", err)
	}
	voter, err := keeper.GetObservedTxInVoter(ctx, hash)
	if err != nil {
		ctx.Logger().Error("fail to get observed tx voter", "error", err)
		return nil, fmt.Errorf("fail to get observed tx voter: %w", err)
	}
	if len(voter.Txs) == 0 {
		return nil, fmt.Errorf("tx voter not exist")
	}
	res, err := codec.MarshalJSONIndent(keeper.Cdc(), voter)
	if err != nil {
		ctx.Logger().Error("fail to marshal tx hash to json", "error", err)
		return nil, fmt.Errorf("fail to marshal tx hash to json: %w", err)
	}
	return res, nil
}

func queryTxIn(ctx cosmos.Context, path []string, req abci.RequestQuery, keeper keeper.Keeper) ([]byte, error) {
	if len(path) == 0 {
		return nil, errors.New("tx id not provided")
	}
	hash, err := common.NewTxID(path[0])
	if err != nil {
		ctx.Logger().Error("fail to parse tx id", "error", err)
		return nil, fmt.Errorf("fail to parse tx id: %w", err)
	}
	voter, err := keeper.GetObservedTxInVoter(ctx, hash)
	if err != nil {
		ctx.Logger().Error("fail to get observed tx voter", "error", err)
		return nil, fmt.Errorf("fail to get observed tx voter: %w", err)
	}
	if len(voter.Txs) == 0 {
		return nil, fmt.Errorf("tx: %s doesn't exist", hash)
	}

	nodeAccounts, err := keeper.ListActiveNodeAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("fail to get node accounts: %w", err)
	}
	res, err := codec.MarshalJSONIndent(keeper.Cdc(), voter.GetTxV13(nodeAccounts))
	if err != nil {
		ctx.Logger().Error("fail to marshal tx hash to json", "error", err)
		return nil, fmt.Errorf("fail to marshal tx hash to json: %w", err)
	}
	return res, nil
}

func queryKeygen(ctx cosmos.Context, kbs KeybaseStore, path []string, req abci.RequestQuery, keeper keeper.Keeper) ([]byte, error) {
	if len(path) == 0 {
		return nil, errors.New("block height not provided")
	}
	var err error
	height, err := strconv.ParseInt(path[0], 0, 64)
	if err != nil {
		ctx.Logger().Error("fail to parse block height", "error", err)
		return nil, fmt.Errorf("fail to parse block height: %w", err)
	}

	if height > common.BlockHeight(ctx) {
		return nil, fmt.Errorf("block height not available yet")
	}

	keygenBlock, err := keeper.GetKeygenBlock(ctx, height)
	if err != nil {
		ctx.Logger().Error("fail to get keygen block", "error", err)
		return nil, fmt.Errorf("fail to get keygen block: %w", err)
	}

	if len(path) > 1 {
		pk, err := common.NewPubKey(path[1])
		if err != nil {
			ctx.Logger().Error("fail to parse pubkey", "error", err)
			return nil, fmt.Errorf("fail to parse pubkey: %w", err)
		}
		// only return those keygen contains the request pub key
		newKeygenBlock := NewKeygenBlock(keygenBlock.Height)
		for _, keygen := range keygenBlock.Keygens {
			if keygen.Members.Contains(pk) {
				newKeygenBlock.Keygens = append(newKeygenBlock.Keygens, keygen)
			}
		}
		keygenBlock = newKeygenBlock
	}

	buf, err := keeper.Cdc().MarshalBinaryBare(keygenBlock)
	if err != nil {
		ctx.Logger().Error("fail to marshal keygen block to json", "error", err)
		return nil, fmt.Errorf("fail to marshal keygen block to json: %w", err)
	}
	sig, _, err := kbs.Keybase.Sign(kbs.SignerName, kbs.SignerPasswd, buf)
	if err != nil {
		ctx.Logger().Error("fail to sign keygen", "error", err)
		return nil, fmt.Errorf("fail to sign keygen: %w", err)
	}

	query := QueryKeygenBlock{
		KeygenBlock: keygenBlock,
		Signature:   base64.StdEncoding.EncodeToString(sig),
	}

	res, err := codec.MarshalJSONIndent(keeper.Cdc(), query)
	if err != nil {
		ctx.Logger().Error("fail to marshal keygen block to json", "error", err)
		return nil, fmt.Errorf("fail to marshal keygen block to json: %w", err)
	}
	return res, nil
}

func queryKeysign(ctx cosmos.Context, kbs KeybaseStore, path []string, req abci.RequestQuery, keeper keeper.Keeper) ([]byte, error) {
	if len(path) == 0 {
		return nil, errors.New("block height not provided")
	}
	var err error
	height, err := strconv.ParseInt(path[0], 0, 64)
	if err != nil {
		ctx.Logger().Error("fail to parse block height", "error", err)
		return nil, fmt.Errorf("fail to parse block height: %w", err)
	}

	if height > common.BlockHeight(ctx) {
		return nil, fmt.Errorf("block height not available yet")
	}

	pk := common.EmptyPubKey
	if len(path) > 1 {
		pk, err = common.NewPubKey(path[1])
		if err != nil {
			ctx.Logger().Error("fail to parse pubkey", "error", err)
			return nil, fmt.Errorf("fail to parse pubkey: %w", err)
		}
	}

	txs, err := keeper.GetTxOut(ctx, height)
	if err != nil {
		ctx.Logger().Error("fail to get tx out array from key value store", "error", err)
		return nil, fmt.Errorf("fail to get tx out array from key value store: %w", err)
	}

	if !pk.IsEmpty() {
		newTxs := &TxOut{
			Height: txs.Height,
		}
		for _, tx := range txs.TxArray {
			if pk.Equals(tx.VaultPubKey) {
				newTxs.TxArray = append(newTxs.TxArray, tx)
			}
		}
		txs = newTxs
	}

	buf, err := keeper.Cdc().MarshalBinaryBare(txs)
	if err != nil {
		ctx.Logger().Error("fail to marshal keysign block to json", "error", err)
		return nil, fmt.Errorf("fail to marshal keysign block to json: %w", err)
	}
	sig, _, err := kbs.Keybase.Sign(kbs.SignerName, kbs.SignerPasswd, buf)
	if err != nil {
		ctx.Logger().Error("fail to sign keysign", "error", err)
		return nil, fmt.Errorf("fail to sign keysign: %w", err)
	}

	query := QueryKeysign{
		Keysign:   *txs,
		Signature: base64.StdEncoding.EncodeToString(sig),
	}

	res, err := codec.MarshalJSONIndent(keeper.Cdc(), query)
	if err != nil {
		ctx.Logger().Error("fail to marshal tx hash to json", "error", err)
		return nil, fmt.Errorf("fail to marshal tx hash to json: %w", err)
	}
	return res, nil
}

// queryOutQueue - iterates over txout, counting how many transactions are waiting to be sent
func queryQueue(ctx cosmos.Context, path []string, req abci.RequestQuery, keeper keeper.Keeper) ([]byte, error) {
	version := keeper.GetLowestActiveVersion(ctx)
	constAccessor := constants.GetConstantValues(version)
	signingTransactionPeriod := constAccessor.GetInt64Value(constants.SigningTransactionPeriod)
	startHeight := common.BlockHeight(ctx) - signingTransactionPeriod
	query := QueryQueue{}

	iterator := keeper.GetSwapQueueIterator(ctx)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var msg MsgSwap
		if err := keeper.Cdc().UnmarshalBinaryBare(iterator.Value(), &msg); err != nil {
			continue
		}
		query.Swap++
	}

	for height := startHeight; height <= common.BlockHeight(ctx); height++ {
		txs, err := keeper.GetTxOut(ctx, height)
		if err != nil {
			ctx.Logger().Error("fail to get tx out array from key value store", "error", err)
			return nil, fmt.Errorf("fail to get tx out array from key value store: %w", err)
		}
		for _, tx := range txs.TxArray {
			if tx.OutHash.IsEmpty() {
				query.Outbound++
			}
		}
	}

	res, err := codec.MarshalJSONIndent(keeper.Cdc(), query)
	if err != nil {
		ctx.Logger().Error("fail to marshal out queue to json", "error", err)
		return nil, fmt.Errorf("fail to marshal out queue to json: %w", err)
	}
	return res, nil
}

func queryHeights(ctx cosmos.Context, path []string, req abci.RequestQuery, keeper keeper.Keeper) ([]byte, error) {
	chain := common.BNBChain
	if len(path) > 0 && len(path[0]) > 0 {
		var err error
		chain, err = common.NewChain(path[0])
		if err != nil {
			ctx.Logger().Error("fail to retrieve chain", "error", err)
			return nil, fmt.Errorf("fail to retrieve chain: %w", err)
		}
	}
	chainHeight, err := keeper.GetLastChainHeight(ctx, chain)
	if err != nil {
		return nil, fmt.Errorf("fail to get last chain height: %w", err)
	}

	signed, err := keeper.GetLastSignedHeight(ctx)
	if err != nil {
		return nil, fmt.Errorf("fail to get last sign height: %w", err)
	}

	res, err := codec.MarshalJSONIndent(keeper.Cdc(), QueryResHeights{
		Chain:            chain,
		LastChainHeight:  chainHeight,
		LastSignedHeight: signed,
		Thorchain:        common.BlockHeight(ctx),
	})
	if err != nil {
		ctx.Logger().Error("fail to marshal events to json", "error", err)
		return nil, fmt.Errorf("fail to marshal events to json: %w", err)
	}
	return res, nil
}

// queryTSSSigner
func queryTSSSigners(ctx cosmos.Context, path []string, req abci.RequestQuery, keeper keeper.Keeper) ([]byte, error) {
	if len(path) == 0 {
		return nil, fmt.Errorf("vault pubkey not provided")
	}
	vaultPubKey := path[0]
	if len(vaultPubKey) == 0 {
		ctx.Logger().Error("empty vault pub key")
		return nil, fmt.Errorf("empty pool pub key")
	}
	pk, err := common.NewPubKey(vaultPubKey)
	if err != nil {
		ctx.Logger().Error("fail to parse pool pub key", "error", err)
		return nil, fmt.Errorf("invalid pool pub key(%s): %w", vaultPubKey, err)
	}
	vault, err := keeper.GetVault(ctx, pk)
	if err != nil {
		ctx.Logger().Error("fail to get vault", "error", err)
		return nil, fmt.Errorf("fail to get vault: %w", err)
	}
	res, err := codec.MarshalJSONIndent(keeper.Cdc(), vault.SigningParty)
	if err != nil {
		ctx.Logger().Error("fail to marshal to json", "error", err)
		return nil, fmt.Errorf("fail to marshal to json: %w", err)
	}

	return res, nil
}

func queryConstantValues(ctx cosmos.Context, path []string, req abci.RequestQuery, keeper keeper.Keeper) ([]byte, error) {
	ver := keeper.GetLowestActiveVersion(ctx)
	constAccessor := constants.GetConstantValues(ver)
	res, err := codec.MarshalJSONIndent(keeper.Cdc(), constAccessor)
	if err != nil {
		ctx.Logger().Error("fail to marshal constant values to json", "error", err)
		return nil, fmt.Errorf("fail to marshal constant values to json: %w", err)
	}
	return res, nil
}

func queryVersion(ctx cosmos.Context, path []string, req abci.RequestQuery, keeper keeper.Keeper) ([]byte, error) {
	ver := QueryVersion{
		Current: keeper.GetLowestActiveVersion(ctx),
		Next:    keeper.GetMinJoinVersion(ctx),
	}
	res, err := codec.MarshalJSONIndent(keeper.Cdc(), ver)
	if err != nil {
		ctx.Logger().Error("fail to marshal version to json", "error", err)
		return nil, fmt.Errorf("fail to marshal version to json: %w", err)
	}
	return res, nil
}

func queryMimirValues(ctx cosmos.Context, path []string, req abci.RequestQuery, keeper keeper.Keeper) ([]byte, error) {
	values := make(map[string]int64, 0)
	iter := keeper.GetMimirIterator(ctx)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var value int64
		if err := keeper.Cdc().UnmarshalBinaryBare(iter.Value(), &value); err != nil {
			ctx.Logger().Error("fail to unmarshal mimir attribute", "error", err)
			return nil, fmt.Errorf("fail to unmarshal mimir attribute:  %w", err)
		}
		values[string(iter.Key())] = value
	}
	res, err := codec.MarshalJSONIndent(keeper.Cdc(), values)
	if err != nil {
		ctx.Logger().Error("fail to marshal mimir values to json", "error", err)
		return nil, fmt.Errorf("fail to marshal mimir values to json: %w", err)
	}
	return res, nil
}

func queryBan(ctx cosmos.Context, path []string, req abci.RequestQuery, keeper keeper.Keeper) ([]byte, error) {
	if len(path) == 0 {
		return nil, errors.New("node address not available")
	}
	addr, err := cosmos.AccAddressFromBech32(path[0])
	if err != nil {
		ctx.Logger().Error("invalid node address", "error", err)
		return nil, fmt.Errorf("invalid node address: %w", err)
	}

	ban, err := keeper.GetBanVoter(ctx, addr)
	if err != nil {
		ctx.Logger().Error("fail to get ban voter", "error", err)
		return nil, fmt.Errorf("fail to get ban voter: %w", err)
	}

	res, err := codec.MarshalJSONIndent(keeper.Cdc(), ban)
	if err != nil {
		ctx.Logger().Error("fail to marshal ban voter to json", "error", err)
		return nil, fmt.Errorf("fail to ban voter to json: %w", err)
	}
	return res, nil
}

func queryPendingOutbound(ctx cosmos.Context, keeper keeper.Keeper) ([]byte, error) {
	version := keeper.GetLowestActiveVersion(ctx)
	constAccessor := constants.GetConstantValues(version)
	signingTransactionPeriod := constAccessor.GetInt64Value(constants.SigningTransactionPeriod)
	startHeight := common.BlockHeight(ctx) - signingTransactionPeriod
	var result []TxOutItem
	for height := startHeight; height <= common.BlockHeight(ctx); height++ {
		txs, err := keeper.GetTxOut(ctx, height)
		if err != nil {
			ctx.Logger().Error("fail to get tx out array from key value store", "error", err)
			return nil, fmt.Errorf("fail to get tx out array from key value store: %w", err)
		}
		for _, tx := range txs.TxArray {
			if tx.OutHash.IsEmpty() {
				result = append(result, *tx)
			}
		}
	}

	res, err := codec.MarshalJSONIndent(keeper.Cdc(), result)
	if err != nil {
		ctx.Logger().Error("fail to marshal pending outbound tx to json", "error", err)
		return nil, fmt.Errorf("fail to marshal pending outbound tx to json: %w", err)
	}
	return res, nil
}
