package generator

import (
	tctypes "github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc/types"
)

type BasePrio uint16

const (
	BasePrioDefault BasePrio = 300
	BasePrioPass    BasePrio = 200
	BasePrioDrop    BasePrio = 100
)

const (
	prioOffsetIPv4 = iota
	prioOffsetIPv6
	prioOffset8021Q
)

var (
	protoToPrioOffset = map[tctypes.FilterProtocol]uint16{
		tctypes.FilterProtocolIPv4:  prioOffsetIPv4,
		tctypes.FilterProtocolIPv6:  prioOffsetIPv6,
		tctypes.FilterProtocol8021Q: prioOffset8021Q,
	}
)

// PrioFromBaseAndProtcol returns Filter priority according to provided BasePrio and FilterProtocol
func PrioFromBaseAndProtcol(basePrio BasePrio, proto tctypes.FilterProtocol) uint16 {
	return uint16(basePrio) + protoToPrioOffset[proto]
}
