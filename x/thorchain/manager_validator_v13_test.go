package thorchain

import (
	"github.com/blang/semver"
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
)

type ValidatorMgrV13TestSuite struct{}

var _ = Suite(&ValidatorMgrV13TestSuite{})

func (vts *ValidatorMgrV13TestSuite) SetUpSuite(c *C) {
	SetupConfigForTest()
}

func (vts *ValidatorMgrV13TestSuite) TestSetupValidatorNodes(c *C) {
	ctx, k := setupKeeperForTest(c)
	ctx = ctx.WithBlockHeight(1)
	mgr := NewDummyMgr()
	vMgr := newValidatorMgrV13(k, mgr.VaultMgr(), mgr.TxOutStore(), mgr.EventMgr())
	c.Assert(vMgr, NotNil)
	ver := constants.SWVersion
	constAccessor := constants.GetConstantValues(ver)
	err := vMgr.setupValidatorNodes(ctx, 0, constAccessor)
	c.Assert(err, IsNil)

	// no node accounts at all
	err = vMgr.setupValidatorNodes(ctx, 1, constAccessor)
	c.Assert(err, NotNil)

	activeNode := GetRandomNodeAccount(NodeActive)
	c.Assert(k.SetNodeAccount(ctx, activeNode), IsNil)

	err = vMgr.setupValidatorNodes(ctx, 1, constAccessor)
	c.Assert(err, IsNil)

	readyNode := GetRandomNodeAccount(NodeReady)
	c.Assert(k.SetNodeAccount(ctx, readyNode), IsNil)

	// one active node and one ready node on start up
	// it should take both of the node as active
	vMgr1 := newValidatorMgrV13(k, mgr.VaultMgr(), mgr.TxOutStore(), mgr.EventMgr())

	c.Assert(vMgr1.BeginBlock(ctx, constAccessor), IsNil)
	activeNodes, err := k.ListActiveNodeAccounts(ctx)
	c.Assert(err, IsNil)
	c.Logf("active nodes:%s", activeNodes)
	c.Assert(len(activeNodes) == 2, Equals, true)

	activeNode1 := GetRandomNodeAccount(NodeActive)
	activeNode2 := GetRandomNodeAccount(NodeActive)
	c.Assert(k.SetNodeAccount(ctx, activeNode1), IsNil)
	c.Assert(k.SetNodeAccount(ctx, activeNode2), IsNil)

	// three active nodes and 1 ready nodes, it should take them all
	vMgr2 := newValidatorMgrV13(k, mgr.VaultMgr(), mgr.TxOutStore(), mgr.EventMgr())
	vMgr2.BeginBlock(ctx, constAccessor)

	activeNodes1, err := k.ListActiveNodeAccounts(ctx)
	c.Assert(err, IsNil)
	c.Assert(len(activeNodes1) == 4, Equals, true)
}

func (vts *ValidatorMgrV13TestSuite) TestRagnarokForChaosnet(c *C) {
	ctx, k := setupKeeperForTest(c)
	mgr := NewManagers(k)
	c.Assert(mgr.BeginBlock(ctx), IsNil)
	vMgr := newValidatorMgrV13(k, mgr.VaultMgr(), mgr.TxOutStore(), mgr.EventMgr())

	constAccessor := constants.NewDummyConstants(map[constants.ConstantName]int64{
		constants.DesireValidatorSet:            12,
		constants.ArtificialRagnarokBlockHeight: 1024,
		constants.BadValidatorRate:              256,
		constants.OldValidatorRate:              256,
		constants.MinimumNodesForBFT:            4,
		constants.RotatePerBlockHeight:          256,
		constants.RotateRetryBlocks:             720,
	}, map[constants.ConstantName]bool{
		constants.StrictBondStakeRatio: false,
	}, map[constants.ConstantName]string{})
	for i := 0; i < 12; i++ {
		node := GetRandomNodeAccount(NodeReady)
		c.Assert(k.SetNodeAccount(ctx, node), IsNil)
	}
	c.Assert(vMgr.setupValidatorNodes(ctx, 1, constAccessor), IsNil)
	nodeAccounts, err := k.ListNodeAccountsByStatus(ctx, NodeActive)
	c.Assert(err, IsNil)
	c.Assert(len(nodeAccounts), Equals, 12)

	// trigger ragnarok
	ctx = ctx.WithBlockHeight(1024)
	c.Assert(vMgr.BeginBlock(ctx, constAccessor), IsNil)
	vault := NewVault(common.BlockHeight(ctx), ActiveVault, AsgardVault, GetRandomPubKey(), common.Chains{common.BNBChain})
	for _, item := range nodeAccounts {
		vault.Membership = append(vault.Membership, item.PubKeySet.Secp256k1)
	}
	c.Assert(k.SetVault(ctx, vault), IsNil)
	updates := vMgr.EndBlock(ctx, mgr, constAccessor)
	// ragnarok , no one leaves
	c.Assert(updates, IsNil)
	ragnarokHeight, err := k.GetRagnarokBlockHeight(ctx)
	c.Assert(err, IsNil)
	c.Assert(ragnarokHeight == 1024, Equals, true, Commentf("%d == %d", ragnarokHeight, 1024))
}

