package types

import (
	"errors"

	"gitlab.com/thorchain/thornode/common/cosmos"
)

// BanVoter is a structure to record ban request and it's voters
type BanVoter struct {
	NodeAddress cosmos.AccAddress   `json:"node_address"`
	BlockHeight int64               `json:"block_height"`
	Signers     []cosmos.AccAddress `json:"signers"` // address of node account who saw this tx and voted for it
}

// NewBanVoter create a new instance of BanVoter
func NewBanVoter(addr cosmos.AccAddress) BanVoter {
	return BanVoter{NodeAddress: addr}
}

// Valid return an error if the node address that need to be banned is empty
func (b BanVoter) Valid() error {
	if b.NodeAddress.Empty() {
		return errors.New("node address is empty")
	}
	if b.BlockHeight <= 0 {
		return errors.New("block height cannot be equal to or less than zero")
	}
	return nil
}

// IsEmpty return true when the node address is empty
func (b BanVoter) IsEmpty() bool {
	return b.NodeAddress.Empty()
}

func (b BanVoter) String() string {
	return b.NodeAddress.String()
}

// HasSigned - check if given address has signed
func (b BanVoter) HasSigned(signer cosmos.AccAddress) bool {
	for _, sign := range b.Signers {
		if sign.Equals(signer) {
			return true
		}
	}
	return false
}

// Sign add the given signer to the signer list
func (b *BanVoter) Sign(signer cosmos.AccAddress) {
	if !b.HasSigned(signer) {
		b.Signers = append(b.Signers, signer)
	}
}

// HasConsensus return true if there are majority accounts sign off the BanVoter
func (b BanVoter) HasConsensus(nodeAccounts NodeAccounts) bool {
	var count int
	for _, signer := range b.Signers {
		if nodeAccounts.IsNodeKeys(signer) {
			count++
		}
	}
	if HasSuperMajority(count, len(nodeAccounts)) {
		return true
	}

	return false
}

// HasConsensusV13 return true if there are majority accounts sign off the BanVoter
func (b BanVoter) HasConsensusV13(nodeAccounts NodeAccounts) bool {
	var count int
	for _, signer := range b.Signers {
		if nodeAccounts.IsNodeKeys(signer) {
			count++
		}
	}
	if HasSuperMajorityV13(count, len(nodeAccounts)) {
		return true
	}

	return false
}
