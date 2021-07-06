package keeper

import (
	"errors"
	"fmt"

	"github.com/blang/semver"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/supply"
	"github.com/tendermint/tendermint/libs/log"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	kvTypes "gitlab.com/thorchain/thornode/x/thorchain/keeper/types"
	"gitlab.com/thorchain/thornode/x/thorchain/types"
)

var kaboom = errors.New("Kaboom!!!")

type KVStoreDummy struct{}

func (k KVStoreDummy) Cdc() *codec.Codec       { return types.MakeTestCodec() }
func (k KVStoreDummy) Supply() supply.Keeper   { return supply.Keeper{} }
func (k KVStoreDummy) CoinKeeper() bank.Keeper { return bank.BaseKeeper{} }
func (k KVStoreDummy) Logger(ctx cosmos.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", ModuleName))
}

func (k KVStoreDummy) Version() int64 { return 0 }
func (k KVStoreDummy) GetKey(_ cosmos.Context, prefix kvTypes.DbPrefix, key string) string {
	return fmt.Sprintf("%s/1/%s", prefix, key)
}

func (k KVStoreDummy) GetStoreVersion(ctx cosmos.Context) int64      { return 1 }
func (k KVStoreDummy) SetStoreVersion(ctx cosmos.Context, ver int64) {}

func (k KVStoreDummy) GetRuneBalanceOfModule(ctx cosmos.Context, moduleName string) cosmos.Uint {
	return cosmos.ZeroUint()
}

func (k KVStoreDummy) SendFromModuleToModule(ctx cosmos.Context, from, to string, coin common.Coin) error {
	return kaboom
}

func (k KVStoreDummy) SendFromAccountToModule(ctx cosmos.Context, from cosmos.AccAddress, to string, coin common.Coin) error {
	return kaboom
}

func (k KVStoreDummy) SendFromModuleToAccount(ctx cosmos.Context, from string, to cosmos.AccAddress, coin common.Coin) error {
	return kaboom
}

func (k KVStoreDummy) SetLastSignedHeight(_ cosmos.Context, _ int64) error { return kaboom }
func (k KVStoreDummy) GetLastSignedHeight(_ cosmos.Context) (int64, error) {
	return 0, kaboom
}

func (k KVStoreDummy) SetLastChainHeight(_ cosmos.Context, _ common.Chain, _ int64) error {
	return kaboom
}

func (k KVStoreDummy) GetLastChainHeight(_ cosmos.Context, _ common.Chain) (int64, error) {
	return 0, kaboom
}

func (k KVStoreDummy) GetLastChainHeights(ctx cosmos.Context) (map[common.Chain]int64, error) {
	return nil, kaboom
}

func (k KVStoreDummy) GetRagnarokBlockHeight(_ cosmos.Context) (int64, error) {
	return 0, kaboom
}
func (k KVStoreDummy) SetRagnarokBlockHeight(_ cosmos.Context, _ int64) {}
func (k KVStoreDummy) GetRagnarokNth(_ cosmos.Context) (int64, error) {
	return 0, kaboom
}
func (k KVStoreDummy) SetRagnarokNth(_ cosmos.Context, _ int64) {}
func (k KVStoreDummy) GetRagnarokPending(_ cosmos.Context) (int64, error) {
	return 0, kaboom
}
func (k KVStoreDummy) SetRagnarokPending(_ cosmos.Context, _ int64) {}
func (k KVStoreDummy) RagnarokInProgress(_ cosmos.Context) bool     { return false }
func (k KVStoreDummy) GetRagnarokUnstakPosition(ctx cosmos.Context) (RagnarokUnstakePosition, error) {
	return RagnarokUnstakePosition{}, kaboom
}
func (k KVStoreDummy) SetRagnarokUnstakPosition(_tx cosmos.Context, _ RagnarokUnstakePosition) {}

