// Code generated by mockery v2.30.1. DO NOT EDIT.

package mocks

import (
	effect "github.com/paketo-buildpacks/libpak/v2/effect"
	mock "github.com/stretchr/testify/mock"
)

// Executor is an autogenerated mock type for the Executor type
type Executor struct {
	mock.Mock
}

// Execute provides a mock function with given fields: execution
func (_m *Executor) Execute(execution effect.Execution) error {
	ret := _m.Called(execution)

	var r0 error
	if rf, ok := ret.Get(0).(func(effect.Execution) error); ok {
		r0 = rf(execution)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewExecutor creates a new instance of Executor. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewExecutor(t interface {
	mock.TestingT
	Cleanup(func())
}) *Executor {
	mock := &Executor{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
