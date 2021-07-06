package bitcoin

import (
	"encoding/base64"
	"math/big"

	"github.com/btcsuite/btcd/btcec"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/tendermint/tendermint/crypto/secp256k1"

	"gitlab.com/thorchain/thornode/bifrost/thorclient"
	"gitlab.com/thorchain/thornode/bifrost/tss"
	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

// TssSignable is a signable implementation backed by tss
type TssSignable struct {
	poolPubKey      common.PubKey
	tssKeyManager   tss.ThorchainKeyManager
	logger          zerolog.Logger
	keySignPartyMgr *thorclient.KeySignPartyMgr
}

// NewTssSignable create a new instance of TssSignable
func NewTssSignable(pubKey common.PubKey, manager tss.ThorchainKeyManager, keySignPartyMgr *thorclient.KeySignPartyMgr) (*TssSignable, error) {
	return &TssSignable{
		poolPubKey:      pubKey,
		tssKeyManager:   manager,
		logger:          log.Logger.With().Str("module", "tss_signable").Logger(),
		keySignPartyMgr: keySignPartyMgr,
	}, nil
}

// Sign the given payload
func (ts *TssSignable) Sign(payload []byte) (*btcec.Signature, error) {
	ts.logger.Debug().Msgf("msg to sign:%s", base64.StdEncoding.EncodeToString(payload))
	keySignParty, err := ts.keySignPartyMgr.GetKeySignParty(ts.poolPubKey)
	if err != nil {
		ts.logger.Error().Err(err).Msg("fail to get keysign party")
		return nil, err
	}
	retry := 3
	var result []byte
	for i := 0; i < retry; i++ {
		result, err = ts.tssKeyManager.RemoteSign(payload, ts.poolPubKey.String(), keySignParty)
		if err == nil {
			break
		}
		ts.keySignPartyMgr.RemoveKeySignParty(ts.poolPubKey)
	}
	if err != nil {
		return nil, err
	}
	var sig btcec.Signature
	sig.R = new(big.Int).SetBytes(result[:32])
	sig.S = new(big.Int).SetBytes(result[32:])
	// let's verify the signature
	if sig.Verify(payload, ts.GetPubKey()) {
		ts.logger.Debug().Msg("we can verify the signature successfully")
	} else {
		ts.logger.Debug().Msg("the signature can't be verified")
	}
	ts.keySignPartyMgr.SaveKeySignParty(ts.poolPubKey, keySignParty)
	return &sig, nil
}

func (ts *TssSignable) GetPubKey() *btcec.PublicKey {
	cpk, err := cosmos.GetPubKeyFromBech32(cosmos.Bech32PubKeyTypeAccPub, ts.poolPubKey.String())
	if err != nil {
		ts.logger.Err(err).Str("pubkey", ts.poolPubKey.String()).Msg("fail to get pubic key from the bech32 pool public key string")
		return nil
	}
	secpPubKey, ok := cpk.(secp256k1.PubKeySecp256k1)
	if !ok {
		ts.logger.Error().Str("pubkey", ts.poolPubKey.String()).Msg("it is not a secp256 k1 public key")
		return nil
	}
	newPubkey, err := btcec.ParsePubKey(secpPubKey[:], btcec.S256())
	if err != nil {
		ts.logger.Err(err).Msg("fail to parse public key")
		return nil
	}
	return newPubkey
}
