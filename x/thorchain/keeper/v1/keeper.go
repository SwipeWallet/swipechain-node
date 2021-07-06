package keeperv1

import (
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/supply"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/x/thorchain/keeper/types"
	kvTypes "gitlab.com/thorchain/thornode/x/thorchain/keeper/types"
)

// NOTE: Always end a dbPrefix with a slash ("/"). This is to ensure that there
// are no prefixes that contain another prefix. In the scenario where this is
// true, an iterator for a specific type, will get more than intended, and may
// include a different type. The slash is used to protect us from this
// scenario.
// Also, use underscores between words and use lowercase characters only

const (
	version                  int64            = 1
	prefixStoreVersion       kvTypes.DbPrefix = "_ver/"
	prefixObservedTxIn       kvTypes.DbPrefix = "observed_tx_in/"
	prefixObservedTxOut      kvTypes.DbPrefix = "observed_tx_out/"
	prefixPool               kvTypes.DbPrefix = "pool/"
	prefixTxOut              kvTypes.DbPrefix = "txout/"
	prefixTotalLiquidityFee  kvTypes.DbPrefix = "total_liquidity_fee/"
	prefixPoolLiquidityFee   kvTypes.DbPrefix = "pool_liquidity_fee/"
	prefixStaker             kvTypes.DbPrefix = "staker/"
	prefixLastChainHeight    kvTypes.DbPrefix = "last_chain_height/"
	prefixLastSignedHeight   kvTypes.DbPrefix = "last_signed_height/"
	prefixNodeAccount        kvTypes.DbPrefix = "node_account/"
	prefixVault              kvTypes.DbPrefix = "vault/"
	prefixVaultAsgardIndex   kvTypes.DbPrefix = "vault_asgard_index/"
	prefixVaultData          kvTypes.DbPrefix = "vault_data/"
	prefixObservingAddresses kvTypes.DbPrefix = "observing_addresses/"
	prefixReserves           kvTypes.DbPrefix = "reserves/"
	prefixTss                kvTypes.DbPrefix = "tss/"
	prefixTssKeysignFailure  kvTypes.DbPrefix = "tssKeysignFailure/"
	prefixKeygen             kvTypes.DbPrefix = "keygen/"
	prefixRagnarokHeight     kvTypes.DbPrefix = "ragnarokHeight/"
	prefixRagnarokNth        kvTypes.DbPrefix = "ragnarokNth/"
	prefixRagnarokPending    kvTypes.DbPrefix = "ragnarokPending/"
	prefixRagnarokPosition   kvTypes.DbPrefix = "ragnarokPosition/"
	prefixGas                kvTypes.DbPrefix = "gas/"
	prefixSupportedTxMarker  kvTypes.DbPrefix = "marker/"
	prefixErrataTx           kvTypes.DbPrefix = "errata/"
	prefixBanVoter           kvTypes.DbPrefix = "ban/"
	prefixNodeSlashPoints    kvTypes.DbPrefix = "slash/"
	prefixNodeJail           kvTypes.DbPrefix = "jail/"
	prefixSwapQueueItem      kvTypes.DbPrefix = "swapitem/"
	prefixMimir              kvTypes.DbPrefix = "mimir/"
	prefixNetworkFee         kvTypes.DbPrefix = "network_fee/"
	prefixNetworkFeeVoter    kvTypes.DbPrefix = "network_fee_voter/"
)

func dbError(ctx cosmos.Context, wrapper string, err error) error {
	err = fmt.Errorf("KVStore Error: %s: %w", wrapper, err)
	ctx.Logger().Error("keeper error", "error", err)
	return err
}

// KVStore Keeper maintains the link to data storage and exposes getter/setter methods for the various parts of the state machine
type KVStore struct {
	coinKeeper   bank.Keeper
	supplyKeeper supply.Keeper
	storeKey     cosmos.StoreKey // Unexposed key to access store from cosmos.Context
	cdc          *codec.Codec    // The wire codec for binary encoding/decoding.
	version      int64
}

// NewKVStore creates new instances of the thorchain Keeper
func NewKVStore(coinKeeper bank.Keeper, supplyKeeper supply.Keeper, storeKey cosmos.StoreKey, cdc *codec.Codec) KVStore {
	return KVStore{
		coinKeeper:   coinKeeper,
		supplyKeeper: supplyKeeper,
		storeKey:     storeKey,
		cdc:          cdc,
		version:      version,
	}
}

// Cdc return the amino codec
func (k KVStore) Cdc() *codec.Codec {
	return k.cdc
}

// Supply return the keeper from supply handler
func (k KVStore) Supply() supply.Keeper {
	return k.supplyKeeper
}

