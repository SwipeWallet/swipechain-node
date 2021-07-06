package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

// PoolStatus is an indication of what the pool state is
type PoolStatus int

// |    State    | Swap | Stake | Unstake  | Refunding |
// | ----------- | ---- | ----- | -------- | --------- |
// | `bootstrap` | no   | yes   | yes      | Refund Invalid Stakes && all Swaps |
// | `enabled`   | yes  | yes   | yes      | Refund Invalid Tx |
// | `suspended` | no   | no    | no       | Refund all |
const (
	Enabled PoolStatus = iota
	Bootstrap
	Suspended
)

var poolStatusStr = map[string]PoolStatus{
	"Enabled":   Enabled,
	"Bootstrap": Bootstrap,
	"Suspended": Suspended,
}

// String implement stringer
func (ps PoolStatus) String() string {
	for key, item := range poolStatusStr {
		if item == ps {
			return key
		}
	}
	return ""
}

// Valid is to check whether the pool status is valid or not
func (ps PoolStatus) Valid() error {
	if ps.String() == "" {
		return errors.New("invalid pool status")
	}
	return nil
}

// MarshalJSON marshal PoolStatus to JSON in string form
func (ps PoolStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(ps.String())
}

// UnmarshalJSON convert string form back to PoolStatus
func (ps *PoolStatus) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	*ps = GetPoolStatus(s)
	return nil
}

// GetPoolStatus from string
func GetPoolStatus(ps string) PoolStatus {
	for key, item := range poolStatusStr {
		if strings.EqualFold(key, ps) {
			return item
		}
	}
	return Suspended
}

// Pool is a struct that contains all the metadata of a pooldata
// This is the structure THORNode will saved to the key value store
type Pool struct {
	BalanceRune  cosmos.Uint  `json:"balance_rune"`  // how many RUNE in the pool
	BalanceAsset cosmos.Uint  `json:"balance_asset"` // how many asset in the pool
	Asset        common.Asset `json:"asset"`         // what's the asset's asset
	PoolUnits    cosmos.Uint  `json:"pool_units"`    // total units of the pool
	Status       PoolStatus   `json:"status"`        // status
}

// Pools represent a list of pools
type Pools []Pool

// NewPool Returns a new Pool
func NewPool() Pool {
	return Pool{
		BalanceRune:  cosmos.ZeroUint(),
		BalanceAsset: cosmos.ZeroUint(),
		PoolUnits:    cosmos.ZeroUint(),
		Status:       Enabled,
	}
}

// Valid check whether the pool is valid or not, if asset is empty then it is not valid
func (ps Pool) Valid() error {
	if ps.IsEmpty() {
		return errors.New("pool asset cannot be empty")
	}
	return nil
}

// IsEnabled check whether the pool is in Enabled status
func (ps Pool) IsEnabled() bool {
	return ps.Status == Enabled
}

// Empty will return true when the asset is empty
func (ps Pool) IsEmpty() bool {
	return ps.Asset.IsEmpty()
}

// String implement fmt.Stringer
func (ps Pool) String() string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintln("rune-balance: " + ps.BalanceRune.String()))
	sb.WriteString(fmt.Sprintln("asset-balance: " + ps.BalanceAsset.String()))
	sb.WriteString(fmt.Sprintln("asset: " + ps.Asset.String()))
	sb.WriteString(fmt.Sprintln("pool-units: " + ps.PoolUnits.String()))
	sb.WriteString(fmt.Sprintln("status: " + ps.Status.String()))
	return sb.String()
}

// EnsureValidPoolStatus make sure the pool is in a valid status otherwise it return an error
func (ps Pool) EnsureValidPoolStatus(msg cosmos.Msg) error {
	switch ps.Status {
	case Enabled:
		return nil
	case Bootstrap:
		switch msg.(type) {
		case MsgSwap:
			return errors.New("pool is in bootstrap status, can't swap")
		default:
			return nil
		}
	case Suspended:
		return errors.New("pool suspended")
	default:
		return fmt.Errorf("unknown pool status,%s", ps.Status)
	}
}

// AssetValueInRune convert a specific amount of asset amt into its rune value
func (ps Pool) AssetValueInRune(amt cosmos.Uint) cosmos.Uint {
	if ps.BalanceRune.IsZero() || ps.BalanceAsset.IsZero() {
		return cosmos.ZeroUint()
	}
	return common.GetShare(ps.BalanceRune, ps.BalanceAsset, amt)
}

// RuneValueInAsset convert a specific amount of rune amt into its asset value
func (ps Pool) RuneValueInAsset(amt cosmos.Uint) cosmos.Uint {
	if ps.BalanceRune.IsZero() || ps.BalanceAsset.IsZero() {
		return cosmos.ZeroUint()
	}
	return common.GetShare(ps.BalanceAsset, ps.BalanceRune, amt)
}
