package tss

import (
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gitlab.com/thorchain/tss/go-tss/blame"
	"gitlab.com/thorchain/tss/go-tss/keygen"
	"gitlab.com/thorchain/tss/go-tss/tss"

	"gitlab.com/thorchain/thornode/bifrost/thorclient"
	"gitlab.com/thorchain/thornode/common"
)

// KeyGen is
type KeyGen struct {
	keys   *thorclient.Keys
	logger zerolog.Logger
	client *http.Client
	server *tss.TssServer
}

// NewTssKeyGen create a new instance of TssKeyGen which will look after TSS key stuff
func NewTssKeyGen(keys *thorclient.Keys, server *tss.TssServer) (*KeyGen, error) {
	if keys == nil {
		return nil, fmt.Errorf("keys is nil")
	}
	return &KeyGen{
		keys:   keys,
		logger: log.With().Str("module", "tss_keygen").Logger(),
		client: &http.Client{
			Timeout: time.Second * 130,
		},
		server: server,
	}, nil
}

func (kg *KeyGen) GenerateNewKey(pKeys common.PubKeys) (common.PubKeySet, blame.Blame, error) {
	// No need to do key gen
	if len(pKeys) == 0 {
		return common.EmptyPubKeySet, blame.Blame{}, nil
	}
	var keys []string
	for _, item := range pKeys {
		keys = append(keys, item.String())
	}
	keyGenReq := keygen.Request{
		Keys: keys,
	}

	ch := make(chan bool, 1)
	defer close(ch)
	timer := time.NewTimer(30 * time.Minute)
	defer timer.Stop()

	var resp keygen.Response
	var err error
	go func() {
		resp, err = kg.server.Keygen(keyGenReq)
		ch <- true
	}()

	select {
	case <-ch:
		// do nothing
	case <-timer.C:
		panic("tss keygen timeout")
	}

	if err != nil {
		// the resp from kg.server.Keygen will not be nil
		if resp.Blame.IsEmpty() {
			resp.Blame.FailReason = err.Error()
		}
		return common.EmptyPubKeySet, resp.Blame, fmt.Errorf("fail to keygen,err:%w", err)
	}

	cpk, err := common.NewPubKey(resp.PubKey)
	if err != nil {
		return common.EmptyPubKeySet, resp.Blame, fmt.Errorf("fail to create common.PubKey,%w", err)
	}

	// TODO later on THORNode need to have both secp256k1 key and ed25519
	return common.NewPubKeySet(cpk, cpk), resp.Blame, nil
}
