package netlink

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netlink/nl"
	"golang.org/x/sys/unix"

	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc/types"
)

/*
Helpers (for converters below)
*/

// u32ValFromPtr returns defaultVal if p is nil, else returns the value of p
func u32ValFromPtr(p *uint32, defaultVal uint32) uint32 {
	var v = defaultVal

	if p != nil {
		v = *p
	}
	return v
}

// u16ValFromPtr returns defaultVal if p is nil, else returns the value of p
func u16ValFromPtr(p *uint16, defaultVal uint16) uint16 {
	var v = defaultVal

	if p != nil {
		v = *p
	}
	return v
}

// filterProtoToUnixProto converts FilterProtocol to unix protocol
func filterProtoToUnixProto(protocol types.FilterProtocol) uint16 {
	switch protocol {
	case types.FilterProtocolIPv4:
		return unix.ETH_P_IP
	case types.FilterProtocolIPv6:
		return unix.ETH_P_IPV6
	case types.FilterProtocol8021Q:
		return unix.ETH_P_8021Q
	case types.FilterProtocolAll:
		return unix.ETH_P_ALL
	}

	// we should not get here
	return 0
}

// unixProtoToFilterProto converts unix protocol to FilterProtocol
func unixProtoToFilterProto(protocol uint16) types.FilterProtocol {
	switch protocol {
	case unix.ETH_P_IP:
		return types.FilterProtocolIPv4
	case unix.ETH_P_IPV6:
		return types.FilterProtocolIPv6
	case unix.ETH_P_8021Q:
		return types.FilterProtocol8021Q
	case unix.ETH_P_ALL:
		return types.FilterProtocolAll
	}

	// we should not get here
	return types.FilterProtocol(fmt.Sprintf("Unknown(%d)", protocol))
}

// flowerVlanEthTypeToUnixProto converts FlowerVlanEthType to unix protocol
func flowerVlanEthTypeToUnixProto(ethtype types.FlowerVlanEthType) uint16 {
	switch ethtype {
	case types.FlowerVlanEthTypeIPv4:
		return unix.ETH_P_IP
	case types.FlowerVlanEthTypeIPv6:
		return unix.ETH_P_IPV6
	}
	// we should not get here
	return 0
}

// unixProtoToFlowerVlanEthType converts unix protocol to FlowerVlanEthType
func unixProtoToFlowerVlanEthType(protocol uint16) types.FlowerVlanEthType {
	switch protocol {
	case unix.ETH_P_IP:
		return types.FlowerVlanEthTypeIPv4
	case unix.ETH_P_IPV6:
		return types.FlowerVlanEthTypeIPv6
	}

	// we should not get here
	return types.FlowerVlanEthType(fmt.Sprintf("Unknown(%d)", protocol))
}

// flowerIPProtoToNlIPProto converts FlowerIPProto to netlink IPProto
func flowerIPProtoToNlIPProto(protocol types.FlowerIPProto) nl.IPProto {
	switch protocol {
	case types.FlowerIPProtoTCP:
		return nl.IPPROTO_TCP
	case types.FlowerIPProtoUDP:
		return nl.IPPROTO_UDP
	}
	return 0
}

// nlIPProtoToFlowerIPProto converts netlink IPProto to FlowerIPProto
func nlIPProtoToFlowerIPProto(protocol nl.IPProto) types.FlowerIPProto {
	switch protocol {
	case nl.IPPROTO_TCP:
		return types.FlowerIPProtoTCP
	case nl.IPPROTO_UDP:
		return types.FlowerIPProtoUDP
	}

	// we should not get here
	return types.FlowerIPProto(fmt.Sprintf("Unknown(%d)", protocol))
}

// actionGenericToTcAction converts ActionGenericType to netlink TcAct
func actionGenericToTcAction(action types.ActionGenericType) netlink.TcAct {
	switch action {
	case types.ActionGenericPass:
		return netlink.TC_ACT_OK
	case types.ActionGenericDrop:
		return netlink.TC_ACT_SHOT
	}
	return netlink.TC_ACT_UNSPEC
}

// tcActionToActionGeneric converts netlink TcAct to ActionGenericType
func tcActionToActionGeneric(action netlink.TcAct) types.ActionGenericType {
	switch action {
	case netlink.TC_ACT_OK:
		return types.ActionGenericPass
	case netlink.TC_ACT_SHOT:
		return types.ActionGenericDrop
	}

	// we should not get here
	return types.ActionGenericType(fmt.Sprintf("Unknown(%d)", action))
}

/*
Converters
*/

