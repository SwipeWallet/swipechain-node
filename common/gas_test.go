package common

import (
	"math/big"

	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
	. "gopkg.in/check.v1"
)

type GasSuite struct{}

var _ = Suite(&GasSuite{})

func (s *GasSuite) TestETHGasFee(c *C) {
	gas := GetETHGasFee(big.NewInt(20), 4)
	amt := gas[0].Amount
	c.Check(
		amt.Equal(cosmos.NewUint(425440)),
		Equals,
		true,
		Commentf("%d", amt.Uint64()),
	)
	gas = MakeETHGas(big.NewInt(20), 50000)
	amt = gas[0].Amount
	c.Check(
		amt.Equal(cosmos.NewUint(1000000)),
		Equals,
		true,
		Commentf("%d", amt.Uint64()),
	)
}

func (s *GasSuite) TestIsEmpty(c *C) {
	gas1 := Gas{
		{Asset: BNBAsset, Amount: cosmos.NewUint(11 * One)},
	}
	c.Check(gas1.IsEmpty(), Equals, false)
	c.Check(Gas{}.IsEmpty(), Equals, true)
}

func (s *GasSuite) TestCombineGas(c *C) {
	gas1 := Gas{
		{Asset: BNBAsset, Amount: cosmos.NewUint(11 * One)},
	}
	gas2 := Gas{
		{Asset: BNBAsset, Amount: cosmos.NewUint(14 * One)},
		{Asset: BTCAsset, Amount: cosmos.NewUint(20 * One)},
	}

	gas := gas1.Add(gas2)
	c.Assert(gas, HasLen, 2)
	c.Check(gas[0].Asset.Equals(BNBAsset), Equals, true)
	c.Check(gas[0].Amount.Equal(cosmos.NewUint(25*One)), Equals, true, Commentf("%d", gas[0].Amount.Uint64()))
	c.Check(gas[1].Asset.Equals(BTCAsset), Equals, true)
	c.Check(gas[1].Amount.Equal(cosmos.NewUint(20*One)), Equals, true)
}

func (s *GasSuite) TestCalcGasPrice(c *C) {
	gasInfo := []cosmos.Uint{cosmos.NewUint(37500), cosmos.NewUint(30000)}
	tx := Tx{
		Coins: Coins{
			NewCoin(BNBAsset, cosmos.NewUint(80808080)),
		},
	}

	gas := CalcBinanceGasPrice(tx, BNBAsset, gasInfo)
	c.Check(gas.Equals(Gas{NewCoin(BNBAsset, cosmos.NewUint(37500))}), Equals, true)

	tx = Tx{
		Coins: Coins{
			NewCoin(BNBAsset, cosmos.NewUint(80808080)),
			NewCoin(BNBAsset, cosmos.NewUint(80808080)),
		},
	}

	gas = CalcBinanceGasPrice(tx, BNBAsset, gasInfo)
	c.Check(gas.Equals(Gas{NewCoin(BNBAsset, cosmos.NewUint(60000))}), Equals, true)
}

func (s *GasSuite) TestUpdateGasPrice(c *C) {
	gasInfo := UpdateGasPrice(Tx{}, BNBAsset, []cosmos.Uint{cosmos.NewUint(33)})
	c.Assert(gasInfo, HasLen, 1)
	c.Check(gasInfo[0].Equal(cosmos.NewUint(33)), Equals, true)

	tx := Tx{
		Coins: Coins{
			NewCoin(BNBAsset, cosmos.NewUint(80808080)),
		},
		Gas: Gas{
			NewCoin(BNBAsset, cosmos.NewUint(222)),
		},
	}

	gasInfo = UpdateGasPrice(tx, BNBAsset, nil)
	c.Assert(gasInfo, HasLen, 2)
	c.Check(gasInfo[0].Equal(cosmos.NewUint(222)), Equals, true)

	tx = Tx{
		Coins: Coins{
			NewCoin(BNBAsset, cosmos.NewUint(80808080)),
			NewCoin(BTCAsset, cosmos.NewUint(80808080)),
		},
		Gas: Gas{
			NewCoin(BNBAsset, cosmos.NewUint(222)),
		},
	}

	gasInfo = UpdateGasPrice(tx, BNBAsset, gasInfo)
	c.Assert(gasInfo, HasLen, 2)
	c.Check(gasInfo[1].Equal(cosmos.NewUint(111)), Equals, true)
}
