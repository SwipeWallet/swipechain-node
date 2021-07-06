package thorclient

import (
	"sync"

	"gitlab.com/thorchain/thornode/common"
)

// KeySignPartyMgr is to cache the keysign party locally
type KeySignPartyMgr struct {
	keySignPartiesLock *sync.Mutex
	keySignParties     map[common.PubKey]common.PubKeys
	bridge             *ThorchainBridge
}

// NewKeySignPartyMgr create a new instance of KeySignPartyMgr
func NewKeySignPartyMgr(bridge *ThorchainBridge) *KeySignPartyMgr {
	return &KeySignPartyMgr{
		keySignPartiesLock: &sync.Mutex{},
		keySignParties:     make(map[common.PubKey]common.PubKeys),
		bridge:             bridge,
	}
}

// GetKeySignParty is going to return a keysign party for the given pool pubkey
// it check internal cache first , if it is available then user the cached version
// when cache is not available, then it will make a call to thorchain
func (mgr *KeySignPartyMgr) GetKeySignParty(poolPubKey common.PubKey) (common.PubKeys, error) {
	keys, ok := mgr.keySignParties[poolPubKey]
	if ok {
		return keys, nil
	}
	return mgr.bridge.GetKeysignParty(poolPubKey)
}

// SaveKeySignParty is going to save the keysign party to local cache
func (mgr *KeySignPartyMgr) SaveKeySignParty(poolPubKey common.PubKey, keySignParty common.PubKeys) {
	mgr.keySignPartiesLock.Lock()
	defer mgr.keySignPartiesLock.Unlock()
	mgr.keySignParties[poolPubKey] = keySignParty
}

// RemoveKeySignParty remove the cached keysign party that match given pool pubkey from memory
func (mgr *KeySignPartyMgr) RemoveKeySignParty(poolPubKey common.PubKey) {
	mgr.keySignPartiesLock.Lock()
	defer mgr.keySignPartiesLock.Unlock()
	delete(mgr.keySignParties, poolPubKey)
}
