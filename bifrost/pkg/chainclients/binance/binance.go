package binance

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/binance-chain/go-sdk/common/types"
	ctypes "github.com/binance-chain/go-sdk/common/types"
	"github.com/binance-chain/go-sdk/keys"
	ttypes "github.com/binance-chain/go-sdk/types"
	"github.com/binance-chain/go-sdk/types/msg"
	btx "github.com/binance-chain/go-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	tssp "gitlab.com/thorchain/tss/go-tss/tss"

	"gitlab.com/thorchain/thornode/bifrost/blockscanner"
	"gitlab.com/thorchain/thornode/bifrost/config"
	"gitlab.com/thorchain/thornode/bifrost/metrics"
	"gitlab.com/thorchain/thornode/bifrost/thorclient"
	stypes "gitlab.com/thorchain/thornode/bifrost/thorclient/types"
	"gitlab.com/thorchain/thornode/bifrost/tss"
	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/x/thorchain"
)

// Binance is a structure to sign and broadcast tx to binance chain used by signer mostly
type Binance struct {
	logger          zerolog.Logger
	cfg             config.ChainConfiguration
	cdc             *codec.Codec
	chainID         string
	isTestNet       bool
	client          *http.Client
	accts           *BinanceMetaDataStore
	tssKeyManager   keys.KeyManager
	localKeyManager *keyManager
	thorchainBridge *thorclient.ThorchainBridge
	storage         *blockscanner.BlockScannerStorage
	blockScanner    *blockscanner.BlockScanner
	bnbScanner      *BinanceBlockScanner
	keysignPartyMgr *thorclient.KeySignPartyMgr
}

// NewBinance create new instance of binance client
func NewBinance(thorKeys *thorclient.Keys, cfg config.ChainConfiguration, server *tssp.TssServer, thorchainBridge *thorclient.ThorchainBridge, m *metrics.Metrics, keySignPartyMgr *thorclient.KeySignPartyMgr) (*Binance, error) {
	tssKm, err := tss.NewKeySign(server)
	if err != nil {
		return nil, fmt.Errorf("fail to create tss signer: %w", err)
	}

	priv, err := thorKeys.GetPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("fail to get private key: %w", err)
	}

	pk, err := common.NewPubKeyFromCrypto(priv.PubKey())
	if err != nil {
		return nil, fmt.Errorf("fail to get pub key: %w", err)
	}
	if thorchainBridge == nil {
		return nil, errors.New("thorchain bridge is nil")
	}
	localKm := &keyManager{
		privKey: priv,
		addr:    ctypes.AccAddress(priv.PubKey().Address()),
		pubkey:  pk,
	}

	b := &Binance{
		logger:          log.With().Str("module", "binance").Logger(),
		cfg:             cfg,
		cdc:             thorclient.MakeCodec(),
		accts:           NewBinanceMetaDataStore(),
		client:          &http.Client{},
		tssKeyManager:   tssKm,
		localKeyManager: localKm,
		thorchainBridge: thorchainBridge,
		keysignPartyMgr: keySignPartyMgr,
	}

	if err := b.checkIsTestNet(); err != nil {
		b.logger.Error().Err(err).Msg("fail to check if is testnet")
		return b, err
	}

	var path string // if not set later, will in memory storage
	if len(b.cfg.BlockScanner.DBPath) > 0 {
		path = fmt.Sprintf("%s/%s", b.cfg.BlockScanner.DBPath, b.cfg.BlockScanner.ChainID)
	}
	b.storage, err = blockscanner.NewBlockScannerStorage(path)
	if err != nil {
		return nil, fmt.Errorf("fail to create scan storage: %w", err)
	}

	b.bnbScanner, err = NewBinanceBlockScanner(b.cfg.BlockScanner, b.storage, b.isTestNet, b.thorchainBridge, m)
	if err != nil {
		return nil, fmt.Errorf("fail to create block scanner: %w", err)
	}

	b.blockScanner, err = blockscanner.NewBlockScanner(b.cfg.BlockScanner, b.storage, m, b.thorchainBridge, b.bnbScanner)
	if err != nil {
		return nil, fmt.Errorf("fail to create block scanner: %w", err)
	}

	return b, nil
}

