package types

import (
	"fmt"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

// ErrataTxVoter is structure to hold information Errata request and it's voters
// In THORChain a request need to get 2/3 majority to sign off before it can be processed
type ErrataTxVoter struct {
	TxID        common.TxID         `json:"tx_id"`
	Chain       common.Chain        `json:"chain"`
	BlockHeight int64               `json:"block_height"`
	Signers     []cosmos.AccAddress `json:"signers"`
}

// NewErrataTxVoter create a new instance of ErrataTxVoter
func NewErrataTxVoter(txID common.TxID, chain common.Chain) ErrataTxVoter {
	return ErrataTxVoter{
		TxID:  txID,
		Chain: chain,
	}
}

// HasSigned - check if given address has signed
func (errata *ErrataTxVoter) HasSigned(signer cosmos.AccAddress) bool {
	for _, sign := range errata.Signers {
		if sign.Equals(signer) {
			return true
		}
	}
	return false
}

// Sign this voter with the given signer address, if the given signer is already signed , it return false
// otherwise it add the given signer to the signers list and return true
func (errata *ErrataTxVoter) Sign(signer cosmos.AccAddress) bool {
	if errata.HasSigned(signer) {
		return false
	}
	errata.Signers = append(errata.Signers, signer)
	return true
}

// HasConsensus determine if this errata has enough signers
func (errata *ErrataTxVoter) HasConsensus(nas NodeAccounts) bool {
	var count int
	for _, signer := range errata.Signers {
		if nas.IsNodeKeys(signer) {
			count++
		}
	}
	if HasSuperMajorityV13(count, len(nas)) {
		return true
	}

	return false
}

// Empty check whether TxID or Chain is empty
func (errata *ErrataTxVoter) Empty() bool {
	if errata.TxID.IsEmpty() || errata.Chain.IsEmpty() {
		return true
	}
	return false
}

// String implement fmt.Stinger , return a string representation of errata tx voter
func (errata *ErrataTxVoter) String() string {
	return fmt.Sprintf("%s-%s", errata.Chain.String(), errata.TxID.String())
}
