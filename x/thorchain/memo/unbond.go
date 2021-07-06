package thorchain

import (
	"fmt"

	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
)

type UnbondMemo struct {
	MemoBase
	NodeAddress cosmos.AccAddress
	Amount      cosmos.Uint
}

func (m UnbondMemo) GetAccAddress() cosmos.AccAddress { return m.NodeAddress }
func (m UnbondMemo) GetAmount() cosmos.Uint           { return m.Amount }

func NewUnbondMemo(addr cosmos.AccAddress, amt cosmos.Uint) UnbondMemo {
	return UnbondMemo{
		MemoBase:    MemoBase{TxType: TxUnbond},
		NodeAddress: addr,
		Amount:      amt,
	}
}

func ParseUnbondMemo(parts []string) (UnbondMemo, error) {
	if len(parts) < 3 {
		return UnbondMemo{}, fmt.Errorf("not enough parameters")
	}
	addr, err := cosmos.AccAddressFromBech32(parts[1])
	if err != nil {
		return UnbondMemo{}, fmt.Errorf("%s is an invalid thorchain address: %w", parts[1], err)
	}
	amt, err := cosmos.ParseUint(parts[2])
	if err != nil {
		return UnbondMemo{}, fmt.Errorf("fail to parse amount (%s): %w", parts[2], err)
	}
	return NewUnbondMemo(addr, amt), nil
}
