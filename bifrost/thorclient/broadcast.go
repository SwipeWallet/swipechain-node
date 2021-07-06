package thorclient

import (
	"bytes"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"gitlab.com/thorchain/thornode/bifrost/metrics"
	"gitlab.com/thorchain/thornode/bifrost/thorclient/types"
	"gitlab.com/thorchain/thornode/common"
)

// Broadcast Broadcasts tx to thorchain
func (b *ThorchainBridge) Broadcast(stdTx authtypes.StdTx, mode types.TxMode) (common.TxID, error) {
	b.broadcastLock.Lock()
	defer b.broadcastLock.Unlock()

	noTxID := common.TxID("")
	if !mode.IsValid() {
		return noTxID, errors.New(fmt.Sprintf("transaction Mode (%s) is invalid", mode))
	}
	start := time.Now()
	defer func() {
		b.m.GetHistograms(metrics.SendToThorchainDuration).Observe(time.Since(start).Seconds())
	}()

	blockHeight, err := b.GetBlockHeight()
	if err != nil {
		return noTxID, err
	}
	if blockHeight > b.blockHeight {
		var seqNum uint64
		b.accountNumber, seqNum, err = b.getAccountNumberAndSequenceNumber()
		if err != nil {
			return noTxID, fmt.Errorf("fail to get account number and sequence number from thorchain : %w", err)
		}
		b.blockHeight = blockHeight
		if seqNum > b.seqNumber {
			b.seqNumber = seqNum
		}
	}

	b.logger.Info().Uint64("account_number", b.accountNumber).Uint64("sequence_number", b.seqNumber).Msg("account info")
	stdMsg := authtypes.StdSignMsg{
		ChainID:       string(b.cfg.ChainID),
		AccountNumber: b.accountNumber,
		Sequence:      b.seqNumber,
		Fee:           stdTx.Fee,
		Msgs:          stdTx.GetMsgs(),
		Memo:          stdTx.GetMemo(),
	}
	sig, err := authtypes.MakeSignature(b.keys.GetKeybase(), b.cfg.SignerName, b.cfg.SignerPasswd, stdMsg)
	if err != nil {
		b.errCounter.WithLabelValues("fail_sign", "").Inc()
		return noTxID, fmt.Errorf("fail to sign the message: %w", err)
	}

	signed := authtypes.NewStdTx(
		stdTx.GetMsgs(),
		stdTx.Fee,
		[]authtypes.StdSignature{sig},
		stdTx.GetMemo(),
	)

	b.m.GetCounter(metrics.TxToThorchainSigned).Inc()

	var setTx types.SetTx
	setTx.Mode = mode.String()
	setTx.Tx.Msg = signed.Msgs
	setTx.Tx.Fee = signed.Fee
	setTx.Tx.Signatures = signed.Signatures
	setTx.Tx.Memo = signed.Memo
	result, err := b.cdc.MarshalJSON(setTx)
	if err != nil {
		b.errCounter.WithLabelValues("fail_marshal_settx", "").Inc()
		return noTxID, fmt.Errorf("fail to marshal settx to json: %w", err)
	}

	b.logger.Info().Int("size", len(result)).Str("payload", string(result)).Msg("post to thorchain")
	body, err := b.post(BroadcastTxsEndpoint, "application/json", bytes.NewBuffer(result))
	if err != nil {
		return noTxID, fmt.Errorf("fail to post tx to thorchain: %w", err)
	}
	var commit sdk.TxResponse
	b.logger.Debug().Str("body", string(body)).Msg("broadcast response from THORChain")
	err = b.cdc.UnmarshalJSON(body, &commit)
	if err != nil {
		b.errCounter.WithLabelValues("fail_unmarshal_commit", "").Inc()
		b.logger.Error().Err(err).Msg("fail unmarshal commit")
		return common.BlankTxID, fmt.Errorf("fail to broadcast: %w", err)
	}
	txHash, err := common.NewTxID(commit.TxHash)
	if err != nil {
		return common.BlankTxID, fmt.Errorf("fail to convert txhash: %w", err)
	}
	// Code will be the tendermint ABICode , it start at 1 , so if it is an error , code will not be zero
	if commit.Code > 0 {
		return txHash, fmt.Errorf("fail to broadcast to THORChain,code:%d", commit.Code)
	}
	b.m.GetCounter(metrics.TxToThorchain).Inc()
	b.logger.Info().Msgf("Received a TxHash of %v from the thorchain", commit.TxHash)

	// increment seqNum
	atomic.AddUint64(&b.seqNumber, 1)

	return txHash, nil
}
