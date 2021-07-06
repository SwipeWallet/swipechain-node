package thorchain

import (
	"fmt"
	"strings"

	"gitlab.com/thorchain/thornode/common"
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
)

// TXTYPE:STATE1:STATE2:STATE3:FINALMEMO

type TxType uint8

const (
	TxUnknown TxType = iota
	TxStake
	TxUnstake
	TxSwap
	TxOutbound
	TxAdd
	TxBond
	TxUnbond
	TxLeave
	TxYggdrasilFund
	TxYggdrasilReturn
	TxReserve
	TxRefund
	TxMigrate
	TxRagnarok
	TxSwitch
)

var stringToTxTypeMap = map[string]TxType{
	"stake":      TxStake,
	"st":         TxStake,
	"+":          TxStake,
	"withdraw":   TxUnstake,
	"unstake":    TxUnstake,
	"wd":         TxUnstake,
	"-":          TxUnstake,
	"swap":       TxSwap,
	"s":          TxSwap,
	"=":          TxSwap,
	"outbound":   TxOutbound,
	"add":        TxAdd,
	"a":          TxAdd,
	"%":          TxAdd,
	"bond":       TxBond,
	"unbond":     TxUnbond,
	"leave":      TxLeave,
	"yggdrasil+": TxYggdrasilFund,
	"yggdrasil-": TxYggdrasilReturn,
	"reserve":    TxReserve,
	"refund":     TxRefund,
	"migrate":    TxMigrate,
	"ragnarok":   TxRagnarok,
	"switch":     TxSwitch,
}

var txToStringMap = map[TxType]string{
	TxStake:           "stake",
	TxUnstake:         "unstake",
	TxSwap:            "swap",
	TxOutbound:        "outbound",
	TxRefund:          "refund",
	TxAdd:             "add",
	TxBond:            "bond",
	TxUnbond:          "unbond",
	TxLeave:           "leave",
	TxYggdrasilFund:   "yggdrasil+",
	TxYggdrasilReturn: "yggdrasil-",
	TxReserve:         "reserve",
	TxMigrate:         "migrate",
	TxRagnarok:        "ragnarok",
	TxSwitch:          "switch",
}

// converts a string into a txType
func StringToTxType(s string) (TxType, error) {
	// THORNode can support Abbreviated MEMOs , usually it is only one character
	sl := strings.ToLower(s)
	if t, ok := stringToTxTypeMap[sl]; ok {
		return t, nil
	}
	return TxUnknown, fmt.Errorf("invalid tx type: %s", s)
}

func (tx TxType) IsInbound() bool {
	switch tx {
	case TxStake, TxUnstake, TxSwap, TxAdd, TxBond, TxUnbond, TxLeave, TxSwitch, TxReserve:
		return true
	default:
		return false
	}
}

func (tx TxType) IsOutbound() bool {
	switch tx {
	case TxOutbound, TxRefund, TxRagnarok:
		return true
	default:
		return false
	}
}

func (tx TxType) IsInternal() bool {
	switch tx {
	case TxYggdrasilFund, TxYggdrasilReturn, TxMigrate:
		return true
	default:
		return false
	}
}

func (tx TxType) IsEmpty() bool {
	return tx == TxUnknown
}

// Check if two txTypes are the same
func (tx TxType) Equals(tx2 TxType) bool {
	return tx == tx2
}

// Converts a txType into a string
func (tx TxType) String() string {
	return txToStringMap[tx]
}

type Memo interface {
	IsType(tx TxType) bool
	GetType() TxType
	IsEmpty() bool
	IsInbound() bool
	IsOutbound() bool
	IsInternal() bool

	String() string
	GetAsset() common.Asset
	GetAmount() cosmos.Uint
	GetDestination() common.Address
	GetSlipLimit() cosmos.Uint
	GetTxID() common.TxID
	GetAccAddress() cosmos.AccAddress
	GetBlockHeight() int64
}

type MemoBase struct {
	TxType TxType
	Asset  common.Asset
}

func (m MemoBase) String() string                   { return "" }
func (m MemoBase) GetType() TxType                  { return m.TxType }
func (m MemoBase) IsType(tx TxType) bool            { return m.TxType.Equals(tx) }
func (m MemoBase) GetAsset() common.Asset           { return m.Asset }
func (m MemoBase) GetAmount() cosmos.Uint           { return cosmos.ZeroUint() }
func (m MemoBase) GetDestination() common.Address   { return "" }
func (m MemoBase) GetSlipLimit() cosmos.Uint        { return cosmos.ZeroUint() }
func (m MemoBase) GetTxID() common.TxID             { return "" }
func (m MemoBase) GetAccAddress() cosmos.AccAddress { return cosmos.AccAddress{} }
func (m MemoBase) GetBlockHeight() int64            { return 0 }
func (m MemoBase) IsOutbound() bool                 { return m.TxType.IsOutbound() }
func (m MemoBase) IsInbound() bool                  { return m.TxType.IsInbound() }
func (m MemoBase) IsInternal() bool                 { return m.TxType.IsInternal() }
func (m MemoBase) IsEmpty() bool                    { return m.TxType.IsEmpty() }

func ParseMemo(memo string) (Memo, error) {
	var err error
	noMemo := MemoBase{TxType: TxUnknown}
	if len(memo) == 0 {
		return noMemo, fmt.Errorf("memo can't be empty")
	}
	parts := strings.Split(memo, ":")
	tx, err := StringToTxType(parts[0])
	if err != nil {
		return noMemo, err
	}

	var asset common.Asset
	switch tx {
	case TxAdd, TxStake, TxSwap, TxUnstake:
		if len(parts) < 2 {
			return noMemo, fmt.Errorf("cannot parse given memo: length %d", len(parts))
		}
		asset, err = common.NewAsset(parts[1])
		if err != nil {
			return noMemo, err
		}
	}

	switch tx {
	case TxLeave:
		return ParseLeaveMemo(parts)
	case TxAdd:
		return NewAddMemo(asset), nil
	case TxStake:
		return ParseStakeMemo(asset, parts)
	case TxUnstake:
		return ParseUnstakeMemo(asset, parts)
	case TxSwap:
		return ParseSwapMemo(asset, parts)
	case TxOutbound:
		return ParseOutboundMemo(parts)
	case TxRefund:
		return ParseRefundMemo(parts)
	case TxBond:
		return ParseBondMemo(parts)
	case TxUnbond:
		return ParseUnbondMemo(parts)
	case TxYggdrasilFund:
		return ParseYggdrasilFundMemo(parts)
	case TxYggdrasilReturn:
		return ParseYggdrasilReturnMemo(parts)
	case TxReserve:
		return NewReserveMemo(), nil
	case TxMigrate:
		return ParseMigrateMemo(parts)
	case TxRagnarok:
		return ParseRagnarokMemo(parts)
	case TxSwitch:
		return ParseSwitchMemo(parts)
	default:
		return noMemo, fmt.Errorf("TxType not supported: %s", tx.String())
	}
}
