package types

import (
	"math/big"

	etypes "github.com/ethereum/go-ethereum/core/types"

	"gitlab.com/thorchain/thornode/common"
)

// BlockMeta is a structure to store the blocks bifrost scanned
type BlockMeta struct {
	PreviousHash string            `json:"previous_hash"`
	Height       int64             `json:"height"`
	BlockHash    string            `json:"block_hash"`
	Transactions []TransactionMeta `json:"transactions"`
}

type TransactionMeta struct {
	Hash        string        `json:"hash"`
	Value       *big.Int      `json:"value"`
	BlockHeight int64         `json:"block_height"`
	VaultPubKey common.PubKey `json:"vault_pub_key"`
}

// NewBlockMeta create a new instance of BlockMeta
func NewBlockMeta(block *etypes.Block) *BlockMeta {
	txsMeta := make([]TransactionMeta, 0)
	for _, tx := range block.Transactions() {
		txsMeta = append(txsMeta, TransactionMeta{
			Hash:        tx.Hash().Hex(),
			Value:       tx.Value(),
			BlockHeight: int64(block.NumberU64()),
		})
	}
	return &BlockMeta{
		PreviousHash: block.ParentHash().Hex(),
		Height:       int64(block.NumberU64()),
		BlockHash:    block.Hash().Hex(),
		Transactions: txsMeta,
	}
}
