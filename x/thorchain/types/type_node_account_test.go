package types

import (
	"encoding/json"
	"sort"
	"strings"

	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
)

type NodeAccountSuite struct{}

var _ = Suite(&NodeAccountSuite{})

func (NodeAccountSuite) TestGetNodeStatus(c *C) {
	input := map[string]NodeStatus{
		"unknown":     Unknown,
		"Unknown":     Unknown,
		"uNknown":     Unknown,
		"WhiteListed": WhiteListed,
		"WHITELISTED": WhiteListed,
		"whitelisted": WhiteListed,
		"Standby":     Standby,
		"standby":     Standby,
		"StanDby":     Standby,
		"active":      Active,
		"Active":      Active,
		"aCtive":      Active,
		"ACTIVE":      Active,
		"disabled":    Disabled,
		"Disabled":    Disabled,
		"disabLed":    Disabled,
		"ready":       Ready,
		"Ready":       Ready,
		"rEady":       Ready,
	}
	for k, v := range input {
		r := GetNodeStatus(k)
		if r != v {
			c.Errorf("expect %s,however THORNode got %s", v, r)
		}
		c.Check(r.String(), Equals, strings.ToLower(k))
		c.Check(v.Valid(), IsNil)
		buf, err := json.Marshal(v)
		c.Assert(err, IsNil)
		c.Assert(len(buf) > 0, Equals, true)
	}
	ns := NodeStatus(255)
	c.Assert(ns.String(), Equals, "")
	c.Check(ns.Valid(), NotNil)
	ns = GetNodeStatus("Whatever")
	c.Assert(ns, Equals, Unknown)
	nodeStatus := Active
	result, err := json.Marshal(nodeStatus)
	c.Check(err, IsNil)
	c.Check(result, NotNil)
	var newNodeStatus NodeStatus
	c.Check(json.Unmarshal(result, &newNodeStatus), IsNil)
	c.Check(newNodeStatus, Equals, nodeStatus)
}

func (NodeAccountSuite) TestNodeAccount(c *C) {
	addr := GetRandomBech32Addr()
	c.Check(addr.Empty(), Equals, false)
	bepConsPubKey := GetRandomBech32ConsensusPubKey()
	nodeAddress := GetRandomBech32Addr()
	bondAddr := GetRandomBNBAddress()
	pubKeys := common.PubKeySet{
		Secp256k1: GetRandomPubKey(),
		Ed25519:   GetRandomPubKey(),
	}

	na := NewNodeAccount(nodeAddress, Active, pubKeys, bepConsPubKey, cosmos.NewUint(common.One), bondAddr, 1)
	c.Assert(na.IsEmpty(), Equals, false)
	c.Assert(na.Valid(), IsNil)
	c.Assert(na.Bond.Uint64(), Equals, uint64(common.One))
	nas := NodeAccounts{
		na,
	}
	c.Assert(nas.IsNodeKeys(addr), Equals, false)
	c.Assert(nas.IsNodeKeys(nodeAddress), Equals, true)
	c.Logf("node account:%s", na)
	naEmpty := NewNodeAccount(cosmos.AccAddress{}, Active, pubKeys, bepConsPubKey, cosmos.NewUint(common.One), bondAddr, 1)
	c.Assert(naEmpty.Valid(), NotNil)
	c.Assert(naEmpty.IsEmpty(), Equals, true)
	invalidBondAddr := NewNodeAccount(cosmos.AccAddress{}, Active, pubKeys, bepConsPubKey, cosmos.NewUint(common.One), "", 1)
	c.Assert(invalidBondAddr.Valid(), NotNil)

	na1 := NewNodeAccount(nodeAddress, Active, pubKeys, bepConsPubKey, cosmos.NewUint(common.One), common.NoAddress, 1)
	c.Check(na1.Valid(), NotNil)

	na2 := NewNodeAccount(nodeAddress, Unknown, pubKeys, bepConsPubKey, cosmos.NewUint(common.One), bondAddr, 1)
	c.Check(na2.Valid(), NotNil)

	na3 := NewNodeAccount(nodeAddress, Active, pubKeys, bepConsPubKey, cosmos.NewUint(common.One), bondAddr, 1)
	c.Check(na3.Equals(na), Equals, true)
	c.Check(na3.Equals(na1), Equals, false)
}

