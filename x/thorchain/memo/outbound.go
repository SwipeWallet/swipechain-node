package thorchain

import (
	"fmt"

	"gitlab.com/thorchain/thornode/common"
)

type OutboundMemo struct {
	MemoBase
	TxID common.TxID
}

func (m OutboundMemo) GetTxID() common.TxID { return m.TxID }
func (m OutboundMemo) String() string {
	return fmt.Sprintf("OUTBOUND:%s", m.TxID.String())
}

func NewOutboundMemo(txID common.TxID) OutboundMemo {
	return OutboundMemo{
		MemoBase: MemoBase{TxType: TxOutbound},
		TxID:     txID,
	}
}

func ParseOutboundMemo(parts []string) (OutboundMemo, error) {
	if len(parts) < 2 {
		return OutboundMemo{}, fmt.Errorf("not enough parameters")
	}
	txID, err := common.NewTxID(parts[1])
	return NewOutboundMemo(txID), err
}
