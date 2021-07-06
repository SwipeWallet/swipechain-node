package blockscanner

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/syndtr/goleveldb/leveldb"
)

const MockErrorBlockHeight = 1024

// MockScannerStorage is to mock scanner storage interface
type MockScannerStorage struct {
	l     *sync.Mutex
	store map[string][]byte
}

// NewMockScannerStorage create a new instance of MockScannerStorage
func NewMockScannerStorage() *MockScannerStorage {
	return &MockScannerStorage{
		store: make(map[string][]byte),
		l:     &sync.Mutex{},
	}
}

func (mss *MockScannerStorage) GetScanPos() (int64, error) {
	buf, ok := mss.store[ScanPosKey]
	if !ok {
		return 0, errors.New("scan pos doesn't exist")
	}
	pos, _ := binary.Varint(buf)
	return pos, nil
}

func (mss *MockScannerStorage) SetScanPos(block int64) error {
	mss.l.Lock()
	defer mss.l.Unlock()
	buf := make([]byte, 8)
	n := binary.PutVarint(buf, block)
	mss.store[ScanPosKey] = buf[:n]
	return nil
}

func (mss *MockScannerStorage) SetBlockScanStatus(block Block, status BlockScanStatus) error {
	blockStatusItem := BlockStatusItem{
		Block:  block,
		Status: status,
	}
	buf, err := json.Marshal(blockStatusItem)
	if err != nil {
		return fmt.Errorf("fail to marshal BlockStatusItem to json: %w", err)
	}
	mss.l.Lock()
	defer mss.l.Unlock()
	mss.store[getBlockStatusKey(block.Height)] = buf
	return nil
}

func (mss *MockScannerStorage) RemoveBlockStatus(block int64) error {
	mss.l.Lock()
	defer mss.l.Unlock()
	delete(mss.store, getBlockStatusKey(block))
	return nil
}

func (mss *MockScannerStorage) GetBlocksForRetry(failedOnly bool) ([]Block, error) {
	return nil, nil
}

func (mss *MockScannerStorage) Close() error {
	return nil
}

func (mss *MockScannerStorage) GetInternalDb() *leveldb.DB {
	return nil
}
