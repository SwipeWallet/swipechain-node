package common

import (
	. "gopkg.in/check.v1"

	"github.com/cosmos/cosmos-sdk/store"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	dbm "github.com/tendermint/tm-db"

	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
)

type BlockHeightSuite struct{}

var _ = Suite(&BlockHeightSuite{})

func (BlockHeightSuite) TestBlockHeight(c *C) {
	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	ctx := cosmos.NewContext(ms, abci.Header{ChainID: "thorchain"}, false, log.NewNopLogger())
	ctx = ctx.WithBlockHeight(18)

	c.Assert(BlockHeight(ctx), Equals, int64(18))
}
