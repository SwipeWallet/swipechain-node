package types

import (
	"gitlab.com/thorchain/thornode/common"
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
	. "gopkg.in/check.v1"
)

type TxOutTestSuite struct{}

var _ = Suite(&TxOutTestSuite{})

func (TxOutTestSuite) TestTxOutItemHash(c *C) {
	// WARNING if those tests are breaking after a change,
	// we need to update Heimdall as well to replicate the changes
	item := TxOutItem{
		Chain:       "BNB",
		ToAddress:   "tbnb1yxfyeda8pnlxlmx0z3cwx74w9xevspwdpzdxpj",
		VaultPubKey: "",
		Coins: common.Coins{
			common.NewCoin(common.BNBAsset, cosmos.NewUint(194765912)),
		},
		Memo:   "REFUND:9999A5A08D8FCF942E1AAAA01AB1E521B699BA3A009FA0591C011DC1FFDC5E68",
		InHash: "9999A5A08D8FCF942E1AAAA01AB1E521B699BA3A009FA0591C011DC1FFDC5E68",
	}
	c.Check(item.Hash(), Equals, "C373A22362891F3B35F4B624E36B2984C33C84C43CA8352ACE085F417307673D")

	item = TxOutItem{
		Chain:       "BNB",
		ToAddress:   "tbnb1yxfyeda8pnlxlmx0z3cwx74w9xevspwdpzdxpj",
		VaultPubKey: "",
		Memo:        "REFUND:9999A5A08D8FCF942E1AAAA01AB1E521B699BA3A009FA0591C011DC1FFDC5E68",
		InHash:      "9999A5A08D8FCF942E1AAAA01AB1E521B699BA3A009FA0591C011DC1FFDC5E68",
	}
	c.Check(item.Hash(), Equals, "5037BD52845B23EEA538248622F0F9625536192A066FDCED91494171BD1EF43D")

	item = TxOutItem{
		Chain:       "BNB",
		ToAddress:   "tbnb1yxfyeda8pnlxlmx0z3cwx74w9xevspwdpzdxpj",
		VaultPubKey: "thorpub1addwnpepqv7kdf473gc4jyls7hlx4rg",
		Memo:        "REFUND:9999A5A08D8FCF942E1AAAA01AB1E521B699BA3A009FA0591C011DC1FFDC5E68",
		InHash:      "9999A5A08D8FCF942E1AAAA01AB1E521B699BA3A009FA0591C011DC1FFDC5E68",
	}
	c.Check(item.Hash(), Equals, "0D920B69BC43443CF58A382BE9714FC67516CD87DD89212A2F2989566E2E632B")
}
