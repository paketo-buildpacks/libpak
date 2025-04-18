// Code generated by mockery v2.53.3. DO NOT EDIT.

package mocks

import (
	libcnb "github.com/buildpacks/libcnb/v2"

	mock "github.com/stretchr/testify/mock"
)

// Contributable is an autogenerated mock type for the Contributable type
type Contributable struct {
	mock.Mock
}

// Contribute provides a mock function with given fields: layer
func (_m *Contributable) Contribute(layer *libcnb.Layer) error {
	ret := _m.Called(layer)

	if len(ret) == 0 {
		panic("no return value specified for Contribute")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(*libcnb.Layer) error); ok {
		r0 = rf(layer)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Name provides a mock function with no fields
func (_m *Contributable) Name() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Name")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// NewContributable creates a new instance of Contributable. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewContributable(t interface {
	mock.TestingT
	Cleanup(func())
}) *Contributable {
	mock := &Contributable{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