func (vts *ValidatorMgrV13TestSuite) TestLowerVersion(c *C) {
	ctx, k := setupKeeperForTest(c)
	ctx = ctx.WithBlockHeight(1440)

	mgr := NewManagers(k)
	c.Assert(mgr.BeginBlock(ctx), IsNil)
	vMgr := newValidatorMgrV13(k, mgr.VaultMgr(), mgr.TxOutStore(), mgr.EventMgr())
	c.Assert(vMgr, NotNil)
	c.Assert(vMgr.markLowerVersion(ctx, 360), IsNil)

	for i := 0; i < 5; i++ {
		activeNode := GetRandomNodeAccount(NodeActive)
		activeNode.Version = semver.MustParse("0.5.0")
		c.Assert(k.SetNodeAccount(ctx, activeNode), IsNil)
	}
	activeNode1 := GetRandomNodeAccount(NodeActive)
	activeNode1.Version = semver.MustParse("0.4.0")
	c.Assert(k.SetNodeAccount(ctx, activeNode1), IsNil)

	c.Assert(vMgr.markLowerVersion(ctx, 360), IsNil)
	na, err := k.GetNodeAccount(ctx, activeNode1.NodeAddress)
	c.Assert(err, IsNil)
	c.Assert(na.LeaveHeight, Equals, int64(1440))
}

func (vts *ValidatorMgrV13TestSuite) TestBadActors(c *C) {
	ctx, k := setupKeeperForTest(c)
	ctx = ctx.WithBlockHeight(1000)

	mgr := NewManagers(k)
	c.Assert(mgr.BeginBlock(ctx), IsNil)
	vMgr := newValidatorMgrV13(k, mgr.VaultMgr(), mgr.TxOutStore(), mgr.EventMgr())
	c.Assert(vMgr, NotNil)

	// no bad actors with active node accounts
	nas, err := vMgr.findBadActors(ctx)
	c.Assert(err, IsNil)
	c.Assert(nas, HasLen, 0)

	activeNode := GetRandomNodeAccount(NodeActive)
	c.Assert(k.SetNodeAccount(ctx, activeNode), IsNil)

	// no bad actors with active node accounts with no slash points
	nas, err = vMgr.findBadActors(ctx)
	c.Assert(err, IsNil)
	c.Assert(nas, HasLen, 0)

	activeNode = GetRandomNodeAccount(NodeActive)
	k.SetNodeAccountSlashPoints(ctx, activeNode.NodeAddress, 25)
	c.Assert(k.SetNodeAccount(ctx, activeNode), IsNil)
	activeNode = GetRandomNodeAccount(NodeActive)
	k.SetNodeAccountSlashPoints(ctx, activeNode.NodeAddress, 50)
	c.Assert(k.SetNodeAccount(ctx, activeNode), IsNil)

	// finds the worse actor
	nas, err = vMgr.findBadActors(ctx)
	c.Assert(err, IsNil)
	c.Assert(nas, HasLen, 1)
	c.Check(nas[0].NodeAddress.Equals(activeNode.NodeAddress), Equals, true)

	// create really bad actors (crossing the redline)
	bad1 := GetRandomNodeAccount(NodeActive)
	k.SetNodeAccountSlashPoints(ctx, bad1.NodeAddress, 1000)
	c.Assert(k.SetNodeAccount(ctx, bad1), IsNil)
	bad2 := GetRandomNodeAccount(NodeActive)
	k.SetNodeAccountSlashPoints(ctx, bad2.NodeAddress, 1000)
	c.Assert(k.SetNodeAccount(ctx, bad2), IsNil)

	nas, err = vMgr.findBadActors(ctx)
	c.Assert(err, IsNil)
	c.Assert(nas, HasLen, 2, Commentf("%d", len(nas)))

	// inconsistent order, workaround
	var count int
	for _, bad := range nas {
		if bad.Equals(bad1) || bad.Equals(bad2) {
			count += 1
		}
	}
	c.Check(count, Equals, 2)
}

