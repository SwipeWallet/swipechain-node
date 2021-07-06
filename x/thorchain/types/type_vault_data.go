package types

import (
	"gitlab.com/thorchain/thornode/common"
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
)

// VaultData keep track of reserve , reward and bond
type VaultData struct {
	BondRewardRune cosmos.Uint `json:"bond_reward_rune"` // The total amount of awarded rune for bonders
	TotalBondUnits cosmos.Uint `json:"total_bond_units"` // Total amount of bond units
	TotalReserve   cosmos.Uint `json:"total_reserve"`    // Total amount of reserves (in rune)
	TotalBEP2Rune  cosmos.Uint `json:"total_bep2_rune"`  // Total amount of BEP2 rune held
}

// NewVaultData create a new instance VaultData it is empty though
func NewVaultData() VaultData {
	return VaultData{
		BondRewardRune: cosmos.ZeroUint(),
		TotalBondUnits: cosmos.ZeroUint(),
		TotalReserve:   cosmos.ZeroUint(),
		TotalBEP2Rune:  cosmos.ZeroUint(),
	}
}

// CalcNodeRewards calculate node rewards
func (v VaultData) CalcNodeRewards(nodeUnits cosmos.Uint) cosmos.Uint {
	return common.GetShare(nodeUnits, v.TotalBondUnits, v.BondRewardRune)
}
