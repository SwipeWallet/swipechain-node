package thorchain

import (
	"io/ioutil"

	. "gopkg.in/check.v1"
)

type GenesisTestSuite struct{}

var _ = Suite(&GenesisTestSuite{})

func (GenesisTestSuite) TestGenesis(c *C) {
	SetupConfigForTest()
	genesisState := DefaultGenesisState()
	c.Assert(ValidateGenesis(genesisState), IsNil)
	ctx, k := setupKeeperForTest(c)
	gs := ExportGenesis(ctx, k)
	c.Assert(ValidateGenesis(gs), IsNil)
	content, err := ioutil.ReadFile("../../test/fixtures/genesis/genesis.json")
	c.Assert(err, IsNil)
	c.Assert(content, NotNil)
	ctx, k = setupKeeperForTest(c)
	var state GenesisState
	c.Assert(ModuleCdc.UnmarshalJSON(content, &state), IsNil)
	result := InitGenesis(ctx, k, state)
	c.Assert(result, NotNil)
	gs1 := ExportGenesis(ctx, k)
	c.Assert(len(gs1.Pools) > 0, Equals, true)
}
