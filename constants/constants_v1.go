package constants

// NewConstantValue010 get new instance of ConstantValue010
func NewConstantValue010() *ConstantVals {
	return &ConstantVals{
		int64values: map[ConstantName]int64{
			EmissionCurve:                   6,
			BlocksPerYear:                   6311390,
			TransactionFee:                  1_00000000,         // A 1.0 Rune fee on all swaps and withdrawals
			NewPoolCycle:                    51840,              // Enable a pool every 3 days
			MinimumNodesForYggdrasil:        6,                  // No yggdrasil pools if THORNode have less than 6 active nodes
			MinimumNodesForBFT:              4,                  // Minimum node count to keep network running. Below this, Ragnarök is performed.
			ValidatorRotateInNumBeforeFull:  2,                  // How many validators should THORNode nominate before THORNode reach the desire validator set
			ValidatorRotateOutNumBeforeFull: 1,                  // How many validators should THORNode queued to be rotate out before THORNode reach the desire validator set)
			ValidatorRotateNumAfterFull:     1,                  // How many validators should THORNode nominate after THORNode reach the desire validator set
			DesireValidatorSet:              30,                 // desire validator set
			FundMigrationInterval:           360,                // number of blocks THORNode will attempt to move funds from a retiring vault to an active one
			RotatePerBlockHeight:            51840,              // How many blocks THORNode try to rotate validators
			RotateRetryBlocks:               720,                // How many blocks until we retry a churn (only if we haven't had a successful churn in RotatePerBlockHeight blocks
			BadValidatorRate:                51840,              // rate to mark a validator to be rotated out for bad behavior
			OldValidatorRate:                51840,              // rate to mark a validator to be rotated out for age
			LackOfObservationPenalty:        2,                  // add two slash point for each block where a node does not observe
			SigningTransactionPeriod:        300,                // how many blocks before a request to sign a tx by yggdrasil pool, is counted as delinquent.
			DoubleSignMaxAge:                24,                 // number of blocks to limit double signing a block
			MinimumBondInRune:               1_000_000_00000000, // 1 million rune
			WhiteListGasAsset:               1000,               // thor coins we will be given to the validator
			FailKeygenSlashPoints:           720,                // slash for 720 blocks , which equals 1 hour
			FailKeySignSlashPoints:          2,                  // slash for 2 blocks
			StakeLockUpBlocks:               17280,              // the number of blocks staker can unstake after their stake
			ObserveSlashPoints:              1,                  // the number of slashpoints for making an observation (redeems later if observation reaches consensus
			ObserveFlex:                     5,                  // number of blocks of flexibility for a validator to get their slash points taken off for making an observation
			YggFundLimit:                    50,                 // percentage of the amount of funds a ygg vault is allowed to have.
			JailTimeKeygen:                  720 * 6,            // blocks a node account is jailed for failing to keygen. DO NOT drop below tss timeout
			JailTimeKeysign:                 60,                 // blocks a node account is jailed for failing to keysign. DO NOT drop below tss timeout
			CliTxCost:                       1_00000000,         // amount of bonded rune to move to the reserve when using a cli command
			MinSlashPointsForBadValidator:   100,                // The minimum slash point
			BadValidatorRedline:             3,                  // redline multiplier to find a multitude of bad actors
		},
		boolValues: map[ConstantName]bool{
			StrictBondStakeRatio: true,
		},
		stringValues: map[ConstantName]string{
			DefaultPoolStatus: "Bootstrap",
		},
	}
}
