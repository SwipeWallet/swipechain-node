package common

import (
	"fmt"

	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
)

// One is useful type so THORNode doesn't need to manage 8 zeroes all the time
const One = 100000000

// GetShare this method will panic if any of the input parameter can't be convert to cosmos.Dec
// which shouldn't happen
func GetShare(part, total, allocation cosmos.Uint) cosmos.Uint {
	if part.IsZero() || total.IsZero() {
		return cosmos.ZeroUint()
	}

	// use string to convert cosmos.Uint to cosmos.Dec is the only way I can find out without being constrain to uint64
	// cosmos.Uint can hold values way larger than uint64 , because it is using big.Int internally
	aD, err := cosmos.NewDecFromStr(allocation.String())
	if err != nil {
		panic(fmt.Errorf("fail to convert %s to cosmos.Dec: %w", allocation.String(), err))
	}

	pD, err := cosmos.NewDecFromStr(part.String())
	if err != nil {
		panic(fmt.Errorf("fatil to convert %s to cosmos.Dec: %w", part.String(), err))
	}
	tD, err := cosmos.NewDecFromStr(total.String())
	if err != nil {
		panic(fmt.Errorf("fail to convert%s to cosmos.Dec: %w", total.String(), err))
	}
	// A / (Total / part) == A * (part/Total) but safer when part < Totals
	result := aD.Quo(tD.Quo(pD))
	return cosmos.NewUintFromBigInt(result.RoundInt().BigInt())
}

// SafeSub subtract input2 from input1, given cosmos.Uint can't be negative , otherwise it will panic
// thus in this method,when input2 is larger than input 1, it will just return cosmos.ZeroUint
func SafeSub(input1, input2 cosmos.Uint) cosmos.Uint {
	if input2.GT(input1) {
		return cosmos.ZeroUint()
	}
	return input1.Sub(input2)
}
