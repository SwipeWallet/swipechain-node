package observer

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"gitlab.com/thorchain/thornode/bifrost/metrics"
	"gitlab.com/thorchain/thornode/bifrost/pkg/chainclients"
	"gitlab.com/thorchain/thornode/bifrost/pubkeymanager"
	"gitlab.com/thorchain/thornode/bifrost/thorclient"
	"gitlab.com/thorchain/thornode/bifrost/thorclient/types"
	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/constants"
	stypes "gitlab.com/thorchain/thornode/x/thorchain/types"
)

const maxTxArrayLen = 100

// Observer observer service
type Observer struct {
	logger            zerolog.Logger
	chains            map[common.Chain]chainclients.ChainClient
	stopChan          chan struct{}
	pubkeyMgr         pubkeymanager.PubKeyValidator
	onDeck            []types.TxIn
	lock              *sync.Mutex
	globalTxsQueue    chan types.TxIn
	globalErrataQueue chan types.ErrataBlock
	m                 *metrics.Metrics
	errCounter        *prometheus.CounterVec
	thorchainBridge   *thorclient.ThorchainBridge
}

// NewObserver create a new instance of Observer for chain
func NewObserver(pubkeyMgr pubkeymanager.PubKeyValidator, chains map[common.Chain]chainclients.ChainClient, thorchainBridge *thorclient.ThorchainBridge, m *metrics.Metrics) (*Observer, error) {
	logger := log.Logger.With().Str("module", "observer").Logger()
	return &Observer{
		logger:            logger,
		chains:            chains,
		stopChan:          make(chan struct{}),
		m:                 m,
		pubkeyMgr:         pubkeyMgr,
		lock:              &sync.Mutex{},
		globalTxsQueue:    make(chan types.TxIn),
		globalErrataQueue: make(chan types.ErrataBlock),
		errCounter:        m.GetCounterVec(metrics.ObserverError),
		thorchainBridge:   thorchainBridge,
	}, nil
}

func (o *Observer) getChain(chainID common.Chain) (chainclients.ChainClient, error) {
	chain, ok := o.chains[chainID]
	if !ok {
		o.logger.Debug().Str("chain", chainID.String()).Msg("is not supported yet")
		return nil, errors.New("Not supported")
	}
	return chain, nil
}

func (o *Observer) Start() error {
	for _, chain := range o.chains {
		chain.Start(o.globalTxsQueue, o.globalErrataQueue)
	}
	go o.processTxIns()
	go o.processErrataTx()
	go o.deck()
	return nil
}

func (o *Observer) deck() {
	for {
		select {
		case <-o.stopChan:
			o.sendDeck()
			return
		case <-time.After(constants.ThorchainBlockTime):
			o.sendDeck()
		}
	}
}

func (o *Observer) sendDeck() {
	o.lock.Lock()
	defer o.lock.Unlock()
	for i, deck := range o.onDeck {
		deck.TxArray = o.filterObservations(deck.Chain, deck.TxArray)
		deck.TxArray = o.filterBinanceMemoFlag(deck.Chain, deck.TxArray)
		for _, txIn := range o.chunkify(deck) {
			if err := o.signAndSendToThorchain(txIn); err != nil {
				o.logger.Error().Err(err).Msg("fail to send to thorchain")
				// retry later
				o.onDeck[i].TxArray = append(o.onDeck[i].TxArray, txIn.TxArray...)
				continue
			}
			// check if chain client has OnObservedTxIn method then call it
			chainClient, err := o.getChain(txIn.Chain)
			if err != nil {
				o.logger.Error().Err(err).Msg("fail to retrieve chain client")
				continue
			}

			i, ok := chainClient.(interface {
				OnObservedTxIn(txIn types.TxInItem, blockHeight int64)
			})
			if ok {
				for _, item := range txIn.TxArray {
					if o.isOutboundMsg(txIn.Chain, item.Sender) {
						continue
					}
					i.OnObservedTxIn(item, item.BlockHeight)
				}
			}
		}
	}
	o.onDeck = make([]types.TxIn, 0)
}

func (o *Observer) processTxIns() {
	for {
		select {
		case <-o.stopChan:
			return
		case txIn := <-o.globalTxsQueue:
			o.lock.Lock()
			found := false
			for i, in := range o.onDeck {
				if in.Chain == txIn.Chain {
					o.onDeck[i].TxArray = append(o.onDeck[i].TxArray, txIn.TxArray...)
					found = true
				}
			}
			if !found {
				o.onDeck = append(o.onDeck, txIn)
			}
			o.lock.Unlock()
		}
	}
}