func (NodeAccountSuite) TestNodeAccountsSort(c *C) {
	var accounts NodeAccounts
	for {
		na := GetRandomNodeAccount(Active)
		dup := false
		for _, node := range accounts {
			if na.NodeAddress.Equals(node.NodeAddress) {
				dup = true
			}
		}
		if dup {
			continue
		}
		accounts = append(accounts, na)
		if len(accounts) == 10 {
			break
		}
	}

	sort.Sort(accounts)

	for i, na := range accounts {
		if i == 0 {
			continue
		}
		if na.NodeAddress.String() < accounts[i].NodeAddress.String() {
			c.Errorf("%s should be before %s", na.NodeAddress, accounts[i].NodeAddress)
		}
	}
	c.Check(accounts.IsEmpty(), Equals, false)
	c.Check(accounts.Contains(accounts[0]), Equals, true)
	var emptyNodeAccounts NodeAccounts
	c.Check(emptyNodeAccounts.Contains(accounts[0]), Equals, false)
	c.Check(emptyNodeAccounts.IsEmpty(), Equals, true)
}

func (NodeAccountSuite) TestNodeAccountUpdateStatusAndSort(c *C) {
	var accounts NodeAccounts
	for i := 0; i < 10; i++ {
		na := GetRandomNodeAccount(Active)
		accounts = append(accounts, na)
	}
	isSorted := sort.SliceIsSorted(accounts, func(i, j int) bool {
		return accounts[i].StatusSince < accounts[j].StatusSince
	})
	c.Assert(isSorted, Equals, true)
}

func (NodeAccountSuite) TestTryAddSignerPubKey(c *C) {
	na := NewNodeAccount(GetRandomBech32Addr(), Active, GetRandomPubKeySet(), GetRandomBech32ConsensusPubKey(), cosmos.NewUint(100*common.One), GetRandomBNBAddress(), 1)
	pk := GetRandomPubKey()
	emptyPK := common.EmptyPubKey
	// make sure it get added
	na.TryAddSignerPubKey(pk)
	c.Assert(na.SignerMembership, NotNil)
	c.Assert(na.SignerMembership, HasLen, 1)
	na.TryAddSignerPubKey(emptyPK)
	c.Assert(na.SignerMembership, HasLen, 1)

	// add the same key again should be a noop
	na.TryAddSignerPubKey(pk)
	c.Assert(len(na.SignerMembership), Equals, 1)
	na.TryRemoveSignerPubKey(emptyPK)
	c.Assert(len(na.SignerMembership), Equals, 1)

	na.TryRemoveSignerPubKey(pk)
	c.Assert(na.SignerMembership, HasLen, 0)
}

func (s *NodeAccountSuite) TestCalcNodeRewards(c *C) {
	na := NodeAccount{
		ActiveBlockHeight: 30,
	}
	blocks := na.CalcBondUnits(50, 2)
	c.Check(blocks.Uint64(), Equals, uint64(18))

	na = NodeAccount{
		ActiveBlockHeight: 30,
	}
	blocks = na.CalcBondUnits(50, 100000)
	c.Check(blocks.Uint64(), Equals, uint64(0))

	na = NodeAccount{
		ActiveBlockHeight: 100,
	}
	blocks = na.CalcBondUnits(50, 0)
	c.Check(blocks.Uint64(), Equals, uint64(0))

	na = NodeAccount{
		ActiveBlockHeight: 30,
	}
	blocks = na.CalcBondUnits(-50, 0)
	c.Check(blocks.Uint64(), Equals, uint64(0))

	na = NodeAccount{
		ActiveBlockHeight: -100,
	}
	blocks = na.CalcBondUnits(50, 0)
	c.Check(blocks.Uint64(), Equals, uint64(0), Commentf("%d", blocks.Uint64()))
}
