package thorchain

import (
	"errors"

	"gitlab.com/thorchain/thornode/common"
)

type SwitchMemo struct {
	MemoBase
	Destination common.Address
}

func (m SwitchMemo) GetDestination() common.Address {
	return m.Destination
}

func NewSwitchMemo(addr common.Address) SwitchMemo {
	return SwitchMemo{
		MemoBase:    MemoBase{TxType: TxSwitch},
		Destination: addr,
	}
}

func ParseSwitchMemo(parts []string) (SwitchMemo, error) {
	if len(parts) < 2 {
		return SwitchMemo{}, errors.New("not enough parameters")
	}
	destination, err := common.NewAddress(parts[1])
	if err != nil {
		return SwitchMemo{}, err
	}
	if destination.IsEmpty() {
		return SwitchMemo{}, errors.New("address cannot be empty")
	}
	return NewSwitchMemo(destination), nil
}