func (o *Observer) isOutboundMsg(chain common.Chain, fromAddr string) bool {
	matchOutbound, _ := o.pubkeyMgr.IsValidPoolAddress(fromAddr, chain)
	if matchOutbound {
		return true
	}
	return false
}

// chunkify - breaks the observations into 100 transactions per observation
func (o *Observer) chunkify(txIn types.TxIn) (result []types.TxIn) {
	// sort it by block height
	sort.SliceStable(txIn.TxArray, func(i, j int) bool {
		return txIn.TxArray[i].BlockHeight < txIn.TxArray[j].BlockHeight
	})
	for len(txIn.TxArray) > 0 {
		newTx := types.TxIn{
			Chain: txIn.Chain,
		}
		if len(txIn.TxArray) > maxTxArrayLen {
			newTx.Count = fmt.Sprintf("%d", maxTxArrayLen)
			newTx.TxArray = txIn.TxArray[:maxTxArrayLen]
			txIn.TxArray = txIn.TxArray[maxTxArrayLen:]
		} else {
			newTx.Count = fmt.Sprintf("%d", len(txIn.TxArray))
			newTx.TxArray = txIn.TxArray
			txIn.TxArray = nil
		}
		result = append(result, newTx)
	}
	return result
}

func (o *Observer) filterObservations(chain common.Chain, items []types.TxInItem) (txs []types.TxInItem) {
	for _, txInItem := range items {
		// NOTE: the following could result in the same tx being added
		// twice, which is expected. We want to make sure we generate both
		// a inbound and outbound txn, if we both apply.

		// check if the from address is a valid pool
		if ok, cpi := o.pubkeyMgr.IsValidPoolAddress(txInItem.Sender, chain); ok {
			txInItem.ObservedVaultPubKey = cpi.PubKey
			txs = append(txs, txInItem)
		}
		// check if the to address is a valid pool address
		if ok, cpi := o.pubkeyMgr.IsValidPoolAddress(txInItem.To, chain); ok {
			txInItem.ObservedVaultPubKey = cpi.PubKey
			txs = append(txs, txInItem)
		}
	}
	return
}

// filterBinanceMemoFlag - on Binance chain , BEP12(https://github.com/binance-chain/BEPs/blob/master/BEP12.md#memo-check-script-for-transfer)
// it allow account to enable memo check flag, with the flag enabled , if a tx doesn't have memo, or doesn't have correct memo will be rejected by the chain ,
// unfortunately THORChain won't be able to deal with these accounts , as THORChain will not know what kind of memo it required to send the tx through
// given that Bifrost have to filter out those txes
// the logic has to be here as THORChain is chain agnostic , customer can swap from BTC/ETC to BNB
func (o *Observer) filterBinanceMemoFlag(chain common.Chain, items []types.TxInItem) (txs []types.TxInItem) {
	bnbClient, ok := o.chains[common.BNBChain]
	if !ok {
		txs = items
		return
	}
	for _, txInItem := range items {
		var addressesToCheck []string
		addr := txInItem.GetAddressToCheck()
		if !addr.IsEmpty() && addr.IsChain(common.BNBChain) {
			addressesToCheck = append(addressesToCheck, addr.String())
		}
		// if it BNB chain let's check the from address as well
		if chain.Equals(common.BNBChain) {
			addressesToCheck = append(addressesToCheck, txInItem.Sender)
		}
		skip := false
		for _, item := range addressesToCheck {
			account, err := bnbClient.GetAccountByAddress(item)
			if err != nil {
				o.logger.Error().Err(err).Msgf("fail to check account for %s", item)
				continue
			}
			if account.HasMemoFlag {
				skip = true
				break
			}
		}
		if !skip {
			txs = append(txs, txInItem)
		}
	}
	return
}

func (o *Observer) processErrataTx() {
	for {
		select {
		case <-o.stopChan:
			return
		case errataBlock, more := <-o.globalErrataQueue:
			if !more {
				return
			}
			o.logger.Info().Msgf("Received a errata block %+v from the Thorchain", errataBlock.Height)
			for _, errataTx := range errataBlock.Txs {
				if err := o.sendErrataTxToThorchain(errataBlock.Height, errataTx.TxID, errataTx.Chain); err != nil {
					o.errCounter.WithLabelValues("fail_to_broadcast_errata_tx", "").Inc()
					o.logger.Error().Err(err).Msg("fail to broadcast errata tx")
				}
			}
		}
	}
}

