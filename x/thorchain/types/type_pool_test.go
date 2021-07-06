package types

import (
	"encoding/json"

	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

type PoolTestSuite struct{}

var _ = Suite(&PoolTestSuite{})

func (PoolTestSuite) TestPool(c *C) {
	p := NewPool()
	c.Check(p.IsEmpty(), Equals, true)
	p.Asset = common.BNBAsset
	c.Check(p.IsEmpty(), Equals, false)
	p.BalanceRune = cosmos.NewUint(100 * common.One)
	p.BalanceAsset = cosmos.NewUint(50 * common.One)
	c.Check(p.AssetValueInRune(cosmos.NewUint(25*common.One)).Equal(cosmos.NewUint(50*common.One)), Equals, true)
	c.Check(p.RuneValueInAsset(cosmos.NewUint(50*common.One)).Equal(cosmos.NewUint(25*common.One)), Equals, true)
	c.Log(p.String())

	signer := GetRandomBech32Addr()
	bnbAddress := GetRandomBNBAddress()
	txID := GetRandomTxHash()

	tx := common.NewTx(
		txID,
		GetRandomBNBAddress(),
		GetRandomBNBAddress(),
		common.Coins{
			common.NewCoin(common.BNBAsset, cosmos.NewUint(1)),
		},
		BNBGasFeeSingleton,
		"",
	)
	m := NewMsgSwap(tx, common.BNBAsset, bnbAddress, cosmos.NewUint(2), signer)

	c.Check(p.EnsureValidPoolStatus(m), IsNil)
	msgNoop := NewMsgNoOp(GetRandomObservedTx(), signer)
	c.Check(p.EnsureValidPoolStatus(msgNoop), IsNil)
	p.Status = Enabled
	c.Check(p.EnsureValidPoolStatus(m), IsNil)
	p.Status = PoolStatus(100)
	c.Check(p.EnsureValidPoolStatus(msgNoop), NotNil)

	p.Status = Suspended
	c.Check(p.EnsureValidPoolStatus(msgNoop), NotNil)
	p1 := NewPool()
	c.Check(p1.Valid(), NotNil)
	p1.Asset = common.BNBAsset
	c.Check(p1.AssetValueInRune(cosmos.NewUint(100)).Uint64(), Equals, cosmos.ZeroUint().Uint64())
	c.Check(p1.RuneValueInAsset(cosmos.NewUint(100)).Uint64(), Equals, cosmos.ZeroUint().Uint64())
	p1.BalanceRune = cosmos.NewUint(100 * common.One)
	p1.BalanceAsset = cosmos.NewUint(50 * common.One)
	c.Check(p1.Valid(), IsNil)

	c.Check(p1.IsEnabled(), Equals, true)

	// When Pool is in bootstrap mode , it can't swap
	p2 := NewPool()
	p2.Status = Bootstrap
	msgSwap := NewMsgSwap(GetRandomTx(), common.BNBAsset, GetRandomBNBAddress(), cosmos.NewUint(1000), GetRandomBech32Addr())
	c.Check(p2.EnsureValidPoolStatus(msgSwap), NotNil)
	c.Check(p2.EnsureValidPoolStatus(msgNoop), IsNil)
}

func (PoolTestSuite) TestPoolStatus(c *C) {
	inputs := []string{
		"enabled", "bootstrap", "suspended", "whatever",
	}
	for _, item := range inputs {
		ps := GetPoolStatus(item)
		c.Assert(ps.Valid(), IsNil)
	}
	var ps PoolStatus
	err := json.Unmarshal([]byte(`"Enabled"`), &ps)
	c.Assert(err, IsNil)
	c.Check(ps == Enabled, Equals, true)
	err = json.Unmarshal([]byte(`{asdf}`), &ps)
	c.Assert(err, NotNil)

	buf, err := json.Marshal(ps)
	c.Check(err, IsNil)
	c.Check(buf, NotNil)
	invalidPoolStatus := PoolStatus(100)
	c.Check(invalidPoolStatus.Valid(), NotNil)
}