func (k KVStoreDummy) GetPoolBalances(_ cosmos.Context, _, _ common.Asset) (cosmos.Uint, cosmos.Uint) {
	return cosmos.ZeroUint(), cosmos.ZeroUint()
}

func (k KVStoreDummy) GetPoolIterator(_ cosmos.Context) cosmos.Iterator {
	return NewDummyIterator()
}
func (k KVStoreDummy) SetPoolData(_ cosmos.Context, _ common.Asset, _ PoolStatus) {}
func (k KVStoreDummy) GetPoolDataIterator(_ cosmos.Context) cosmos.Iterator {
	return NewDummyIterator()
}
func (k KVStoreDummy) EnableAPool(_ cosmos.Context) {}

func (k KVStoreDummy) GetPool(_ cosmos.Context, _ common.Asset) (Pool, error) {
	return Pool{}, kaboom
}
func (k KVStoreDummy) GetPools(_ cosmos.Context) (Pools, error)                           { return nil, kaboom }
func (k KVStoreDummy) SetPool(_ cosmos.Context, _ Pool) error                             { return kaboom }
func (k KVStoreDummy) PoolExist(_ cosmos.Context, _ common.Asset) bool                    { return false }
func (k KVStoreDummy) RemovePool(_ cosmos.Context, _ common.Asset)                        {}
func (k KVStoreDummy) GetStakerIterator(_ cosmos.Context, _ common.Asset) cosmos.Iterator { return nil }
func (k KVStoreDummy) GetStaker(_ cosmos.Context, _ common.Asset, _ common.Address) (Staker, error) {
	return Staker{}, kaboom
}
func (k KVStoreDummy) SetStaker(_ cosmos.Context, _ Staker)                 {}
func (k KVStoreDummy) RemoveStaker(_ cosmos.Context, _ Staker)              {}
func (k KVStoreDummy) TotalActiveNodeAccount(_ cosmos.Context) (int, error) { return 0, kaboom }
func (k KVStoreDummy) ListNodeAccountsWithBond(_ cosmos.Context) (NodeAccounts, error) {
	return nil, kaboom
}

func (k KVStoreDummy) ListNodeAccountsByStatus(_ cosmos.Context, _ NodeStatus) (NodeAccounts, error) {
	return nil, kaboom
}

func (k KVStoreDummy) ListActiveNodeAccounts(_ cosmos.Context) (NodeAccounts, error) {
	return nil, kaboom
}

func (k KVStoreDummy) GetLowestActiveVersion(_ cosmos.Context) semver.Version {
	return semver.Version{
		Major: 0,
		Minor: 1,
		Patch: 0,
	}
}
func (k KVStoreDummy) GetMinJoinVersion(_ cosmos.Context) semver.Version   { return semver.Version{} }
func (k KVStoreDummy) GetMinJoinVersionV1(_ cosmos.Context) semver.Version { return semver.Version{} }
func (k KVStoreDummy) GetNodeAccount(_ cosmos.Context, _ cosmos.AccAddress) (NodeAccount, error) {
	return NodeAccount{}, kaboom
}

func (k KVStoreDummy) GetNodeAccountByPubKey(_ cosmos.Context, _ common.PubKey) (NodeAccount, error) {
	return NodeAccount{}, kaboom
}

func (k KVStoreDummy) SetNodeAccount(_ cosmos.Context, _ NodeAccount) error { return kaboom }
func (k KVStoreDummy) EnsureNodeKeysUnique(_ cosmos.Context, _ string, _ common.PubKeySet) error {
	return kaboom
}
func (k KVStoreDummy) GetNodeAccountIterator(_ cosmos.Context) cosmos.Iterator { return nil }