func (vts *ValidatorMgrV13TestSuite) TestRagnarokBond(c *C) {
	ctx, k := setupKeeperForTest(c)
	ctx = ctx.WithBlockHeight(1)
	ver := constants.SWVersion

	mgr := NewDummyMgr()
	vMgr := newValidatorMgrV13(k, mgr.VaultMgr(), mgr.TxOutStore(), mgr.EventMgr())
	c.Assert(vMgr, NotNil)

	constAccessor := constants.GetConstantValues(ver)
	err := vMgr.setupValidatorNodes(ctx, 0, constAccessor)
	c.Assert(err, IsNil)

	activeNode := GetRandomNodeAccount(NodeActive)
	activeNode.Bond = cosmos.NewUint(100)
	c.Assert(k.SetNodeAccount(ctx, activeNode), IsNil)

	disabledNode := GetRandomNodeAccount(NodeDisabled)
	disabledNode.Bond = cosmos.ZeroUint()
	c.Assert(k.SetNodeAccount(ctx, disabledNode), IsNil)

	// no unbonding for first 10
	c.Assert(vMgr.ragnarokBond(ctx, 1, mgr), IsNil)
	activeNode, err = k.GetNodeAccount(ctx, activeNode.NodeAddress)
	c.Assert(err, IsNil)
	c.Check(activeNode.Bond.Equal(cosmos.NewUint(100)), Equals, true)

	c.Assert(vMgr.ragnarokBond(ctx, 11, mgr), IsNil)
	activeNode, err = k.GetNodeAccount(ctx, activeNode.NodeAddress)
	c.Assert(err, IsNil)
	c.Check(activeNode.Bond.Equal(cosmos.NewUint(90)), Equals, true)
	items, err := mgr.TxOutStore().GetOutboundItems(ctx)
	c.Assert(err, IsNil)
	if common.RuneAsset().Chain.Equals(common.THORChain) {
		c.Check(items, HasLen, 0, Commentf("Len %d", items))
	} else {
		c.Check(items, HasLen, 1, Commentf("Len %d", items))
	}
	mgr.TxOutStore().ClearOutboundItems(ctx)

	c.Assert(vMgr.ragnarokBond(ctx, 12, mgr), IsNil)
	activeNode, err = k.GetNodeAccount(ctx, activeNode.NodeAddress)
	c.Assert(err, IsNil)
	c.Check(activeNode.Bond.Equal(cosmos.NewUint(72)), Equals, true)
	items, err = mgr.TxOutStore().GetOutboundItems(ctx)
	c.Assert(err, IsNil)
	if common.RuneAsset().Chain.Equals(common.THORChain) {
		c.Check(items, HasLen, 0, Commentf("Len %d", items))
	} else {
		c.Check(items, HasLen, 1, Commentf("Len %d", items))
	}
}