// qdiscToNlQdisc converts Qdisc to netlink Qdisc
func qdiscToNlQdisc(qd types.QDisc, linkIdx int) netlink.Qdisc {
	return &netlink.Ingress{
		QdiscAttrs: netlink.QdiscAttrs{
			LinkIndex: linkIdx,
			Handle:    u32ValFromPtr(qd.Attrs().Handle, 0),
			Parent:    u32ValFromPtr(qd.Attrs().Parent, netlink.HANDLE_INGRESS),
		},
	}
}

// nlQdiscToQdisc converts netlink Qdisc to QDisc
func nlQdiscToQdisc(qd netlink.Qdisc) types.QDisc {
	return types.NewIngressQDiscBuilder().
		WithParent(qd.Attrs().Parent).
		WithHandle(qd.Attrs().Handle).Build()
}

// chainToNlChain converts Chain to netlink Chain
func chainToNlChain(chain types.Chain, parent uint32) netlink.Chain {
	return netlink.Chain{
		Parent: parent,
		Chain:  u32ValFromPtr(chain.Attrs().Chain, 0),
	}
}

// nlChainToChain converts netlink Chain to Chain
func nlChainToChain(chain *netlink.Chain) types.Chain {
	return types.NewChainBuilder().
		WithParent(chain.Parent).
		WithChain(chain.Chain).Build()
}

// flowerFilterToNlFlowerFilter converts FlowerFilter to netlink Flower
func flowerFilterToNlFlowerFilter(filter *types.FlowerFilter, parent uint32, linkIdx int) *netlink.Flower {
	// ATM Generators dont utilize chains in filters and rely on default chain being 0

	// Handle Filter attributes
	nlFlowerFilter := &netlink.Flower{
		FilterAttrs: netlink.FilterAttrs{
			LinkIndex: linkIdx,
			Handle:    u32ValFromPtr(filter.Attrs().Handle, 0),
			Parent:    parent,
			Chain:     filter.Attrs().Chain,
			Priority:  u16ValFromPtr(filter.Attrs().Priority, 0),
			Protocol:  filterProtoToUnixProto(filter.Attrs().Protocol),
		},
	}

	// Handle matches
	if filter.Flower != nil {
		if filter.Flower.DstIP != nil {
			nlFlowerFilter.DestIP = filter.Flower.DstIP.IP
			nlFlowerFilter.DestIPMask = filter.Flower.DstIP.Mask
		}

		if filter.Flower.DstPort != nil {
			nlFlowerFilter.DestPort = *filter.Flower.DstPort
		}

		if filter.Flower.IPProto != nil {
			ipp := flowerIPProtoToNlIPProto(*filter.Flower.IPProto)
			nlFlowerFilter.IPProto = &ipp
		}

		if filter.Flower.VlanEthType != nil {
			nlFlowerFilter.EthType = flowerVlanEthTypeToUnixProto(*filter.Flower.VlanEthType)
		} else {
			nlFlowerFilter.EthType = nlFlowerFilter.Protocol
		}
	}

	// Handle action
	for idx, act := range filter.Actions {
		nlAct := netlink.GenericAction{
			ActionAttrs: netlink.ActionAttrs{
				Index:  idx,
				Action: actionGenericToTcAction(types.ActionGenericType(act.Spec()["control_action"])),
			},
		}
		nlFlowerFilter.Actions = append(nlFlowerFilter.Actions, &nlAct)
	}

	return nlFlowerFilter
}

// nlFlowerFilterToFlowerFilter converts netlink Flower filter to FlowerFilter
func nlFlowerFilterToFlowerFilter(filter *netlink.Flower) *types.FlowerFilter {
	fb := types.NewFlowerFilterBuilder().
		WithHandle(filter.Handle).
		WithProtocol(unixProtoToFilterProto(filter.Protocol)).
		WithPriority(filter.Priority)

	if filter.Chain != nil {
		fb.WithChain(*filter.Chain)
	}

	if filter.DestIP != nil {
		fb.WithMatchKeyDstIP(&net.IPNet{
			IP:   filter.DestIP,
			Mask: filter.DestIPMask})
	}

	if filter.IPProto != nil {
		fb.WithMatchKeyIPProto(nlIPProtoToFlowerIPProto(*filter.IPProto))
	}

	if filter.DestPort != 0 {
		fb.WithMatchKeyDstPort(filter.DestPort)
	}

	if filter.Protocol == unix.ETH_P_8021Q {
		fb.WithMatchKeyVlanEthType(unixProtoToFlowerVlanEthType(filter.EthType))
	}

	for _, act := range filter.Actions {
		if act.Type() != "generic" {
			// Note(adrianc): we should not get here
			continue
		}

		fb.WithAction(types.NewGenericAction(tcActionToActionGeneric(act.Attrs().Action)))
	}

	return fb.Build()
}