func (k KVStoreDummy) GetNodeAccountSlashPoints(_ cosmos.Context, _ cosmos.AccAddress) (int64, error) {
	return 0, kaboom
}
func (k KVStoreDummy) SetNodeAccountSlashPoints(_ cosmos.Context, _ cosmos.AccAddress, _ int64) {}
func (k KVStoreDummy) ResetNodeAccountSlashPoints(_ cosmos.Context, _ cosmos.AccAddress)        {}
func (k KVStoreDummy) IncNodeAccountSlashPoints(_ cosmos.Context, _ cosmos.AccAddress, _ int64) error {
	return kaboom
}

func (k KVStoreDummy) DecNodeAccountSlashPoints(_ cosmos.Context, _ cosmos.AccAddress, _ int64) error {
	return kaboom
}

func (k KVStoreDummy) GetNodeAccountJail(ctx cosmos.Context, addr cosmos.AccAddress) (Jail, error) {
	return Jail{}, kaboom
}

func (k KVStoreDummy) SetNodeAccountJail(ctx cosmos.Context, addr cosmos.AccAddress, height int64, reason string) error {
	return kaboom
}

func (k KVStoreDummy) GetObservingAddresses(_ cosmos.Context) ([]cosmos.AccAddress, error) {
	return nil, kaboom
}

func (k KVStoreDummy) AddObservingAddresses(_ cosmos.Context, _ []cosmos.AccAddress) error {
	return kaboom
}
func (k KVStoreDummy) ClearObservingAddresses(_ cosmos.Context)                      {}
func (k KVStoreDummy) SetObservedTxInVoter(_ cosmos.Context, _ ObservedTxVoter)      {}
func (k KVStoreDummy) GetObservedTxInVoterIterator(_ cosmos.Context) cosmos.Iterator { return nil }
func (k KVStoreDummy) GetObservedTxInVoter(_ cosmos.Context, _ common.TxID) (ObservedTxVoter, error) {
	return ObservedTxVoter{}, kaboom
}
func (k KVStoreDummy) SetObservedTxOutVoter(_ cosmos.Context, _ ObservedTxVoter)      {}
func (k KVStoreDummy) GetObservedTxOutVoterIterator(_ cosmos.Context) cosmos.Iterator { return nil }
func (k KVStoreDummy) GetObservedTxOutVoter(_ cosmos.Context, _ common.TxID) (ObservedTxVoter, error) {
	return ObservedTxVoter{}, kaboom
}
func (k KVStoreDummy) SetTssVoter(_ cosmos.Context, _ TssVoter)             {}
func (k KVStoreDummy) GetTssVoterIterator(_ cosmos.Context) cosmos.Iterator { return nil }
func (k KVStoreDummy) GetTssVoter(_ cosmos.Context, _ string) (TssVoter, error) {
	return TssVoter{}, kaboom
}

func (k KVStoreDummy) GetKeygenBlock(_ cosmos.Context, _ int64) (KeygenBlock, error) {
	return KeygenBlock{}, kaboom
}
func (k KVStoreDummy) SetKeygenBlock(_ cosmos.Context, _ KeygenBlock)            { return }
func (k KVStoreDummy) GetKeygenBlockIterator(_ cosmos.Context) cosmos.Iterator   { return nil }
func (k KVStoreDummy) GetTxOut(_ cosmos.Context, _ int64) (*TxOut, error)        { return nil, kaboom }
func (k KVStoreDummy) SetTxOut(_ cosmos.Context, _ *TxOut) error                 { return kaboom }
func (k KVStoreDummy) AppendTxOut(_ cosmos.Context, _ int64, _ *TxOutItem) error { return kaboom }
func (k KVStoreDummy) ClearTxOut(_ cosmos.Context, _ int64) error                { return kaboom }
func (k KVStoreDummy) GetTxOutIterator(_ cosmos.Context) cosmos.Iterator         { return nil }
func (k KVStoreDummy) AddToLiquidityFees(_ cosmos.Context, _ common.Asset, _ cosmos.Uint) error {
	return kaboom
}

func (k KVStoreDummy) GetTotalLiquidityFees(_ cosmos.Context, _ uint64) (cosmos.Uint, error) {
	return cosmos.ZeroUint(), kaboom
}

