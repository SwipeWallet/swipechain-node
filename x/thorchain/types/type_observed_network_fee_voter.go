package types

import (
	"fmt"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

// ObservedNetworkFeeVoter is used to book keep who voted for the network fee message, whether it has majority consensus or not
type ObservedNetworkFeeVoter struct {
	BlockHeight       int64               `json:"block_height"`        // the THORNode block height which the voter reach consensus
	ReportBlockHeight int64               `json:"report_block_height"` // Block height related to the chain
	Chain             common.Chain        `json:"chain"`
	Signers           []cosmos.AccAddress `json:"signers"`
}

// NewObservedNetworkFeeVoter create a new instance of ObservedNetworkFeeVoter
func NewObservedNetworkFeeVoter(reportBlockHeight int64, chain common.Chain) ObservedNetworkFeeVoter {
	return ObservedNetworkFeeVoter{
		ReportBlockHeight: reportBlockHeight,
		Chain:             chain,
	}
}

// HasSigned - check if given address has signed
func (f *ObservedNetworkFeeVoter) HasSigned(signer cosmos.AccAddress) bool {
	for _, sign := range f.Signers {
		if sign.Equals(signer) {
			return true
		}
	}
	return false
}

// Sign this voter with given signer address
func (f *ObservedNetworkFeeVoter) Sign(signer cosmos.AccAddress) bool {
	if f.HasSigned(signer) {
		return false
	}
	f.Signers = append(f.Signers, signer)
	return true
}

// HasConsensus Determine if this errata has enough signers
func (f *ObservedNetworkFeeVoter) HasConsensus(nas NodeAccounts) bool {
	var count int
	for _, signer := range f.Signers {
		if nas.IsNodeKeys(signer) {
			count++
		}
	}
	if HasSuperMajority(count, len(nas)) {
		return true
	}

	return false
}

// HasConsensusV13 Determine if this errata has enough signers
func (f *ObservedNetworkFeeVoter) HasConsensusV13(nas NodeAccounts) bool {
	var count int
	for _, signer := range f.Signers {
		if nas.IsNodeKeys(signer) {
			count++
		}
	}
	if HasSuperMajorityV13(count, len(nas)) {
		return true
	}

	return false
}

// Empty return true when chain is empty and block height is 0
func (f *ObservedNetworkFeeVoter) IsEmpty() bool {
	return f.Chain.IsEmpty() && f.ReportBlockHeight == 0
}

// String implement fmt.Stringer
func (f *ObservedNetworkFeeVoter) String() string {
	return fmt.Sprintf("%s-%d", f.Chain.String(), f.ReportBlockHeight)
}
