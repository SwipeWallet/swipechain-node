package types

import (
	"errors"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

// ReserveContributor track contributor
type ReserveContributor struct {
	Address common.Address `json:"address"`
	Amount  cosmos.Uint    `json:"amount"`
}

// ReserveContributors a list of reserve contributors
type ReserveContributors []ReserveContributor

// NewReserveContributor create a new instance of Reserve Contributor
func NewReserveContributor(addr common.Address, amt cosmos.Uint) ReserveContributor {
	return ReserveContributor{
		Address: addr,
		Amount:  amt,
	}
}

// IsEmpty return true when the reserve contributor's address is empty
func (res ReserveContributor) IsEmpty() bool {
	return res.Address.IsEmpty()
}

// Valid check whether reserve contributor has all necessary values
func (res ReserveContributor) Valid() error {
	if res.Amount.IsZero() {
		return errors.New("amount cannot be zero")
	}
	if res.Address.IsEmpty() {
		return errors.New("address cannot be empty")
	}
	return nil
}

// Add the given reserve contributor to list
func (reses ReserveContributors) Add(res ReserveContributor) ReserveContributors {
	for i, r := range reses {
		if r.Address.Equals(res.Address) {
			reses[i].Amount = reses[i].Amount.Add(res.Amount)
			return reses
		}
	}

	return append(reses, res)
}
