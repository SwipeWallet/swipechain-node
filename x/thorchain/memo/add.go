package thorchain

import (
	"fmt"

	"gitlab.com/thorchain/thornode/common"
)

type AddMemo struct{ MemoBase }

func (m AddMemo) String() string {
	return fmt.Sprintf("ADD:%s", m.Asset)
}

func NewAddMemo(asset common.Asset) AddMemo {
	return AddMemo{
		MemoBase: MemoBase{TxType: TxAdd, Asset: asset},
	}
}
