package types

import (
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
)

type VaultDataSuite struct{}

var _ = Suite(&VaultDataSuite{})

func (s *VaultDataSuite) TestCalcNodeRewards(c *C) {
	vaultdata1 := NewVaultData()
	c.Check(vaultdata1.BondRewardRune.Uint64(), Equals, cosmos.ZeroUint().Uint64())

	vault := VaultData{
		TotalBondUnits: cosmos.NewUint(100),
		BondRewardRune: cosmos.NewUint(3000),
	}
	reward := vault.CalcNodeRewards(cosmos.NewUint(5))
	c.Check(reward.Uint64(), Equals, uint64(150))

	vault = VaultData{
		TotalBondUnits: cosmos.NewUint(7357),
		BondRewardRune: cosmos.NewUint(275.357 * common.One),
	}
	reward = vault.CalcNodeRewards(cosmos.NewUint(78))
	c.Check(reward.Uint64(), Equals, uint64(291937556))

	vault = VaultData{
		TotalBondUnits: cosmos.NewUint(7357),
		BondRewardRune: cosmos.ZeroUint(),
	}
	reward = vault.CalcNodeRewards(cosmos.NewUint(78))
	c.Check(reward.Uint64(), Equals, uint64(0))
}
