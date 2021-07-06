package types

import (
	"gitlab.com/thorchain/thornode/common/cosmos"
)

// TssKeysignFailVoter a voter structure to store TssKeySign Failure information
type TssKeysignFailVoter struct {
	ID      string              `json:"id"` // checksum of sorted input pubkeys
	Height  int64               `json:"height"`
	Signers []cosmos.AccAddress `json:"signers"`
}

// NewTssKeysignFailVoter create a new instance of TssKeysignFailVoter
func NewTssKeysignFailVoter(id string, height int64) TssKeysignFailVoter {
	return TssKeysignFailVoter{
		ID:     id,
		Height: height,
	}
}

// HasSigned - check if given address has signed
func (tss TssKeysignFailVoter) HasSigned(signer cosmos.AccAddress) bool {
	for _, sign := range tss.Signers {
		if sign.Equals(signer) {
			return true
		}
	}
	return false
}

// Sign this voter with given signer address
func (tss *TssKeysignFailVoter) Sign(signer cosmos.AccAddress) bool {
	if tss.HasSigned(signer) {
		return false
	}
	tss.Signers = append(tss.Signers, signer)
	return true
}

// HasConsensus determine if this tss pool has enough signers
func (tss *TssKeysignFailVoter) HasConsensus(nas NodeAccounts) bool {
	var count int
	for _, signer := range tss.Signers {
		if nas.IsNodeKeys(signer) {
			count++
		}
	}
	return HasSimpleMajority(count, len(nas))
}

// HasConsensusV13 determine if this tss pool has enough signers
// this method introduced at version 0.13.0 as a replacement for HasConsensus
func (tss *TssKeysignFailVoter) HasConsensusV13(nas NodeAccounts) bool {
	var count int
	for _, signer := range tss.Signers {
		if nas.IsNodeKeys(signer) {
			count++
		}
	}
	return HasSimpleMajorityV13(count, len(nas))
}

// HasConsensusV18 determine if this tss pool has enough signers
// this method introduced at version 0.18.0 as a replacement for HasConsensus
func (tss *TssKeysignFailVoter) HasConsensusV18(nas NodeAccounts) bool {
	var count int
	for _, signer := range tss.Signers {
		for _, item := range nas {
			if signer.Equals(item.NodeAddress) {
				count++
				break
			}
		}
	}
	return HasSimpleMajorityV13(count, len(nas))
}

// Empty to check whether this Voter is empty or not
func (tss *TssKeysignFailVoter) Empty() bool {
	return len(tss.ID) == 0 || tss.Height == 0
}

// String implement fmt.Stringer , return's the ID
func (tss *TssKeysignFailVoter) String() string {
	return tss.ID
}
