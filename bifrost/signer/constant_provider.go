package signer

import (
	"fmt"
	"sync"

	"gitlab.com/thorchain/thornode/bifrost/thorclient"
	"gitlab.com/thorchain/thornode/constants"
)

// ConstantProvider which will query thorchain to get the constants value per request
// it will also cache the constant values internally
type ConstantsProvider struct {
	requestHeight int64 // the block height last request to thorchain to retrieve constant values
	bridge        *thorclient.ThorchainBridge
	constantsLock *sync.Mutex
	constants     map[string]int64 // the constant values get from thorchain and cached in memory
}

// NewConstantsProvider create a new instance of ConstantsProvider
func NewConstantsProvider(bridge *thorclient.ThorchainBridge) *ConstantsProvider {
	return &ConstantsProvider{
		constants:     make(map[string]int64),
		requestHeight: 0,
		bridge:        bridge,
		constantsLock: &sync.Mutex{},
	}
}

// GetInt64Value get the constant value that match the given key
func (cp *ConstantsProvider) GetInt64Value(thorchainBlockHeight int64, key constants.ConstantName) (int64, error) {
	if err := cp.EnsureConstants(thorchainBlockHeight); err != nil {
		return 0, fmt.Errorf("fail to get constants from thorchain: %w", err)
	}
	return cp.constants[key.String()], nil
}

func (cp *ConstantsProvider) EnsureConstants(thorchainBlockHeight int64) error {
	if cp.requestHeight == 0 {
		return cp.getConstantsFromThorchain(thorchainBlockHeight)
	}
	rotateBlockHeight := cp.constants[constants.RotatePerBlockHeight.String()]
	// Thorchain will have new version and constants only when new node get rotated in , and the new version get consensus
	if thorchainBlockHeight-cp.requestHeight < rotateBlockHeight {
		return nil
	}
	return cp.getConstantsFromThorchain(thorchainBlockHeight)
}

func (cp *ConstantsProvider) getConstantsFromThorchain(height int64) error {
	constants, err := cp.bridge.GetConstants()
	if err != nil {
		return fmt.Errorf("fail to get constants: %w", err)
	}
	cp.constantsLock.Lock()
	defer cp.constantsLock.Unlock()
	cp.constants = constants
	cp.requestHeight = height
	return nil
}
