package thorchain

import (
	"fmt"

	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
)

type BondMemo struct {
	MemoBase
	NodeAddress cosmos.AccAddress
}

func (m BondMemo) GetAccAddress() cosmos.AccAddress { return m.NodeAddress }

func NewBondMemo(addr cosmos.AccAddress) BondMemo {
	return BondMemo{
		MemoBase:    MemoBase{TxType: TxBond},
		NodeAddress: addr,
	}
}

func ParseBondMemo(parts []string) (BondMemo, error) {
	if len(parts) < 2 {
		return BondMemo{}, fmt.Errorf("not enough parameters")
	}
	addr, err := cosmos.AccAddressFromBech32(parts[1])
	if err != nil {
		return BondMemo{}, fmt.Errorf("%s is an invalid thorchain address: %w", parts[1], err)
	}
	return NewBondMemo(addr), nil
}
