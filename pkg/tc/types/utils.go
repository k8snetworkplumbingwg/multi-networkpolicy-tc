package types

import (
	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/policyrules"
)

// compare first with second. They are equal if:
//  1. first and second point to the same address (nil or otherwise)
//  2. first and second contain the same value
//  3. if nilVal != nil
//     3.1 first is not nil and *nilVal equals to *first
//     3.2 second is not nil and *nilVal equals to *second
func compare[C comparable](first *C, second *C, nilVal *C) bool {
	if first == second {
		return true
	}

	if first != nil && second != nil {
		return *first == *second
	}

	if nilVal != nil {
		if first != nil && *first == *nilVal {
			return true
		}
		if second != nil && *second == *nilVal {
			return true
		}
	}
	return false
}

// ProtoToFlowerVlanEthType converts FilterProtocol to FlowerVlanEthType, returns "" if conversion is invalid.
func ProtoToFlowerVlanEthType(proto FilterProtocol) FlowerVlanEthType {
	var vlanEthType FlowerVlanEthType

	switch proto {
	case FilterProtocolIPv4:
		vlanEthType = FlowerVlanEthTypeIPv4
	case FilterProtocolIPv6:
		vlanEthType = FlowerVlanEthTypeIPv6
	}
	return vlanEthType
}

// PortProtocolToFlowerIPProto converts policyrules.PolicyPortProtocol to FlowerIPProto,
// returns "" if conversion is invalid.
func PortProtocolToFlowerIPProto(proto policyrules.PolicyPortProtocol) FlowerIPProto {
	var ipProto FlowerIPProto

	switch proto {
	case policyrules.ProtocolTCP:
		ipProto = FlowerIPProtoTCP
	case policyrules.ProtocolUDP:
		ipProto = FlowerIPProtoUDP
	}
	return ipProto
}
