package ethereum

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	ecommon "github.com/ethereum/go-ethereum/common"
	etypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	tssp "gitlab.com/thorchain/tss/go-tss/tss"

	"gitlab.com/thorchain/thornode/bifrost/blockscanner"
	"gitlab.com/thorchain/thornode/bifrost/config"
	"gitlab.com/thorchain/thornode/bifrost/metrics"
	"gitlab.com/thorchain/thornode/bifrost/pkg/chainclients/ethereum/types"
	"gitlab.com/thorchain/thornode/bifrost/thorclient"
	stypes "gitlab.com/thorchain/thornode/bifrost/thorclient/types"
	"gitlab.com/thorchain/thornode/bifrost/tss"
	"gitlab.com/thorchain/thornode/common"
)

// Client is a structure to sign and broadcast tx to Ethereum chain used by signer mostly
type Client struct {
	logger          zerolog.Logger
	cfg             config.ChainConfiguration
	chainID         types.ChainID
	pk              common.PubKey
	client          *ethclient.Client
	kw              *KeySignWrapper
	ethScanner      *BlockScanner
	thorchainBridge *thorclient.ThorchainBridge
	blockScanner    *blockscanner.BlockScanner
	keySignPartyMgr *thorclient.KeySignPartyMgr
}

// NewClient create new instance of Ethereum client
func NewClient(thorKeys *thorclient.Keys, cfg config.ChainConfiguration, server *tssp.TssServer, thorchainBridge *thorclient.ThorchainBridge, m *metrics.Metrics, keySignPartyMgr *thorclient.KeySignPartyMgr) (*Client, error) {
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

	ethPrivateKey, err := getETHPrivateKey(priv)
	if err != nil {
		return nil, err
	}

	keysignWrapper := &KeySignWrapper{
		privKey:       ethPrivateKey,
		pubKey:        pk,
		tssKeyManager: tssKm,
		logger:        log.With().Str("module", "local_signer").Str("chain", common.ETHChain.String()).Logger(),
	}
	ethClient, err := ethclient.Dial(cfg.RPCHost)
	if err != nil {
		return nil, err
	}
	c := &Client{
		logger:          log.With().Str("module", "ethereum").Logger(),
		cfg:             cfg,
		client:          ethClient,
		pk:              pk,
		kw:              keysignWrapper,
		thorchainBridge: thorchainBridge,
		keySignPartyMgr: keySignPartyMgr,
	}
	c.InitChainID()

	var path string // if not set later, will in memory storage
	if len(c.cfg.BlockScanner.DBPath) > 0 {
		path = fmt.Sprintf("%s/%s", c.cfg.BlockScanner.DBPath, c.cfg.BlockScanner.ChainID)
	}
	storage, err := blockscanner.NewBlockScannerStorage(path)
	if err != nil {
		return c, fmt.Errorf("fail to create blockscanner storage: %w", err)
	}

	c.ethScanner, err = NewBlockScanner(c.cfg.BlockScanner, storage, c.chainID, c.client, c.thorchainBridge, m)
	if err != nil {
		return c, fmt.Errorf("fail to create eth block scanner: %w", err)
	}

	c.blockScanner, err = blockscanner.NewBlockScanner(c.cfg.BlockScanner, storage, m, c.thorchainBridge, c.ethScanner)
	if err != nil {
		return c, fmt.Errorf("fail to create block scanner: %w", err)
	}

	return c, nil
}

func (c *Client) Start(globalTxsQueue chan stypes.TxIn, globalErrataQueue chan stypes.ErrataBlock) {
	c.blockScanner.Start(globalTxsQueue)
	c.ethScanner.globalErrataQueue = globalErrataQueue
}

func (c *Client) Stop() {
	c.blockScanner.Stop()
	c.client.Close()
}

func (c *Client) GetConfig() config.ChainConfiguration {
	return c.cfg
}

// IsTestNet determinate whether we are running on test net by checking the status
func (c *Client) InitChainID() {
	chainID, err := c.client.ChainID(context.Background())
	if err != nil {
		c.logger.Error().Err(err).Msg("Unable to get chain id")
		chainID = big.NewInt(types.Localnet)
	}
	c.chainID = types.ChainID(chainID.Int64())
	vByte = byte(int(vByte) + int(2*c.chainID))
	eipSigner = etypes.NewEIP155Signer(chainID)
}

func (c *Client) GetChain() common.Chain {
	return common.ETHChain
}

func (c *Client) GetHeight() (int64, error) {
	return c.ethScanner.GetHeight()
}

// GetAddress return current signer address, it will be bech32 encoded address
func (c *Client) GetAddress(poolPubKey common.PubKey) string {
	addr, err := poolPubKey.GetAddress(common.ETHChain)
	if err != nil {
		c.logger.Error().Err(err).Str("pool_pub_key", poolPubKey.String()).Msg("fail to get pool address")
		return ""
	}
	return addr.String()
}

func (c *Client) GetGasFee(count uint64) common.Gas {
	return common.GetETHGasFee(big.NewInt(1), count)
}

func (c *Client) GetGasPrice() (*big.Int, error) {
	return c.client.SuggestGasPrice(context.Background())
}

