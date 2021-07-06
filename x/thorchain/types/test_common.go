// Please put all the test related function to here
package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/x/auth"
	atypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/supply"
	"github.com/tendermint/tendermint/crypto"
	tmtypes "github.com/tendermint/tendermint/types"

	"gitlab.com/thorchain/thornode/cmd"
	"gitlab.com/thorchain/thornode/common"
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
)

// GetRandomNodeAccount create a random generated node account , used for test purpose
func GetRandomNodeAccount(status NodeStatus) NodeAccount {
	v, _ := tmtypes.RandValidator(true, 100)
	k, _ := cosmos.Bech32ifyPubKey(cosmos.Bech32PubKeyTypeConsPub, v.PubKey)
	pubKeys := common.PubKeySet{
		Secp256k1: GetRandomPubKey(),
		Ed25519:   GetRandomPubKey(),
	}
	addr, _ := pubKeys.Secp256k1.GetThorAddress()
	bondAddr := GetRandomBNBAddress()
	if common.RuneAsset().Chain.Equals(common.THORChain) {
		bondAddr = common.Address(addr.String())
	}
	na := NewNodeAccount(addr, status, pubKeys, k, cosmos.NewUint(100*common.One), bondAddr, 1)
	na.Version = constants.SWVersion
	if na.Status == Active {
		na.ActiveBlockHeight = 10
		na.Bond = cosmos.NewUint(1000 * common.One)
	}
	na.IPAddress = "192.168.0.1"

	return na
}

func GetRandomObservedTx() ObservedTx {
	return NewObservedTx(GetRandomTx(), 33, GetRandomPubKey())
}

// GetRandomTx
func GetRandomTx() common.Tx {
	return common.NewTx(
		GetRandomTxHash(),
		GetRandomBNBAddress(),
		GetRandomBNBAddress(),
		common.Coins{common.NewCoin(common.BNBAsset, cosmos.OneUint())},
		common.Gas{
			{Asset: common.BNBAsset, Amount: cosmos.NewUint(37500)},
		},
		"",
	)
}

// GetRandomBech32Addr is an account address used for test
func GetRandomBech32Addr() cosmos.AccAddress {
	name := common.RandStringBytesMask(10)
	return cosmos.AccAddress(crypto.AddressHash([]byte(name)))
}

func GetRandomBech32ConsensusPubKey() string {
	_, pubKey, _ := atypes.KeyTestPubAddr()
	result, err := cosmos.Bech32ifyPubKey(cosmos.Bech32PubKeyTypeConsPub, pubKey)
	if err != nil {
		panic(err)
	}
	return result
}

// GetRandomRuneAddress will just create a random rune address used for test purpose
func GetRandomRUNEAddress() common.Address {
	if common.RuneAsset().Chain.Equals(common.THORChain) {
		return GetRandomTHORAddress()
	}
	return GetRandomBNBAddress()
}

// GetRandomTHORAddress will just create a random thor address used for test purpose
func GetRandomTHORAddress() common.Address {
	name := common.RandStringBytesMask(10)
	str, _ := common.ConvertAndEncode(cmd.Bech32PrefixAccAddr, crypto.AddressHash([]byte(name)))
	thor, _ := common.NewAddress(str)
	return thor
}

// GetRandomBNBAddress will just create a random bnb address used for test purpose
func GetRandomBNBAddress() common.Address {
	name := common.RandStringBytesMask(10)
	str, _ := common.ConvertAndEncode("tbnb", crypto.AddressHash([]byte(name)))
	bnb, _ := common.NewAddress(str)
	return bnb
}

func GetRandomBTCAddress() common.Address {
	pubKey := GetRandomPubKey()
	addr, _ := pubKey.GetAddress(common.BTCChain)
	return addr
}

// GetRandomTxHash create a random txHash used for test purpose
func GetRandomTxHash() common.TxID {
	txHash, _ := common.NewTxID(common.RandStringBytesMask(64))
	return txHash
}

// GetRandomPubKeySet return a random common.PubKeySet for test purpose
func GetRandomPubKeySet() common.PubKeySet {
	return common.NewPubKeySet(GetRandomPubKey(), GetRandomPubKey())
}

func GetRandomVault() Vault {
	return NewVault(32, ActiveVault, AsgardVault, GetRandomPubKey(), common.Chains{common.BNBChain})
}

func GetRandomPubKey() common.PubKey {
	_, pubKey, _ := atypes.KeyTestPubAddr()
	bech32PubKey, _ := cosmos.Bech32ifyPubKey(cosmos.Bech32PubKeyTypeAccPub, pubKey)
	pk, _ := common.NewPubKey(bech32PubKey)
	return pk
}

// SetupConfigForTest used for test purpose
func SetupConfigForTest() {
	config := cosmos.GetConfig()
	config.SetBech32PrefixForAccount(cmd.Bech32PrefixAccAddr, cmd.Bech32PrefixAccPub)
	config.SetBech32PrefixForValidator(cmd.Bech32PrefixValAddr, cmd.Bech32PrefixValPub)
	config.SetBech32PrefixForConsensusNode(cmd.Bech32PrefixConsAddr, cmd.Bech32PrefixConsPub)
}

// nolint: deadcode unused
// create a codec used only for testing
func MakeTestCodec() *codec.Codec {
	cdc := codec.New()
	bank.RegisterCodec(cdc)
	auth.RegisterCodec(cdc)
	RegisterCodec(cdc)
	supply.RegisterCodec(cdc)
	cosmos.RegisterCodec(cdc)
	codec.RegisterCrypto(cdc)
	return cdc
}
