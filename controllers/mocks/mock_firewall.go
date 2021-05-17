// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/metal-stack/firewall-controller/controllers (interfaces: FirewallInterface)

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockFirewallInterface is a mock of FirewallInterface interface.
type MockFirewallInterface struct {
	ctrl     *gomock.Controller
	recorder *MockFirewallInterfaceMockRecorder
}

// MockFirewallInterfaceMockRecorder is the mock recorder for MockFirewallInterface.
type MockFirewallInterfaceMockRecorder struct {
	mock *MockFirewallInterface
}

// NewMockFirewallInterface creates a new mock instance.
func NewMockFirewallInterface(ctrl *gomock.Controller) *MockFirewallInterface {
	mock := &MockFirewallInterface{ctrl: ctrl}
	mock.recorder = &MockFirewallInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockFirewallInterface) EXPECT() *MockFirewallInterfaceMockRecorder {
	return m.recorder
}

// Reconcile mocks base method.
func (m *MockFirewallInterface) Reconcile() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Reconcile")
	ret0, _ := ret[0].(error)
	return ret0
}

// Reconcile indicates an expected call of Reconcile.
func (mr *MockFirewallInterfaceMockRecorder) Reconcile() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Reconcile", reflect.TypeOf((*MockFirewallInterface)(nil).Reconcile))
}