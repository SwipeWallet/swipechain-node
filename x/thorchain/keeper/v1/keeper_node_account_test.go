package keeperv1

import (
	"github.com/blang/semver"
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
)

type KeeperNodeAccountSuite struct{}

var _ = Suite(&KeeperNodeAccountSuite{})

func (s *KeeperNodeAccountSuite) TestNodeAccount(c *C) {
	ctx, k := setupKeeperForTest(c)
	ctx = ctx.WithBlockHeight(10)

	na1 := GetRandomNodeAccount(NodeActive)
	na2 := GetRandomNodeAccount(NodeStandby)
	c.Assert(k.SetNodeAccount(ctx, na1), IsNil)
	c.Assert(k.SetNodeAccount(ctx, na2), IsNil)
	c.Check(na1.ActiveBlockHeight, Equals, int64(10))
	c.Check(na2.ActiveBlockHeight, Equals, int64(0))

	count, err := k.TotalActiveNodeAccount(ctx)
	c.Assert(err, IsNil)
	c.Check(count, Equals, 1)

	na, err := k.GetNodeAccount(ctx, na1.NodeAddress)
	c.Assert(err, IsNil)
	c.Check(na.Equals(na1), Equals, true)

	na, err = k.GetNodeAccountByPubKey(ctx, na1.PubKeySet.Secp256k1)
	c.Assert(err, IsNil)
	c.Check(na.Equals(na1), Equals, true)

	valCon := "im unique!"
	pubkeys := GetRandomPubKeySet()
	err = k.EnsureNodeKeysUnique(ctx, na1.ValidatorConsPubKey, common.EmptyPubKeySet)
	c.Assert(err, NotNil)
	err = k.EnsureNodeKeysUnique(ctx, "", pubkeys)
	c.Assert(err, NotNil)
	err = k.EnsureNodeKeysUnique(ctx, na1.ValidatorConsPubKey, pubkeys)
	c.Assert(err, NotNil)
	err = k.EnsureNodeKeysUnique(ctx, valCon, na1.PubKeySet)
	c.Assert(err, NotNil)
	err = k.EnsureNodeKeysUnique(ctx, valCon, pubkeys)
	c.Assert(err, IsNil)
	addr := GetRandomBech32Addr()
	na, err = k.GetNodeAccount(ctx, addr)
	c.Assert(err, IsNil)
	c.Assert(na.Status, Equals, NodeUnknown)
	c.Assert(na.ValidatorConsPubKey, Equals, "")
	nodeAccounts, err := k.ListNodeAccountsWithBond(ctx)
	c.Check(err, IsNil)
	c.Check(nodeAccounts.Len() > 0, Equals, true)
}

func (s *KeeperNodeAccountSuite) TestGetMinJoinVersion(c *C) {
	type nodeInfo struct {
		status  NodeStatus
		version semver.Version
	}
	inputs := []struct {
		nodeInfoes            []nodeInfo
		expectedVersion       semver.Version
		expectedActiveVersion semver.Version
	}{
		{
			nodeInfoes: []nodeInfo{
				{
					status:  NodeActive,
					version: semver.MustParse("0.2.0"),
				},
				{
					status:  NodeActive,
					version: semver.MustParse("0.3.0"),
				},
				{
					status:  NodeActive,
					version: semver.MustParse("0.3.0"),
				},
				{
					status:  NodeStandby,
					version: semver.MustParse("0.2.0"),
				},
				{
					status:  NodeStandby,
					version: semver.MustParse("0.2.0"),
				},
			},
			expectedVersion:       semver.MustParse("0.3.0"),
			expectedActiveVersion: semver.MustParse("0.2.0"),
		},
		{
			nodeInfoes: []nodeInfo{
				{
					status:  NodeActive,
					version: semver.MustParse("0.2.0"),
				},
				{
					status:  NodeActive,
					version: semver.MustParse("1.3.0"),
				},
				{
					status:  NodeActive,
					version: semver.MustParse("0.3.0"),
				},
				{
					status:  NodeStandby,
					version: semver.MustParse("0.2.0"),
				},
				{
					status:  NodeStandby,
					version: semver.MustParse("0.2.0"),
				},
			},
			expectedVersion:       semver.MustParse("0.3.0"),
			expectedActiveVersion: semver.MustParse("0.2.0"),
		},
		{
			nodeInfoes: []nodeInfo{
				{
					status:  NodeActive,
					version: semver.MustParse("0.2.0"),
				},
				{
					status:  NodeActive,
					version: semver.MustParse("1.3.0"),
				},
				{
					status:  NodeActive,
					version: semver.MustParse("0.3.0"),
				},
				{
					status:  NodeActive,
					version: semver.MustParse("0.2.0"),
				},
				{
					status:  NodeActive,
					version: semver.MustParse("0.2.0"),
				},
			},
			expectedVersion:       semver.MustParse("0.2.0"),
			expectedActiveVersion: semver.MustParse("0.2.0"),
		},
	}

	for _, item := range inputs {
		ctx, k := setupKeeperForTest(c)
		for _, ni := range item.nodeInfoes {
			na1 := GetRandomNodeAccount(ni.status)
			na1.Version = ni.version
			c.Assert(k.SetNodeAccount(ctx, na1), IsNil)
		}
		c.Check(k.GetMinJoinVersion(ctx).Equals(item.expectedVersion), Equals, true, Commentf("%+v", k.GetMinJoinVersion(ctx)))
		c.Check(k.GetLowestActiveVersion(ctx).Equals(item.expectedActiveVersion), Equals, true)
	}
}

