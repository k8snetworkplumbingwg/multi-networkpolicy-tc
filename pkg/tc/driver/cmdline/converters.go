package cmdline

import (
	"strings"

	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc/types"
)

const (
	allStr       = "all"
	ipStr        = "ip"
	ipv6Str      = "ipv6"
	vlanProtoStr = "802.1q"

	tcpStr = "tcp"
	udpStr = "udp"
)

// sToFilterProtocol converts given string to types.FilterProtocol. returns "" in case of an invalid conversion
func sToFilterProtocol(proto string) types.FilterProtocol {
	var fp types.FilterProtocol

	switch strings.ToLower(proto) {
	case allStr:
		fp = types.FilterProtocolAll
	case ipStr:
		fp = types.FilterProtocolIPv4
	case ipv6Str:
		fp = types.FilterProtocolIPv6
	case vlanProtoStr:
		fp = types.FilterProtocol8021Q
	}

	return fp
}

// sToFlowerIPProto converts given string to types.FlowerIPProto. returns "" in case of an invalid conversion
func sToFlowerIPProto(ipp string) types.FlowerIPProto {
	var fp types.FlowerIPProto

	switch strings.ToLower(ipp) {
	case tcpStr:
		fp = types.FlowerIPProtoTCP
	case udpStr:
		fp = types.FlowerIPProtoUDP
	}

	return fp
}

// sToFilterProtocol converts given string to types.FlowerVlanEthType. returns "" in case of an invalid conversion
func sToFlowerVlanEthType(ethtype string) types.FlowerVlanEthType {
	var vlanEthType types.FlowerVlanEthType

	switch strings.ToLower(ethtype) {
	case ipStr:
		vlanEthType = types.FlowerVlanEthTypeIPv4
	case ipv6Str:
		vlanEthType = types.FlowerVlanEthTypeIPv6
	}

	return vlanEthType
}