func (o *Observer) sendErrataTxToThorchain(height int64, txID common.TxID, chain common.Chain) error {
	stdTx, err := o.thorchainBridge.GetErrataStdTx(txID, chain)
	strHeight := strconv.FormatInt(height, 10)
	if err != nil {
		o.errCounter.WithLabelValues("fail_to_sign", strHeight).Inc()
		return fmt.Errorf("fail to sign the tx: %w", err)
	}
	txID, err = o.thorchainBridge.Broadcast(*stdTx, types.TxSync)
	if err != nil {
		o.errCounter.WithLabelValues("fail_to_send_to_thorchain", strHeight).Inc()
		return fmt.Errorf("fail to send the tx to thorchain: %w", err)
	}
	o.logger.Info().Int64("block", height).Str("thorchain hash", txID.String()).Msg("sign and send to thorchain successfully")
	return nil
}

func (o *Observer) signAndSendToThorchain(txIn types.TxIn) error {
	nodeStatus, err := o.thorchainBridge.FetchNodeStatus()
	if err != nil {
		return fmt.Errorf("failed to get node status: %w", err)
	}
	if nodeStatus != stypes.Active {
		return nil
	}
	txs, err := o.getThorchainTxIns(txIn)
	if err != nil {
		return fmt.Errorf("fail to convert txin to thorchain txin: %w", err)
	}
	stdTx, err := o.thorchainBridge.GetObservationsStdTx(txs)
	if err != nil {
		return fmt.Errorf("fail to sign the tx: %w", err)
	}
	bf := backoff.NewExponentialBackOff()
	return backoff.Retry(func() error {
		txID, err := o.thorchainBridge.Broadcast(*stdTx, types.TxSync)
		if err != nil {
			return fmt.Errorf("fail to send the tx to thorchain: %w", err)
		}
		o.logger.Info().Str("thorchain hash", txID.String()).Msg("sign and send to thorchain successfully")
		return nil
	}, bf)
}

// getThorchainTxIns convert to the type thorchain expected
// maybe in later THORNode can just refactor this to use the type in thorchain
func (o *Observer) getThorchainTxIns(txIn types.TxIn) (stypes.ObservedTxs, error) {
	txs := make(stypes.ObservedTxs, len(txIn.TxArray))
	o.logger.Debug().Msgf("len %d", len(txIn.TxArray))
	for i, item := range txIn.TxArray {
		o.logger.Debug().Str("tx-hash", item.Tx).Msg("txInItem")
		blockHeight := strconv.FormatInt(item.BlockHeight, 10)
		txID, err := common.NewTxID(item.Tx)
		if err != nil {
			o.errCounter.WithLabelValues("fail_to_parse_tx_hash", blockHeight).Inc()
			return nil, fmt.Errorf("fail to parse tx hash, %s is invalid: %w", item.Tx, err)
		}
		sender, err := common.NewAddress(item.Sender)
		if err != nil {
			o.errCounter.WithLabelValues("fail_to_parse_sender", item.Sender).Inc()
			return nil, fmt.Errorf("fail to parse sender,%s is invalid sender address: %w", item.Sender, err)
		}

		to, err := common.NewAddress(item.To)
		if err != nil {
			o.errCounter.WithLabelValues("fail_to_parse_sender", item.Sender).Inc()
			return nil, fmt.Errorf("fail to parse sender,%s is invalid sender address: %w", item.Sender, err)
		}

		o.logger.Debug().Msgf("pool pubkey %s", item.ObservedVaultPubKey)
		chainAddr, _ := item.ObservedVaultPubKey.GetAddress(txIn.Chain)
		o.logger.Debug().Msgf("%s address %s", txIn.Chain.String(), chainAddr)
		if err != nil {
			o.errCounter.WithLabelValues("fail to parse observed pool address", item.ObservedVaultPubKey.String()).Inc()
			return nil, fmt.Errorf("fail to parse observed pool address: %s: %w", item.ObservedVaultPubKey.String(), err)
		}
		txs[i] = stypes.NewObservedTx(
			common.NewTx(txID, sender, to, item.Coins, item.Gas, item.Memo),
			item.BlockHeight,
			item.ObservedVaultPubKey,
		)
	}
	return txs, nil
}

// Stop the observer
func (o *Observer) Stop() error {
	o.logger.Debug().Msg("request to stop observer")
	defer o.logger.Debug().Msg("observer stopped")

	for _, chain := range o.chains {
		chain.Stop()
	}

	close(o.stopChan)
	if err := o.pubkeyMgr.Stop(); err != nil {
		o.logger.Error().Err(err).Msg("fail to stop pool address manager")
	}
	return o.m.Stop()
}
