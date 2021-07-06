package types

import (
	"sort"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

// TssVoter keep track of tss message
type TssVoter struct {
	ID          string              `json:"id"` // checksum of sorted input pubkeys
	PoolPubKey  common.PubKey       `json:"pool_pub_key"`
	PubKeys     common.PubKeys      `json:"pubkeys"`
	BlockHeight int64               `json:"block_height"`
	Chains      common.Chains       `json:"chains"`
	Signers     []cosmos.AccAddress `json:"signers"`
}

// NewTssVoter create a new instance of TssVoter
func NewTssVoter(id string, pks common.PubKeys, pool common.PubKey) TssVoter {
	return TssVoter{
		ID:         id,
		PubKeys:    pks,
		PoolPubKey: pool,
	}
}

// HasSigned - check if given address has signed
func (tss TssVoter) HasSigned(signer cosmos.AccAddress) bool {
	for _, sign := range tss.Signers {
		if sign.Equals(signer) {
			return true
		}
	}
	return false
}

// Sign this voter with given signer address
func (tss *TssVoter) Sign(signer cosmos.AccAddress, chains common.Chains) bool {
	if tss.HasSigned(signer) {
		return false
	}
	for _, pk := range tss.PubKeys {
		addr, err := pk.GetThorAddress()
		if addr.Equals(signer) && err == nil {
			tss.Signers = append(tss.Signers, signer)
			tss.Chains = append(tss.Chains, chains...)
			return true
		}
	}
	return false
}

// ConsensusChains - get a list of chains that have 2/3rds majority
func (tss *TssVoter) ConsensusChains() common.Chains {
	chainCount := make(map[common.Chain]int, 0)
	for _, chain := range tss.Chains {
		if _, ok := chainCount[chain]; !ok {
			chainCount[chain] = 0
		}
		chainCount[chain]++
	}

	chains := make(common.Chains, 0)
	for chain, count := range chainCount {
		if HasSuperMajority(count, len(tss.PubKeys)) {
			chains = append(chains, chain)
		}
	}

	// sort chains for consistency
	sort.SliceStable(chains, func(i, j int) bool {
		return chains[i].String() < chains[j].String()
	})

	return chains
}

// ConsensusChainsV13 - get a list of chains that have 2/3rds majority
func (tss *TssVoter) ConsensusChainsV13() common.Chains {
	chainCount := make(map[common.Chain]int, 0)
	for _, chain := range tss.Chains {
		if _, ok := chainCount[chain]; !ok {
			chainCount[chain] = 0
		}
		chainCount[chain]++
	}

	chains := make(common.Chains, 0)
	for chain, count := range chainCount {
		if HasSuperMajorityV13(count, len(tss.PubKeys)) {
			chains = append(chains, chain)
		}
	}

	// sort chains for consistency
	sort.SliceStable(chains, func(i, j int) bool {
		return chains[i].String() < chains[j].String()
	})

	return chains
}

// HasConsensus determine if this tss pool has enough signers
func (tss *TssVoter) HasConsensus() bool {
	return HasSuperMajority(len(tss.Signers), len(tss.PubKeys))
}

// HasConsensusV13 determine if this tss pool has enough signers
func (tss *TssVoter) HasConsensusV13() bool {
	return HasSuperMajorityV13(len(tss.Signers), len(tss.PubKeys))
}

// Empty check whether TssVoter represent empty info
func (tss *TssVoter) IsEmpty() bool {
	return len(tss.ID) == 0 || len(tss.PoolPubKey) == 0 || len(tss.PubKeys) == 0
}

// String implement fmt.Stringer
func (tss *TssVoter) String() string {
	return tss.ID
}