func (s *KeeperNodeAccountSuite) TestNodeAccountSlashPoints(c *C) {
	ctx, k := setupKeeperForTest(c)
	addr := GetRandomBech32Addr()

	pts, err := k.GetNodeAccountSlashPoints(ctx, addr)
	c.Assert(err, IsNil)
	c.Check(pts, Equals, int64(0))

	pts = 5
	k.SetNodeAccountSlashPoints(ctx, addr, pts)
	pts, err = k.GetNodeAccountSlashPoints(ctx, addr)
	c.Assert(err, IsNil)
	c.Check(pts, Equals, int64(5))

	c.Assert(k.IncNodeAccountSlashPoints(ctx, addr, 12), IsNil)
	pts, err = k.GetNodeAccountSlashPoints(ctx, addr)
	c.Assert(err, IsNil)
	c.Check(pts, Equals, int64(17))

	c.Assert(k.DecNodeAccountSlashPoints(ctx, addr, 7), IsNil)
	pts, err = k.GetNodeAccountSlashPoints(ctx, addr)
	c.Assert(err, IsNil)
	c.Check(pts, Equals, int64(10))
	k.ResetNodeAccountSlashPoints(ctx, GetRandomBech32Addr())
}

func (s *KeeperNodeAccountSuite) TestJail(c *C) {
	ctx, k := setupKeeperForTest(c)
	addr := GetRandomBech32Addr()

	jail, err := k.GetNodeAccountJail(ctx, addr)
	c.Assert(err, IsNil)
	c.Check(jail.NodeAddress.Equals(addr), Equals, true)
	c.Check(jail.ReleaseHeight, Equals, int64(0))
	c.Check(jail.Reason, Equals, "")

	// ensure setting it works
	err = k.SetNodeAccountJail(ctx, addr, 50, "foo")
	c.Assert(err, IsNil)
	jail, err = k.GetNodeAccountJail(ctx, addr)
	c.Assert(err, IsNil)
	c.Check(jail.NodeAddress.Equals(addr), Equals, true)
	c.Check(jail.ReleaseHeight, Equals, int64(50))
	c.Check(jail.Reason, Equals, "foo")

	// ensure we won't reduce sentence
	err = k.SetNodeAccountJail(ctx, addr, 20, "bar")
	c.Assert(err, IsNil)
	jail, err = k.GetNodeAccountJail(ctx, addr)
	c.Assert(err, IsNil)
	c.Check(jail.NodeAddress.Equals(addr), Equals, true)
	c.Check(jail.ReleaseHeight, Equals, int64(50))
	c.Check(jail.Reason, Equals, "foo")

	// ensure we can update
	err = k.SetNodeAccountJail(ctx, addr, 70, "bar")
	c.Assert(err, IsNil)
	jail, err = k.GetNodeAccountJail(ctx, addr)
	c.Assert(err, IsNil)
	c.Check(jail.NodeAddress.Equals(addr), Equals, true)
	c.Check(jail.ReleaseHeight, Equals, int64(70))
	c.Check(jail.Reason, Equals, "bar")
}
