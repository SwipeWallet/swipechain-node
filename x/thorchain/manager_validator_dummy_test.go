package thorchain

import (
	abci "github.com/tendermint/tendermint/abci/types"

	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
)

type ValidatorDummyMgr struct{}

// NewValidatorDummyMgr create a new instance of ValidatorDummyMgr
func NewValidatorDummyMgr() *ValidatorDummyMgr {
	return &ValidatorDummyMgr{}
}

func (vm *ValidatorDummyMgr) BeginBlock(_ cosmos.Context, _ constants.ConstantValues) error {
	return kaboom
}

func (vm *ValidatorDummyMgr) EndBlock(_ cosmos.Context, _ Manager, _ constants.ConstantValues) []abci.ValidatorUpdate {
	return nil
}

func (vm *ValidatorDummyMgr) RequestYggReturn(_ cosmos.Context, _ NodeAccount, _ Manager) error {
	return kaboom
}

func (vm *ValidatorDummyMgr) processRagnarok(_ cosmos.Context, _ Manager, _ constants.ConstantValues) error {
	return kaboom
}

func (vm *ValidatorDummyMgr) NodeAccountPreflightCheck(ctx cosmos.Context, na NodeAccount, constAccessor constants.ConstantValues) (NodeStatus, error) {
	return NodeDisabled, kaboom
}