func (k KVStoreDummy) GetPoolLiquidityFees(_ cosmos.Context, _ uint64, _ common.Asset) (cosmos.Uint, error) {
	return cosmos.ZeroUint(), kaboom
}

func (k KVStoreDummy) GetChains(_ cosmos.Context) (common.Chains, error)  { return nil, kaboom }
func (k KVStoreDummy) SetChains(_ cosmos.Context, _ common.Chains)        {}
func (k KVStoreDummy) GetVaultIterator(_ cosmos.Context) cosmos.Iterator  { return nil }
func (k KVStoreDummy) VaultExists(_ cosmos.Context, _ common.PubKey) bool { return false }
func (k KVStoreDummy) FindPubKeyOfAddress(_ cosmos.Context, _ common.Address, _ common.Chain) (common.PubKey, error) {
	return common.EmptyPubKey, kaboom
}
func (k KVStoreDummy) SetVault(_ cosmos.Context, _ Vault) error { return kaboom }
func (k KVStoreDummy) GetVault(_ cosmos.Context, _ common.PubKey) (Vault, error) {
	return Vault{}, kaboom
}
func (k KVStoreDummy) GetAsgardVaults(_ cosmos.Context) (Vaults, error) { return nil, kaboom }
func (k KVStoreDummy) GetAsgardVaultsByStatus(_ cosmos.Context, _ VaultStatus) (Vaults, error) {
	return nil, kaboom
}
func (k KVStoreDummy) DeleteVault(_ cosmos.Context, _ common.PubKey) error { return kaboom }

func (k KVStoreDummy) GetReservesContributors(_ cosmos.Context) (ReserveContributors, error) {
	return nil, kaboom
}

func (k KVStoreDummy) SetReserveContributors(_ cosmos.Context, _ ReserveContributors) error {
	return kaboom
}

func (k KVStoreDummy) HasValidVaultPools(_ cosmos.Context) (bool, error)     { return false, kaboom }
func (k KVStoreDummy) AddFeeToReserve(_ cosmos.Context, _ cosmos.Uint) error { return kaboom }
func (k KVStoreDummy) GetVaultData(_ cosmos.Context) (VaultData, error)      { return VaultData{}, kaboom }
func (k KVStoreDummy) SetVaultData(_ cosmos.Context, _ VaultData) error      { return kaboom }

func (k KVStoreDummy) SetTssKeysignFailVoter(_ cosmos.Context, tss TssKeysignFailVoter) {
}

func (k KVStoreDummy) GetTssKeysignFailVoterIterator(_ cosmos.Context) cosmos.Iterator {
	return nil
}

func (k KVStoreDummy) GetTssKeysignFailVoter(_ cosmos.Context, _ string) (TssKeysignFailVoter, error) {
	return TssKeysignFailVoter{}, kaboom
}

func (k KVStoreDummy) GetGas(_ cosmos.Context, _ common.Asset) ([]cosmos.Uint, error) {
	return nil, kaboom
}
func (k KVStoreDummy) SetGas(_ cosmos.Context, _ common.Asset, _ []cosmos.Uint) {}
func (k KVStoreDummy) GetGasIterator(ctx cosmos.Context) cosmos.Iterator        { return nil }

func (k KVStoreDummy) ListTxMarker(_ cosmos.Context, _ string) (TxMarkers, error) {
	return nil, kaboom
}
func (k KVStoreDummy) SetTxMarkers(_ cosmos.Context, _ string, _ TxMarkers) error  { return kaboom }
func (k KVStoreDummy) AppendTxMarker(_ cosmos.Context, _ string, _ TxMarker) error { return kaboom }
func (k KVStoreDummy) GetAllTxMarkers(ctx cosmos.Context) (map[string]TxMarkers, error) {
	return nil, kaboom
}