func (c *Client) GetNonce(addr string) (uint64, error) {
	nonce, err := c.client.PendingNonceAt(context.Background(), ecommon.HexToAddress(addr))
	if err != nil {
		return 0, fmt.Errorf("fail to get account nonce: %w", err)
	}
	return nonce, nil
}

// SignTx sign the the given TxArrayItem
func (c *Client) SignTx(tx stypes.TxOutItem, height int64) ([]byte, error) {
	toAddr := tx.ToAddress.String()

	value := big.NewInt(0)
	for _, coin := range tx.Coins {
		value.Add(value, coin.Amount.BigInt())
	}
	if len(toAddr) == 0 || value.Uint64() == 0 {
		c.logger.Error().Msg("invalid tx params")
		return nil, nil
	}
	fromAddr := c.GetAddress(tx.VaultPubKey)

	nonce, err := c.GetNonce(fromAddr)
	if err != nil {
		c.logger.Error().Err(err).Msg("fail to fetch latest nonce")
		return nil, err
	}
	c.logger.Info().Uint64("nonce", nonce).Msg("account info")

	gasPrice := c.ethScanner.GetGasPrice()
	gasOut := big.NewInt(0)
	for _, coin := range tx.MaxGas {
		gasOut.Add(gasOut, coin.Amount.BigInt())
	}
	encodedData := []byte(hex.EncodeToString([]byte(tx.Memo)))
	// calculate gas based on memo and gas price and compare against max gas
	gasFee := common.GetETHGasFee(big.NewInt(1), uint64(len(tx.Memo)))[0].Amount.BigInt()
	if gasOut.Cmp(gasFee.Mul(gasFee, gasPrice)) == -1 {
		return nil, fmt.Errorf("not enough max gas: %s", gasOut.String())
	}
	gasOut.Div(gasOut, gasPrice)

	createdTx := etypes.NewTransaction(nonce, ecommon.HexToAddress(toAddr), value, gasOut.Uint64(), gasPrice, encodedData)

	rawTx, err := c.sign(createdTx, fromAddr, tx.VaultPubKey, height, tx)
	if err != nil || len(rawTx) == 0 {
		return nil, fmt.Errorf("fail to sign message: %w", err)
	}
	return rawTx, nil
}

// sign is design to sign a given message with keysign party and keysign wrapper
func (c *Client) sign(tx *etypes.Transaction, from string, poolPubKey common.PubKey, height int64, txOutItem stypes.TxOutItem) ([]byte, error) {
	keySignParty, err := c.keySignPartyMgr.GetKeySignParty(poolPubKey)
	if err != nil {
		c.logger.Error().Err(err).Msg("fail to get keysign party")
		return nil, err
	}
	// let's retry before we give up on the signing party
	retryMax := 3
	var finalErr error
	for i := 0; i < retryMax; i++ {
		rawBytes, err := c.kw.Sign(tx, poolPubKey, keySignParty)
		if err == nil && rawBytes != nil {
			c.keySignPartyMgr.SaveKeySignParty(poolPubKey, keySignParty)
			return rawBytes, nil
		}
		c.keySignPartyMgr.RemoveKeySignParty(poolPubKey)
		finalErr = err
	}
	var keysignError tss.KeysignError
	if errors.As(finalErr, &keysignError) {
		if len(keysignError.Blame.BlameNodes) == 0 {
			// TSS doesn't know which node to blame
			return nil, finalErr
		}
		// key sign error forward the keysign blame to thorchain
		txID, errPostKeysignFail := c.thorchainBridge.PostKeysignFailure(keysignError.Blame, height, txOutItem.Memo, txOutItem.Coins, txOutItem.VaultPubKey)
		if errPostKeysignFail != nil {
			c.logger.Error().Err(errPostKeysignFail).Msg("fail to post keysign failure to thorchain")
			return nil, multierror.Append(finalErr, errPostKeysignFail)
		}
		c.logger.Info().Str("tx_id", txID.String()).Msgf("post keysign failure to thorchain")
	}
	c.logger.Error().Err(err).Msg("fail to sign tx")
	return nil, err
}

// GetAccount gets account by address in eth client
func (c *Client) GetAccount(pkey common.PubKey) (common.Account, error) {
	addr := c.GetAddress(pkey)
	nonce, err := c.GetNonce(addr)
	if err != nil {
		return common.Account{}, err
	}
	balance, err := c.client.BalanceAt(context.Background(), ecommon.HexToAddress(addr), nil)
	if err != nil {
		return common.Account{}, fmt.Errorf("fail to get account nonce: %w", err)
	}
	account := common.NewAccount(int64(nonce), 0, common.AccountCoins{common.AccountCoin{Amount: balance.Uint64(), Denom: "ETH.ETH"}}, false)
	return account, nil
}

func (c *Client) GetAccountByAddress(address string) (common.Account, error) {
	return common.Account{}, nil
}

// BroadcastTx decodes tx using rlp and broadcasts too Ethereum chain
func (c *Client) BroadcastTx(stx stypes.TxOutItem, hexTx []byte) error {
	var tx *etypes.Transaction = &etypes.Transaction{}
	if err := json.Unmarshal(hexTx, tx); err != nil {
		return err
	}
	if err := c.client.SendTransaction(context.Background(), tx); err != nil {
		return err
	}
	return nil
}
