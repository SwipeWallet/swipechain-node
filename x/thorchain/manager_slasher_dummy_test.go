package thorchain

import (
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"
	"gitlab.com/thorchain/thornode/common"
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
)

type DummySlasher struct {
	pts map[string]int64
}

func NewDummySlasher() *DummySlasher {
	return &DummySlasher{
		pts: make(map[string]int64, 0),
	}
}

func (d DummySlasher) BeginBlock(ctx cosmos.Context, req abci.RequestBeginBlock, constAccessor constants.ConstantValues) {
}

func (d DummySlasher) HandleDoubleSign(ctx cosmos.Context, addr crypto.Address, infractionHeight int64, constAccessor constants.ConstantValues) error {
	return kaboom
}

func (d DummySlasher) LackObserving(ctx cosmos.Context, constAccessor constants.ConstantValues) error {
	return kaboom
}

func (d DummySlasher) LackSigning(ctx cosmos.Context, constAccessor constants.ConstantValues, mgr Manager) error {
	return kaboom
}

func (d DummySlasher) SlashNodeAccount(ctx cosmos.Context, observedPubKey common.PubKey, asset common.Asset, slashAmount cosmos.Uint, mgr Manager) error {
	return kaboom
}

func (d DummySlasher) IncSlashPoints(ctx cosmos.Context, point int64, addresses ...cosmos.AccAddress) {
	for _, addr := range addresses {
		found := false
		for k := range d.pts {
			if k == addr.String() {
				d.pts[k] += point
				found = true
				break
			}
		}
		if !found {
			d.pts[addr.String()] = point
		}
	}
}

func (d DummySlasher) DecSlashPoints(ctx cosmos.Context, point int64, addresses ...cosmos.AccAddress) {
	for _, addr := range addresses {
		found := false
		for k := range d.pts {
			if k == addr.String() {
				d.pts[k] -= point
				found = true
				break
			}
		}
		if !found {
			d.pts[addr.String()] = -point
		}
	}
}
