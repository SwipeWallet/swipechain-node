package thorchain

import (
	"errors"
	"fmt"
	"strconv"
)

type MigrateMemo struct {
	MemoBase
	BlockHeight int64
}

func (m MigrateMemo) String() string {
	return fmt.Sprintf("MIGRATE:%d", m.BlockHeight)
}

func (m MigrateMemo) GetBlockHeight() int64 {
	return m.BlockHeight
}

func NewMigrateMemo(blockHeight int64) MigrateMemo {
	return MigrateMemo{
		MemoBase:    MemoBase{TxType: TxMigrate},
		BlockHeight: blockHeight,
	}
}

func ParseMigrateMemo(parts []string) (MigrateMemo, error) {
	if len(parts) < 2 {
		return MigrateMemo{}, errors.New("not enough parameters")
	}
	blockHeight, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return MigrateMemo{}, fmt.Errorf("fail to convert (%s) to a valid block height: %w", parts[1], err)
	}
	return NewMigrateMemo(blockHeight), nil
}
