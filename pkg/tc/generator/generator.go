package generator

import (
	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/policyrules"
	tctypes "github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc/types"
)

// Objects is a struct containing TC objects
type Objects struct {
	// QDisc is the TC QDisc where rules should be applied
	QDisc tctypes.QDisc
	// Filters are the TC filters that should be applied
	Filters []tctypes.Filter
}

// Generator is an interface to generate Objects from PolicyRuleSet
type Generator interface {
	// GenerateFromPolicyRuleSet creates Objects that correspond to the provided ruleSet
	GenerateFromPolicyRuleSet(ruleSet policyrules.PolicyRuleSet) (*Objects, error)
}
