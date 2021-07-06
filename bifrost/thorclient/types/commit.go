package types

import (
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
)

type BroadcastResult struct {
	JSONRPC string                            `json:"jsonrpc"`
	Result  coretypes.ResultBroadcastTxCommit `json:"result"`
}
