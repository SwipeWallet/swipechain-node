package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/blang/semver"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

// NodeStatus Represent the Node status
type NodeStatus uint8

// As soon as user paid a certain amount of asset(defined later)
// their node address will be whitelisted
// once THORNode discover their observer had send tx in , then their status will be standby
// once THORNode rotate them in , then they will be active
const (
	Unknown NodeStatus = iota
	WhiteListed
	Standby
	Ready
	Active
	Disabled
)

var nodeStatusStr = map[string]NodeStatus{
	"unknown":     Unknown,
	"whitelisted": WhiteListed,
	"standby":     Standby,
	"ready":       Ready,
	"active":      Active,
	"disabled":    Disabled,
}

// String implement stringer
func (ps NodeStatus) String() string {
	for key, item := range nodeStatusStr {
		if item == ps {
			return key
		}
	}
	return ""
}

// Valid check whether the node status is valid or not
func (ps NodeStatus) Valid() error {
	if ps.String() == "" {
		return fmt.Errorf("invalid node status")
	}
	return nil
}

// MarshalJSON marshal NodeStatus to JSON in string form
func (ps NodeStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(ps.String())
}

// UnmarshalJSON convert string form back to NodeStatus
func (ps *NodeStatus) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	*ps = GetNodeStatus(s)
	return nil
}

// GetNodeStatus from string
func GetNodeStatus(ps string) NodeStatus {
	for key, item := range nodeStatusStr {
		if strings.EqualFold(key, ps) {
			return item
		}
	}

	return Unknown
}

// NodeAccount everything about node account
type NodeAccount struct {
	NodeAddress         cosmos.AccAddress `json:"node_address"` // Thor address which is an operator address
	Status              NodeStatus        `json:"status"`
	PubKeySet           common.PubKeySet  `json:"pub_key_set"`
	ValidatorConsPubKey string            `json:"validator_cons_pub_key"`
	Bond                cosmos.Uint       `json:"bond"`
	ActiveBlockHeight   int64             `json:"active_block_height"` // The block height when this node account became active status
	BondAddress         common.Address    `json:"bond_address"`        // BNB Address to send bond from. It also indicates the operator address to whilelist and associate.
	StatusSince         int64             `json:"status_since"`
	SignerMembership    common.PubKeys    `json:"signer_membership"`
	RequestedToLeave    bool              `json:"requested_to_leave"`
	ForcedToLeave       bool              `json:"forced_to_leave"`
	LeaveHeight         int64             `json:"leave_height"`
	IPAddress           string            `json:"ip_address"`
	Version             semver.Version    `json:"version"`
}

// NewNodeAccount create new instance of NodeAccount
func NewNodeAccount(nodeAddress cosmos.AccAddress, status NodeStatus, nodePubKeySet common.PubKeySet, validatorConsPubKey string, bond cosmos.Uint, bondAddress common.Address, height int64) NodeAccount {
	na := NodeAccount{
		NodeAddress:         nodeAddress,
		PubKeySet:           nodePubKeySet,
		ValidatorConsPubKey: validatorConsPubKey,
		Bond:                bond,
		BondAddress:         bondAddress,
	}
	na.UpdateStatus(status, height)
	return na
}

// IsEmpty decide whether NodeAccount is empty
func (n NodeAccount) IsEmpty() bool {
	return n.NodeAddress.Empty() || n.Status == Unknown
}

// Valid check whether NodeAccount has all necessary values
func (n NodeAccount) Valid() error {
	if n.NodeAddress.Empty() {
		return errors.New("node thor address is empty")
	}
	if n.BondAddress.IsEmpty() {
		return errors.New("bond address is empty")
	}
	if n.Status == Unknown {
		return errors.New("node status cannot be unknown")
	}

	return nil
}

// UpdateStatus change the status of node account, in the mean time update StatusSince field
func (n *NodeAccount) UpdateStatus(status NodeStatus, height int64) {
	if n.Status == status {
		return
	}
	n.Status = status
	n.StatusSince = height
}

