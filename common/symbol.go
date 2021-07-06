package common

import (
	"errors"
	"regexp"
	"strings"
)

const (
	// BNBSymbol BNB
	BNBSymbol = Symbol("BNB")
	// Rune67CSymbol RUNE on binance testnet
	Rune67CSymbol = Symbol("RUNE-67C")
	// RuneB1ASymbol RUNE on binance mainnet
	RuneB1ASymbol = Symbol("RUNE-B1A")
)

var isAlphaNumeric = regexp.MustCompile(`^[A-Za-z0-9-._]+$`).MatchString

// Symbol represent an asset
type Symbol string

// NewSymbol parse the input as symbol
func NewSymbol(input string) (Symbol, error) {
	if !isAlphaNumeric(input) {
		return "", errors.New("invalid symbol")
	}

	return Symbol(input), nil
}

// Ticker return the ticker part of symbol
func (s Symbol) Ticker() Ticker {
	parts := strings.Split(s.String(), "-")
	ticker, _ := NewTicker(parts[0])
	return ticker
}

// Equals check whether two symbol are the same
func (s Symbol) Equals(s2 Symbol) bool {
	return strings.EqualFold(s.String(), s2.String())
}

// IsEmpty return true when symbol is just empty string
func (s Symbol) IsEmpty() bool {
	return strings.TrimSpace(s.String()) == ""
}

// String implement fmt.Stringer
func (s Symbol) String() string {
	// uppercasing again just in case someone created a ticker via Chain("rune")
	return strings.ToUpper(string(s))
}

// IsMiniToken is to determine whether it is a mini token on binance chain
func (s Symbol) IsMiniToken() bool {
	parts := strings.Split(s.String(), "-")
	if len(parts) == 1 {
		return false
	}
	return len(parts[1]) == 4 && strings.HasSuffix(s.String(), "M")
}
