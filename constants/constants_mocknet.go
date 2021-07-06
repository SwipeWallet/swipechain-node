// +build mocknet

// For internal testing and mockneting
package constants

func init() {
	int64Overrides = map[ConstantName]int64{
		// ArtificialRagnarokBlockHeight: 200,
		DesireValidatorSet:    12,
		RotatePerBlockHeight:  60,          // 5 min
		BadValidatorRate:      60,          // 5 min
		OldValidatorRate:      60,          // 5 min
		MinimumBondInRune:     100_000_000, // 1 rune
		FundMigrationInterval: 10,
		StakeLockUpBlocks:     0,
		CliTxCost:             0,
		JailTimeKeygen:        10,
		JailTimeKeysign:       10,
	}
	boolOverrides = map[ConstantName]bool{
		StrictBondStakeRatio: false,
	}
	stringOverrides = map[ConstantName]string{
		DefaultPoolStatus: "Enabled",
	}
}