// Start Binance chain client
func (b *Binance) Start(globalTxsQueue chan stypes.TxIn, globalErrataQueue chan stypes.ErrataBlock) {
	b.blockScanner.Start(globalTxsQueue)
}

// Stop Binance chain client
func (b *Binance) Stop() {
	b.blockScanner.Stop()
}

// GetConfig return the configuration used by Binance chain client
func (b *Binance) GetConfig() config.ChainConfiguration {
	return b.cfg
}

// checkIsTestNet determinate whether we are running on test net by checking the status
func (b *Binance) checkIsTestNet() error {
	// Cached data after first call
	if b.isTestNet {
		return nil
	}

	u, err := url.Parse(b.cfg.RPCHost)
	if err != nil {
		return fmt.Errorf("unable to parse rpc host: %s: %w", b.cfg.RPCHost, err)
	}

	u.Path = "/status"

	resp, err := b.client.Get(u.String())
	if err != nil {
		return err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			b.logger.Error().Err(err).Msg("fail to close resp body")
		}
	}()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal().Err(err).Msg("fail to read body")
	}

	type Status struct {
		Jsonrpc string `json:"jsonrpc"`
		ID      string `json:"id"`
		Result  struct {
			NodeInfo struct {
				Network string `json:"network"`
			} `json:"node_info"`
		} `json:"result"`
	}

	var status Status
	if err := json.Unmarshal(data, &status); err != nil {
		return fmt.Errorf("fail to unmarshal body: %w", err)
	}

	b.chainID = status.Result.NodeInfo.Network
	b.isTestNet = b.chainID == "Binance-Chain-Ganges"

	if b.isTestNet {
		types.Network = types.TestNetwork
	} else {
		types.Network = types.ProdNetwork
	}

	return nil
}

func (b *Binance) GetChain() common.Chain {
	return common.BNBChain
}

func (b *Binance) GetHeight() (int64, error) {
	return b.bnbScanner.GetHeight()
}

func (b *Binance) input(addr types.AccAddress, coins types.Coins) msg.Input {
	return msg.Input{
		Address: addr,
		Coins:   coins,
	}
}

func (b *Binance) output(addr types.AccAddress, coins types.Coins) msg.Output {
	return msg.Output{
		Address: addr,
		Coins:   coins,
	}
}

func (b *Binance) msgToSend(in []msg.Input, out []msg.Output) msg.SendMsg {
	return msg.SendMsg{Inputs: in, Outputs: out}
}

func (b *Binance) createMsg(from types.AccAddress, fromCoins types.Coins, transfers []msg.Transfer) msg.SendMsg {
	input := b.input(from, fromCoins)
	output := make([]msg.Output, 0, len(transfers))
	for _, t := range transfers {
		t.Coins = t.Coins.Sort()
		output = append(output, b.output(t.ToAddr, t.Coins))
	}
	return b.msgToSend([]msg.Input{input}, output)
}

func (b *Binance) parseTx(fromAddr string, transfers []msg.Transfer) msg.SendMsg {
	addr, err := types.AccAddressFromBech32(fromAddr)
	if err != nil {
		b.logger.Error().Str("address", fromAddr).Err(err).Msg("fail to parse address")
	}
	fromCoins := types.Coins{}
	for _, t := range transfers {
		t.Coins = t.Coins.Sort()
		fromCoins = fromCoins.Plus(t.Coins)
	}
	return b.createMsg(addr, fromCoins, transfers)
}

// GetAddress return current signer address, it will be bech32 encoded address
func (b *Binance) GetAddress(poolPubKey common.PubKey) string {
	addr, err := poolPubKey.GetAddress(common.BNBChain)
	if err != nil {
		b.logger.Error().Err(err).Str("pool_pub_key", poolPubKey.String()).Msg("fail to get pool address")
		return ""
	}
	return addr.String()
}

