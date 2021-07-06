package types

import (
	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

// Jail keep the information about node account been jailed
type Jail struct {
	NodeAddress   cosmos.AccAddress `json:"node_address"`
	ReleaseHeight int64             `json:"release_height"`
	Reason        string            `json:"reason"`
}

// NewJail create a new Jail instance
func NewJail(addr cosmos.AccAddress) Jail {
	return Jail{
		NodeAddress: addr,
	}
}

// IsJailed on a given height , check whether a node is jailed or not
func (j Jail) IsJailed(ctx cosmos.Context) bool {
	return j.ReleaseHeight > common.BlockHeight(ctx)
}
