package common

import (
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
)

// This shifts the block height to be a larger number. The reason why this is
// added is because when/if we do a hard fork of the code, the new fork starts
// at block height 1 (although there is an issue to change this behavior within
// tendermint/cosmos sdk,
// https://github.com/tendermint/tendermint/issues/4646). This is a problem for
// our app because we store the block height in many objects. For example,
// txout uses block height, and bifrost nodes may attempt to sign the same
// transaction twice. We also track validator rewards using block height, which
// those maths assume block height is only ever increasing.
const heightShift int64 = 0

// BlockHeight return the adjusted block height
func BlockHeight(ctx cosmos.Context) int64 {
	return heightShift + ctx.BlockHeight()
}
