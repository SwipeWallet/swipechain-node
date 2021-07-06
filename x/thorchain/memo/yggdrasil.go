package thorchain

import (
	"errors"
	"fmt"
	"strconv"
)

type YggdrasilFundMemo struct {
	MemoBase
	BlockHeight int64
}

func (m YggdrasilFundMemo) String() string {
	return fmt.Sprintf("YGGDRASIL+:%d", m.BlockHeight)
}

func (m YggdrasilFundMemo) GetBlockHeight() int64 {
	return m.BlockHeight
}

type YggdrasilReturnMemo struct {
	MemoBase
	BlockHeight int64
}

func (m YggdrasilReturnMemo) String() string {
	return fmt.Sprintf("YGGDRASIL-:%d", m.BlockHeight)
}

func (m YggdrasilReturnMemo) GetBlockHeight() int64 {
	return m.BlockHeight
}

func NewYggdrasilFund(blockHeight int64) YggdrasilFundMemo {
	return YggdrasilFundMemo{
		MemoBase:    MemoBase{TxType: TxYggdrasilFund},
		BlockHeight: blockHeight,
	}
}

func NewYggdrasilReturn(blockHeight int64) YggdrasilReturnMemo {
	return YggdrasilReturnMemo{
		MemoBase:    MemoBase{TxType: TxYggdrasilReturn},
		BlockHeight: blockHeight,
	}
}

func ParseYggdrasilFundMemo(parts []string) (YggdrasilFundMemo, error) {
	if len(parts) < 2 {
		return YggdrasilFundMemo{}, errors.New("not enough parameters")
	}
	blockHeight, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return YggdrasilFundMemo{}, fmt.Errorf("fail to convert (%s) to a valid block height: %w", parts[1], err)
	}
	return NewYggdrasilFund(blockHeight), nil
}

func ParseYggdrasilReturnMemo(parts []string) (YggdrasilReturnMemo, error) {
	if len(parts) < 2 {
		return YggdrasilReturnMemo{}, errors.New("not enough parameters")
	}
	blockHeight, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return YggdrasilReturnMemo{}, fmt.Errorf("fail to convert (%s) to a valid block height: %w", parts[1], err)
	}
	return NewYggdrasilReturn(blockHeight), nil
}
