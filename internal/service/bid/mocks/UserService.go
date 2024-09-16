// Code generated by mockery v2.45.1. DO NOT EDIT.

package mocks

import (
	context "context"

	uuid "github.com/google/uuid"
	mock "github.com/stretchr/testify/mock"
)

// UserService is an autogenerated mock type for the UserService type
type UserService struct {
	mock.Mock
}

// OrgSize provides a mock function with given fields: ctx, orgId
func (_m *UserService) OrgSize(ctx context.Context, orgId uuid.UUID) (int64, error) {
	ret := _m.Called(ctx, orgId)

	if len(ret) == 0 {
		panic("no return value specified for OrgSize")
	}

	var r0 int64
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, uuid.UUID) (int64, error)); ok {
		return rf(ctx, orgId)
	}
	if rf, ok := ret.Get(0).(func(context.Context, uuid.UUID) int64); ok {
		r0 = rf(ctx, orgId)
	} else {
		r0 = ret.Get(0).(int64)
	}

	if rf, ok := ret.Get(1).(func(context.Context, uuid.UUID) error); ok {
		r1 = rf(ctx, orgId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Permission provides a mock function with given fields: ctx, username, orgId
func (_m *UserService) Permission(ctx context.Context, username string, orgId uuid.UUID) error {
	ret := _m.Called(ctx, username, orgId)

	if len(ret) == 0 {
		panic("no return value specified for Permission")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, uuid.UUID) error); ok {
		r0 = rf(ctx, username, orgId)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UserId provides a mock function with given fields: ctx, username
func (_m *UserService) UserId(ctx context.Context, username string) (uuid.UUID, error) {
	ret := _m.Called(ctx, username)

	if len(ret) == 0 {
		panic("no return value specified for UserId")
	}

	var r0 uuid.UUID
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (uuid.UUID, error)); ok {
		return rf(ctx, username)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) uuid.UUID); ok {
		r0 = rf(ctx, username)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(uuid.UUID)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, username)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Validate provides a mock function with given fields: ctx, username
func (_m *UserService) Validate(ctx context.Context, username string) error {
	ret := _m.Called(ctx, username)

	if len(ret) == 0 {
		panic("no return value specified for Validate")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, username)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ValidateOrgId provides a mock function with given fields: ctx, orgId
func (_m *UserService) ValidateOrgId(ctx context.Context, orgId uuid.UUID) error {
	ret := _m.Called(ctx, orgId)

	if len(ret) == 0 {
		panic("no return value specified for ValidateOrgId")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, uuid.UUID) error); ok {
		r0 = rf(ctx, orgId)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ValidateUserId provides a mock function with given fields: ctx, userId
func (_m *UserService) ValidateUserId(ctx context.Context, userId uuid.UUID) error {
	ret := _m.Called(ctx, userId)

	if len(ret) == 0 {
		panic("no return value specified for ValidateUserId")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, uuid.UUID) error); ok {
		r0 = rf(ctx, userId)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewUserService creates a new instance of UserService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewUserService(t interface {
	mock.TestingT
	Cleanup(func())
}) *UserService {
	mock := &UserService{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
