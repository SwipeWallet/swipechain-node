package thorclient

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"time"

	cKeys "github.com/cosmos/cosmos-sdk/crypto/keys"
	sdk "github.com/cosmos/cosmos-sdk/types"
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/x/thorchain"
)

type KeysSuite struct{}

var _ = Suite(&KeysSuite{})

func (*KeysSuite) SetUpSuite(c *C) {
	thorchain.SetupConfigForTest()
}

const (
	signerNameForTest     = `jack`
	signerPasswordForTest = `password`
)

func (*KeysSuite) setupKeysForTest(c *C) string {
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

func (ks *KeysSuite) TestNewKeys(c *C) {
	oldStdIn := os.Stdin
	defer func() {
		os.Stdin = oldStdIn
	}()
	os.Stdin = nil
	folder := ks.setupKeysForTest(c)
	defer func() {
		err := os.RemoveAll(folder)
		c.Assert(err, IsNil)
	}()

	k, info, err := GetKeyringKeybase(folder, signerNameForTest, signerPasswordForTest)
	c.Assert(err, IsNil)
	c.Assert(k, NotNil)
	c.Assert(info, NotNil)
	ki := NewKeysWithKeybase(k, info, signerPasswordForTest)
	kInfo := ki.GetSignerInfo()
	c.Assert(kInfo, NotNil)
	c.Assert(kInfo.GetName(), Equals, signerNameForTest)
	priKey, err := ki.GetPrivateKey()
	c.Assert(err, IsNil)
	c.Assert(priKey, NotNil)
	c.Assert(priKey.Bytes(), HasLen, 37)
	kb := ki.GetKeybase()
	c.Assert(kb, NotNil)
	kb.CloseDB()
}
