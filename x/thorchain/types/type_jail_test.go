package types

import (
	. "gopkg.in/check.v1"

	"github.com/cosmos/cosmos-sdk/store"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/params"
	"github.com/cosmos/cosmos-sdk/x/supply"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	dbm "github.com/tendermint/tm-db"
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
)

type JailSuite struct{}

var _ = Suite(&JailSuite{})

func (s JailSuite) TestNewJail(c *C) {
	addr := GetRandomBech32Addr()
	jail := NewJail(addr)
	c.Check(jail.NodeAddress.Equals(addr), Equals, true)
	c.Check(jail.ReleaseHeight, Equals, int64(0))
	c.Check(jail.Reason, Equals, "")
}

func (s JailSuite) TestIsJailed(c *C) {
	addr := GetRandomBech32Addr()
	jail := NewJail(addr)

	keyAcc := cosmos.NewKVStoreKey(auth.StoreKey)
	keyParams := cosmos.NewKVStoreKey(params.StoreKey)
	tkeyParams := cosmos.NewTransientStoreKey(params.TStoreKey)
	keySupply := cosmos.NewKVStoreKey(supply.StoreKey)

	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(keyAcc, cosmos.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keySupply, cosmos.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyParams, cosmos.StoreTypeIAVL, db)
	ms.MountStoreWithDB(cosmos.NewKVStoreKey("thorchain"), cosmos.StoreTypeIAVL, db)
	ms.MountStoreWithDB(tkeyParams, cosmos.StoreTypeTransient, db)
	err := ms.LoadLatestVersion()
	c.Assert(err, IsNil)

	ctx := cosmos.NewContext(ms, abci.Header{ChainID: "thorchain"}, false, log.NewNopLogger())
	ctx = ctx.WithBlockHeight(100)

	c.Check(jail.IsJailed(ctx), Equals, false)
	jail.ReleaseHeight = 100
	c.Check(jail.IsJailed(ctx), Equals, false)
	jail.ReleaseHeight = 101
	c.Check(jail.IsJailed(ctx), Equals, true)
}
