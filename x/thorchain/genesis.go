package thorchain

import (
	"errors"
	"fmt"
	"strings"

	abci "github.com/tendermint/tendermint/abci/types"
	tmtypes "github.com/tendermint/tendermint/types"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/x/thorchain/keeper"
)

// GenesisState strcture that used to store the data THORNode put in genesis
type GenesisState struct {
	Pools                []Pool                    `json:"pools"`
	Stakers              []Staker                  `json:"stakers"`
	ObservedTxInVoters   ObservedTxVoters          `json:"observed_tx_in_voters"`
	ObservedTxOutVoters  ObservedTxVoters          `json:"observed_tx_out_voters"`
	TxOuts               []TxOut                   `json:"txouts"`
	NodeAccounts         NodeAccounts              `json:"node_accounts"`
	Vaults               Vaults                    `json:"vaults"`
	Gas                  map[string][]cosmos.Uint  `json:"gas"`
	Reserve              uint64                    `json:"reserve"`
	BanVoters            []BanVoter                `json:"ban_voters"`
	LastSignedHeight     int64                     `json:"last_signed_height"`
	LastChainHeights     map[string]int64          `json:"last_chain_heights"`
	ReserveContributors  ReserveContributors       `json:"reserve_contributors"`
	VaultData            VaultData                 `json:"vault_data"`
	TssVoters            []TssVoter                `json:"tss_voters"`
	TssKeysignFailVoters []TssKeysignFailVoter     `json:"tss_keysign_fail_voters"`
	KeygenBlocks         []KeygenBlock             `json:"keygen_blocks"`
	AllTxMarkers         map[string]TxMarkers      `json:"all_tx_markers"`
	ErrataTxVoters       []ErrataTxVoter           `json:"errata_tx_voters"`
	MsgSwaps             []MsgSwap                 `json:"msg_swaps"`
	NetworkFees          []NetworkFee              `json:"network_fees"`
	NetworkFeeVoters     []ObservedNetworkFeeVoter `json:"network_fee_voters"`
}

// NewGenesisState create a new instance of GenesisState
func NewGenesisState() GenesisState {
	return GenesisState{}
}

// ValidateGenesis validate genesis is valid or not
func ValidateGenesis(data GenesisState) error {
	for _, record := range data.Pools {
		if err := record.Valid(); err != nil {
			return err
		}
	}

	for _, voter := range data.ObservedTxInVoters {
		if err := voter.Valid(); err != nil {
			return err
		}
	}

	for _, voter := range data.ObservedTxOutVoters {
		if err := voter.Valid(); err != nil {
			return err
		}
	}

	for _, out := range data.TxOuts {
		if err := out.Valid(); err != nil {
			return err
		}
	}

	for _, ta := range data.NodeAccounts {
		if err := ta.Valid(); err != nil {
			return err
		}
	}

	for _, vault := range data.Vaults {
		if err := vault.Valid(); err != nil {
			return err
		}
	}

	for k, v := range data.Gas {
		if len(v) == 0 {
			return fmt.Errorf("gas %s cannot have empty units", k)
		}
	}
	for _, bv := range data.BanVoters {
		if err := bv.Valid(); err != nil {
			return fmt.Errorf("invalid ban voter: %w", err)
		}
	}

	if data.LastSignedHeight < 0 {
		return errors.New("last signed height cannot be negative")
	}
	for c, h := range data.LastChainHeights {
		if h < 0 {
			return fmt.Errorf("invalid chain(%s) height", c)
		}
	}
	for _, r := range data.ReserveContributors {
		if err := r.Valid(); err != nil {
			return fmt.Errorf("invalid reserve contributor:%w", err)
		}
	}

	for _, b := range data.KeygenBlocks {
		for _, kb := range b.Keygens {
			if err := kb.Valid(); err != nil {
				return fmt.Errorf("invalid keygen: %w", err)
			}
		}
	}
	for _, item := range data.MsgSwaps {
		if err := item.ValidateBasic(); err != nil {
			return fmt.Errorf("invalid swap msg: %w", err)
		}
	}
	for _, nf := range data.NetworkFees {
		if err := nf.Valid(); err != nil {
			return fmt.Errorf("invalid network fee: %w", err)
		}
	}

	return nil
}

