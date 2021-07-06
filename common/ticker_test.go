package common

import (
	. "gopkg.in/check.v1"
)

type TickerSuite struct{}

var _ = Suite(&TickerSuite{})

func (s TickerSuite) TestTicker(c *C) {
	runeTicker, err := NewTicker("rune")
	c.Assert(err, IsNil)
	bnbTicker, err := NewTicker("bnb")
	c.Assert(err, IsNil)
	c.Check(runeTicker.IsEmpty(), Equals, false)
	c.Check(runeTicker.Equals(RuneTicker), Equals, true)
	c.Check(bnbTicker.Equals(RuneTicker), Equals, false)
	c.Check(runeTicker.String(), Equals, "RUNE")

	tomobTicker, err := NewTicker("TOMOB-1E1")
	c.Assert(err, IsNil)
	c.Assert(tomobTicker.String(), Equals, "TOMOB-1E1")
	_, err = NewTicker("t") // too short
	c.Assert(err, NotNil)

	maxCharacterTicker, err := NewTicker("TICKER789-XXX")
	c.Assert(err, IsNil)
	c.Assert(maxCharacterTicker.IsEmpty(), Equals, false)
	_, err = NewTicker("too long of a ticker") // too long
	c.Assert(err, NotNil)
}