func (b *Binance) getGasFee(count uint64) common.Gas {
	coins := make(common.Coins, count)
	gasInfo := []cosmos.Uint{
		cosmos.NewUint(b.bnbScanner.singleFee),
		cosmos.NewUint(b.bnbScanner.multiFee),
	}
	return common.CalcBinanceGasPrice(common.Tx{Coins: coins}, common.BNBAsset, gasInfo)
}

// SignTx sign the the given TxArrayItem
func (b *Binance) SignTx(tx stypes.TxOutItem, thorchainHeight int64) ([]byte, error) {
	var payload []msg.Transfer

	toAddr, err := types.AccAddressFromBech32(tx.ToAddress.String())
	if err != nil {
		return nil, fmt.Errorf("fail to parse account address(%s) :%w", tx.ToAddress.String(), err)
	}

	var gasCoin common.Coins

	// for yggdrasil, need to left some coin to pay for fee, this logic is per chain, given different chain charge fees differently
	if strings.EqualFold(tx.Memo, thorchain.NewYggdrasilReturn(thorchainHeight).String()) {
		gas := b.getGasFee(uint64(len(tx.Coins)))
		gasCoin = gas.ToCoins()
	}
	var coins types.Coins
	for _, coin := range tx.Coins {
		// deduct gas coin
		for _, gc := range gasCoin {
			if coin.Asset.Equals(gc.Asset) {
				coin.Amount = common.SafeSub(coin.Amount, gc.Amount)
			}
		}

		coins = append(coins, types.Coin{
			Denom:  coin.Asset.Symbol.String(),
			Amount: int64(coin.Amount.Uint64()),
		})
	}

	payload = append(payload, msg.Transfer{
		ToAddr: toAddr,
		Coins:  coins,
	})

	if len(payload) == 0 {
		b.logger.Error().Msg("payload is empty , this should not happen")
		return nil, nil
	}
	fromAddr := b.GetAddress(tx.VaultPubKey)
	sendMsg := b.parseTx(fromAddr, payload)
	if err := sendMsg.ValidateBasic(); err != nil {
		return nil, fmt.Errorf("invalid send msg: %w", err)
	}

	currentHeight, err := b.bnbScanner.GetHeight()
	if err != nil {
		b.logger.Error().Err(err).Msg("fail to get current binance block height")
		return nil, err
	}
	meta := b.accts.Get(tx.VaultPubKey)
	if currentHeight > meta.BlockHeight {
		acc, err := b.GetAccount(tx.VaultPubKey)
		if err != nil {
			return nil, fmt.Errorf("fail to get account info: %w", err)
		}
		meta = BinanceMetadata{
			AccountNumber: acc.AccountNumber,
			SeqNumber:     acc.Sequence,
			BlockHeight:   currentHeight,
		}
		b.accts.Set(tx.VaultPubKey, meta)
	}
	b.logger.Info().Int64("account_number", meta.AccountNumber).Int64("sequence_number", meta.SeqNumber).Int64("block height", meta.BlockHeight).Msg("account info")
	signMsg := btx.StdSignMsg{
		ChainID:       b.chainID,
		Memo:          tx.Memo,
		Msgs:          []msg.Msg{sendMsg},
		Source:        btx.Source,
		Sequence:      meta.SeqNumber,
		AccountNumber: meta.AccountNumber,
	}
	rawBz, err := b.signMsg(signMsg, fromAddr, tx.VaultPubKey, thorchainHeight, tx)
	if err != nil {
		return nil, fmt.Errorf("fail to sign message: %w", err)
	}

	if len(rawBz) == 0 {
		b.logger.Warn().Msg("this should not happen, the message is empty")
		// the transaction was already signed
		return nil, nil
	}

	hexTx := []byte(hex.EncodeToString(rawBz))
	return hexTx, nil
}

func (b *Binance) sign(signMsg btx.StdSignMsg, poolPubKey common.PubKey, signerPubKeys common.PubKeys) ([]byte, error) {
	if b.localKeyManager.Pubkey().Equals(poolPubKey) {
		return b.localKeyManager.Sign(signMsg)
	}
	k := b.tssKeyManager.(tss.ThorchainKeyManager)
	return k.SignWithPool(signMsg, poolPubKey, signerPubKeys)
}

