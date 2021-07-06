package thorchain

import (
	"errors"

	"github.com/blang/semver"
	. "gopkg.in/check.v1"
)

type ManagersTestSuite struct{}

var _ = Suite(&ManagersTestSuite{})

func (ManagersTestSuite) TestManagers(c *C) {
	ctx, k := setupKeeperForTest(c)
	mgr := NewManagers(k)
	mgr.BeginBlock(ctx)
	ver := semver.MustParse("0.0.1")

	gasMgr, err := GetGasManager(ver)
	c.Assert(gasMgr, IsNil)
	c.Assert(err, NotNil)
	c.Assert(errors.Is(err, errInvalidVersion), Equals, true)

	eventMgr, err := GetEventManager(ver)
	c.Assert(eventMgr, IsNil)
	c.Assert(err, NotNil)
	c.Assert(errors.Is(err, errInvalidVersion), Equals, true)

	txOutStore, err := GetTxOutStore(k, ver, mgr.EventMgr())
	c.Assert(txOutStore, IsNil)
	c.Assert(err, NotNil)
	c.Assert(errors.Is(err, errInvalidVersion), Equals, true)

	vaultMgr, err := GetVaultManager(k, ver, mgr.TxOutStore(), mgr.EventMgr())
	c.Assert(vaultMgr, IsNil)
	c.Assert(err, NotNil)
	c.Assert(errors.Is(err, errInvalidVersion), Equals, true)

	validatorManager, err := GetValidatorManager(k, ver, mgr.VaultMgr(), mgr.TxOutStore(), mgr.EventMgr())
	c.Assert(validatorManager, IsNil)
	c.Assert(err, NotNil)
	c.Assert(errors.Is(err, errInvalidVersion), Equals, true)

	observerMgr, err := GetObserverManager(ver)
	c.Assert(observerMgr, IsNil)
	c.Assert(err, NotNil)
	c.Assert(errors.Is(err, errInvalidVersion), Equals, true)

	swapQueue, err := GetSwapQueue(k, ver)
	c.Assert(swapQueue, IsNil)
	c.Assert(err, NotNil)
	c.Assert(errors.Is(err, errInvalidVersion), Equals, true)

	slasher, err := GetSlasher(k, ver)
	c.Assert(slasher, IsNil)
	c.Assert(err, NotNil)
	c.Assert(errors.Is(err, errInvalidVersion), Equals, true)

	yggMgr, err := GetYggManager(k, ver)
	c.Assert(yggMgr, IsNil)
	c.Assert(err, NotNil)
	c.Assert(errors.Is(err, errInvalidVersion), Equals, true)
}
