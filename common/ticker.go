package common

import (
	"errors"
	"strings"
)

const (
	// BNBTicker BNB
	BNBTicker = Ticker("BNB")
	// RuneTicker RUNE
	RuneTicker = Ticker("RUNE")
)

type (
	// Ticker The trading 'symbol' or shortened name (typically in capital
	// letters) that refer to a coin on a trading platform. For example: BNB
	Ticker string
	// Tickers a list of ticker
	Tickers []Ticker
)

// NewTicker parse the given string as ticker, return error if it is not
// legitimate ticker
func NewTicker(ticker string) (Ticker, error) {
	noTicker := Ticker("")
	if len(ticker) < 3 {
		return noTicker, errors.New("ticker error: not enough characters")
	}

	if len(ticker) > 13 {
		return noTicker, errors.New("ticker error: too many characters")
	}
	return Ticker(strings.ToUpper(ticker)), nil
}

// Equals compare whether two ticker is the same
func (t Ticker) Equals(t2 Ticker) bool {
	return strings.EqualFold(t.String(), t2.String())
}

// IsEmpty return true when the ticker is an empty string
func (t Ticker) IsEmpty() bool {
	return strings.TrimSpace(t.String()) == ""
}

// String implement fmt.Stringer
func (t Ticker) String() string {
	// upper casing again just in case someone created a ticker via
	// Ticker("rune")
	return strings.ToUpper(string(t))
}
