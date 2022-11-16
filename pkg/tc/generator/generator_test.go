package generator_test

import (
	"net"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/policyrules"
	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc"
	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc/generator"
	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc/types"
	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/utils"
)

func ensureCallAndQdisc(tcObj *generator.Objects, err error) {
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	ExpectWithOffset(1, tcObj.QDisc).ToNot(BeNil())
	ExpectWithOffset(1, tcObj.QDisc.Type()).To(Equal(types.QDiscIngressType))
}

func ipnetFromStr(ipCidr string) *net.IPNet {
	_, ipn, err := net.ParseCIDR(ipCidr)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	ExpectWithOffset(1, ipn).ToNot(BeNil())
	return ipn
}

func filtersEqual(actualFilters, expectedFilters tc.FilterSet) {
	// Note(adrianc) we do this (and not simply calling filterSet.Equals)
	// to provide visibility into what is NOT equal in case of failure
	ExpectWithOffset(1, actualFilters.Difference(expectedFilters).List()).To(BeEmpty())
	ExpectWithOffset(1, expectedFilters.Difference(actualFilters).List()).To(BeEmpty())
}

func ipToProto(ip net.IP) types.FilterProtocol {
	proto := types.FilterProtocolIPv6
	if utils.IsIPv4(ip) {
		proto = types.FilterProtocolIPv4
	}
	return proto
}

func filterSetFromFilters(filters []types.Filter) tc.FilterSet {
	fs := tc.NewFilterSetImpl()

	for _, f := range filters {
		fs.Add(f)
	}

	return fs
}

var _ = Describe("filter priority tests", func() {
	DescribeTable("returns expected priority for BasePrio and Protocol",
		func(basePrio generator.BasePrio, proto types.FilterProtocol, expectedPrio int) {
			Expect(generator.PrioFromBaseAndProtcol(basePrio, proto)).To(Equal(uint16(expectedPrio)))
		},
		Entry("Default priority IPv4 = 300", generator.BasePrioDefault, types.FilterProtocolIPv4, 300),
		Entry("Default priority IPv6 = 301", generator.BasePrioDefault, types.FilterProtocolIPv6, 301),
		Entry("Default priority 802.1Q = 302", generator.BasePrioDefault, types.FilterProtocol8021Q, 302),
		Entry("Pass priority IPv4 = 200", generator.BasePrioPass, types.FilterProtocolIPv4, 200),
		Entry("Pass priority IPv6 = 201", generator.BasePrioPass, types.FilterProtocolIPv6, 201),
		Entry("Pass priority 802.1Q = 202", generator.BasePrioPass, types.FilterProtocol8021Q, 202),
		Entry("Drop priority IPv4 = 100", generator.BasePrioDrop, types.FilterProtocolIPv4, 100),
		Entry("Drop priority IPv6 = 101", generator.BasePrioDrop, types.FilterProtocolIPv6, 101),
		Entry("Drop priority 802.1Q = 102", generator.BasePrioDrop, types.FilterProtocol8021Q, 102),
	)
})

