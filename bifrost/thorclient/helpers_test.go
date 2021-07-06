package thorclient

import (
	"time"

	"github.com/cosmos/cosmos-sdk/client/keys"
	cKeys "github.com/cosmos/cosmos-sdk/crypto/keys"
	hd "github.com/cosmos/cosmos-sdk/crypto/keys/hd"
	sdk "github.com/cosmos/cosmos-sdk/types"
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/bifrost/config"
	"gitlab.com/thorchain/thornode/bifrost/metrics"
	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/x/thorchain"
)

var m *metrics.Metrics

func SetupThorchainForTest(c *C) (config.ClientConfiguration, cKeys.Info, cKeys.Keybase) {
	thorchain.SetupConfigForTest()
	cfg := config.ClientConfiguration{
		ChainID:         "thorchain",
		ChainHost:       "localhost",
		ChainRPC:        "localhost",
		SignerName:      "thorchain",
		SignerPasswd:    "password",
		ChainHomeFolder: "",
	}
	kb := keys.NewInMemoryKeyBase()

	params := *hd.NewFundraiserParams(0, sdk.CoinType, 0)
	hdPath := params.String()

	// create a consistent user
	info, err := kb.CreateAccount(cfg.SignerName, "industry segment educate height inject hover bargain offer employ select speak outer video tornado story slow chief object junk vapor venue large shove behave", cfg.SignerPasswd, cfg.SignerPasswd, hdPath, cKeys.Secp256k1)
	c.Assert(err, IsNil)

	return cfg, info, kb
}

func GetMetricForTest(c *C) *metrics.Metrics {
	if m == nil {
		var err error
		m, err = metrics.NewMetrics(config.MetricsConfiguration{
			Enabled:      false,
			ListenPort:   9000,
			ReadTimeout:  time.Second,
			WriteTimeout: time.Second,
			Chains:       common.Chains{common.BNBChain},
		})
		c.Assert(m, NotNil)
		c.Assert(err, IsNil)
	}
	return m
}
