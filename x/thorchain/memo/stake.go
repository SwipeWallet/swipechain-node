package thorchain

import (
	"fmt"

	"gitlab.com/thorchain/thornode/common"
)

type StakeMemo struct {
	MemoBase
	Address common.Address
}

func (m StakeMemo) GetDestination() common.Address { return m.Address }

func NewStakeMemo(asset common.Asset, addr common.Address) StakeMemo {
	return StakeMemo{
		MemoBase: MemoBase{TxType: TxStake, Asset: asset},
		Address:  addr,
	}
}

func ParseStakeMemo(asset common.Asset, parts []string) (StakeMemo, error) {
	var addr common.Address
	var err error
	if !asset.Chain.Equals(common.RuneAsset().Chain) {
		if len(parts) < 3 {
			// cannot stake into a non THOR-based pool when THORNode don't have an
			// associated address
			return StakeMemo{}, fmt.Errorf("invalid stake. Cannot stake to a non THOR-based pool without providing an associated address")
		}
		addr, err = common.NewAddress(parts[2])
		if err != nil {
			return StakeMemo{}, err
		}
	}
	return NewStakeMemo(asset, addr), nil
}