var _ = Describe("SimpleTCGenerator tests", func() {
	var generatorInst generator.Generator
	defaultFilters := []types.Filter{
		types.NewFlowerFilterBuilder().
			WithPriority(generator.PrioFromBaseAndProtcol(generator.BasePrioDefault, types.FilterProtocolIPv4)).
			WithProtocol(types.FilterProtocolIPv4).
			WithAction(types.NewGenericActionBuiler().WithDrop().Build()).
			Build(),
		types.NewFlowerFilterBuilder().
			WithPriority(generator.PrioFromBaseAndProtcol(generator.BasePrioDefault, types.FilterProtocolIPv6)).
			WithProtocol(types.FilterProtocolIPv6).
			WithAction(types.NewGenericActionBuiler().WithDrop().Build()).
			Build(),
		types.NewFlowerFilterBuilder().
			WithPriority(generator.PrioFromBaseAndProtcol(generator.BasePrioDefault, types.FilterProtocol8021Q)).
			WithProtocol(types.FilterProtocol8021Q).
			WithMatchKeyVlanEthType(types.FlowerVlanEthTypeIPv4).
			WithAction(types.NewGenericActionBuiler().WithDrop().Build()).
			Build(),
		types.NewFlowerFilterBuilder().
			WithPriority(generator.PrioFromBaseAndProtcol(generator.BasePrioDefault, types.FilterProtocol8021Q)).
			WithProtocol(types.FilterProtocol8021Q).
			WithMatchKeyVlanEthType(types.FlowerVlanEthTypeIPv6).
			WithAction(types.NewGenericActionBuiler().WithDrop().Build()).
			Build(),
	}

	BeforeEach(func() {
		generatorInst = generator.NewSimpleTCGenerator()
	})

	Context("GenerateFromPolicyRuleSet() Basic", func() {
		It("generates no objects if PolicyRuleSet with nil rules", func() {
			rs := policyrules.PolicyRuleSet{
				IfcInfo: policyrules.InterfaceInfo{},
				Type:    policyrules.PolicyTypeEgress,
				Rules:   nil,
			}
			tcObj, err := generatorInst.GenerateFromPolicyRuleSet(rs)

			ensureCallAndQdisc(tcObj, err)
			Expect(tcObj.Filters).To(BeEmpty())
		})

		It("generates objects with default drop filter if PolicyRuleSet with zero rules", func() {
			rs := policyrules.PolicyRuleSet{
				IfcInfo: policyrules.InterfaceInfo{},
				Type:    policyrules.PolicyTypeEgress,
				Rules:   make([]policyrules.Rule, 0),
			}
			tcObj, err := generatorInst.GenerateFromPolicyRuleSet(rs)

			ensureCallAndQdisc(tcObj, err)
			Expect(tcObj.Filters).To(HaveLen(len(defaultFilters)))
			expectedFilters := filterSetFromFilters(defaultFilters)
			actualFilters := filterSetFromFilters(tcObj.Filters)
			filtersEqual(actualFilters, expectedFilters)
		})

		It("fails to generate objects if PolicyRuleSet is Ingress", func() {
			rs := policyrules.PolicyRuleSet{
				IfcInfo: policyrules.InterfaceInfo{},
				Type:    policyrules.PolicyTypeIngress,
				Rules:   make([]policyrules.Rule, 0),
			}
			_, err := generatorInst.GenerateFromPolicyRuleSet(rs)
			Expect(err).To(HaveOccurred())
		})

		Context("GenerateFromPolicyRuleSet() With policy Rules", func() {
			var rs policyrules.PolicyRuleSet
			var actualFilters tc.FilterSet
			var ips = []*net.IPNet{
				ipnetFromStr("192.168.1.2/32"),
				ipnetFromStr("10.100.1.1/24"),
				ipnetFromStr("2001::1/128"),
				ipnetFromStr("2001::1000:0/112"),
			}
			var ports = []policyrules.Port{
				{
					Protocol: policyrules.ProtocolTCP,
					Number:   6666,
				},
				{
					Protocol: policyrules.ProtocolUDP,
					Number:   7777,
				},
			}

			BeforeEach(func() {
				rs = policyrules.PolicyRuleSet{
					IfcInfo: policyrules.InterfaceInfo{},
					Type:    policyrules.PolicyTypeEgress,
				}
				actualFilters = tc.NewFilterSetImpl()
			})

			It("generates tc objects for pass rule with IP", func() {
				rules := []policyrules.Rule{{
					IPCidrs: ips,
					Action:  policyrules.PolicyActionPass,
				}}
				rs.Rules = rules

				tcObj, err := generatorInst.GenerateFromPolicyRuleSet(rs)
				ensureCallAndQdisc(tcObj, err)
				for i := range tcObj.Filters {
					actualFilters.Add(tcObj.Filters[i])
				}

				expectedFilters := filterSetFromFilters(defaultFilters)
				for _, ip := range ips {
					proto := ipToProto(ip.IP)
					prio := generator.PrioFromBaseAndProtcol(generator.BasePrioPass, proto)
					expectedFilters.Add(
						types.NewFlowerFilterBuilder().
							WithPriority(prio).
							WithProtocol(proto).
							WithMatchKeyDstIP(ip).
							WithAction(types.NewGenericActionBuiler().WithPass().Build()).
							Build())
					expectedFilters.Add(
						types.NewFlowerFilterBuilder().
							WithPriority(generator.PrioFromBaseAndProtcol(generator.BasePrioPass,
								types.FilterProtocol8021Q)).
							WithProtocol(types.FilterProtocol8021Q).
							WithMatchKeyVlanEthType(types.ProtoToFlowerVlanEthType(proto)).
							WithMatchKeyDstIP(ip).
							WithAction(types.NewGenericActionBuiler().WithPass().Build()).
							Build())
				}

				filtersEqual(actualFilters, expectedFilters)
			})

			It("generates tc objects for pass rule with Port", func() {
				rules := []policyrules.Rule{{
					Ports:  ports,
					Action: policyrules.PolicyActionPass,
				}}
				rs.Rules = rules

				tcObj, err := generatorInst.GenerateFromPolicyRuleSet(rs)
				ensureCallAndQdisc(tcObj, err)
				for i := range tcObj.Filters {
					actualFilters.Add(tcObj.Filters[i])
				}

				expectedFilters := filterSetFromFilters(defaultFilters)
				for _, port := range ports {
					expectedFilters.Add(
						types.NewFlowerFilterBuilder().
							WithPriority(generator.PrioFromBaseAndProtcol(generator.BasePrioPass,
								types.FilterProtocolIPv4)).
							WithProtocol(types.FilterProtocolIPv4).
							WithMatchKeyIPProto(types.PortProtocolToFlowerIPProto(port.Protocol)).
							WithMatchKeyDstPort(port.Number).
							WithAction(types.NewGenericActionBuiler().WithPass().Build()).
							Build())
					expectedFilters.Add(
						types.NewFlowerFilterBuilder().
							WithPriority(generator.PrioFromBaseAndProtcol(generator.BasePrioPass,
								types.FilterProtocolIPv6)).
							WithProtocol(types.FilterProtocolIPv6).
							WithMatchKeyIPProto(types.PortProtocolToFlowerIPProto(port.Protocol)).
							WithMatchKeyDstPort(port.Number).
							WithAction(types.NewGenericActionBuiler().WithPass().Build()).
							Build())
					expectedFilters.Add(
						types.NewFlowerFilterBuilder().
							WithPriority(generator.PrioFromBaseAndProtcol(generator.BasePrioPass,
								types.FilterProtocol8021Q)).
							WithProtocol(types.FilterProtocol8021Q).
							WithMatchKeyVlanEthType("ip").
							WithMatchKeyIPProto(types.PortProtocolToFlowerIPProto(port.Protocol)).
							WithMatchKeyDstPort(port.Number).
							WithAction(types.NewGenericActionBuiler().WithPass().Build()).
							Build())
					expectedFilters.Add(
						types.NewFlowerFilterBuilder().
							WithPriority(generator.PrioFromBaseAndProtcol(generator.BasePrioPass,
								types.FilterProtocol8021Q)).
							WithProtocol(types.FilterProtocol8021Q).
							WithMatchKeyVlanEthType("ipv6").
							WithMatchKeyIPProto(types.PortProtocolToFlowerIPProto(port.Protocol)).
							WithMatchKeyDstPort(port.Number).
							WithAction(types.NewGenericActionBuiler().WithPass().Build()).
							Build())
				}

				filtersEqual(actualFilters, expectedFilters)
			})

			It("generates tc objects for pass rule with IP and port", func() {
				rules := []policyrules.Rule{{
					IPCidrs: ips,
					Ports:   ports,
					Action:  policyrules.PolicyActionPass,
				}}
				rs.Rules = rules

				tcObj, err := generatorInst.GenerateFromPolicyRuleSet(rs)
				ensureCallAndQdisc(tcObj, err)
				for i := range tcObj.Filters {
					actualFilters.Add(tcObj.Filters[i])
				}

				expectedFilters := filterSetFromFilters(defaultFilters)
				for _, ip := range ips {
					proto := ipToProto(ip.IP)
					prio := generator.PrioFromBaseAndProtcol(generator.BasePrioPass, proto)
					for _, port := range ports {
						expectedFilters.Add(
							types.NewFlowerFilterBuilder().
								WithPriority(prio).
								WithProtocol(proto).
								WithMatchKeyDstIP(ip).
								WithMatchKeyIPProto(types.PortProtocolToFlowerIPProto(port.Protocol)).
								WithMatchKeyDstPort(port.Number).
								WithAction(types.NewGenericActionBuiler().WithPass().Build()).
								Build())
						expectedFilters.Add(
							types.NewFlowerFilterBuilder().
								WithPriority(generator.PrioFromBaseAndProtcol(generator.BasePrioPass,
									types.FilterProtocol8021Q)).
								WithProtocol(types.FilterProtocol8021Q).
								WithMatchKeyVlanEthType(types.ProtoToFlowerVlanEthType(proto)).
								WithMatchKeyDstIP(ip).
								WithMatchKeyIPProto(types.PortProtocolToFlowerIPProto(port.Protocol)).
								WithMatchKeyDstPort(port.Number).
								WithAction(types.NewGenericActionBuiler().WithPass().Build()).
								Build())
					}
				}

				filtersEqual(actualFilters, expectedFilters)
			})

			It("generates tc objects for pass rule with no IP and Port", func() {
				rules := []policyrules.Rule{{
					Action: policyrules.PolicyActionPass,
				}}
				rs.Rules = rules

				tcObj, err := generatorInst.GenerateFromPolicyRuleSet(rs)
				ensureCallAndQdisc(tcObj, err)
				for i := range tcObj.Filters {
					actualFilters.Add(tcObj.Filters[i])
				}

				expectedFilters := filterSetFromFilters(defaultFilters)
				expectedFilters.Add(types.NewFlowerFilterBuilder().
					WithPriority(generator.PrioFromBaseAndProtcol(generator.BasePrioPass, types.FilterProtocolIPv4)).
					WithProtocol(types.FilterProtocolIPv4).
					WithAction(types.NewGenericActionBuiler().WithPass().Build()).
					Build())
				expectedFilters.Add(types.NewFlowerFilterBuilder().
					WithPriority(generator.PrioFromBaseAndProtcol(generator.BasePrioPass, types.FilterProtocolIPv6)).
					WithProtocol(types.FilterProtocolIPv6).
					WithAction(types.NewGenericActionBuiler().WithPass().Build()).
					Build())
				expectedFilters.Add(types.NewFlowerFilterBuilder().
					WithPriority(generator.PrioFromBaseAndProtcol(generator.BasePrioPass, types.FilterProtocol8021Q)).
					WithProtocol(types.FilterProtocol8021Q).
					WithMatchKeyVlanEthType(types.FlowerVlanEthTypeIPv4).
					WithAction(types.NewGenericActionBuiler().WithPass().Build()).
					Build())
				expectedFilters.Add(types.NewFlowerFilterBuilder().
					WithPriority(generator.PrioFromBaseAndProtcol(generator.BasePrioPass, types.FilterProtocol8021Q)).
					WithProtocol(types.FilterProtocol8021Q).
					WithMatchKeyVlanEthType(types.FlowerVlanEthTypeIPv6).
					WithAction(types.NewGenericActionBuiler().WithPass().Build()).
					Build())

				filtersEqual(actualFilters, expectedFilters)
			})

			It("generates tc objects for drop rule with IP", func() {
				rules := []policyrules.Rule{{
					IPCidrs: ips,
					Action:  policyrules.PolicyActionDrop,
				}}
				rs.Rules = rules

				tcObj, err := generatorInst.GenerateFromPolicyRuleSet(rs)
				ensureCallAndQdisc(tcObj, err)
				for i := range tcObj.Filters {
					actualFilters.Add(tcObj.Filters[i])
				}

				expectedFilters := filterSetFromFilters(defaultFilters)
				for _, ip := range ips {
					proto := ipToProto(ip.IP)
					prio := generator.PrioFromBaseAndProtcol(generator.BasePrioDrop, proto)
					expectedFilters.Add(
						types.NewFlowerFilterBuilder().
							WithPriority(prio).
							WithProtocol(proto).
							WithMatchKeyDstIP(ip).
							WithAction(types.NewGenericActionBuiler().WithDrop().Build()).
							Build())
					expectedFilters.Add(
						types.NewFlowerFilterBuilder().
							WithPriority(generator.PrioFromBaseAndProtcol(generator.BasePrioDrop,
								types.FilterProtocol8021Q)).
							WithProtocol(types.FilterProtocol8021Q).
							WithMatchKeyVlanEthType(types.ProtoToFlowerVlanEthType(proto)).
							WithMatchKeyDstIP(ip).
							WithAction(types.NewGenericActionBuiler().WithDrop().Build()).
							Build())
				}

				filtersEqual(actualFilters, expectedFilters)
			})

			It("generates tc objects for drop rule with Port", func() {
				rules := []policyrules.Rule{{
					Ports:  ports,
					Action: policyrules.PolicyActionDrop,
				}}
				rs.Rules = rules

				tcObj, err := generatorInst.GenerateFromPolicyRuleSet(rs)
				ensureCallAndQdisc(tcObj, err)
				for i := range tcObj.Filters {
					actualFilters.Add(tcObj.Filters[i])
				}

				expectedFilters := filterSetFromFilters(defaultFilters)
				for _, port := range ports {
					expectedFilters.Add(
						types.NewFlowerFilterBuilder().
							WithPriority(generator.PrioFromBaseAndProtcol(generator.BasePrioDrop,
								types.FilterProtocolIPv4)).
							WithProtocol(types.FilterProtocolIPv4).
							WithMatchKeyIPProto(types.PortProtocolToFlowerIPProto(port.Protocol)).
							WithMatchKeyDstPort(port.Number).
							WithAction(types.NewGenericActionBuiler().WithDrop().Build()).
							Build())
					expectedFilters.Add(
						types.NewFlowerFilterBuilder().
							WithPriority(generator.PrioFromBaseAndProtcol(generator.BasePrioDrop,
								types.FilterProtocolIPv6)).
							WithProtocol(types.FilterProtocolIPv6).
							WithMatchKeyIPProto(types.PortProtocolToFlowerIPProto(port.Protocol)).
							WithMatchKeyDstPort(port.Number).
							WithAction(types.NewGenericActionBuiler().WithDrop().Build()).
							Build())
					expectedFilters.Add(
						types.NewFlowerFilterBuilder().
							WithPriority(generator.PrioFromBaseAndProtcol(generator.BasePrioDrop,
								types.FilterProtocol8021Q)).
							WithProtocol(types.FilterProtocol8021Q).
							WithMatchKeyVlanEthType(types.FlowerVlanEthTypeIPv4).
							WithMatchKeyIPProto(types.PortProtocolToFlowerIPProto(port.Protocol)).
							WithMatchKeyDstPort(port.Number).
							WithAction(types.NewGenericActionBuiler().WithDrop().Build()).
							Build())
					expectedFilters.Add(
						types.NewFlowerFilterBuilder().
							WithPriority(generator.PrioFromBaseAndProtcol(generator.BasePrioDrop,
								types.FilterProtocol8021Q)).
							WithProtocol(types.FilterProtocol8021Q).
							WithMatchKeyVlanEthType(types.FlowerVlanEthTypeIPv6).
							WithMatchKeyIPProto(types.PortProtocolToFlowerIPProto(port.Protocol)).
							WithMatchKeyDstPort(port.Number).
							WithAction(types.NewGenericActionBuiler().WithDrop().Build()).
							Build())
				}

				filtersEqual(actualFilters, expectedFilters)
			})

			It("generates tc objects for drop rule with IP and port", func() {
				rules := []policyrules.Rule{{
					IPCidrs: ips,
					Ports:   ports,
					Action:  policyrules.PolicyActionDrop,
				}}
				rs.Rules = rules

				tcObj, err := generatorInst.GenerateFromPolicyRuleSet(rs)
				ensureCallAndQdisc(tcObj, err)
				for i := range tcObj.Filters {
					actualFilters.Add(tcObj.Filters[i])
				}

				expectedFilters := filterSetFromFilters(defaultFilters)
				for _, ip := range ips {
					proto := ipToProto(ip.IP)
					prio := generator.PrioFromBaseAndProtcol(generator.BasePrioDrop, proto)
					for _, port := range ports {
						expectedFilters.Add(
							types.NewFlowerFilterBuilder().
								WithPriority(prio).
								WithProtocol(proto).
								WithMatchKeyDstIP(ip).
								WithMatchKeyIPProto(types.PortProtocolToFlowerIPProto(port.Protocol)).
								WithMatchKeyDstPort(port.Number).
								WithAction(types.NewGenericActionBuiler().WithDrop().Build()).
								Build())
						expectedFilters.Add(
							types.NewFlowerFilterBuilder().
								WithPriority(generator.PrioFromBaseAndProtcol(generator.BasePrioDrop,
									types.FilterProtocol8021Q)).
								WithProtocol(types.FilterProtocol8021Q).
								WithMatchKeyVlanEthType(types.ProtoToFlowerVlanEthType(proto)).
								WithMatchKeyDstIP(ip).
								WithMatchKeyIPProto(types.PortProtocolToFlowerIPProto(port.Protocol)).
								WithMatchKeyDstPort(port.Number).
								WithAction(types.NewGenericActionBuiler().WithDrop().Build()).
								Build())
					}
				}

				filtersEqual(actualFilters, expectedFilters)
			})

			It("generates tc objects for drop rule with no IP and Port", func() {
				rules := []policyrules.Rule{{
					Action: policyrules.PolicyActionDrop,
				}}
				rs.Rules = rules

				tcObj, err := generatorInst.GenerateFromPolicyRuleSet(rs)
				ensureCallAndQdisc(tcObj, err)
				for i := range tcObj.Filters {
					actualFilters.Add(tcObj.Filters[i])
				}

				expectedFilters := filterSetFromFilters(defaultFilters)
				expectedFilters.Add(types.NewFlowerFilterBuilder().
					WithPriority(generator.PrioFromBaseAndProtcol(generator.BasePrioDrop, types.FilterProtocolIPv4)).
					WithProtocol(types.FilterProtocolIPv4).
					WithAction(types.NewGenericActionBuiler().WithDrop().Build()).
					Build())
				expectedFilters.Add(types.NewFlowerFilterBuilder().
					WithPriority(generator.PrioFromBaseAndProtcol(generator.BasePrioDrop, types.FilterProtocolIPv6)).
					WithProtocol(types.FilterProtocolIPv6).
					WithAction(types.NewGenericActionBuiler().WithDrop().Build()).
					Build())
				expectedFilters.Add(types.NewFlowerFilterBuilder().
					WithPriority(generator.PrioFromBaseAndProtcol(generator.BasePrioDrop, types.FilterProtocol8021Q)).
					WithProtocol(types.FilterProtocol8021Q).
					WithMatchKeyVlanEthType(types.FlowerVlanEthTypeIPv4).
					WithAction(types.NewGenericActionBuiler().WithDrop().Build()).
					Build())
				expectedFilters.Add(types.NewFlowerFilterBuilder().
					WithPriority(generator.PrioFromBaseAndProtcol(generator.BasePrioDrop, types.FilterProtocol8021Q)).
					WithProtocol(types.FilterProtocol8021Q).
					WithMatchKeyVlanEthType(types.FlowerVlanEthTypeIPv6).
					WithAction(types.NewGenericActionBuiler().WithDrop().Build()).
					Build())

				filtersEqual(actualFilters, expectedFilters)
			})

			It("generates tc objects for multiple rules", func() {
				rules := []policyrules.Rule{
					{
						IPCidrs: []*net.IPNet{ips[0]},
						Ports:   []policyrules.Port{ports[0]},
						Action:  policyrules.PolicyActionPass,
					},
					{
						IPCidrs: []*net.IPNet{ips[2]},
						Ports:   []policyrules.Port{ports[1]},
						Action:  policyrules.PolicyActionDrop,
					},
				}
				rs.Rules = rules

				tcObj, err := generatorInst.GenerateFromPolicyRuleSet(rs)
				ensureCallAndQdisc(tcObj, err)
				for i := range tcObj.Filters {
					actualFilters.Add(tcObj.Filters[i])
				}

				expectedFilters := filterSetFromFilters(defaultFilters)
				expectedFilters.Add(
					types.NewFlowerFilterBuilder().
						WithPriority(generator.PrioFromBaseAndProtcol(generator.BasePrioPass,
							types.FilterProtocolIPv4)).
						WithProtocol(types.FilterProtocolIPv4).
						WithMatchKeyDstIP(ips[0]).
						WithMatchKeyIPProto(types.PortProtocolToFlowerIPProto(ports[0].Protocol)).
						WithMatchKeyDstPort(ports[0].Number).
						WithAction(types.NewGenericActionBuiler().WithPass().Build()).
						Build())
				expectedFilters.Add(
					types.NewFlowerFilterBuilder().
						WithPriority(generator.PrioFromBaseAndProtcol(generator.BasePrioPass,
							types.FilterProtocol8021Q)).
						WithProtocol(types.FilterProtocol8021Q).
						WithMatchKeyVlanEthType(types.FlowerVlanEthTypeIPv4).
						WithMatchKeyDstIP(ips[0]).
						WithMatchKeyIPProto(types.PortProtocolToFlowerIPProto(ports[0].Protocol)).
						WithMatchKeyDstPort(ports[0].Number).
						WithAction(types.NewGenericActionBuiler().WithPass().Build()).
						Build())
				expectedFilters.Add(
					types.NewFlowerFilterBuilder().
						WithPriority(generator.PrioFromBaseAndProtcol(generator.BasePrioDrop,
							types.FilterProtocolIPv6)).
						WithProtocol(types.FilterProtocolIPv6).
						WithMatchKeyDstIP(ips[2]).
						WithMatchKeyIPProto(types.PortProtocolToFlowerIPProto(ports[1].Protocol)).
						WithMatchKeyDstPort(ports[1].Number).
						WithAction(types.NewGenericActionBuiler().WithDrop().Build()).
						Build())
				expectedFilters.Add(
					types.NewFlowerFilterBuilder().
						WithPriority(generator.PrioFromBaseAndProtcol(generator.BasePrioDrop,
							types.FilterProtocol8021Q)).
						WithProtocol(types.FilterProtocol8021Q).
						WithMatchKeyVlanEthType(types.FlowerVlanEthTypeIPv6).
						WithMatchKeyDstIP(ips[2]).
						WithMatchKeyIPProto(types.PortProtocolToFlowerIPProto(ports[1].Protocol)).
						WithMatchKeyDstPort(ports[1].Number).
						WithAction(types.NewGenericActionBuiler().WithDrop().Build()).
						Build())

				filtersEqual(actualFilters, expectedFilters)
			})
		})
	})
})
