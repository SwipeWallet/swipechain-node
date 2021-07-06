package thorchain

import (
	"fmt"
	"testing"

	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/x/thorchain/types"
)

func TestPackage(t *testing.T) { TestingT(t) }

type MemoSuite struct{}

var _ = Suite(&MemoSuite{})

func (s *MemoSuite) SetUpSuite(c *C) {
	types.SetupConfigForTest()
}

func (s *MemoSuite) TestTxType(c *C) {
	for _, trans := range []TxType{TxStake, TxUnstake, TxSwap, TxOutbound, TxAdd, TxBond, TxUnbond, TxLeave, TxSwitch} {
		tx, err := StringToTxType(trans.String())
		c.Assert(err, IsNil)
		c.Check(tx, Equals, trans)
		c.Check(tx.IsEmpty(), Equals, false)
	}
}

func (s *MemoSuite) TestParseWithAbbreviated(c *C) {
	// happy paths
	memo, err := ParseMemo("%:" + common.RuneAsset().String())
	c.Assert(err, IsNil)
	c.Check(memo.GetAsset().String(), Equals, common.RuneAsset().String())
	c.Check(memo.IsType(TxAdd), Equals, true, Commentf("MEMO: %+v", memo))
	c.Check(memo.IsInbound(), Equals, true)
	c.Check(memo.IsInternal(), Equals, false)
	c.Check(memo.IsOutbound(), Equals, false)

	memo, err = ParseMemo("+:" + common.RuneAsset().String())
	c.Assert(err, IsNil)
	c.Check(memo.GetAsset().String(), Equals, common.RuneAsset().String())
	c.Check(memo.IsType(TxStake), Equals, true, Commentf("MEMO: %+v", memo))
	c.Check(memo.IsInbound(), Equals, true)
	c.Check(memo.IsInternal(), Equals, false)
	c.Check(memo.IsOutbound(), Equals, false)

	memo, err = ParseMemo(fmt.Sprintf("-:%s:25", common.RuneAsset().String()))
	c.Assert(err, IsNil)
	c.Check(memo.GetAsset().String(), Equals, common.RuneAsset().String())
	c.Check(memo.IsType(TxUnstake), Equals, true, Commentf("MEMO: %+v", memo))
	c.Check(memo.GetAmount().Uint64(), Equals, uint64(25), Commentf("%d", memo.GetAmount().Uint64()))
	c.Check(memo.IsInbound(), Equals, true)
	c.Check(memo.IsInternal(), Equals, false)
	c.Check(memo.IsOutbound(), Equals, false)

	memo, err = ParseMemo("=:" + common.RuneAsset().String() + ":bnb1lejrrtta9cgr49fuh7ktu3sddhe0ff7wenlpn6:870000000")
	c.Assert(err, IsNil)
	c.Check(memo.GetAsset().String(), Equals, common.RuneAsset().String())
	c.Check(memo.IsType(TxSwap), Equals, true, Commentf("MEMO: %+v", memo))
	c.Check(memo.GetDestination().String(), Equals, "bnb1lejrrtta9cgr49fuh7ktu3sddhe0ff7wenlpn6")
	c.Log(memo.GetSlipLimit().Uint64())
	c.Check(memo.GetSlipLimit().Equal(cosmos.NewUint(870000000)), Equals, true)
	c.Check(memo.IsInbound(), Equals, true)
	c.Check(memo.IsInternal(), Equals, false)
	c.Check(memo.IsOutbound(), Equals, false)

	memo, err = ParseMemo("=:" + common.RuneAsset().String() + ":bnb1lejrrtta9cgr49fuh7ktu3sddhe0ff7wenlpn6")
	c.Assert(err, IsNil)
	c.Check(memo.GetAsset().String(), Equals, common.RuneAsset().String())
	c.Check(memo.IsType(TxSwap), Equals, true, Commentf("MEMO: %+v", memo))
	c.Check(memo.GetDestination().String(), Equals, "bnb1lejrrtta9cgr49fuh7ktu3sddhe0ff7wenlpn6")
	c.Check(memo.GetSlipLimit().Uint64(), Equals, uint64(0))
	c.Check(memo.IsInbound(), Equals, true)

	memo, err = ParseMemo("=:" + common.RuneAsset().String() + ":bnb1lejrrtta9cgr49fuh7ktu3sddhe0ff7wenlpn6:")
	c.Assert(err, IsNil)
	c.Check(memo.GetAsset().String(), Equals, common.RuneAsset().String())
	c.Check(memo.IsType(TxSwap), Equals, true, Commentf("MEMO: %+v", memo))
	c.Check(memo.GetDestination().String(), Equals, "bnb1lejrrtta9cgr49fuh7ktu3sddhe0ff7wenlpn6")
	c.Check(memo.GetSlipLimit().Equal(cosmos.ZeroUint()), Equals, true)

	memo, err = ParseMemo("OUTBOUND:MUKVQILIHIAUSEOVAXBFEZAJKYHFJYHRUUYGQJZGFYBYVXCXYNEMUOAIQKFQLLCX")
	c.Assert(err, IsNil)
	c.Check(memo.IsType(TxOutbound), Equals, true, Commentf("%s", memo.GetType()))
	c.Check(memo.IsOutbound(), Equals, true)
	c.Check(memo.IsInbound(), Equals, false)
	c.Check(memo.IsInternal(), Equals, false)

	memo, err = ParseMemo("REFUND:MUKVQILIHIAUSEOVAXBFEZAJKYHFJYHRUUYGQJZGFYBYVXCXYNEMUOAIQKFQLLCX")
	c.Assert(err, IsNil)
	c.Check(memo.IsType(TxRefund), Equals, true)
	c.Check(memo.IsOutbound(), Equals, true)

	memo, err = ParseMemo("leave:whatever")
	c.Assert(err, NotNil)
	c.Check(memo.IsType(TxUnknown), Equals, true)

	addr := types.GetRandomBech32Addr()
	memo, err = ParseMemo(fmt.Sprintf("leave:%s", addr.String()))
	c.Assert(err, IsNil)
	c.Check(memo.IsType(TxLeave), Equals, true)
	c.Check(memo.GetAccAddress().String(), Equals, addr.String())

	memo, err = ParseMemo("yggdrasil+:30")
	c.Assert(err, IsNil)
	c.Check(memo.IsType(TxYggdrasilFund), Equals, true)
	c.Check(memo.IsInbound(), Equals, false)
	c.Check(memo.IsInternal(), Equals, true)
	memo, err = ParseMemo("yggdrasil-:30")
	c.Assert(err, IsNil)
	c.Check(memo.IsType(TxYggdrasilReturn), Equals, true)
	c.Check(memo.IsInternal(), Equals, true)
	memo, err = ParseMemo("migrate:100")
	c.Assert(err, IsNil)
	c.Check(memo.IsType(TxMigrate), Equals, true)
	c.Check(memo.IsInternal(), Equals, true)

	memo, err = ParseMemo("ragnarok:100")
	c.Assert(err, IsNil)
	c.Check(memo.IsType(TxRagnarok), Equals, true)
	c.Check(memo.IsOutbound(), Equals, true)

	mem := fmt.Sprintf("switch:%s", types.GetRandomBech32Addr())
	memo, err = ParseMemo(mem)
	c.Assert(err, IsNil)
	c.Check(memo.IsType(TxSwitch), Equals, true)
	c.Check(memo.IsInbound(), Equals, true)

	memo, err = ParseMemo("reserve")
	c.Check(err, IsNil)
	c.Check(memo.IsType(TxReserve), Equals, true)
	c.Check(memo.IsInbound(), Equals, true)
	c.Check(memo.IsInternal(), Equals, false)
	c.Check(memo.IsOutbound(), Equals, false)

	// unhappy paths
	_, err = ParseMemo("")
	c.Assert(err, NotNil)
	_, err = ParseMemo("bogus")
	c.Assert(err, NotNil)
	_, err = ParseMemo("CREATE") // missing symbol
	c.Assert(err, NotNil)
	_, err = ParseMemo("c:") // bad symbol
	c.Assert(err, NotNil)
	_, err = ParseMemo("-:bnb") // withdraw basis points is optional
	c.Assert(err, IsNil)
	_, err = ParseMemo("-:bnb:twenty-two") // bad amount
	c.Assert(err, NotNil)
	_, err = ParseMemo("=:bnb:bad_DES:5.6") // bad destination
	c.Assert(err, NotNil)
	_, err = ParseMemo(">:bnb:bnb1lejrrtta9cgr49fuh7ktu3sddhe0ff7wenlpn6:five") // bad slip limit
	c.Assert(err, NotNil)
	_, err = ParseMemo("!:key:val") // not enough arguments
	c.Assert(err, NotNil)
	_, err = ParseMemo("!:bogus:key:value") // bogus admin command type
	c.Assert(err, NotNil)
	_, err = ParseMemo("nextpool:whatever")
	c.Assert(err, NotNil)
	_, err = ParseMemo("migrate")
	c.Assert(err, NotNil)
	_, err = ParseMemo("switch")
	c.Assert(err, NotNil)
	_, err = ParseMemo("switch:")
	c.Assert(err, NotNil)
}

