package tss

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	cKeys "github.com/cosmos/cosmos-sdk/crypto/keys"
	sdk "github.com/cosmos/cosmos-sdk/types"
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/bifrost/thorclient"
	"gitlab.com/thorchain/thornode/x/thorchain"
)

func TestTSSKeyGen(t *testing.T) { TestingT(t) }

type KeyGenTestSuite struct{}

var _ = Suite(&KeyGenTestSuite{})

func (*KeyGenTestSuite) SetUpSuite(c *C) {
	thorchain.SetupConfigForTest()
}

const (
	signerNameForTest     = `jack`
	signerPasswordForTest = `password`
)

func (*KeyGenTestSuite) setupKeysForTest(c *C) string {
	ns := strconv.Itoa(time.Now().Nanosecond())
	thorcliDir := filepath.Join(os.TempDir(), ns, ".thorcli")
	c.Logf("thorcliDir:%s", thorcliDir)
	buf := bytes.NewBufferString(signerPasswordForTest)
	// the library used by keyring is using ReadLine , which expect a new line
	buf.WriteByte('\n')
	buf.WriteString(signerPasswordForTest)
	buf.WriteByte('\n')
	kb, err := cKeys.NewKeyring(sdk.KeyringServiceName(), cKeys.BackendFile, thorcliDir, buf)
	c.Assert(err, IsNil)
	info, _, err := kb.CreateMnemonic(signerNameForTest, cKeys.English, signerPasswordForTest, cKeys.Secp256k1)
	c.Logf("name:%s", info.GetName())
	c.Assert(err, IsNil)
	kb.CloseDB()
	return thorcliDir
}

func (kts *KeyGenTestSuite) TestNewTssKenGen(c *C) {
	oldStdIn := os.Stdin
	defer func() {
		os.Stdin = oldStdIn
	}()
	os.Stdin = nil
	folder := kts.setupKeysForTest(c)
	defer func() {
		err := os.RemoveAll(folder)
		c.Assert(err, IsNil)
	}()
	kb, info, err := thorclient.GetKeyringKeybase(folder, signerNameForTest, signerPasswordForTest)
	c.Assert(err, IsNil)
	k := thorclient.NewKeysWithKeybase(kb, info, signerPasswordForTest)
	c.Assert(k, NotNil)
	kg, err := NewTssKeyGen(k, nil)
	c.Assert(err, IsNil)
	c.Assert(kg, NotNil)
}
