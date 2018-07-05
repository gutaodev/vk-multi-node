// Code generated by mockery v1.0.0. DO NOT EDIT.
package mocks

import mock "github.com/stretchr/testify/mock"
import proxy "github.com/virtual-kubelet/virtual-kubelet/providers/vic/proxy"
import trace "github.com/vmware/vic/pkg/trace"

// IsolationProxy is an autogenerated mock type for the IsolationProxy type
type IsolationProxy struct {
	mock.Mock
}

// AddHandleToScope provides a mock function with given fields: op, handle, config
func (_m *IsolationProxy) AddHandleToScope(op trace.Operation, handle string, config proxy.IsolationContainerConfig) (string, error) {
	ret := _m.Called(op, handle, config)

	var r0 string
	if rf, ok := ret.Get(0).(func(trace.Operation, string, proxy.IsolationContainerConfig) string); ok {
		r0 = rf(op, handle, config)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(trace.Operation, string, proxy.IsolationContainerConfig) error); ok {
		r1 = rf(op, handle, config)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// AddImageToHandle provides a mock function with given fields: op, handle, deltaID, layerID, imageID, imageName
func (_m *IsolationProxy) AddImageToHandle(op trace.Operation, handle string, deltaID string, layerID string, imageID string, imageName string) (string, error) {
	ret := _m.Called(op, handle, deltaID, layerID, imageID, imageName)

	var r0 string
	if rf, ok := ret.Get(0).(func(trace.Operation, string, string, string, string, string) string); ok {
		r0 = rf(op, handle, deltaID, layerID, imageID, imageName)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(trace.Operation, string, string, string, string, string) error); ok {
		r1 = rf(op, handle, deltaID, layerID, imageID, imageName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// AddInteractionToHandle provides a mock function with given fields: op, handle
func (_m *IsolationProxy) AddInteractionToHandle(op trace.Operation, handle string) (string, error) {
	ret := _m.Called(op, handle)

	var r0 string
	if rf, ok := ret.Get(0).(func(trace.Operation, string) string); ok {
		r0 = rf(op, handle)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(trace.Operation, string) error); ok {
		r1 = rf(op, handle)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// AddLoggingToHandle provides a mock function with given fields: op, handle
func (_m *IsolationProxy) AddLoggingToHandle(op trace.Operation, handle string) (string, error) {
	ret := _m.Called(op, handle)

	var r0 string
	if rf, ok := ret.Get(0).(func(trace.Operation, string) string); ok {
		r0 = rf(op, handle)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(trace.Operation, string) error); ok {
		r1 = rf(op, handle)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// BindScope provides a mock function with given fields: op, handle, name
func (_m *IsolationProxy) BindScope(op trace.Operation, handle string, name string) (string, interface{}, error) {
	ret := _m.Called(op, handle, name)

	var r0 string
	if rf, ok := ret.Get(0).(func(trace.Operation, string, string) string); ok {
		r0 = rf(op, handle, name)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 interface{}
	if rf, ok := ret.Get(1).(func(trace.Operation, string, string) interface{}); ok {
		r1 = rf(op, handle, name)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(interface{})
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(trace.Operation, string, string) error); ok {
		r2 = rf(op, handle, name)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// CommitHandle provides a mock function with given fields: op, handle, containerID, waitTime
func (_m *IsolationProxy) CommitHandle(op trace.Operation, handle string, containerID string, waitTime int32) error {
	ret := _m.Called(op, handle, containerID, waitTime)

	var r0 error
	if rf, ok := ret.Get(0).(func(trace.Operation, string, string, int32) error); ok {
		r0 = rf(op, handle, containerID, waitTime)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CreateHandle provides a mock function with given fields: op
func (_m *IsolationProxy) CreateHandle(op trace.Operation) (string, string, error) {
	ret := _m.Called(op)

	var r0 string
	if rf, ok := ret.Get(0).(func(trace.Operation) string); ok {
		r0 = rf(op)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 string
	if rf, ok := ret.Get(1).(func(trace.Operation) string); ok {
		r1 = rf(op)
	} else {
		r1 = ret.Get(1).(string)
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(trace.Operation) error); ok {
		r2 = rf(op)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// CreateHandleTask provides a mock function with given fields: op, handle, id, layerID, config
func (_m *IsolationProxy) CreateHandleTask(op trace.Operation, handle string, id string, layerID string, config proxy.IsolationContainerConfig) (string, error) {
	ret := _m.Called(op, handle, id, layerID, config)

	var r0 string
	if rf, ok := ret.Get(0).(func(trace.Operation, string, string, string, proxy.IsolationContainerConfig) string); ok {
		r0 = rf(op, handle, id, layerID, config)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(trace.Operation, string, string, string, proxy.IsolationContainerConfig) error); ok {
		r1 = rf(op, handle, id, layerID, config)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// EpAddresses provides a mock function with given fields: op, id, name
func (_m *IsolationProxy) EpAddresses(op trace.Operation, id string, name string) ([]string, error) {
	ret := _m.Called(op, id, name)

	var r0 []string
	if rf, ok := ret.Get(0).(func(trace.Operation, string, string) []string); ok {
		r0 = rf(op, id, name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(trace.Operation, string, string) error); ok {
		r1 = rf(op, id, name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Handle provides a mock function with given fields: op, id, name
func (_m *IsolationProxy) Handle(op trace.Operation, id string, name string) (string, error) {
	ret := _m.Called(op, id, name)

	var r0 string
	if rf, ok := ret.Get(0).(func(trace.Operation, string, string) string); ok {
		r0 = rf(op, id, name)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(trace.Operation, string, string) error); ok {
		r1 = rf(op, id, name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Remove provides a mock function with given fields: op, id, force
func (_m *IsolationProxy) Remove(op trace.Operation, id string, force bool) error {
	ret := _m.Called(op, id, force)

	var r0 error
	if rf, ok := ret.Get(0).(func(trace.Operation, string, bool) error); ok {
		r0 = rf(op, id, force)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetState provides a mock function with given fields: op, handle, name, state
func (_m *IsolationProxy) SetState(op trace.Operation, handle string, name string, state string) (string, error) {
	ret := _m.Called(op, handle, name, state)

	var r0 string
	if rf, ok := ret.Get(0).(func(trace.Operation, string, string, string) string); ok {
		r0 = rf(op, handle, name, state)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(trace.Operation, string, string, string) error); ok {
		r1 = rf(op, handle, name, state)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// State provides a mock function with given fields: op, id, name
func (_m *IsolationProxy) State(op trace.Operation, id string, name string) (string, error) {
	ret := _m.Called(op, id, name)

	var r0 string
	if rf, ok := ret.Get(0).(func(trace.Operation, string, string) string); ok {
		r0 = rf(op, id, name)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(trace.Operation, string, string) error); ok {
		r1 = rf(op, id, name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UnbindScope provides a mock function with given fields: op, handle, name
func (_m *IsolationProxy) UnbindScope(op trace.Operation, handle string, name string) (string, interface{}, error) {
	ret := _m.Called(op, handle, name)

	var r0 string
	if rf, ok := ret.Get(0).(func(trace.Operation, string, string) string); ok {
		r0 = rf(op, handle, name)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 interface{}
	if rf, ok := ret.Get(1).(func(trace.Operation, string, string) interface{}); ok {
		r1 = rf(op, handle, name)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(interface{})
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(trace.Operation, string, string) error); ok {
		r2 = rf(op, handle, name)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}