// DefaultGenesisState the default values THORNode put in the Genesis
func DefaultGenesisState() GenesisState {
	return GenesisState{
		Pools:                []Pool{},
		NodeAccounts:         NodeAccounts{},
		TxOuts:               make([]TxOut, 0),
		Stakers:              make([]Staker, 0),
		Vaults:               make(Vaults, 0),
		ObservedTxInVoters:   make(ObservedTxVoters, 0),
		ObservedTxOutVoters:  make(ObservedTxVoters, 0),
		Gas:                  make(map[string][]cosmos.Uint),
		BanVoters:            make([]BanVoter, 0),
		LastSignedHeight:     0,
		LastChainHeights:     make(map[string]int64),
		ReserveContributors:  ReserveContributors{},
		VaultData:            NewVaultData(),
		TssVoters:            make([]TssVoter, 0),
		TssKeysignFailVoters: make([]TssKeysignFailVoter, 0),
		KeygenBlocks:         make([]KeygenBlock, 0),
		AllTxMarkers:         make(map[string]TxMarkers),
		ErrataTxVoters:       make([]ErrataTxVoter, 0),
		MsgSwaps:             make([]MsgSwap, 0),
		NetworkFees:          make([]NetworkFee, 0),
		NetworkFeeVoters:     make([]ObservedNetworkFeeVoter, 0),
	}
}

