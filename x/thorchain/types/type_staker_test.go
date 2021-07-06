package types

import (
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
)

type StakerSuite struct{}

var _ = Suite(&StakerSuite{})

func (StakerSuite) TestStaker(c *C) {
	staker := Staker{
		Asset:           common.BNBAsset,
		RuneAddress:     GetRandomBNBAddress(),
		AssetAddress:    GetRandomBTCAddress(),
		LastStakeHeight: 12,
	}
	c.Check(staker.Valid(), IsNil)
	c.Check(len(staker.Key()) > 0, Equals, true)
	staker1 := Staker{
		Asset:           common.BNBAsset,
		RuneAddress:     GetRandomBNBAddress(),
		AssetAddress:    GetRandomBTCAddress(),
		LastStakeHeight: 0,
	}
	c.Check(staker1.Valid(), NotNil)

	staker2 := Staker{
		Asset:           common.BNBAsset,
		RuneAddress:     common.NoAddress,
		AssetAddress:    GetRandomBTCAddress(),
		LastStakeHeight: 100,
	}
	c.Check(staker2.Valid(), NotNil)

	staker3 := Staker{
		Asset:           common.BNBAsset,
		RuneAddress:     GetRandomBNBAddress(),
		AssetAddress:    common.NoAddress,
		LastStakeHeight: 100,
	}
	c.Check(staker3.Valid(), NotNil)
}