// CoinKeeper return the keeper from bank handler
func (k KVStore) CoinKeeper() bank.Keeper {
	return k.coinKeeper
}

// Version return the current version
func (k KVStore) Version() int64 {
	return k.version
}

// GetKey return a key that can be used to store into key value store
func (k KVStore) GetKey(ctx cosmos.Context, prefix kvTypes.DbPrefix, key string) string {
	return fmt.Sprintf("%s/%s", prefix, strings.ToUpper(key))
}

// GetStoreVersion get the current key value store version
func (k KVStore) GetStoreVersion(ctx cosmos.Context) int64 {
	key := k.GetKey(ctx, prefixStoreVersion, "")
	store := ctx.KVStore(k.storeKey)
	if !store.Has([]byte(key)) {
		// thornode start at version 0.6.0, thus when there is no store version , it return 6
		return 6
	}
	var value int64
	buf := store.Get([]byte(key))
	k.cdc.MustUnmarshalBinaryBare(buf, &value)
	return value
}

// getIterator - get an iterator for given prefix
func (k KVStore) getIterator(ctx cosmos.Context, prefix types.DbPrefix) cosmos.Iterator {
	store := ctx.KVStore(k.storeKey)
	return cosmos.KVStorePrefixIterator(store, []byte(prefix))
}

// set - save data from the kvstore
func (k KVStore) set(ctx cosmos.Context, key string, record interface{}) {
	store := ctx.KVStore(k.storeKey)
	buf := k.cdc.MustMarshalBinaryBare(record)
	if buf == nil {
		store.Delete([]byte(key))
	} else {
		store.Set([]byte(key), buf)
	}
}

// del - delete data from the kvstore
func (k KVStore) del(ctx cosmos.Context, key string) {
	store := ctx.KVStore(k.storeKey)
	if store.Has([]byte(key)) {
		store.Delete([]byte(key))
	}
}

// get - fetches data from the kvstore
func (k KVStore) get(ctx cosmos.Context, key string, record interface{}) (bool, error) {
	store := ctx.KVStore(k.storeKey)
	if !store.Has([]byte(key)) {
		return false, nil
	}

	bz := store.Get([]byte(key))
	if err := k.cdc.UnmarshalBinaryBare(bz, record); err != nil {
		return true, dbError(ctx, fmt.Sprintf("Unmarshal kvstore: %s", key), err)
	}
	return true, nil
}

// has - kvstore has key
func (k KVStore) has(ctx cosmos.Context, key string) bool {
	store := ctx.KVStore(k.storeKey)
	return store.Has([]byte(key))
}

// SetStoreVersion save the store version
func (k KVStore) SetStoreVersion(ctx cosmos.Context, value int64) {
	key := k.GetKey(ctx, prefixStoreVersion, "")
	store := ctx.KVStore(k.storeKey)
	store.Set([]byte(key), k.cdc.MustMarshalBinaryBare(value))
}

// GetRuneBalanceOfModule get the RUNE balance
func (k KVStore) GetRuneBalanceOfModule(ctx cosmos.Context, moduleName string) cosmos.Uint {
	addr := k.supplyKeeper.GetModuleAddress(moduleName)
	coins := k.coinKeeper.GetCoins(ctx, addr)
	amt := coins.AmountOf(common.RuneNative.Native())
	return cosmos.NewUintFromBigInt(amt.BigInt())
}

// SendFromModuleToModule transfer asset from one module to another
func (k KVStore) SendFromModuleToModule(ctx cosmos.Context, from, to string, coin common.Coin) error {
	coins := cosmos.NewCoins(
		cosmos.NewCoin(coin.Asset.Native(), cosmos.NewIntFromBigInt(coin.Amount.BigInt())),
	)
	return k.Supply().SendCoinsFromModuleToModule(ctx, from, to, coins)
}

// SendFromAccountToModule transfer fund from one account to a module
func (k KVStore) SendFromAccountToModule(ctx cosmos.Context, from cosmos.AccAddress, to string, coin common.Coin) error {
	coins := cosmos.NewCoins(
		cosmos.NewCoin(coin.Asset.Native(), cosmos.NewIntFromBigInt(coin.Amount.BigInt())),
	)
	return k.Supply().SendCoinsFromAccountToModule(ctx, from, to, coins)
}

// SendFromModuleToAccount transfer fund from module to an account
func (k KVStore) SendFromModuleToAccount(ctx cosmos.Context, from string, to cosmos.AccAddress, coin common.Coin) error {
	coins := cosmos.NewCoins(
		cosmos.NewCoin(coin.Asset.Native(), cosmos.NewIntFromBigInt(coin.Amount.BigInt())),
	)
	return k.Supply().SendCoinsFromModuleToAccount(ctx, from, to, coins)
}