// InitGenesis read the data in GenesisState and apply it to data store
func InitGenesis(ctx cosmos.Context, keeper keeper.Keeper, data GenesisState) []abci.ValidatorUpdate {
	for _, record := range data.Pools {
		if err := keeper.SetPool(ctx, record); err != nil {
			panic(err)
		}
	}

	for _, stake := range data.Stakers {
		keeper.SetStaker(ctx, stake)
	}

	validators := make([]abci.ValidatorUpdate, 0, len(data.NodeAccounts))
	for _, nodeAccount := range data.NodeAccounts {
		if nodeAccount.Status == NodeActive {
			// Only Active node will become validator
			pk, err := cosmos.GetPubKeyFromBech32(cosmos.Bech32PubKeyTypeConsPub, nodeAccount.ValidatorConsPubKey)
			if err != nil {
				ctx.Logger().Error("fail to parse consensus public key", "key", nodeAccount.ValidatorConsPubKey, "error", err)
				panic(err)
			}
			validators = append(validators, abci.ValidatorUpdate{
				PubKey: tmtypes.TM2PB.PubKey(pk),
				Power:  100,
			})
		}

		if err := keeper.SetNodeAccount(ctx, nodeAccount); err != nil {
			// we should panic
			panic(err)
		}
	}

	for _, vault := range data.Vaults {
		if err := keeper.SetVault(ctx, vault); err != nil {
			panic(err)
		}
	}

	for _, voter := range data.ObservedTxInVoters {
		keeper.SetObservedTxInVoter(ctx, voter)
	}

	for _, voter := range data.ObservedTxOutVoters {
		keeper.SetObservedTxOutVoter(ctx, voter)
	}

	for _, bv := range data.BanVoters {
		keeper.SetBanVoter(ctx, bv)
	}

	for _, out := range data.TxOuts {
		if err := keeper.SetTxOut(ctx, &out); err != nil {
			ctx.Logger().Error("fail to save tx out during genesis", "error", err)
			panic(err)
		}
	}

	for k, v := range data.Gas {
		keyParts := strings.Split(k, "/")
		asset, err := common.NewAsset(keyParts[len(keyParts)-1])
		if err != nil {
			panic(err)
		}
		keeper.SetGas(ctx, asset, v)
	}
	if data.LastSignedHeight > 0 {
		if err := keeper.SetLastSignedHeight(ctx, data.LastSignedHeight); err != nil {
			panic(err)
		}
	}

	for c, h := range data.LastChainHeights {
		chain, err := common.NewChain(c)
		if err != nil {
			panic(err)
		}
		if err := keeper.SetLastChainHeight(ctx, chain, h); err != nil {
			panic(err)
		}
	}
	if len(data.ReserveContributors) > 0 {
		if err := keeper.SetReserveContributors(ctx, data.ReserveContributors); err != nil {
			panic(err)
		}
	}
	if err := keeper.SetVaultData(ctx, data.VaultData); err != nil {
		panic(err)
	}

	for _, tv := range data.TssVoters {
		if tv.IsEmpty() {
			continue
		}
		keeper.SetTssVoter(ctx, tv)
	}
	for _, item := range data.TssKeysignFailVoters {
		if item.Empty() {
			continue
		}
		keeper.SetTssKeysignFailVoter(ctx, item)
	}

	for _, item := range data.KeygenBlocks {
		if item.IsEmpty() {
			continue
		}
		keeper.SetKeygenBlock(ctx, item)
	}

	for hash, item := range data.AllTxMarkers {
		if err := keeper.SetTxMarkers(ctx, hash, item); err != nil {
			panic(err)
		}
	}
	for _, item := range data.ErrataTxVoters {
		if item.Empty() {
			continue
		}
		keeper.SetErrataTxVoter(ctx, item)
	}

	for _, item := range data.MsgSwaps {
		if err := keeper.SetSwapQueueItem(ctx, item); err != nil {
			panic(err)
		}
	}
	for _, nf := range data.NetworkFees {
		if err := keeper.SaveNetworkFee(ctx, nf.Chain, nf); err != nil {
			panic(err)
		}
	}

	for _, nf := range data.NetworkFeeVoters {
		keeper.SetObservedNetworkFeeVoter(ctx, nf)
	}

	if common.RuneAsset().Chain.Equals(common.THORChain) {
		// Mint coins into the reserve
		coin, err := common.NewCoin(common.RuneNative, cosmos.NewUint(data.Reserve)).Native()
		if err != nil {
			panic(err)
		}
		coins := cosmos.NewCoins(coin)
		if err := keeper.Supply().MintCoins(ctx, ModuleName, coins); err != nil {
			panic(err)
		}
		if err := keeper.Supply().SendCoinsFromModuleToModule(ctx, ModuleName, ReserveName, coins); err != nil {
			panic(err)
		}
	}

	for _, admin := range ADMINS {
		addr, err := cosmos.AccAddressFromBech32(admin)
		if err != nil {
			panic(err)
		}
		// give mimir gas
		coinsToMint, err := cosmos.ParseCoins("1000thor")
		if err != nil {
			panic(err)
		}
		// mint some gas asset
		err = keeper.Supply().MintCoins(ctx, ModuleName, coinsToMint)
		if err != nil {
			panic(err)
		}
		if err := keeper.Supply().SendCoinsFromModuleToAccount(ctx, ModuleName, addr, coinsToMint); err != nil {
			panic(err)
		}
	}

	if common.RuneAsset().Chain.Equals(common.THORChain) {
		ctx.Logger().Info("Reserve Module", "address", keeper.Supply().GetModuleAddress(ReserveName).String())
		ctx.Logger().Info("Bond    Module", "address", keeper.Supply().GetModuleAddress(BondName).String())
		ctx.Logger().Info("Asgard  Module", "address", keeper.Supply().GetModuleAddress(AsgardName).String())
	}

	return validators
}

