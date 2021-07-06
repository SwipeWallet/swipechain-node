package thorchain

import (
	"errors"
	"fmt"
	"strconv"
)

type RagnarokMemo struct {
	MemoBase
	BlockHeight int64
}

func (m RagnarokMemo) String() string {
	return fmt.Sprintf("RAGNAROK:%d", m.BlockHeight)
}

func (m RagnarokMemo) GetBlockHeight() int64 {
	return m.BlockHeight
}

func NewRagnarokMemo(blockHeight int64) RagnarokMemo {
	return RagnarokMemo{
		MemoBase:    MemoBase{TxType: TxRagnarok},
		BlockHeight: blockHeight,
	}
}

func ParseRagnarokMemo(parts []string) (RagnarokMemo, error) {
	if len(parts) < 2 {
		return RagnarokMemo{}, errors.New("not enough parameters")
	}
	blockHeight, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return RagnarokMemo{}, fmt.Errorf("fail to convert (%s) to a valid block height: %w", parts[1], err)
	}
	return NewRagnarokMemo(blockHeight), nil
}
