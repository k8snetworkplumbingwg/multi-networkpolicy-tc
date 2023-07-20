package policyrules

import (
	"net"
	"strings"
)

const (
	PolicyTypeIngress PolicyType = "Ingress"
	PolicyTypeEgress  PolicyType = "Egress"

	PolicyActionPass PolicyAction = "Pass"
	PolicyActionDrop PolicyAction = "Drop"

	ProtocolTCP PolicyPortProtocol = "TCP"
	ProtocolUDP PolicyPortProtocol = "UDP"
)

// PolicyType is the type of policy either PolicyTypeIngress or PolicyTypeEgress
type PolicyType string

// PolicyAction is Action needed to be performed for the given Rule
type PolicyAction string

// PolicyPortProtocol is the Port Protocol
type PolicyPortProtocol string

// InterfaceInfo holds information about the interface
type InterfaceInfo struct {
	// Network is the network interfaceInfo is associated with
	Network string
	// Pod Interface same
	InterfaceName string
	// IPs are the IPs assigned to the interface
	IPs []net.IP
	// DeviceID is the Device ID associated with the interface
	DeviceID string
}

// GetUID returns a unique ID for InterfaceInfo in the following format:
//
//	<network-namespace>/<network-name>/<interface-name>
func (i *InterfaceInfo) GetUID() string {
	return strings.Join([]string{i.Network, i.InterfaceName}, "/")
}

// Port holds port information
type Port struct {
	Protocol PolicyPortProtocol
	Number   uint16
}

// Rule represents a single Policy Rule
type Rule struct {
	IPCidrs []*net.IPNet
	Ports   []Port
	Action  PolicyAction
}

// PolicyRuleSet holds the set of Rules of the given Type that should apply to the interface identified by IfcInfo
type PolicyRuleSet struct {
	IfcInfo InterfaceInfo
	Type    PolicyType
	Rules   []Rule
}