// signMsg is design to sign a given message until it success or the same message had been send out by other signer
func (b *Binance) signMsg(signMsg btx.StdSignMsg, from string, poolPubKey common.PubKey, thorchainHeight int64, txOutItem stypes.TxOutItem) ([]byte, error) {
	keySignParty, err := b.keysignPartyMgr.GetKeySignParty(poolPubKey)
	if err != nil {
		b.logger.Error().Err(err).Msg("fail to get keysign party")
		return nil, err
	}
	// let's retry before we give up on the signing party
	retryMax := 3
	var finalErr error
	for i := 0; i < retryMax; i++ {
		rawBytes, err := b.sign(signMsg, poolPubKey, keySignParty)
		if err == nil && rawBytes != nil {
			b.keysignPartyMgr.SaveKeySignParty(poolPubKey, keySignParty)
			return rawBytes, nil
		}
		b.keysignPartyMgr.RemoveKeySignParty(poolPubKey)
		finalErr = err
	}
	var keysignError tss.KeysignError
	if errors.As(finalErr, &keysignError) {
		if len(keysignError.Blame.BlameNodes) == 0 {
			// TSS doesn't know which node to blame
			return nil, finalErr
		}

		// key sign error forward the keysign blame to thorchain
		txID, errPostKeysignFail := b.thorchainBridge.PostKeysignFailure(keysignError.Blame, thorchainHeight, txOutItem.Memo, txOutItem.Coins, poolPubKey)
		if errPostKeysignFail != nil {
			b.logger.Error().Err(errPostKeysignFail).Msg("fail to post keysign failure to thorchain")
			return nil, multierror.Append(finalErr, errPostKeysignFail)
		}
		b.logger.Info().Str("tx_id", txID.String()).Msgf("post keysign failure to thorchain")
		// back off a block time, so it has more chance to pick up the updated signer party
		time.Sleep(time.Second * 5)
	}
	b.logger.Error().Err(finalErr).Msgf("fail to sign msg with memo: %s", signMsg.Memo)
	return nil, finalErr
}

func (b *Binance) GetAccount(pkey common.PubKey) (common.Account, error) {
	addr := b.GetAddress(pkey)
	address, err := types.AccAddressFromBech32(addr)
	if err != nil {
		b.logger.Error().Err(err).Msgf("fail to get parse address: %s", addr)
		return common.Account{}, err
	}
	return b.GetAccountByAddress(address.String())
}

func (b *Binance) GetAccountByAddress(address string) (common.Account, error) {
	u, err := url.Parse(b.cfg.RPCHost)
	if err != nil {
		log.Fatal().Msgf("Error parsing rpc (%s): %s", b.cfg.RPCHost, err)
		return common.Account{}, err
	}
	u.Path = "/abci_query"
	v := u.Query()
	v.Set("path", fmt.Sprintf("\"/account/%s\"", address))
	u.RawQuery = v.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return common.Account{}, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			b.logger.Error().Err(err).Msg("fail to close response body")
		}
	}()

	type queryResult struct {
		Jsonrpc string `json:"jsonrpc"`
		ID      string `json:"id"`
		Result  struct {
			Response struct {
				Key         string `json:"key"`
				Value       string `json:"value"`
				BlockHeight string `json:"height"`
			} `json:"response"`
		} `json:"result"`
	}

	var result queryResult
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return common.Account{}, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return common.Account{}, err
	}

	data, err := base64.StdEncoding.DecodeString(result.Result.Response.Value)
	if err != nil {
		return common.Account{}, err
	}

	cdc := ttypes.NewCodec()
	var acc types.AppAccount
	err = cdc.UnmarshalBinaryBare(data, &acc)
	if err != nil {
		return common.Account{}, err
	}
	account := common.NewAccount(acc.BaseAccount.Sequence, acc.BaseAccount.AccountNumber, common.GetCoins(acc.BaseAccount.Coins), acc.Flags > 0)
	return account, nil
}

