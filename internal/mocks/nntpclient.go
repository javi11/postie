// Code generated manually for nntppool v4 migration. DO NOT EDIT.

package mocks

import (
	"context"
	"io"
	"reflect"

	"github.com/javi11/nntppool/v4"
	"github.com/mnightingale/rapidyenc"
	"go.uber.org/mock/gomock"
)

// MockNNTPClient is a mock of pool.NNTPClient interface.
type MockNNTPClient struct {
	ctrl     *gomock.Controller
	recorder *MockNNTPClientMockRecorder
}

// MockNNTPClientMockRecorder is the mock recorder for MockNNTPClient.
type MockNNTPClientMockRecorder struct {
	mock *MockNNTPClient
}

// NewMockNNTPClient creates a new mock instance.
func NewMockNNTPClient(ctrl *gomock.Controller) *MockNNTPClient {
	mock := &MockNNTPClient{ctrl: ctrl}
	mock.recorder = &MockNNTPClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockNNTPClient) EXPECT() *MockNNTPClientMockRecorder {
	return m.recorder
}

// PostYenc mocks base method.
func (m *MockNNTPClient) PostYenc(ctx context.Context, headers nntppool.PostHeaders, body io.Reader, meta rapidyenc.Meta) (*nntppool.PostResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PostYenc", ctx, headers, body, meta)
	ret0, _ := ret[0].(*nntppool.PostResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PostYenc indicates an expected call of PostYenc.
func (mr *MockNNTPClientMockRecorder) PostYenc(ctx, headers, body, meta any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PostYenc", reflect.TypeOf((*MockNNTPClient)(nil).PostYenc), ctx, headers, body, meta)
}

// Stat mocks base method.
func (m *MockNNTPClient) Stat(ctx context.Context, messageID string) (*nntppool.StatResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Stat", ctx, messageID)
	ret0, _ := ret[0].(*nntppool.StatResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Stat indicates an expected call of Stat.
func (mr *MockNNTPClientMockRecorder) Stat(ctx, messageID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stat", reflect.TypeOf((*MockNNTPClient)(nil).Stat), ctx, messageID)
}

// Stats mocks base method.
func (m *MockNNTPClient) Stats() nntppool.ClientStats {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Stats")
	ret0, _ := ret[0].(nntppool.ClientStats)
	return ret0
}

// Stats indicates an expected call of Stats.
func (mr *MockNNTPClientMockRecorder) Stats() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stats", reflect.TypeOf((*MockNNTPClient)(nil).Stats))
}

// AddProvider mocks base method.
func (m *MockNNTPClient) AddProvider(p nntppool.Provider) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddProvider", p)
	ret0, _ := ret[0].(error)
	return ret0
}

// AddProvider indicates an expected call of AddProvider.
func (mr *MockNNTPClientMockRecorder) AddProvider(p any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddProvider", reflect.TypeOf((*MockNNTPClient)(nil).AddProvider), p)
}

// RemoveProvider mocks base method.
func (m *MockNNTPClient) RemoveProvider(name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RemoveProvider", name)
	ret0, _ := ret[0].(error)
	return ret0
}

// RemoveProvider indicates an expected call of RemoveProvider.
func (mr *MockNNTPClientMockRecorder) RemoveProvider(name any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveProvider", reflect.TypeOf((*MockNNTPClient)(nil).RemoveProvider), name)
}

// Close mocks base method.
func (m *MockNNTPClient) Close() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close.
func (mr *MockNNTPClientMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockNNTPClient)(nil).Close))
}
