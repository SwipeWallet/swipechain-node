package thorchain

type DummyMgr struct {
	gasMgr       GasManager
	eventMgr     EventManager
	txOutStore   TxOutStore
	vaultMgr     VaultManager
	validatorMgr ValidatorManager
	obMgr        ObserverManager
	swapQ        SwapQueue
	slasher      Slasher
	yggManager   YggManager
}

func NewDummyMgr() *DummyMgr {
	return &DummyMgr{
		gasMgr:       NewDummyGasManager(),
		eventMgr:     NewDummyEventMgr(),
		txOutStore:   NewTxStoreDummy(),
		vaultMgr:     NewVaultMgrDummy(),
		validatorMgr: NewValidatorDummyMgr(),
		obMgr:        NewDummyObserverManager(),
		slasher:      NewDummySlasher(),
		yggManager:   NewDummyYggManger(),
		// TODO add dummy swap queue
	}
}

func (m DummyMgr) GasMgr() GasManager             { return m.gasMgr }
func (m DummyMgr) EventMgr() EventManager         { return m.eventMgr }
func (m DummyMgr) TxOutStore() TxOutStore         { return m.txOutStore }
func (m DummyMgr) VaultMgr() VaultManager         { return m.vaultMgr }
func (m DummyMgr) ValidatorMgr() ValidatorManager { return m.validatorMgr }
func (m DummyMgr) ObMgr() ObserverManager         { return m.obMgr }
func (m DummyMgr) SwapQ() SwapQueue               { return m.swapQ }
func (m DummyMgr) Slasher() Slasher               { return m.slasher }
func (m DummyMgr) YggManager() YggManager         { return m.yggManager }