// Equals compare two node account, to see whether they are equal
func (n NodeAccount) Equals(n1 NodeAccount) bool {
	if n.NodeAddress.Equals(n1.NodeAddress) &&
		n.PubKeySet.Equals(n1.PubKeySet) &&
		n.ValidatorConsPubKey == n1.ValidatorConsPubKey &&
		n.BondAddress.Equals(n1.BondAddress) &&
		n.Bond.Equal(n1.Bond) &&
		n.Version.Equals(n1.Version) {
		return true
	}
	return false
}

// String implement fmt.Stringer interface
func (n NodeAccount) String() string {
	sb := strings.Builder{}
	sb.WriteString("node:" + n.NodeAddress.String() + "\n")
	sb.WriteString("status:" + n.Status.String() + "\n")
	sb.WriteString("node pubkeys:" + n.PubKeySet.String() + "\n")
	sb.WriteString("validator consensus pub key:" + n.ValidatorConsPubKey + "\n")
	sb.WriteString("bond:" + n.Bond.String() + "\n")
	sb.WriteString("version:" + n.Version.String() + "\n")
	sb.WriteString("bond address:" + n.BondAddress.String() + "\n")
	sb.WriteString("requested to leave:" + strconv.FormatBool(n.RequestedToLeave) + "\n")
	return sb.String()
}

// CalcBondUnits calculate bond
func (n *NodeAccount) CalcBondUnits(height, slashpoints int64) cosmos.Uint {
	// ensure slashpoints is not negative
	slashpoints = int64(math.Max(float64(0), float64(slashpoints)))
	if height < 0 || n.ActiveBlockHeight < 0 || slashpoints < 0 {
		return cosmos.ZeroUint()
	}

	blockCount := height - (n.ActiveBlockHeight + slashpoints)
	if blockCount < 0 { // ensure we're never negative
		blockCount = 0
	}

	return cosmos.NewUint(uint64(blockCount))
}

// TryAddSignerPubKey add a key to node account
func (n *NodeAccount) TryAddSignerPubKey(key common.PubKey) {
	if key.IsEmpty() {
		return
	}
	for _, item := range n.SignerMembership {
		if item.Equals(key) {
			return
		}
	}
	n.SignerMembership = append(n.SignerMembership, key)
}

// TryRemoveSignerPubKey remove the given pubkey from signer membership
func (n *NodeAccount) TryRemoveSignerPubKey(key common.PubKey) {
	if key.IsEmpty() {
		return
	}
	idxToDelete := -1
	for idx, item := range n.SignerMembership {
		if item.Equals(key) {
			idxToDelete = idx
		}
	}
	if idxToDelete != -1 {
		n.SignerMembership = append(n.SignerMembership[:idxToDelete], n.SignerMembership[idxToDelete+1:]...)
	}
}

// NodeAccounts just a list of NodeAccount
type NodeAccounts []NodeAccount

// IsEmpty to check whether the NodeAccounts is empty
func (nas NodeAccounts) IsEmpty() bool {
	return len(nas) == 0
}

// IsNodeKeys validate whether the given account address belongs to an currently active validator
func (nas NodeAccounts) IsNodeKeys(addr cosmos.AccAddress) bool {
	for _, na := range nas {
		if na.Status == Active && addr.Equals(na.NodeAddress) {
			return true
		}
	}
	return false
}

// NodeAccount sort interface , it will sort by StatusSince field, and then by SignerBNBAddress
func (nas NodeAccounts) Less(i, j int) bool {
	if nas[i].StatusSince < nas[j].StatusSince {
		return true
	}
	if nas[i].StatusSince > nas[j].StatusSince {
		return false
	}
	return nas[i].NodeAddress.String() < nas[j].NodeAddress.String()
}

// Len return the number of accounts in it
func (nas NodeAccounts) Len() int { return len(nas) }

// Swap node account
func (nas NodeAccounts) Swap(i, j int) {
	nas[i], nas[j] = nas[j], nas[i]
}

// Contains will check whether the given node account is in the list
func (nas NodeAccounts) Contains(na NodeAccount) bool {
	for _, item := range nas {
		if item.Equals(na) {
			return true
		}
	}
	return false
}