func getStakers(ctx cosmos.Context, k keeper.Keeper, asset common.Asset) []Staker {
	stakers := make([]Staker, 0)
	iterator := k.GetStakerIterator(ctx, asset)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var ps Staker
		k.Cdc().MustUnmarshalBinaryBare(iterator.Value(), &ps)
		stakers = append(stakers, ps)
	}
	return stakers
}

// ExportGenesis export the data in Genesis
func ExportGenesis(ctx cosmos.Context, k keeper.Keeper) GenesisState {
	var iterator cosmos.Iterator

	pools, err := k.GetPools(ctx)
	if err != nil {
		panic(err)
	}

	var stakers []Staker
	for _, pool := range pools {
		stakers = append(stakers, getStakers(ctx, k, pool.Asset)...)
	}

	var nodeAccounts NodeAccounts
	iterator = k.GetNodeAccountIterator(ctx)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var na NodeAccount
		k.Cdc().MustUnmarshalBinaryBare(iterator.Value(), &na)
		nodeAccounts = append(nodeAccounts, na)
	}

	var observedTxInVoters ObservedTxVoters
	iterator = k.GetObservedTxInVoterIterator(ctx)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var vote ObservedTxVoter
		k.Cdc().MustUnmarshalBinaryBare(iterator.Value(), &vote)
		observedTxInVoters = append(observedTxInVoters, vote)
	}

	var observedTxOutVoters ObservedTxVoters
	iterator = k.GetObservedTxOutVoterIterator(ctx)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var vote ObservedTxVoter
		k.Cdc().MustUnmarshalBinaryBare(iterator.Value(), &vote)
		observedTxOutVoters = append(observedTxOutVoters, vote)
	}

	var outs []TxOut
	iterator = k.GetTxOutIterator(ctx)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var out TxOut
		k.Cdc().MustUnmarshalBinaryBare(iterator.Value(), &out)
		outs = append(outs, out)
	}

	gas := make(map[string][]cosmos.Uint, 0)
	iterator = k.GetGasIterator(ctx)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var g []cosmos.Uint
		k.Cdc().MustUnmarshalBinaryBare(iterator.Value(), &g)
		gas[string(iterator.Key())] = g
	}

	banVoters := make([]BanVoter, 0)
	iteratorBanVoters := k.GetBanVoterIterator(ctx)
	defer iteratorBanVoters.Close()
	for ; iteratorBanVoters.Valid(); iteratorBanVoters.Next() {
		var bv BanVoter
		k.Cdc().MustUnmarshalBinaryBare(iteratorBanVoters.Value(), &bv)
		banVoters = append(banVoters, bv)
	}

	lastSignedHeight, err := k.GetLastSignedHeight(ctx)
	if err != nil {
		panic(err)
	}

	chainHeights, err := k.GetLastChainHeights(ctx)
	if err != nil {
		panic(err)
	}
	lastChainHeights := make(map[string]int64, 0)
	for k, v := range chainHeights {
		lastChainHeights[k.String()] = v
	}

	reserveContributors, err := k.GetReservesContributors(ctx)
	if err != nil {
		panic(err)
	}

	vaultData, err := k.GetVaultData(ctx)
	if err != nil {
		panic(err)
	}

	vaults := make(Vaults, 0)
	iterVault := k.GetVaultIterator(ctx)
	defer iterVault.Close()
	for ; iterVault.Valid(); iterVault.Next() {
		var vault Vault
		k.Cdc().MustUnmarshalBinaryBare(iterVault.Value(), &vault)
		vaults = append(vaults, vault)
	}

	tssVoters := make([]TssVoter, 0)
	iterTssVoter := k.GetTssVoterIterator(ctx)
	defer iterTssVoter.Close()
	for ; iterTssVoter.Valid(); iterTssVoter.Next() {
		var tv TssVoter
		k.Cdc().MustUnmarshalBinaryBare(iterTssVoter.Value(), &tv)
		tssVoters = append(tssVoters, tv)
	}

	tssKeySignFailVoters := make([]TssKeysignFailVoter, 0)
	iterTssKeysignFailVoter := k.GetTssKeysignFailVoterIterator(ctx)
	defer iterTssKeysignFailVoter.Close()
	for ; iterTssKeysignFailVoter.Valid(); iterTssKeysignFailVoter.Next() {
		var t TssKeysignFailVoter
		k.Cdc().MustUnmarshalBinaryBare(iterTssKeysignFailVoter.Value(), &t)
		tssKeySignFailVoters = append(tssKeySignFailVoters, t)
	}

	keygenBlocks := make([]KeygenBlock, 0)
	iterKeygenBlocks := k.GetKeygenBlockIterator(ctx)
	for ; iterKeygenBlocks.Valid(); iterKeygenBlocks.Next() {
		var kb KeygenBlock
		k.Cdc().MustUnmarshalBinaryBare(iterKeygenBlocks.Value(), &kb)
		keygenBlocks = append(keygenBlocks, kb)
	}

	allTxMarkers, err := k.GetAllTxMarkers(ctx)
	if err != nil {
		panic(err)
	}

	errataVoters := make([]ErrataTxVoter, 0)
	iterErrata := k.GetErrataTxVoterIterator(ctx)
	defer iterErrata.Close()
	for ; iterErrata.Valid(); iterErrata.Next() {
		var et ErrataTxVoter
		k.Cdc().MustUnmarshalBinaryBare(iterErrata.Value(), &et)
		errataVoters = append(errataVoters, et)
	}

	swapMsgs := make([]MsgSwap, 0)
	iterMsgSwap := k.GetSwapQueueIterator(ctx)
	defer iterMsgSwap.Close()
	for ; iterMsgSwap.Valid(); iterMsgSwap.Next() {
		var m MsgSwap
		k.Cdc().MustUnmarshalBinaryBare(iterMsgSwap.Value(), &m)
		swapMsgs = append(swapMsgs, m)
	}

	networkFees := make([]NetworkFee, 0)
	iterNetworkFee := k.GetNetworkFeeIterator(ctx)
	defer iterNetworkFee.Close()
	for ; iterNetworkFee.Valid(); iterNetworkFee.Next() {
		var nf NetworkFee
		k.Cdc().MustUnmarshalBinaryBare(iterNetworkFee.Value(), &nf)
		networkFees = append(networkFees, nf)
	}

	networkFeeVoters := make([]ObservedNetworkFeeVoter, 0)
	iterNetworkFeeVoter := k.GetObservedNetworkFeeVoterIterator(ctx)
	defer iterNetworkFeeVoter.Close()
	for ; iterNetworkFeeVoter.Valid(); iterNetworkFeeVoter.Next() {
		var nf ObservedNetworkFeeVoter
		k.Cdc().MustUnmarshalBinaryBare(iterNetworkFeeVoter.Value(), &nf)
		networkFeeVoters = append(networkFeeVoters, nf)
	}
	return GenesisState{
		Pools:                pools,
		Stakers:              stakers,
		ObservedTxInVoters:   observedTxInVoters,
		ObservedTxOutVoters:  observedTxOutVoters,
		TxOuts:               outs,
		NodeAccounts:         nodeAccounts,
		Vaults:               vaults,
		Gas:                  gas,
		BanVoters:            banVoters,
		LastSignedHeight:     lastSignedHeight,
		LastChainHeights:     lastChainHeights,
		ReserveContributors:  reserveContributors,
		VaultData:            vaultData,
		TssVoters:            tssVoters,
		TssKeysignFailVoters: tssKeySignFailVoters,
		KeygenBlocks:         keygenBlocks,
		AllTxMarkers:         allTxMarkers,
		ErrataTxVoters:       errataVoters,
		MsgSwaps:             swapMsgs,
		NetworkFees:          networkFees,
		NetworkFeeVoters:     networkFeeVoters,
	}
}
