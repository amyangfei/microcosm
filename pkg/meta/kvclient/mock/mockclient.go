// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/hanfei1991/microcosm/pkg/meta/metaclient (interfaces: KVClient)

// Package mock is a generated GoMock package.
package mock

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	metaclient "github.com/hanfei1991/microcosm/pkg/meta/metaclient"
)

// MockKVClient is a mock of KVClient interface.
type MockKVClient struct {
	ctrl     *gomock.Controller
	recorder *MockKVClientMockRecorder
}

// MockKVClientMockRecorder is the mock recorder for MockKVClient.
type MockKVClientMockRecorder struct {
	mock *MockKVClient
}

// NewMockKVClient creates a new mock instance.
func NewMockKVClient(ctrl *gomock.Controller) *MockKVClient {
	mock := &MockKVClient{ctrl: ctrl}
	mock.recorder = &MockKVClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockKVClient) EXPECT() *MockKVClientMockRecorder {
	return m.recorder
}

// Close mocks base method.
func (m *MockKVClient) Close() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close.
func (mr *MockKVClientMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockKVClient)(nil).Close))
}

// Delete mocks base method.
func (m *MockKVClient) Delete(arg0 context.Context, arg1 string, arg2 ...metaclient.OpOption) (*metaclient.DeleteResponse, metaclient.Error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Delete", varargs...)
	ret0, _ := ret[0].(*metaclient.DeleteResponse)
	ret1, _ := ret[1].(metaclient.Error)
	return ret0, ret1
}

// Delete indicates an expected call of Delete.
func (mr *MockKVClientMockRecorder) Delete(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockKVClient)(nil).Delete), varargs...)
}

// Get mocks base method.
func (m *MockKVClient) Get(arg0 context.Context, arg1 string, arg2 ...metaclient.OpOption) (*metaclient.GetResponse, metaclient.Error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Get", varargs...)
	ret0, _ := ret[0].(*metaclient.GetResponse)
	ret1, _ := ret[1].(metaclient.Error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockKVClientMockRecorder) Get(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockKVClient)(nil).Get), varargs...)
}

// Put mocks base method.
func (m *MockKVClient) Put(arg0 context.Context, arg1, arg2 string) (*metaclient.PutResponse, metaclient.Error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Put", arg0, arg1, arg2)
	ret0, _ := ret[0].(*metaclient.PutResponse)
	ret1, _ := ret[1].(metaclient.Error)
	return ret0, ret1
}

// Put indicates an expected call of Put.
func (mr *MockKVClientMockRecorder) Put(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Put", reflect.TypeOf((*MockKVClient)(nil).Put), arg0, arg1, arg2)
}

// Txn mocks base method.
func (m *MockKVClient) Txn(arg0 context.Context) metaclient.Txn {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Txn", arg0)
	ret0, _ := ret[0].(metaclient.Txn)
	return ret0
}

// Txn indicates an expected call of Txn.
func (mr *MockKVClientMockRecorder) Txn(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Txn", reflect.TypeOf((*MockKVClient)(nil).Txn), arg0)
}
