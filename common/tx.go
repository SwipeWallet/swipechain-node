package common

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"

	"gitlab.com/thorchain/thornode/common/cosmos"
)

type (
	// TxID is a string that can uniquely represent a transaction on different
	// block chain
	TxID string
	// TxIDs is a slice of TxID
	TxIDs []TxID
)

// BlankTxID represent blank
var BlankTxID = TxID("0000000000000000000000000000000000000000000000000000000000000000")

// NewTxID parse the input hash as TxID
func NewTxID(hash string) (TxID, error) {
	switch len(hash) {
	case 64:
		// do nothing
	case 66: // ETH check
		if !strings.HasPrefix(hash, "0x") {
			err := fmt.Errorf("txid error: must be 66 characters (got %d)", len(hash))
			return TxID(""), err
		}
	default:
		err := fmt.Errorf("txid error: must be 64 characters (got %d)", len(hash))
		return TxID(""), err
	}

	return TxID(strings.ToUpper(hash)), nil
}

// Equals check whether two TxID are the same
func (tx TxID) Equals(tx2 TxID) bool {
	return strings.EqualFold(tx.String(), tx2.String())
}

// IsEmpty return true when the tx represent empty string
func (tx TxID) IsEmpty() bool {
	return strings.TrimSpace(tx.String()) == ""
}

// String implement fmt.Stringer
func (tx TxID) String() string {
	return string(tx)
}

// Tx transaction
type Tx struct {
	ID          TxID    `json:"id"`
	Chain       Chain   `json:"chain"`
	FromAddress Address `json:"from_address"`
	ToAddress   Address `json:"to_address"`
	Coins       Coins   `json:"coins"`
	Gas         Gas     `json:"gas"`
	Memo        string  `json:"memo"`
}

// Txs a list of Tx
type Txs []Tx

// GetRagnarokTx return a tx used for ragnarok
func GetRagnarokTx(chain Chain, fromAddr, toAddr Address) Tx {
	return Tx{
		Chain:       chain,
		ID:          BlankTxID,
		FromAddress: fromAddr,
		ToAddress:   toAddr,
		Coins: Coins{
			// used for ragnarok, so doesn't really matter
			NewCoin(BNBAsset, cosmos.OneUint()),
		},
		Gas: Gas{
			// used for ragnarok, so doesn't really matter
			NewCoin(BNBAsset, cosmos.OneUint()),
		},
		Memo: "Ragnarok",
	}
}

// NewTx create a new instance of Tx based on the input information
func NewTx(txID TxID, from, to Address, coins Coins, gas Gas, memo string) Tx {
	var chain Chain
	for _, coin := range coins {
		chain = coin.Asset.Chain
		break
	}
	return Tx{
		ID:          txID,
		Chain:       chain,
		FromAddress: from,
		ToAddress:   to,
		Coins:       coins,
		Gas:         gas,
		Memo:        memo,
	}
}

// Hash calculate a hash based on from address, coins and to address
func (tx Tx) Hash() string {
	str := fmt.Sprintf("%s|%s|%s", tx.FromAddress, tx.Coins, tx.ToAddress)
	return fmt.Sprintf("%X", sha256.Sum256([]byte(str)))
}

// String implement fmt.Stringer return a string representation of the tx
func (tx Tx) String() string {
	return fmt.Sprintf("%s: %s ==> %s (Memo: %s) %s", tx.ID, tx.FromAddress, tx.ToAddress, tx.Memo, tx.Coins)
}

// IsEmpty check whether the ID field is empty or not
func (tx Tx) IsEmpty() bool {
	return tx.ID.IsEmpty()
}

// Equals compare two Tx to see whether they represent the same Tx
func (tx Tx) Equals(tx2 Tx) bool {
	if !tx.ID.Equals(tx2.ID) {
		return false
	}
	if !tx.Chain.Equals(tx2.Chain) {
		return false
	}
	if !tx.FromAddress.Equals(tx2.FromAddress) {
		return false
	}
	if !tx.ToAddress.Equals(tx2.ToAddress) {
		return false
	}
	if !tx.Coins.Equals(tx2.Coins) {
		return false
	}
	if !tx.Gas.Equals(tx2.Gas) {
		return false
	}
	if !strings.EqualFold(tx.Memo, tx2.Memo) {
		return false
	}
	return true
}

// Valid do some data sanity check , if the tx contains invalid information
// it will return an none nil error
func (tx Tx) Valid() error {
	if tx.ID.IsEmpty() {
		return errors.New("Tx ID cannot be empty")
	}
	if tx.FromAddress.IsEmpty() {
		return errors.New("from address cannot be empty")
	}
	if tx.ToAddress.IsEmpty() {
		return errors.New("to address cannot be empty")
	}
	if tx.Chain.IsEmpty() {
		return errors.New("chain cannot be empty")
	}
	if len(tx.Coins) == 0 {
		return errors.New("must have at least 1 coin")
	}
	if err := tx.Coins.Valid(); err != nil {
		return err
	}
	if !tx.Chain.Equals(THORChain) && len(tx.Gas) == 0 {
		return errors.New("must have at least 1 gas coin")
	}
	if err := tx.Gas.Valid(); err != nil {
		return err
	}

	if len([]byte(tx.Memo)) > 150 {
		return fmt.Errorf("memo must not exceed 150 bytes: %d", len([]byte(tx.Memo)))
	}
	return nil
}

// ToAttributes push all the tx fields into a slice of cosmos Attribute(key value pairs)
func (tx Tx) ToAttributes() []cosmos.Attribute {
	return []cosmos.Attribute{
		cosmos.NewAttribute("id", tx.ID.String()),
		cosmos.NewAttribute("chain", tx.Chain.String()),
		cosmos.NewAttribute("from", tx.FromAddress.String()),
		cosmos.NewAttribute("to", tx.ToAddress.String()),
		cosmos.NewAttribute("coin", tx.Coins.String()),
		cosmos.NewAttribute("memo", tx.Memo),
	}
}