// broadcastTx is to broadcast the tx to binance chain
func (b *Binance) BroadcastTx(tx stypes.TxOutItem, hexTx []byte) error {
	u, err := url.Parse(b.cfg.RPCHost)
	if err != nil {
		log.Error().Msgf("Error parsing rpc (%s): %s", b.cfg.RPCHost, err)
		return err
	}
	u.Path = "broadcast_tx_commit"
	values := u.Query()
	values.Set("tx", "0x"+string(hexTx))
	u.RawQuery = values.Encode()
	resp, err := http.Post(u.String(), "", nil)
	if err != nil {
		return fmt.Errorf("fail to broadcast tx to binance chain: %w", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("fail to read response body: %w", err)
	}

	// NOTE: we can actually see two different json responses for the same end.
	// This complicates things pretty well.
	// Sample 1: { "height": "0", "txhash": "D97E8A81417E293F5B28DDB53A4AD87B434CA30F51D683DA758ECC2168A7A005", "raw_log": "[{\"msg_index\":0,\"success\":true,\"log\":\"\",\"events\":[{\"type\":\"message\",\"attributes\":[{\"key\":\"action\",\"value\":\"set_observed_txout\"}]}]}]", "logs": [ { "msg_index": 0, "success": true, "log": "", "events": [ { "type": "message", "attributes": [ { "key": "action", "value": "set_observed_txout" } ] } ] } ] }
	// Sample 2: { "height": "0", "txhash": "6A9AA734374D567D1FFA794134A66D3BF614C4EE5DDF334F21A52A47C188A6A2", "code": 4, "raw_log": "{\"codespace\":\"sdk\",\"code\":4,\"message\":\"signature verification failed; verify correct account sequence and chain-id\"}" }
	// Sample 3: {\"jsonrpc\": \"2.0\",\"id\": \"\",\"result\": {  \"check_tx\": {    \"code\": 65541,    \"log\": \"{\\\"codespace\\\":1,\\\"code\\\":5,\\\"abci_code\\\":65541,\\\"message\\\":\\\"insufficient fund. you got 29602BNB,351873676FSN-F1B,1094620960FTM-585,10119750400LOK-3C0,191723639522RUNE-67C,13629773TATIC-E9C,4169469575TCAN-014,10648250188TOMOB-1E1,1155074377TUSDB-000, but 37500BNB fee needed.\\\"}\",    \"events\": [      {}    ]  },  \"deliver_tx\": {},  \"hash\": \"406A3F68B17544F359DF8C94D4E28A626D249BC9C4118B51F7B4CE16D45AF616\",  \"height\": \"0\"}\n}

	b.logger.Info().Str("body", string(body)).Msgf("broadcast response from Binance Chain,memo:%s", tx.Memo)
	var commit stypes.BroadcastResult
	err = b.cdc.UnmarshalJSON(body, &commit)
	if err != nil {
		b.logger.Error().Err(err).Msgf("fail unmarshal commit: %s", string(body))
		return fmt.Errorf("fail to unmarshal commit: %w", err)
	}
	// check for any failure logs
	// Error code 4 is used for bad account sequence number. We expect to
	// see this often because in TSS, multiple nodes will broadcast the
	// same sequence number but only one will be successful. We can just
	// drop and ignore in these scenarios. In 1of1 signing, we can also
	// drop and ignore. The reason being, thorchain will attempt to again
	// later.
	checkTx := commit.Result.CheckTx
	if checkTx.Code > 0 && checkTx.Code != cosmos.CodeUnauthorized {
		err := errors.New(checkTx.Log)
		b.logger.Info().Str("body", string(body)).Msg("broadcast response from Binance Chain")
		b.logger.Error().Err(err).Msg("fail to broadcast")
		return fmt.Errorf("fail to broadcast: %w", err)
	}

	deliverTx := commit.Result.DeliverTx
	if deliverTx.Code > 0 {
		err := errors.New(deliverTx.Log)
		b.logger.Error().Err(err).Msg("fail to broadcast")
		return fmt.Errorf("fail to broadcast: %w", err)
	}

	// increment sequence number
	b.accts.SeqInc(tx.VaultPubKey)

	return nil
}
