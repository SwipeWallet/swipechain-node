package keeperv1

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/blang/semver"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
)

// TotalActiveNodeAccount count the number of active node account
func (k KVStore) TotalActiveNodeAccount(ctx cosmos.Context) (int, error) {
	activeNodes, err := k.ListActiveNodeAccounts(ctx)
	return len(activeNodes), err
}

// ListNodeAccountsWithBond - gets a list of all node accounts that have bond
// Note: the order of node account in the result is not defined
func (k KVStore) ListNodeAccountsWithBond(ctx cosmos.Context) (NodeAccounts, error) {
	nodeAccounts := make(NodeAccounts, 0)
	naIterator := k.GetNodeAccountIterator(ctx)
	defer naIterator.Close()
	for ; naIterator.Valid(); naIterator.Next() {
		var na NodeAccount
		if err := k.cdc.UnmarshalBinaryBare(naIterator.Value(), &na); err != nil {
			return nodeAccounts, dbError(ctx, "Unmarshal: node account", err)
		}
		if !na.Bond.IsZero() {
			nodeAccounts = append(nodeAccounts, na)
		}
	}
	return nodeAccounts, nil
}

// ListNodeAccountsByStatus - get a list of node accounts with the given status
func (k KVStore) ListNodeAccountsByStatus(ctx cosmos.Context, status NodeStatus) (NodeAccounts, error) {
	nodeAccounts := make(NodeAccounts, 0)
	naIterator := k.GetNodeAccountIterator(ctx)
	defer naIterator.Close()
	for ; naIterator.Valid(); naIterator.Next() {
		var na NodeAccount
		if err := k.cdc.UnmarshalBinaryBare(naIterator.Value(), &na); err != nil {
			return nodeAccounts, dbError(ctx, "Unmarshal: node account", err)
		}
		if na.Status == status {
			nodeAccounts = append(nodeAccounts, na)
		}
	}
	return nodeAccounts, nil
}

// ListActiveNodeAccounts - get a list of active node accounts
func (k KVStore) ListActiveNodeAccounts(ctx cosmos.Context) (NodeAccounts, error) {
	return k.ListNodeAccountsByStatus(ctx, NodeActive)
}

// GetMinJoinVersion - get min version to join. Min version is the most popular version
func (k KVStore) GetMinJoinVersion(ctx cosmos.Context) semver.Version {
	type tmpVersionInfo struct {
		version semver.Version
		count   int
	}
	vCount := make(map[string]tmpVersionInfo, 0)
	nodes, err := k.ListActiveNodeAccounts(ctx)
	if err != nil {
		_ = dbError(ctx, "Unable to list active node accounts", err)
		return semver.Version{}
	}
	sort.SliceStable(nodes, func(i, j int) bool {
		return nodes[i].Version.LT(nodes[j].Version)
	})
	for _, na := range nodes {
		v, ok := vCount[na.Version.String()]
		if ok {
			v.count = v.count + 1
			vCount[na.Version.String()] = v
		} else {
			vCount[na.Version.String()] = tmpVersionInfo{
				version: na.Version,
				count:   1,
			}
		}
		// assume all versions are backward compatible
		for k, v := range vCount {
			if v.version.LT(na.Version) {
				v.count = v.count + 1
				vCount[k] = v
			}
		}
	}
	totalCount := len(nodes)
	version := semver.Version{}

	for _, info := range vCount {
		// skip those version that doesn't have majority
		if !HasSuperMajority(info.count, totalCount) {
			continue
		}
		if info.version.GT(version) {
			version = info.version
		}

	}
	return version
}

// GetMinJoinVersion - get min version to join. Min version is the most popular version
func (k KVStore) GetMinJoinVersionV1(ctx cosmos.Context) semver.Version {
	type tmpVersionInfo struct {
		version semver.Version
		count   int
	}
	vCount := make(map[string]tmpVersionInfo, 0)
	nodes, err := k.ListActiveNodeAccounts(ctx)
	if err != nil {
		_ = dbError(ctx, "Unable to list active node accounts", err)
		return semver.Version{}
	}
	sort.SliceStable(nodes, func(i, j int) bool {
		return nodes[i].Version.LT(nodes[j].Version)
	})
	for _, na := range nodes {
		v, ok := vCount[na.Version.String()]
		if ok {
			v.count = v.count + 1
			vCount[na.Version.String()] = v
		} else {
			vCount[na.Version.String()] = tmpVersionInfo{
				version: na.Version,
				count:   1,
			}
		}
		// assume all versions are backward compatible
		for k, v := range vCount {
			if v.version.LT(na.Version) {
				v.count = v.count + 1
				vCount[k] = v
			}
		}
	}
	totalCount := len(nodes)
	version := semver.Version{}

	for _, info := range vCount {
		// skip those version that doesn't have majority
		if !HasSuperMajorityV13(info.count, totalCount) {
			continue
		}
		if info.version.GT(version) {
			version = info.version
		}

	}
	return version
}

// GetLowestActiveVersion - get version number of lowest active node
func (k KVStore) GetLowestActiveVersion(ctx cosmos.Context) semver.Version {
	nodes, err := k.ListActiveNodeAccounts(ctx)
	if err != nil {
		_ = dbError(ctx, "Unable to list active node accounts", err)
		return constants.SWVersion
	}
	if len(nodes) > 0 {
		version := nodes[0].Version
		for _, na := range nodes {
			if na.Version.LT(version) {
				version = na.Version
			}
		}
		return version
	}
	return constants.SWVersion
}

