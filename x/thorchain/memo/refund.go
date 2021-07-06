package thorchain

import (
	"fmt"

	"gitlab.com/thorchain/thornode/common"
)

type RefundMemo struct {
	MemoBase
	TxID common.TxID
}

func (m RefundMemo) GetTxID() common.TxID { return m.TxID }

// String implement fmt.Stringer
func (m RefundMemo) String() string {
	return fmt.Sprintf("REFUND:%s", m.TxID.String())
}

// NewRefundMemo create a new RefundMemo
func NewRefundMemo(txID common.TxID) RefundMemo {
	return RefundMemo{
		MemoBase: MemoBase{TxType: TxRefund},
		TxID:     txID,
	}
}

func ParseRefundMemo(parts []string) (RefundMemo, error) {
	if len(parts) < 2 {
		return RefundMemo{}, fmt.Errorf("not enough parameters")
	}
	txID, err := common.NewTxID(parts[1])
	return NewRefundMemo(txID), err
}
