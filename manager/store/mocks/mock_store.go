// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/figment-networks/indexer-manager/manager/store (interfaces: DataStore)

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	params "github.com/figment-networks/indexer-manager/manager/store/params"
	structs "github.com/figment-networks/indexer-manager/structs"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
	time "time"
)

// MockDataStore is a mock of DataStore interface
type MockDataStore struct {
	ctrl     *gomock.Controller
	recorder *MockDataStoreMockRecorder
}

// MockDataStoreMockRecorder is the mock recorder for MockDataStore
type MockDataStoreMockRecorder struct {
	mock *MockDataStore
}

// NewMockDataStore creates a new mock instance
func NewMockDataStore(ctrl *gomock.Controller) *MockDataStore {
	mock := &MockDataStore{ctrl: ctrl}
	mock.recorder = &MockDataStoreMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockDataStore) EXPECT() *MockDataStoreMockRecorder {
	return m.recorder
}

// BlockContinuityCheck mocks base method
func (m *MockDataStore) BlockContinuityCheck(arg0 context.Context, arg1 structs.BlockWithMeta, arg2, arg3 uint64) ([][2]uint64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BlockContinuityCheck", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].([][2]uint64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// BlockContinuityCheck indicates an expected call of BlockContinuityCheck
func (mr *MockDataStoreMockRecorder) BlockContinuityCheck(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BlockContinuityCheck", reflect.TypeOf((*MockDataStore)(nil).BlockContinuityCheck), arg0, arg1, arg2, arg3)
}

// BlockTransactionCheck mocks base method
func (m *MockDataStore) BlockTransactionCheck(arg0 context.Context, arg1 structs.BlockWithMeta, arg2, arg3 uint64) ([]uint64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BlockTransactionCheck", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].([]uint64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// BlockTransactionCheck indicates an expected call of BlockTransactionCheck
func (mr *MockDataStoreMockRecorder) BlockTransactionCheck(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BlockTransactionCheck", reflect.TypeOf((*MockDataStore)(nil).BlockTransactionCheck), arg0, arg1, arg2, arg3)
}

// GetBlockForMinTime mocks base method
func (m *MockDataStore) GetBlockForMinTime(arg0 context.Context, arg1 structs.BlockWithMeta, arg2 time.Time) (structs.Block, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBlockForMinTime", arg0, arg1, arg2)
	ret0, _ := ret[0].(structs.Block)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBlockForMinTime indicates an expected call of GetBlockForMinTime
func (mr *MockDataStoreMockRecorder) GetBlockForMinTime(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBlockForMinTime", reflect.TypeOf((*MockDataStore)(nil).GetBlockForMinTime), arg0, arg1, arg2)
}

// GetBlocksHeightsWithNumTx mocks base method
func (m *MockDataStore) GetBlocksHeightsWithNumTx(arg0 context.Context, arg1 structs.BlockWithMeta, arg2, arg3 uint64) ([][2]uint64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBlocksHeightsWithNumTx", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].([][2]uint64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBlocksHeightsWithNumTx indicates an expected call of GetBlocksHeightsWithNumTx
func (mr *MockDataStoreMockRecorder) GetBlocksHeightsWithNumTx(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBlocksHeightsWithNumTx", reflect.TypeOf((*MockDataStore)(nil).GetBlocksHeightsWithNumTx), arg0, arg1, arg2, arg3)
}

// GetLatestBlock mocks base method
func (m *MockDataStore) GetLatestBlock(arg0 context.Context, arg1 structs.BlockWithMeta) (structs.Block, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLatestBlock", arg0, arg1)
	ret0, _ := ret[0].(structs.Block)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetLatestBlock indicates an expected call of GetLatestBlock
func (mr *MockDataStoreMockRecorder) GetLatestBlock(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLatestBlock", reflect.TypeOf((*MockDataStore)(nil).GetLatestBlock), arg0, arg1)
}

// GetLatestTransaction mocks base method
func (m *MockDataStore) GetLatestTransaction(arg0 context.Context, arg1 structs.TransactionWithMeta) (structs.Transaction, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLatestTransaction", arg0, arg1)
	ret0, _ := ret[0].(structs.Transaction)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetLatestTransaction indicates an expected call of GetLatestTransaction
func (mr *MockDataStoreMockRecorder) GetLatestTransaction(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLatestTransaction", reflect.TypeOf((*MockDataStore)(nil).GetLatestTransaction), arg0, arg1)
}

// GetTransactions mocks base method
func (m *MockDataStore) GetTransactions(arg0 context.Context, arg1 params.TransactionSearch) ([]structs.Transaction, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTransactions", arg0, arg1)
	ret0, _ := ret[0].([]structs.Transaction)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTransactions indicates an expected call of GetTransactions
func (mr *MockDataStoreMockRecorder) GetTransactions(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTransactions", reflect.TypeOf((*MockDataStore)(nil).GetTransactions), arg0, arg1)
}

// GetTransactionsHeightsWithTxCount mocks base method
func (m *MockDataStore) GetTransactionsHeightsWithTxCount(arg0 context.Context, arg1 structs.BlockWithMeta, arg2, arg3 uint64) ([][2]uint64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTransactionsHeightsWithTxCount", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].([][2]uint64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTransactionsHeightsWithTxCount indicates an expected call of GetTransactionsHeightsWithTxCount
func (mr *MockDataStoreMockRecorder) GetTransactionsHeightsWithTxCount(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTransactionsHeightsWithTxCount", reflect.TypeOf((*MockDataStore)(nil).GetTransactionsHeightsWithTxCount), arg0, arg1, arg2, arg3)
}

// StoreBlock mocks base method
func (m *MockDataStore) StoreBlock(arg0 structs.BlockWithMeta) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StoreBlock", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// StoreBlock indicates an expected call of StoreBlock
func (mr *MockDataStoreMockRecorder) StoreBlock(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StoreBlock", reflect.TypeOf((*MockDataStore)(nil).StoreBlock), arg0)
}

// StoreTransaction mocks base method
func (m *MockDataStore) StoreTransaction(arg0 structs.TransactionWithMeta) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StoreTransaction", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// StoreTransaction indicates an expected call of StoreTransaction
func (mr *MockDataStoreMockRecorder) StoreTransaction(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StoreTransaction", reflect.TypeOf((*MockDataStore)(nil).StoreTransaction), arg0)
}

// StoreTransactions mocks base method
func (m *MockDataStore) StoreTransactions(arg0 []structs.TransactionWithMeta) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StoreTransactions", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// StoreTransactions indicates an expected call of StoreTransactions
func (mr *MockDataStoreMockRecorder) StoreTransactions(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StoreTransactions", reflect.TypeOf((*MockDataStore)(nil).StoreTransactions), arg0)
}
