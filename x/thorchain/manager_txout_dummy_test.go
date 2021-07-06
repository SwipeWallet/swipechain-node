package thorchain

import (
	"gitlab.com/thorchain/thornode/common"
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
)

// TxOutStoreDummy is going to manage all the outgoing tx
type TxOutStoreDummy struct {
	blockOut *TxOut
	asgard   common.PubKey
}

// NewTxOutStoreDummy will create a new instance of TxOutStore.
func NewTxStoreDummy() *TxOutStoreDummy {
	return &TxOutStoreDummy{
		blockOut: NewTxOut(100),
		asgard:   GetRandomPubKey(),
	}
}

func (tos *TxOutStoreDummy) GetBlockOut(_ cosmos.Context) (*TxOut, error) {
	return tos.blockOut, nil
}

func (tos *TxOutStoreDummy) ClearOutboundItems(ctx cosmos.Context) {
	tos.blockOut = NewTxOut(tos.blockOut.Height)
}

func (tos *TxOutStoreDummy) GetOutboundItems(ctx cosmos.Context) ([]*TxOutItem, error) {
	return tos.blockOut.TxArray, nil
}

func (tos *TxOutStoreDummy) GetOutboundItemByToAddress(_ cosmos.Context, to common.Address) []TxOutItem {
	items := make([]TxOutItem, 0)
	for _, item := range tos.blockOut.TxArray {
		if item.ToAddress.Equals(to) {
			items = append(items, *item)
		}
	}
	return items
}

// AddTxOutItem add an item to internal structure
func (tos *TxOutStoreDummy) TryAddTxOutItem(ctx cosmos.Context, mgr Manager, toi *TxOutItem) (bool, error) {
	if !toi.Chain.Equals(common.THORChain) {
		tos.addToBlockOut(ctx, toi)
	}
	return true, nil
}

func (tos *TxOutStoreDummy) UnSafeAddTxOutItem(ctx cosmos.Context, mgr Manager, toi *TxOutItem) error {
	if !toi.Chain.Equals(common.THORChain) {
		tos.addToBlockOut(ctx, toi)
	}
	return nil
}

func (tos *TxOutStoreDummy) addToBlockOut(_ cosmos.Context, toi *TxOutItem) {
	tos.blockOut.TxArray = append(tos.blockOut.TxArray, toi)
}

// TxOutStoreFailDummy
type TxOutStoreFailDummy struct {
	blockOut *TxOut
	asgard   common.PubKey
}

// NewTxOutStoreFailDummy will create a new instance of TxOutStore.
func NewTxStoreFailDummy() *TxOutStoreFailDummy {
	return &TxOutStoreFailDummy{
		blockOut: NewTxOut(100),
		asgard:   GetRandomPubKey(),
	}
}

func (tos *TxOutStoreFailDummy) GetBlockOut(_ cosmos.Context) (*TxOut, error) {
	return tos.blockOut, kaboom
}

func (tos *TxOutStoreFailDummy) ClearOutboundItems(ctx cosmos.Context) {
	tos.blockOut = NewTxOut(tos.blockOut.Height)
}

func (tos *TxOutStoreFailDummy) GetOutboundItems(ctx cosmos.Context) ([]*TxOutItem, error) {
	return tos.blockOut.TxArray, kaboom
}

func (tos *TxOutStoreFailDummy) GetOutboundItemByToAddress(_ cosmos.Context, to common.Address) []TxOutItem {
	items := make([]TxOutItem, 0)
	for _, item := range tos.blockOut.TxArray {
		if item.ToAddress.Equals(to) {
			items = append(items, *item)
		}
	}
	return items
}

// AddTxOutItem add an item to internal structure
func (tos *TxOutStoreFailDummy) TryAddTxOutItem(ctx cosmos.Context, mgr Manager, toi *TxOutItem) (bool, error) {
	if !toi.Chain.Equals(common.THORChain) {
		tos.addToBlockOut(ctx, toi)
	}
	return false, kaboom
}

func (tos *TxOutStoreFailDummy) UnSafeAddTxOutItem(ctx cosmos.Context, mgr Manager, toi *TxOutItem) error {
	if !toi.Chain.Equals(common.THORChain) {
		tos.addToBlockOut(ctx, toi)
	}
	return kaboom
}

func (tos *TxOutStoreFailDummy) addToBlockOut(_ cosmos.Context, toi *TxOutItem) {
	tos.blockOut.TxArray = append(tos.blockOut.TxArray, toi)
}