func (vts *ValidatorMgrV13TestSuite) TestGetChangedNodes(c *C) {
	ctx, k := setupKeeperForTest(c)
	ctx = ctx.WithBlockHeight(1)
	ver := constants.SWVersion

	mgr := NewDummyMgr()
	vMgr := newValidatorMgrV13(k, mgr.VaultMgr(), mgr.TxOutStore(), mgr.EventMgr())
	c.Assert(vMgr, NotNil)

	constAccessor := constants.GetConstantValues(ver)
	err := vMgr.setupValidatorNodes(ctx, 0, constAccessor)
	c.Assert(err, IsNil)

	activeNode := GetRandomNodeAccount(NodeActive)
	activeNode.Bond = cosmos.NewUint(100)
	activeNode.ForcedToLeave = true
	c.Assert(k.SetNodeAccount(ctx, activeNode), IsNil)

	disabledNode := GetRandomNodeAccount(NodeDisabled)
	disabledNode.Bond = cosmos.ZeroUint()
	c.Assert(k.SetNodeAccount(ctx, disabledNode), IsNil)

	vault := NewVault(common.BlockHeight(ctx), ActiveVault, AsgardVault, GetRandomPubKey(), common.Chains{common.BNBChain})
	vault.Membership = append(vault.Membership, activeNode.PubKeySet.Secp256k1)
	c.Assert(k.SetVault(ctx, vault), IsNil)

	newNodes, removedNodes, err := vMgr.getChangedNodes(ctx, NodeAccounts{activeNode})
	c.Assert(err, IsNil)
	c.Assert(newNodes, HasLen, 0)
	c.Assert(removedNodes, HasLen, 1)
}

func (vts *ValidatorMgrV13TestSuite) TestFindCounToRemove(c *C) {
	// remove one
	c.Check(findCountToRemove(0, NodeAccounts{
		NodeAccount{LeaveHeight: 12},
		NodeAccount{},
		NodeAccount{},
		NodeAccount{},
		NodeAccount{},
	}), Equals, 1)

	// don't remove one
	c.Check(findCountToRemove(0, NodeAccounts{
		NodeAccount{LeaveHeight: 12},
		NodeAccount{LeaveHeight: 12},
		NodeAccount{},
		NodeAccount{},
	}), Equals, 0)

	// remove one because of request to leave
	c.Check(findCountToRemove(0, NodeAccounts{
		NodeAccount{LeaveHeight: 12, RequestedToLeave: true},
		NodeAccount{},
		NodeAccount{},
		NodeAccount{},
	}), Equals, 1)

	// remove one because of banned
	c.Check(findCountToRemove(0, NodeAccounts{
		NodeAccount{LeaveHeight: 12, ForcedToLeave: true},
		NodeAccount{},
		NodeAccount{},
		NodeAccount{},
	}), Equals, 1)

	// don't remove more than 1/3rd of node accounts
	c.Check(findCountToRemove(0, NodeAccounts{
		NodeAccount{LeaveHeight: 12},
		NodeAccount{LeaveHeight: 12},
		NodeAccount{LeaveHeight: 12},
		NodeAccount{LeaveHeight: 12},
		NodeAccount{LeaveHeight: 12},
		NodeAccount{LeaveHeight: 12},
		NodeAccount{LeaveHeight: 12},
		NodeAccount{LeaveHeight: 12},
		NodeAccount{LeaveHeight: 12},
		NodeAccount{LeaveHeight: 12},
		NodeAccount{LeaveHeight: 12},
		NodeAccount{LeaveHeight: 12},
	}), Equals, 3)
}

func (vts *ValidatorMgrV13TestSuite) TestFindMaxAbleToLeave(c *C) {
	c.Check(findMaxAbleToLeave(-1), Equals, 0)
	c.Check(findMaxAbleToLeave(0), Equals, 0)
	c.Check(findMaxAbleToLeave(1), Equals, 0)
	c.Check(findMaxAbleToLeave(2), Equals, 0)
	c.Check(findMaxAbleToLeave(3), Equals, 0)
	c.Check(findMaxAbleToLeave(4), Equals, 0)

	c.Check(findMaxAbleToLeave(5), Equals, 1)
	c.Check(findMaxAbleToLeave(6), Equals, 1)
	c.Check(findMaxAbleToLeave(7), Equals, 2)
	c.Check(findMaxAbleToLeave(8), Equals, 2)
	c.Check(findMaxAbleToLeave(9), Equals, 2)
	c.Check(findMaxAbleToLeave(10), Equals, 3)
	c.Check(findMaxAbleToLeave(11), Equals, 3)
	c.Check(findMaxAbleToLeave(12), Equals, 3)
}
