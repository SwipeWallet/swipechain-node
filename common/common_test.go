package common

import (
	"testing"

	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
	. "gopkg.in/check.v1"
)

func TestPackage(t *testing.T) { TestingT(t) }

type CommonSuite struct{}

var _ = Suite(&CommonSuite{})

func (s CommonSuite) TestGetShare(c *C) {
	part := cosmos.NewUint(149506590)
	total := cosmos.NewUint(50165561086)
	alloc := cosmos.NewUint(50000000)
	share := GetShare(part, total, alloc)
	c.Assert(share.Equal(cosmos.NewUint(149013)), Equals, true)
}
