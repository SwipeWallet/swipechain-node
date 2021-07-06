package thorchain

import (
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
)

type DummyYggManager struct{}

func NewDummyYggManger() *DummyYggManager {
	return &DummyYggManager{}
}

func (DummyYggManager) Fund(ctx cosmos.Context, mgr Manager, constAccessor constants.ConstantValues) error {
	return kaboom
}
