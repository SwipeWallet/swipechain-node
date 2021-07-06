package thorchain

import (
	"fmt"

	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
)

type LeaveMemo struct {
	MemoBase
	NodeAddress cosmos.AccAddress
}

func (m LeaveMemo) GetAccAddress() cosmos.AccAddress { return m.NodeAddress }

func NewLeaveMemo(addr cosmos.AccAddress) LeaveMemo {
	return LeaveMemo{
		MemoBase:    MemoBase{TxType: TxLeave},
		NodeAddress: addr,
	}
}

func ParseLeaveMemo(parts []string) (LeaveMemo, error) {
	if len(parts) < 2 {
		return LeaveMemo{}, fmt.Errorf("not enough parameters")
	}
	addr, err := cosmos.AccAddressFromBech32(parts[1])
	if err != nil {
		return LeaveMemo{}, fmt.Errorf("%s is an invalid thorchain address: %w", parts[1], err)
	}
	return NewLeaveMemo(addr), nil
}