// GetNodeAccount try to get node account with the given address from db
func (k KVStore) GetNodeAccount(ctx cosmos.Context, addr cosmos.AccAddress) (NodeAccount, error) {
	emptyPubKeySet := common.PubKeySet{
		Secp256k1: common.EmptyPubKey,
		Ed25519:   common.EmptyPubKey,
	}
	record := NewNodeAccount(addr, NodeUnknown, emptyPubKeySet, "", cosmos.ZeroUint(), "", common.BlockHeight(ctx))
	_, err := k.get(ctx, k.GetKey(ctx, prefixNodeAccount, addr.String()), &record)
	return record, err
}

// GetNodeAccountByPubKey try to get node account with the given pubkey from db
func (k KVStore) GetNodeAccountByPubKey(ctx cosmos.Context, pk common.PubKey) (NodeAccount, error) {
	addr, err := pk.GetThorAddress()
	if err != nil {
		return NodeAccount{}, err
	}
	return k.GetNodeAccount(ctx, addr)
}

// SetNodeAccount save the given node account into data store
func (k KVStore) SetNodeAccount(ctx cosmos.Context, na NodeAccount) error {
	if na.IsEmpty() {
		return nil
	}
	if na.Status == NodeActive {
		if na.ActiveBlockHeight == 0 {
			// the na is active, and does not have a block height when they
			// became active. This must be the first block they are active, so
			// THORNode will set it now.
			na.ActiveBlockHeight = common.BlockHeight(ctx)
			k.ResetNodeAccountSlashPoints(ctx, na.NodeAddress) // reset slash points
		}
	}

	k.set(ctx, k.GetKey(ctx, prefixNodeAccount, na.NodeAddress.String()), na)

	return nil
}

// EnsureNodeKeysUnique check the given consensus pubkey and pubkey set against all the the node account
// return an error when it is overlap with any existing account
func (k KVStore) EnsureNodeKeysUnique(ctx cosmos.Context, consensusPubKey string, pubKeys common.PubKeySet) error {
	iter := k.GetNodeAccountIterator(ctx)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var na NodeAccount
		if err := k.cdc.UnmarshalBinaryBare(iter.Value(), &na); err != nil {
			return dbError(ctx, "Unmarshal: node account", err)
		}
		if strings.EqualFold("", consensusPubKey) {
			return dbError(ctx, "", errors.New("Validator Consensus Key cannot be empty"))
		}
		if na.ValidatorConsPubKey == consensusPubKey {
			return dbError(ctx, "", fmt.Errorf("%s already exist", na.ValidatorConsPubKey))
		}
		if pubKeys.Equals(common.EmptyPubKeySet) {
			return dbError(ctx, "", errors.New("PubKeySet cannot be empty"))
		}
		if na.PubKeySet.Equals(pubKeys) {
			return dbError(ctx, "", fmt.Errorf("%s already exist", pubKeys))
		}
	}

	return nil
}

// GetNodeAccountIterator iterate node account
func (k KVStore) GetNodeAccountIterator(ctx cosmos.Context) cosmos.Iterator {
	return k.getIterator(ctx, prefixNodeAccount)
}

// GetNodeAccountSlashPoints - get the slash points associated with the given
// node address
func (k KVStore) GetNodeAccountSlashPoints(ctx cosmos.Context, addr cosmos.AccAddress) (int64, error) {
	record := int64(0)
	_, err := k.get(ctx, k.GetKey(ctx, prefixNodeSlashPoints, addr.String()), &record)
	return record, err
}

// SetNodeAccountSlashPoints - set the slash points associated with the given
// node address and uint
func (k KVStore) SetNodeAccountSlashPoints(ctx cosmos.Context, addr cosmos.AccAddress, pts int64) {
	// make sure slash point doesn't go to negative
	if pts < 0 {
		return
	}
	k.set(ctx, k.GetKey(ctx, prefixNodeSlashPoints, addr.String()), pts)
}

// ResetNodeAccountSlashPoints - reset the slash points to zero for associated
// with the given node address
func (k KVStore) ResetNodeAccountSlashPoints(ctx cosmos.Context, addr cosmos.AccAddress) {
	k.del(ctx, k.GetKey(ctx, prefixNodeSlashPoints, addr.String()))
}

// IncNodeAccountSlashPoints - increments the slash points associated with the
// given node address and uint
func (k KVStore) IncNodeAccountSlashPoints(ctx cosmos.Context, addr cosmos.AccAddress, pts int64) error {
	current, err := k.GetNodeAccountSlashPoints(ctx, addr)
	if err != nil {
		return err
	}
	k.SetNodeAccountSlashPoints(ctx, addr, current+pts)
	return nil
}

// DecNodeAccountSlashPoints - decrements the slash points associated with the
// given node address and uint
func (k KVStore) DecNodeAccountSlashPoints(ctx cosmos.Context, addr cosmos.AccAddress, pts int64) error {
	current, err := k.GetNodeAccountSlashPoints(ctx, addr)
	if err != nil {
		return err
	}
	k.SetNodeAccountSlashPoints(ctx, addr, current-pts)
	return nil
}

// GetNodeAccountJail - gets jail details for a given node address
func (k KVStore) GetNodeAccountJail(ctx cosmos.Context, addr cosmos.AccAddress) (Jail, error) {
	record := NewJail(addr)
	_, err := k.get(ctx, k.GetKey(ctx, prefixNodeJail, addr.String()), &record)
	return record, err
}

// SetNodeAccountJail - update the jail details of a node account
func (k KVStore) SetNodeAccountJail(ctx cosmos.Context, addr cosmos.AccAddress, height int64, reason string) error {
	jail, err := k.GetNodeAccountJail(ctx, addr)
	if err != nil {
		return err
	}
	// never reduce sentence
	if jail.ReleaseHeight > height {
		return nil
	}
	jail.ReleaseHeight = height
	jail.Reason = reason

	k.set(ctx, k.GetKey(ctx, prefixNodeJail, addr.String()), jail)
	return nil
}
