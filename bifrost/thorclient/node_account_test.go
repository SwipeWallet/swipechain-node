package thorclient

import (
	"net/http"
	"net/http/httptest"
	"strings"

	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/bifrost/config"
)

type NodeAccountSuite struct {
	server  *httptest.Server
	bridge  *ThorchainBridge
	cfg     config.ClientConfiguration
	fixture string
}

var _ = Suite(&NodeAccountSuite{})

func (s *NodeAccountSuite) SetUpSuite(c *C) {
	s.server = httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		switch {
		case strings.HasPrefix(req.RequestURI, NodeAccountEndpoint):
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

func (s *NodeAccountSuite) TearDownSuite(c *C) {
	s.server.Close()
}

func (s *NodeAccountSuite) TestGetNodeAccount(c *C) {
	s.fixture = "../../test/fixtures/endpoints/nodeaccount/template.json"
	na, err := s.bridge.GetNodeAccount(s.bridge.keys.GetSignerInfo().GetAddress().String())
	c.Assert(err, IsNil)
	c.Assert(na, NotNil)
}
