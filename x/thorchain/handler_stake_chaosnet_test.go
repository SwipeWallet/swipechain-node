// +build chaosnet

package thorchain

import (
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
)

func (HandlerStakeSuite) TestStakeRUNEOverLimit(c *C) {
	ctx, _ := setupKeeperForTest(c)
	activeNodeAccount := GetRandomNodeAccount(NodeActive)
	k := &MockStakeKeeper{
		activeNodeAccount: activeNodeAccount,
		currentPool: Pool{
			BalanceRune:  cosmos.ZeroUint(),
			BalanceAsset: cosmos.ZeroUint(),
			Asset:        common.BNBAsset,
			PoolUnits:    cosmos.ZeroUint(),
			PoolAddress:  "",
			Status:       PoolEnabled,
		},
	}
	// happy path
	stakeHandler := NewStakeHandler(k)
	bnbAddr := GetRandomBNBAddress()
	stakeTxHash := GetRandomTxHash()
	tx := common.NewTx(
		stakeTxHash,
		bnbAddr,
		GetRandomBNBAddress(),
		common.Coins{common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One*5))},
		common.BNBGasFeeSingleton,
		"stake:BNB",
	)
	ver := constants.SWVersion
	msgSetStake := NewMsgSetStakeData(
		tx,
		common.BNBAsset,
		cosmos.NewUint(1000_000*common.One),
		cosmos.NewUint(100_000*common.One),
		bnbAddr,
		bnbAddr,
		activeNodeAccount.NodeAddress)
	constAccessor := constants.NewConstantValue010()
	result, err := stakeHandler.Run(ctx, msgSetStake, ver, constAccessor)
	c.Assert(err, IsNil)
	c.Assert(result.Code, Equals, CodeStakeRUNEOverLimit)
}
