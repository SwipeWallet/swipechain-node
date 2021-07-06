// +build testnet

// For Public TestNet
package constants

func init() {
	int64Overrides = map[ConstantName]int64{
		NewPoolCycle:         1000,
		DesireValidatorSet:   30,
		RotatePerBlockHeight: 240,
		BadValidatorRate:     17280,
		OldValidatorRate:     17280,
		MinimumBondInRune:    10000_00000000, // 1 rune
		StakeLockUpBlocks:    0,
		CliTxCost:            0,
	}
	boolOverrides = map[ConstantName]bool{
		StrictBondStakeRatio: false,
	}
}
