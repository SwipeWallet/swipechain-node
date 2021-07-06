package types

import (
	"fmt"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

// MaxUnstakeBasisPoints basis points for unstake
const MaxUnstakeBasisPoints = 10_000

// MsgUnStake is used to withdraw
type MsgUnStake struct {
	Tx                 common.Tx         `json:"tx"`
	RuneAddress        common.Address    `json:"rune_address"`          // it should be the rune address
	UnstakeBasisPoints cosmos.Uint       `json:"withdraw_basis_points"` // withdraw basis points
	Asset              common.Asset      `json:"asset"`                 // asset asset asset
	Signer             cosmos.AccAddress `json:"signer"`
}

// NewMsgUnStake is a constructor function for MsgSetPoolData
func NewMsgUnStake(tx common.Tx, runeAddress common.Address, withdrawBasisPoints cosmos.Uint, asset common.Asset, signer cosmos.AccAddress) MsgUnStake {
	return MsgUnStake{
		Tx:                 tx,
		RuneAddress:        runeAddress,
		UnstakeBasisPoints: withdrawBasisPoints,
		Asset:              asset,
		Signer:             signer,
	}
}

// Route should return the route key of the module
func (msg MsgUnStake) Route() string { return RouterKey }

// Type should return the action
func (msg MsgUnStake) Type() string { return "unstake" }

// ValidateBasic runs stateless checks on the message
func (msg MsgUnStake) ValidateBasic() error {
	if msg.Signer.Empty() {
		return cosmos.ErrInvalidAddress(msg.Signer.String())
	}
	if err := msg.Tx.Valid(); err != nil {
		return cosmos.ErrUnknownRequest(err.Error())
	}
	if msg.Asset.IsEmpty() {
		return cosmos.ErrUnknownRequest("Pool Asset cannot be empty")
	}
	if msg.RuneAddress.IsEmpty() {
		return cosmos.ErrUnknownRequest("Address cannot be empty")
	}
	if !msg.RuneAddress.IsChain(common.RuneAsset().Chain) {
		return cosmos.ErrUnknownRequest(fmt.Sprintf("Address must be a %s address", common.RuneAsset().Chain))
	}
	if msg.UnstakeBasisPoints.IsZero() {
		return cosmos.ErrUnknownRequest("UnstakeBasicPoints can't be zero")
	}
	if msg.UnstakeBasisPoints.GT(cosmos.NewUint(MaxUnstakeBasisPoints)) {
		return cosmos.ErrUnknownRequest("UnstakeBasisPoints is larger than maximum withdraw basis points")
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgUnStake) GetSignBytes() []byte {
	return cosmos.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgUnStake) GetSigners() []cosmos.AccAddress {
	return []cosmos.AccAddress{msg.Signer}
}
