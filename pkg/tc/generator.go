package tc

import (
	"fmt"
	"net"

	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/policyrules"
	tctypes "github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc/types"
	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/utils"
)

const (
	PrioDefault = 300
	PrioPass    = 200
	PrioDrop    = 100
)

var (
	allProtocols = [...]tctypes.FilterProtocol{tctypes.FilterProtocolIPv4, tctypes.FilterProtocolIPv6}
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

// NewSimpleTCGenerator creates a new SimpleTCGenerator instance
func NewSimpleTCGenerator() *SimpleTCGenerator {
	return &SimpleTCGenerator{}
}

// SimpleTCGenerator is a simple implementation for Generator interface
type SimpleTCGenerator struct{}

// GenerateFromPolicyRuleSet implements Generator interface
// It renders TC objects needed to satisfy the rules in the provided PolicyRuleSet
// QDisc is Ingress QDisc
// Filters is a list of filters which satisfy the PolicyRuleSet. They are generated as follows
//  1. Drop rule at chain 0, priority 300 for all traffic
//  2. Accept rules per CIDR X Port for every Pass Rule in PolicyRuleSet at chain 0, priority 200
//  3. Drop rules per CIDR X Port for every Drop Rule in PolicyRuleSet at chain 0, prioirty 100
//     Note: only Egress Policy type is supported
func (s *SimpleTCGenerator) GenerateFromPolicyRuleSet(ruleSet policyrules.PolicyRuleSet) (*Objects, error) {
	tcObj := &Objects{
		QDisc:   nil,
		Filters: make([]tctypes.Filter, 0),
	}
	// only egress policy is supported
	if ruleSet.Type != policyrules.PolicyTypeEgress {
		return nil, fmt.Errorf("unsupported policy type. %s", ruleSet.Type)
	}

	// create qdisc obj
	tcObj.QDisc = tctypes.NewIngressQDiscBuilder().Build()

	if ruleSet.Rules == nil {
		// no rules
		return tcObj, nil
	}

	// create filters
	// 1. default drop rule for supported protocols at priority 3xx
	tcObj.Filters = append(tcObj.Filters, s.genFilters(nil, nil, PrioDefault,
		tctypes.NewGenericActionBuiler().WithDrop().Build())...)

	for _, rule := range ruleSet.Rules {
		// 2. accept rules at priority 2xx
		// 3. drop rules at priority 1xx
		switch rule.Action {
		case policyrules.PolicyActionPass:
			tcObj.Filters = append(tcObj.Filters, s.genPassFilters(rule)...)
		case policyrules.PolicyActionDrop:
			tcObj.Filters = append(tcObj.Filters, s.genDropFilters(rule)...)
		default:
			// we should not get here
			return nil, fmt.Errorf("unknown policy action for rule. %s", rule.Action)
		}
	}
	return tcObj, nil
}

// genPassFilters generates Filters with Pass action
func (s *SimpleTCGenerator) genPassFilters(rule policyrules.Rule) []tctypes.Filter {
	return s.genFilters(rule.IPCidrs, rule.Ports, PrioPass, tctypes.NewGenericActionBuiler().WithPass().Build())
}

// genPassFilters generates Filters with Drop action
func (s *SimpleTCGenerator) genDropFilters(rule policyrules.Rule) []tctypes.Filter {
	return s.genFilters(rule.IPCidrs, rule.Ports, PrioDrop, tctypes.NewGenericActionBuiler().WithDrop().Build())
}

// genFilters generates (flower) Filters based on provided ipCidrs, ports on the given prio with the given action
// the filters generated are: matching on {ipCidrs} [X {Ports}] With priority `prio`, and action `action`
// if no IPs and Ports provided, returned filters will match all ipv4, ipv6 traffic
func (s *SimpleTCGenerator) genFilters(ipCidrs []*net.IPNet, ports []policyrules.Port, prio uint16,
	action tctypes.Action) []tctypes.Filter {
	hasIPs := len(ipCidrs) > 0
	hasPorts := len(ports) > 0
	filters := make([]tctypes.Filter, 0)

	switch {
	case hasIPs:
		for _, ipCidr := range ipCidrs {
			var proto tctypes.FilterProtocol
			var actualPrio uint16

			if utils.IsIPv4(ipCidr.IP) {
				proto = tctypes.FilterProtocolIPv4
				actualPrio = prio
			} else {
				proto = tctypes.FilterProtocolIPv6
				actualPrio = prio + 1
			}

			if hasPorts {
				for _, port := range ports {
					filters = append(filters,
						tctypes.NewFlowerFilterBuilder().
							WithProtocol(proto).
							WithPriority(actualPrio).
							WithMatchKeyDstIP(ipCidr.String()).
							WithMatchKeyIPProto(string(port.Protocol)).
							WithMatchKeyDstPort(port.Number).
							WithAction(action).
							Build())
				}
			} else {
				filters = append(filters,
					tctypes.NewFlowerFilterBuilder().
						WithProtocol(proto).
						WithPriority(actualPrio).
						WithMatchKeyDstIP(ipCidr.String()).
						WithAction(action).
						Build())
			}
		}
	case hasPorts: // ports without IPs
		for _, port := range ports {
			// match all protocols with given port
			for idx, proto := range allProtocols {
				actualPrio := prio + uint16(idx)
				filters = append(filters,
					tctypes.NewFlowerFilterBuilder().
						WithProtocol(proto).
						WithPriority(actualPrio).
						WithMatchKeyIPProto(string(port.Protocol)).
						WithMatchKeyDstPort(port.Number).
						WithAction(action).
						Build())
			}
		}
	default:
		// match all protocols with action
		for idx, proto := range allProtocols {
			actualPrio := prio + uint16(idx)
			filters = append(filters,
				tctypes.NewFlowerFilterBuilder().
					WithProtocol(proto).
					WithPriority(actualPrio).
					WithAction(action).
					Build())
		}
	}

	return filters
}
