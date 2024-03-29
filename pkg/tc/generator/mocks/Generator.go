// Code generated by mockery v2.14.0. DO NOT EDIT.

package mocks

import (
	policyrules "github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/policyrules"
	generator "github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc/generator"
	mock "github.com/stretchr/testify/mock"
)

// Generator is an autogenerated mock type for the Generator type
type Generator struct {
	mock.Mock
}

// GenerateFromPolicyRuleSet provides a mock function with given fields: ruleSet
func (_m *Generator) GenerateFromPolicyRuleSet(ruleSet policyrules.PolicyRuleSet) (*generator.Objects, error) {
	ret := _m.Called(ruleSet)

	var r0 *generator.Objects
	if rf, ok := ret.Get(0).(func(policyrules.PolicyRuleSet) *generator.Objects); ok {
		r0 = rf(ruleSet)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*generator.Objects)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(policyrules.PolicyRuleSet) error); ok {
		r1 = rf(ruleSet)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewTCGenerator interface {
	mock.TestingT
	Cleanup(func())
}

// NewTCGenerator creates a new instance of Generator. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewTCGenerator(t mockConstructorTestingTNewTCGenerator) *Generator {
	mock := &Generator{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
