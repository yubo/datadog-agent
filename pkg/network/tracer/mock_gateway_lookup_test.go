// Code generated by MockGen. DO NOT EDIT.
// Source: gateway_lookup.go

// Package tracer is a generated GoMock package.
package tracer

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockcloudProvider is a mock of cloudProvider interface.
type MockcloudProvider struct {
	ctrl     *gomock.Controller
	recorder *MockcloudProviderMockRecorder
}

// MockcloudProviderMockRecorder is the mock recorder for MockcloudProvider.
type MockcloudProviderMockRecorder struct {
	mock *MockcloudProvider
}

// NewMockcloudProvider creates a new mock instance.
func NewMockcloudProvider(ctrl *gomock.Controller) *MockcloudProvider {
	mock := &MockcloudProvider{ctrl: ctrl}
	mock.recorder = &MockcloudProviderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockcloudProvider) EXPECT() *MockcloudProviderMockRecorder {
	return m.recorder
}

// IsAWS mocks base method.
func (m *MockcloudProvider) IsAWS() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsAWS")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsAWS indicates an expected call of IsAWS.
func (mr *MockcloudProviderMockRecorder) IsAWS() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsAWS", reflect.TypeOf((*MockcloudProvider)(nil).IsAWS))
}