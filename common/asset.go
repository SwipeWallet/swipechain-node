package common

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

var (
	// EmptyAsset empty asset, not valid
	EmptyAsset = Asset{Chain: EmptyChain, Symbol: "", Ticker: ""}
	// BNBAsset BNB
	BNBAsset = Asset{Chain: BNBChain, Symbol: "BNB", Ticker: "BNB"}
	// BTCAsset BTC
	BTCAsset = Asset{Chain: BTCChain, Symbol: "BTC", Ticker: "BTC"}
	// ETHAsset ETH
	ETHAsset = Asset{Chain: ETHChain, Symbol: "ETH", Ticker: "ETH"}
	// Rune67CAsset RUNE on Binance test net
	Rune67CAsset = Asset{Chain: BNBChain, Symbol: "RUNE-67C", Ticker: "RUNE"} // testnet asset on binance ganges
	// RuneB1AAsset RUNE on Binance main net
	RuneB1AAsset = Asset{Chain: BNBChain, Symbol: "RUNE-B1A", Ticker: "RUNE"} // mainnet
	// RuneNative RUNE on thorchain
	RuneNative = Asset{Chain: THORChain, Symbol: "RUNE", Ticker: "RUNE"}
)

// Asset represent an asset in THORChain it is in BNB.BNB format
// for example BNB.RUNE-67C , BNB.RUNE-B1A
type Asset struct {
	Chain  Chain  `json:"chain"`
	Symbol Symbol `json:"symbol"`
	Ticker Ticker `json:"ticker"`
}

// NewAsset parse the given input into Asset object
func NewAsset(input string) (Asset, error) {
	var err error
	var asset Asset
	var sym string
	parts := strings.SplitN(input, ".", 2)
	if len(parts) == 1 {
		asset.Chain = THORChain
		sym = parts[0]
	} else {
		asset.Chain, err = NewChain(parts[0])
		if err != nil {
			return EmptyAsset, err
		}
		sym = parts[1]
	}

	asset.Symbol, err = NewSymbol(sym)
	if err != nil {
		return EmptyAsset, err
	}

	parts = strings.SplitN(sym, "-", 2)
	asset.Ticker, err = NewTicker(parts[0])
	if err != nil {
		return EmptyAsset, err
	}

	return asset, nil
}

// Equals determinate whether two assets are equivalent
func (a Asset) Equals(a2 Asset) bool {
	return a.Chain.Equals(a2.Chain) && a.Symbol.Equals(a2.Symbol) && a.Ticker.Equals(a2.Ticker)
}

// Native return native asset, only relevant on THORChain
func (a Asset) Native() string {
	if !a.Chain.Equals(THORChain) {
		return ""
	}
	return strings.ToLower(a.Symbol.String())
}

// IsEmpty will be true when any of the field is empty, chain,symbol or ticker
func (a Asset) IsEmpty() bool {
	return a.Chain.IsEmpty() || a.Symbol.IsEmpty() || a.Ticker.IsEmpty()
}

// String implement fmt.Stringer , return the string representation of Asset
func (a Asset) String() string {
	return fmt.Sprintf("%s.%s", a.Chain.String(), a.Symbol.String())
}

// IsRune is a helper function ,return true only when the asset represent RUNE
func (a Asset) IsRune() bool {
	return a.Equals(Rune67CAsset) || a.Equals(RuneB1AAsset) || a.Equals(RuneNative)
}

// IsBNB is a helper function, return true only when the asset represent BNB
func (a Asset) IsBNB() bool {
	return a.Equals(BNBAsset)
}

// MarshalJSON implement Marshaler interface
func (a Asset) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}

// UnmarshalJSON implement Unmarshaler interface
func (a *Asset) UnmarshalJSON(data []byte) error {
	var err error
	var assetStr string
	if err := json.Unmarshal(data, &assetStr); err != nil {
		return err
	}
	*a, err = NewAsset(assetStr)
	return err
}

// RuneAsset return RUNE Asset depends on different environment
func RuneAsset() Asset {
	if strings.EqualFold(os.Getenv("NATIVE"), "true") {
		return RuneNative
	}
	if strings.EqualFold(os.Getenv("NET"), "testnet") || strings.EqualFold(os.Getenv("NET"), "mocknet") {
		return Rune67CAsset
	}
	return RuneB1AAsset
}

// BEP2RuneAsset is RUNE on BEP2
func BEP2RuneAsset() Asset {
	if strings.EqualFold(os.Getenv("NET"), "testnet") || strings.EqualFold(os.Getenv("NET"), "mocknet") {
		return Rune67CAsset
	}
	return RuneB1AAsset
}
