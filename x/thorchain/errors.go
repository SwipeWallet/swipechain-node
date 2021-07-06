package thorchain

import (
	"fmt"

	se "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/hashicorp/go-multierror"
)

// THORChain error code start at 99
const (
	// CodeBadVersion error code for bad version
	CodeInternalError         uint32 = 99
	CodeTxFail                uint32 = 100
	CodeBadVersion            uint32 = 101
	CodeInvalidMessage        uint32 = 102
	CodeConstantsNotAvailable uint32 = 103
	CodeInvalidVault          uint32 = 104
	CodeInvalidMemo           uint32 = 105
	CodeInvalidPoolStatus     uint32 = 107

	CodeSwapFail                 uint32 = 108
	CodeSwapFailNotEnoughFee     uint32 = 110
	CodeSwapFailInvalidAmount    uint32 = 113
	CodeSwapFailInvalidBalance   uint32 = 114
	CodeSwapFailNotEnoughBalance uint32 = 115

	CodeStakeFailValidation    uint32 = 120
	CodeFailGetStaker          uint32 = 122
	CodeStakeMismatchAssetAddr uint32 = 123
	CodeStakeInvalidPoolAsset  uint32 = 124
	CodeStakeRUNEOverLimit     uint32 = 125
	CodeStakeRUNEMoreThanBond  uint32 = 126

	CodeUnstakeFailValidation uint32 = 130
	CodeFailAddOutboundTx     uint32 = 131
	CodeFailSaveEvent         uint32 = 132
	CodeNoStakeUnitLeft       uint32 = 135
	CodeUnstakeWithin24Hours  uint32 = 136
	CodeUnstakeFail           uint32 = 137
	CodeEmptyChain            uint32 = 138
)

var (
	notAuthorized               = fmt.Errorf("not authorized")
	errInvalidVersion           = fmt.Errorf("bad version")
	errBadVersion               = se.Register(DefaultCodespace, CodeBadVersion, errInvalidVersion.Error())
	errInvalidMessage           = se.Register(DefaultCodespace, CodeInvalidMessage, "invalid message")
	errConstNotAvailable        = se.Register(DefaultCodespace, CodeConstantsNotAvailable, "constant values not available")
	errInvalidMemo              = se.Register(DefaultCodespace, CodeInvalidMemo, "invalid memo")
	errFailSaveEvent            = se.Register(DefaultCodespace, CodeFailSaveEvent, "fail to save add events")
	errStakeFailValidation      = se.Register(DefaultCodespace, CodeStakeFailValidation, "fail to validate stake")
	errStakeRUNEOverLimit       = se.Register(DefaultCodespace, CodeStakeRUNEOverLimit, "stake rune is over limit")
	errStakeRUNEMoreThanBond    = se.Register(DefaultCodespace, CodeStakeRUNEMoreThanBond, "stake rune is more than bond")
	errInvalidPoolStatus        = se.Register(DefaultCodespace, CodeInvalidPoolStatus, "invalid pool status")
	errFailAddOutboundTx        = se.Register(DefaultCodespace, CodeFailAddOutboundTx, "prepare outbound tx not successful")
	errUnstakeFailValidation    = se.Register(DefaultCodespace, CodeUnstakeFailValidation, "fail to validate unstake")
	errFailGetStaker            = se.Register(DefaultCodespace, CodeFailGetStaker, "fail to get staker")
	errStakeMismatchAssetAddr   = se.Register(DefaultCodespace, CodeStakeMismatchAssetAddr, "mismatch of asset address")
	errSwapFailNotEnoughFee     = se.Register(DefaultCodespace, CodeSwapFailNotEnoughFee, "fail swap, not enough fee")
	errSwapFail                 = se.Register(DefaultCodespace, CodeSwapFail, "fail swap")
	errSwapFailInvalidAmount    = se.Register(DefaultCodespace, CodeSwapFailInvalidAmount, "fail swap, invalid amount")
	errSwapFailInvalidBalance   = se.Register(DefaultCodespace, CodeSwapFailInvalidBalance, "fail swap, invalid balance")
	errSwapFailNotEnoughBalance = se.Register(DefaultCodespace, CodeSwapFailNotEnoughBalance, "fail swap, not enough balance")
	errNoStakeUnitLeft          = se.Register(DefaultCodespace, CodeNoStakeUnitLeft, "nothing to withdraw")
	errUnstakeWithin24Hours     = se.Register(DefaultCodespace, CodeUnstakeWithin24Hours, "you cannot unstake for 24 hours after staking for this blockchain")
	errUnstakeFail              = se.Register(DefaultCodespace, CodeUnstakeFail, "fail to unstake")
	errInternal                 = se.Register(DefaultCodespace, CodeInternalError, "internal error")
)

// ErrInternal return an error  of errInternal with additional message
func ErrInternal(err error, msg string) error {
	return se.Wrap(multierror.Append(errInternal, err), msg)
}
