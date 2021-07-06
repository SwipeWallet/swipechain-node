package thorclient

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"

	ckeys "github.com/cosmos/cosmos-sdk/crypto/keys"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/crypto"
)

const (
	// folder name for thorchain thorcli
	thorchainCliFolderName = `.thorcli`
)

// Keys manages all the keys used by thorchain
type Keys struct {
	signerName string
	password   string // TODO this is a bad way , need to fix it
	signerInfo ckeys.Info
	kb         ckeys.Keybase
}

// NewKeysWithKeybase create a new instance of Keys
func NewKeysWithKeybase(kb ckeys.Keybase, signerInfo ckeys.Info, password string) *Keys {
	return &Keys{
		signerName: signerInfo.GetName(),
		password:   password,
		signerInfo: signerInfo,
		kb:         kb,
	}
}

// NewKeys create a new instance of keys
func GetKeyringKeybase(chainHomeFolder, signerName, password string) (ckeys.Keybase, ckeys.Info, error) {
	if len(signerName) == 0 {
		return nil, nil, fmt.Errorf("signer name is empty")
	}
	if len(password) == 0 {
		return nil, nil, fmt.Errorf("password is empty")
	}

	buf := bytes.NewBufferString(password)
	// the library used by keyring is using ReadLine , which expect a new line
	buf.WriteByte('\n')
	kb, err := getKeybase(chainHomeFolder, buf)
	if err != nil {
		return nil, nil, fmt.Errorf("fail to get keybase,err:%w", err)
	}
	// the keyring library which used by cosmos sdk , will use interactive terminal if it detect it has one
	// this will temporary trick it think there is no interactive terminal, thus will read the password from the buffer provided
	oldStdIn := os.Stdin
	defer func() {
		os.Stdin = oldStdIn
	}()
	os.Stdin = nil
	si, err := kb.Get(signerName)
	if err != nil {
		return nil, nil, fmt.Errorf("fail to get signer info(%s): %w", signerName, err)
	}
	return kb, si, nil
}

// getKeybase will create an instance of Keybase
func getKeybase(thorchainHome string, reader io.Reader) (ckeys.Keybase, error) {
	cliDir := thorchainHome
	if len(thorchainHome) == 0 {
		usr, err := user.Current()
		if err != nil {
			return nil, fmt.Errorf("fail to get current user,err:%w", err)
		}
		cliDir = filepath.Join(usr.HomeDir, thorchainCliFolderName)
	}

	return ckeys.NewKeyring(sdk.KeyringServiceName(), ckeys.BackendFile, cliDir, reader)
}

// GetSignerInfo return signer info
func (k *Keys) GetSignerInfo() ckeys.Info {
	return k.signerInfo
}

// GetPrivateKey return the private key
func (k *Keys) GetPrivateKey() (crypto.PrivKey, error) {
	return k.kb.ExportPrivateKeyObject(k.signerName, k.password)
}

// GetKeybase return the keybase
func (k *Keys) GetKeybase() ckeys.Keybase {
	return k.kb
}
