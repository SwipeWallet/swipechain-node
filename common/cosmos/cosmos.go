package cosmos

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	se "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/hashicorp/go-multierror"
)

var (
	KeyringServiceName      = sdk.KeyringServiceName
	NewUint                 = sdk.NewUint
	ParseUint               = sdk.ParseUint
	NewInt                  = sdk.NewInt
	NewDec                  = sdk.NewDec
	ZeroUint                = sdk.ZeroUint
	ZeroDec                 = sdk.ZeroDec
	OneUint                 = sdk.OneUint
	NewCoin                 = sdk.NewCoin
	NewCoins                = sdk.NewCoins
	ParseCoins              = sdk.ParseCoins
	NewDecWithPrec          = sdk.NewDecWithPrec
	NewDecFromBigInt        = sdk.NewDecFromBigInt
	NewIntFromBigInt        = sdk.NewIntFromBigInt
	NewUintFromBigInt       = sdk.NewUintFromBigInt
	AccAddressFromBech32    = sdk.AccAddressFromBech32
	GetFromBech32           = sdk.GetFromBech32
	NewAttribute            = sdk.NewAttribute
	NewDecFromStr           = sdk.NewDecFromStr
	GetConfig               = sdk.GetConfig
	NewEvent                = sdk.NewEvent
	RegisterCodec           = sdk.RegisterCodec
	NewEventManager         = sdk.NewEventManager
	EventTypeMessage        = sdk.EventTypeMessage
	AttributeKeyModule      = sdk.AttributeKeyModule
	KVStorePrefixIterator   = sdk.KVStorePrefixIterator
	NewKVStoreKey           = sdk.NewKVStoreKey
	NewTransientStoreKey    = sdk.NewTransientStoreKey
	StoreTypeTransient      = sdk.StoreTypeTransient
	StoreTypeIAVL           = sdk.StoreTypeIAVL
	NewContext              = sdk.NewContext
	GetPubKeyFromBech32     = sdk.GetPubKeyFromBech32
	Bech32ifyPubKey         = sdk.Bech32ifyPubKey
	Bech32PubKeyTypeConsPub = sdk.Bech32PubKeyTypeConsPub
	Bech32PubKeyTypeAccPub  = sdk.Bech32PubKeyTypeAccPub
	Wrapf                   = se.Wrapf
	MustSortJSON            = sdk.MustSortJSON
	CodeUnauthorized        = uint32(4)
	CodeInsufficientFunds   = uint32(5)
)

type (
	Context    = sdk.Context
	Uint       = sdk.Uint
	Coin       = sdk.Coin
	Coins      = sdk.Coins
	AccAddress = sdk.AccAddress
	Attribute  = sdk.Attribute
	Result     = sdk.Result
	Event      = sdk.Event
	Events     = sdk.Events
	Dec        = sdk.Dec
	Msg        = sdk.Msg
	Iterator   = sdk.Iterator
	Handler    = sdk.Handler
	StoreKey   = sdk.StoreKey
	Querier    = sdk.Querier
	TxResponse = sdk.TxResponse
)

func ErrUnknownRequest(msg string) error {
	return se.Wrap(se.ErrUnknownRequest, msg)
}

func ErrInvalidAddress(addr string) error {
	return se.Wrap(se.ErrInvalidAddress, addr)
}

func ErrInvalidCoins(msg string) error {
	return se.Wrap(se.ErrInvalidCoins, msg)
}

func ErrUnauthorized(msg string) error {
	return se.Wrap(se.ErrUnauthorized, msg)
}

func ErrInsufficientCoins(err error, msg string) error {
	return se.Wrap(multierror.Append(se.ErrInsufficientFunds, err), msg)
}