func (s *MemoSuite) TestParse(c *C) {
	// happy paths
	memo, err := ParseMemo("add:" + common.RuneAsset().String())
	c.Assert(err, IsNil)
	c.Check(memo.GetAsset().String(), Equals, common.RuneAsset().String())
	c.Check(memo.IsType(TxAdd), Equals, true, Commentf("MEMO: %+v", memo))
	c.Check(memo.String(), Equals, "ADD:"+common.RuneAsset().String())

	memo, err = ParseMemo("STAKE:" + common.RuneAsset().String())
	c.Assert(err, IsNil)
	c.Check(memo.GetAsset().String(), Equals, common.RuneAsset().String())
	c.Check(memo.IsType(TxStake), Equals, true, Commentf("MEMO: %+v", memo))
	c.Check(memo.String(), Equals, "")

	memo, err = ParseMemo("STAKE:BTC.BTC")
	c.Assert(err, NotNil)
	memo, err = ParseMemo("STAKE:BTC.BTC:bc1qwqdg6squsna38e46795at95yu9atm8azzmyvckulcc7kytlcckxswvvzej")
	c.Assert(err, IsNil)
	c.Check(memo.GetDestination().String(), Equals, "bc1qwqdg6squsna38e46795at95yu9atm8azzmyvckulcc7kytlcckxswvvzej")
	c.Check(memo.IsType(TxStake), Equals, true, Commentf("MEMO: %+v", memo))

	memo, err = ParseMemo("WITHDRAW:" + common.RuneAsset().String() + ":25")
	c.Assert(err, IsNil)
	c.Check(memo.GetAsset().String(), Equals, common.RuneAsset().String())
	c.Check(memo.IsType(TxUnstake), Equals, true, Commentf("MEMO: %+v", memo))
	c.Check(memo.GetAmount().Equal(cosmos.NewUint(25)), Equals, true, Commentf("%d", memo.GetAmount().Uint64()))

	memo, err = ParseMemo("SWAP:" + common.RuneAsset().String() + ":bnb1lejrrtta9cgr49fuh7ktu3sddhe0ff7wenlpn6:870000000")
	c.Assert(err, IsNil)
	c.Check(memo.GetAsset().String(), Equals, common.RuneAsset().String())
	c.Check(memo.IsType(TxSwap), Equals, true, Commentf("MEMO: %+v", memo))
	c.Check(memo.GetDestination().String(), Equals, "bnb1lejrrtta9cgr49fuh7ktu3sddhe0ff7wenlpn6")
	c.Log(memo.GetSlipLimit().String())
	c.Check(memo.GetSlipLimit().Equal(cosmos.NewUint(870000000)), Equals, true)

	memo, err = ParseMemo("SWAP:" + common.RuneAsset().String() + ":bnb1lejrrtta9cgr49fuh7ktu3sddhe0ff7wenlpn6")
	c.Assert(err, IsNil)
	c.Check(memo.GetAsset().String(), Equals, common.RuneAsset().String())
	c.Check(memo.IsType(TxSwap), Equals, true, Commentf("MEMO: %+v", memo))
	c.Check(memo.GetDestination().String(), Equals, "bnb1lejrrtta9cgr49fuh7ktu3sddhe0ff7wenlpn6")
	c.Check(memo.GetSlipLimit().Uint64(), Equals, uint64(0))

	memo, err = ParseMemo("SWAP:" + common.RuneAsset().String() + ":bnb1lejrrtta9cgr49fuh7ktu3sddhe0ff7wenlpn6:")
	c.Assert(err, IsNil)
	c.Check(memo.GetAsset().String(), Equals, common.RuneAsset().String())
	c.Check(memo.IsType(TxSwap), Equals, true, Commentf("MEMO: %+v", memo))
	c.Check(memo.GetDestination().String(), Equals, "bnb1lejrrtta9cgr49fuh7ktu3sddhe0ff7wenlpn6")
	c.Check(memo.GetSlipLimit().Uint64(), Equals, uint64(0))

	whiteListAddr := types.GetRandomBech32Addr()
	memo, err = ParseMemo("bond:" + whiteListAddr.String())
	c.Assert(err, IsNil)
	c.Assert(memo.IsType(TxBond), Equals, true)
	c.Assert(memo.GetAccAddress().String(), Equals, whiteListAddr.String())

	memo, err = ParseMemo("leave:" + types.GetRandomBech32Addr().String())
	c.Assert(err, IsNil)
	c.Assert(memo.IsType(TxLeave), Equals, true)

	memo, err = ParseMemo("unbond:" + whiteListAddr.String() + ":300")
	c.Assert(err, IsNil)
	c.Assert(memo.IsType(TxUnbond), Equals, true)
	c.Assert(memo.GetAccAddress().String(), Equals, whiteListAddr.String())
	c.Assert(memo.GetAmount().Equal(cosmos.NewUint(300)), Equals, true)

	memo, err = ParseMemo("migrate:100")
	c.Assert(err, IsNil)
	c.Check(memo.IsType(TxMigrate), Equals, true)
	c.Check(memo.GetBlockHeight(), Equals, int64(100))
	c.Check(memo.String(), Equals, "MIGRATE:100")

	txID := types.GetRandomTxHash()
	memo, err = ParseMemo("OUTBOUND:" + txID.String())
	c.Check(err, IsNil)
	c.Check(memo.IsOutbound(), Equals, true)
	c.Check(memo.GetTxID(), Equals, txID)
	c.Check(memo.String(), Equals, "OUTBOUND:"+txID.String())

	refundMemo := "REFUND:" + txID.String()
	memo, err = ParseMemo(refundMemo)
	c.Check(err, IsNil)
	c.Check(memo.GetTxID(), Equals, txID)
	c.Check(memo.String(), Equals, refundMemo)

	yggFundMemo := "YGGDRASIL+:100"
	memo, err = ParseMemo(yggFundMemo)
	c.Check(err, IsNil)
	c.Check(memo.GetBlockHeight(), Equals, int64(100))
	c.Check(memo.String(), Equals, yggFundMemo)

	yggReturnMemo := "YGGDRASIL-:100"
	memo, err = ParseMemo(yggReturnMemo)
	c.Check(err, IsNil)
	c.Check(memo.GetBlockHeight(), Equals, int64(100))
	c.Check(memo.String(), Equals, yggReturnMemo)

	ragnarokMemo := "RAGNAROK:1024"
	memo, err = ParseMemo(ragnarokMemo)
	c.Check(err, IsNil)
	c.Check(memo.IsType(TxRagnarok), Equals, true)
	c.Check(memo.GetBlockHeight(), Equals, int64(1024))
	c.Check(memo.String(), Equals, ragnarokMemo)

	baseMemo := MemoBase{}
	c.Check(baseMemo.String(), Equals, "")
	c.Check(baseMemo.GetAmount().Uint64(), Equals, cosmos.ZeroUint().Uint64())
	c.Check(baseMemo.GetDestination(), Equals, common.NoAddress)
	c.Check(baseMemo.GetSlipLimit().Uint64(), Equals, cosmos.ZeroUint().Uint64())
	c.Check(baseMemo.GetTxID(), Equals, common.TxID(""))
	c.Check(baseMemo.GetAccAddress().Empty(), Equals, true)
	c.Check(baseMemo.IsEmpty(), Equals, true)
	c.Check(baseMemo.GetBlockHeight(), Equals, int64(0))

	// unhappy paths
	_, err = ParseMemo("")
	c.Assert(err, NotNil)
	_, err = ParseMemo("bogus")
	c.Assert(err, NotNil)
	_, err = ParseMemo("CREATE") // missing symbol
	c.Assert(err, NotNil)
	_, err = ParseMemo("CREATE:") // bad symbol
	c.Assert(err, NotNil)
	_, err = ParseMemo("withdraw") // not enough parameters
	c.Assert(err, NotNil)
	_, err = ParseMemo("withdraw:bnb") // withdraw basis points is optional
	c.Assert(err, IsNil)
	_, err = ParseMemo("withdraw:bnb:twenty-two") // bad amount
	c.Assert(err, NotNil)
	_, err = ParseMemo("swap") // not enough parameters
	c.Assert(err, NotNil)
	_, err = ParseMemo("swap:bnb:STAKER-1:5.6") // bad destination
	c.Assert(err, NotNil)
	_, err = ParseMemo("swap:bnb:bad_DES:5.6") // bad destination
	c.Assert(err, NotNil)
	_, err = ParseMemo("swap:bnb:bnb1lejrrtta9cgr49fuh7ktu3sddhe0ff7wenlpn6:five") // bad slip limit
	c.Assert(err, NotNil)
	_, err = ParseMemo("admin:key:val") // not enough arguments
	c.Assert(err, NotNil)
	_, err = ParseMemo("admin:bogus:key:value") // bogus admin command type
	c.Assert(err, NotNil)
	_, err = ParseMemo("migrate:abc")
	c.Assert(err, NotNil)

	_, err = ParseMemo("withdraw:A")
	c.Assert(err, NotNil)
	_, err = ParseMemo("leave")
	c.Assert(err, NotNil)
	_, err = ParseMemo("outbound") // not enough parameter
	c.Assert(err, NotNil)
	_, err = ParseMemo("bond") // not enough parameter
	c.Assert(err, NotNil)
	_, err = ParseMemo("refund") // not enough parameter
	c.Assert(err, NotNil)
	_, err = ParseMemo("yggdrasil+") // not enough parameter
	c.Assert(err, NotNil)
	_, err = ParseMemo("yggdrasil+:A") // invalid block height
	c.Assert(err, NotNil)
	_, err = ParseMemo("yggdrasil-") // not enough parameter
	c.Assert(err, NotNil)
	_, err = ParseMemo("yggdrasil-:B") // invalid block height
	c.Assert(err, NotNil)
	_, err = ParseMemo("ragnarok") // not enough parameter
	c.Assert(err, NotNil)
	_, err = ParseMemo("ragnarok:what") // not enough parameter
	c.Assert(err, NotNil)
	_, err = ParseMemo("bond:what") // invalid address
	c.Assert(err, NotNil)
	_, err = ParseMemo("switch:what") // invalid address
	c.Assert(err, NotNil)
	_, err = ParseMemo("whatever") // not support
	c.Assert(err, NotNil)
}