func (k KVStoreDummy) SetErrataTxVoter(_ cosmos.Context, _ ErrataTxVoter)        {}
func (k KVStoreDummy) GetErrataTxVoterIterator(_ cosmos.Context) cosmos.Iterator { return nil }
func (k KVStoreDummy) GetErrataTxVoter(_ cosmos.Context, _ common.TxID, _ common.Chain) (ErrataTxVoter, error) {
	return ErrataTxVoter{}, kaboom
}
func (k KVStoreDummy) SetBanVoter(_ cosmos.Context, _ BanVoter) {}
func (k KVStoreDummy) GetBanVoter(_ cosmos.Context, _ cosmos.AccAddress) (BanVoter, error) {
	return BanVoter{}, kaboom
}

func (k KVStoreDummy) GetBanVoterIterator(ctx cosmos.Context) cosmos.Iterator {
	return nil
}
func (k KVStoreDummy) SetSwapQueueItem(ctx cosmos.Context, msg MsgSwap) error  { return kaboom }
func (k KVStoreDummy) GetSwapQueueIterator(ctx cosmos.Context) cosmos.Iterator { return nil }
func (k KVStoreDummy) RemoveSwapQueueItem(ctx cosmos.Context, _ common.TxID)   {}
func (k KVStoreDummy) GetSwapQueueItem(ctx cosmos.Context, txID common.TxID) (MsgSwap, error) {
	return MsgSwap{}, kaboom
}
func (k KVStoreDummy) GetMimir(_ cosmos.Context, key string) (int64, error) { return 0, kaboom }
func (k KVStoreDummy) SetMimir(_ cosmos.Context, key string, value int64)   {}
func (k KVStoreDummy) GetMimirIterator(ctx cosmos.Context) cosmos.Iterator  { return nil }
func (k KVStoreDummy) GetNetworkFee(ctx cosmos.Context, chain common.Chain) (NetworkFee, error) {
	return NetworkFee{}, kaboom
}

func (k KVStoreDummy) SaveNetworkFee(ctx cosmos.Context, chain common.Chain, networkFee NetworkFee) error {
	return kaboom
}

func (k KVStoreDummy) GetNetworkFeeIterator(ctx cosmos.Context) cosmos.Iterator {
	return nil
}

func (k KVStoreDummy) SetObservedNetworkFeeVoter(ctx cosmos.Context, networkFeeVoter ObservedNetworkFeeVoter) {
}

func (k KVStoreDummy) GetObservedNetworkFeeVoterIterator(ctx cosmos.Context) cosmos.Iterator {
	return nil
}

func (k KVStoreDummy) GetObservedNetworkFeeVoter(ctx cosmos.Context, height int64, chain common.Chain) (ObservedNetworkFeeVoter, error) {
	return ObservedNetworkFeeVoter{}, nil
}

// a mock cosmos.Iterator implementation for testing purposes
type DummyIterator struct {
	cosmos.Iterator
	placeholder int
	keys        [][]byte
	values      [][]byte
	err         error
}

func NewDummyIterator() *DummyIterator {
	return &DummyIterator{
		keys:   make([][]byte, 0),
		values: make([][]byte, 0),
	}
}

func (iter *DummyIterator) AddItem(key, value []byte) {
	iter.keys = append(iter.keys, key)
	iter.values = append(iter.values, value)
}

func (iter *DummyIterator) Next() {
	iter.placeholder++
}

func (iter *DummyIterator) Valid() bool {
	return iter.placeholder < len(iter.keys)
}

func (iter *DummyIterator) Key() []byte {
	return iter.keys[iter.placeholder]
}

func (iter *DummyIterator) Value() []byte {
	return iter.values[iter.placeholder]
}

func (iter *DummyIterator) Close() {
	iter.placeholder = 0
}

func (iter *DummyIterator) Error() error {
	return iter.err
}

func (iter *DummyIterator) Domain() (start, end []byte) {
	return nil, nil
}
