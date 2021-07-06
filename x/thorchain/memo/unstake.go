package thorchain

import (
	"fmt"

	"gitlab.com/thorchain/thornode/common"
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/x/thorchain/types"
)

type UnstakeMemo struct {
	MemoBase
	Amount cosmos.Uint
}

func (m UnstakeMemo) GetAmount() cosmos.Uint { return m.Amount }

func NewUnstakeMemo(asset common.Asset, amt cosmos.Uint) UnstakeMemo {
	return UnstakeMemo{
		MemoBase: MemoBase{TxType: TxUnstake, Asset: asset},
		Amount:   amt,
	}
}

func ParseUnstakeMemo(asset common.Asset, parts []string) (UnstakeMemo, error) {
	var err error
	if len(parts) < 2 {
		return UnstakeMemo{}, fmt.Errorf("not enough parameters")
	}
	withdrawlBasisPts := cosmos.ZeroUint()
	if len(parts) > 2 {
		withdrawlBasisPts, err = cosmos.ParseUint(parts[2])
		if err != nil {
			return UnstakeMemo{}, err
		}
		if withdrawlBasisPts.IsZero() || withdrawlBasisPts.GT(cosmos.NewUint(types.MaxUnstakeBasisPoints)) {
			return UnstakeMemo{}, fmt.Errorf("withdraw amount %s is invalid", parts[2])
		}
	}
	return NewUnstakeMemo(asset, withdrawlBasisPts), nil
}
