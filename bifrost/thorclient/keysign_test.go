package thorclient

import (
	"net/http"
	"net/http/httptest"
	"strings"

	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/bifrost/config"
	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/x/thorchain/types"
)

type KeysignSuite struct {
	server  *httptest.Server
	bridge  *ThorchainBridge
	cfg     config.ClientConfiguration
	fixture string
}

var _ = Suite(&KeysignSuite{})

func (s *KeysignSuite) SetUpSuite(c *C) {
	s.server = httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		switch {
		case strings.HasPrefix(req.RequestURI, KeysignEndpoint):
			httpTestHandler(c, rw, s.fixture)
		}
	}))

	cfg, info, kb := SetupThorchainForTest(c)
	s.cfg = cfg
	s.cfg.ChainHost = s.server.Listener.Addr().String()
	var err error
	s.bridge, err = NewThorchainBridge(s.cfg, GetMetricForTest(c), NewKeysWithKeybase(kb, info, cfg.SignerPasswd))
	s.bridge.httpClient.RetryMax = 1
	c.Assert(err, IsNil)
	c.Assert(s.bridge, NotNil)
}

func (s *KeysignSuite) TearDownSuite(c *C) {
	s.server.Close()
}

func (s *KeysignSuite) TestGetKeysign(c *C) {
	s.fixture = "../../test/fixtures/endpoints/keysign/template.json"
	pk := types.GetRandomPubKey()
	keysign, err := s.bridge.GetKeysign(1718, pk.String())
	c.Assert(err, IsNil)
	c.Assert(keysign, NotNil)
	c.Assert(keysign.Height, Equals, int64(1718))
	c.Assert(keysign.TxArray[0].Chain, Equals, common.BNBChain)
}
