package thorchain

import (
	"fmt"

	"gitlab.com/thorchain/thornode/common"
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
)

type SwapMemo struct {
	MemoBase
	Destination common.Address
	SlipLimit   cosmos.Uint
}

func (m SwapMemo) GetDestination() common.Address { return m.Destination }
func (m SwapMemo) GetSlipLimit() cosmos.Uint      { return m.SlipLimit }

func NewSwapMemo(asset common.Asset, dest common.Address, slip cosmos.Uint) SwapMemo {
	return SwapMemo{
		MemoBase:    MemoBase{TxType: TxSwap, Asset: asset},
		Destination: dest,
		SlipLimit:   slip,
	}
}

func ParseSwapMemo(asset common.Asset, parts []string) (SwapMemo, error) {
	var err error
	if len(parts) < 2 {
		return SwapMemo{}, fmt.Errorf("not enough parameters")
	}
	// DESTADDR can be empty , if it is empty , it will swap to the sender address
	destination := common.NoAddress
	if len(parts) > 2 {
		if len(parts[2]) > 0 {
			destination, err = common.NewAddress(parts[2])
			if err != nil {
				return SwapMemo{}, err
			}
		}
	}
	// price limit can be empty , when it is empty , there is no price protection
	slip := cosmos.ZeroUint()
	if len(parts) > 3 && len(parts[3]) > 0 {
		amount, err := cosmos.ParseUint(parts[3])
		if err != nil {
			return SwapMemo{}, fmt.Errorf("swap price limit:%s is invalid", parts[3])
		}

		slip = amount
	}
	return NewSwapMemo(asset, destination, slip), nil
}
