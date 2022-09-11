package tc_test

import (
	"net"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Mellanox/multi-networkpolicy-tc/pkg/policyrules"
	"github.com/Mellanox/multi-networkpolicy-tc/pkg/tc"
	"github.com/Mellanox/multi-networkpolicy-tc/pkg/tc/types"
)

func ensureCallAndQdisc(tcObj *tc.Objects, err error) {
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

var _ = Describe("SimpleTCGenerator tests", func() {
	var generator tc.Generator
	defaultDropFliter := types.NewFlowerFilterBuilder().
		WithPriority(tc.PrioDefault).
		WithProtocol(types.FilterProtocolIP).
		WithAction(types.NewGenericActionBuiler().WithDrop().Build()).
		Build()

	BeforeEach(func() {
		generator = tc.NewSimpleTCGenerator()
	})

	Context("GenerateFromPolicyRuleSet() Basic", func() {
		It("generates no objects if PolicyRuleSet with nil rules", func() {
			rs := policyrules.PolicyRuleSet{
				IfcInfo: policyrules.InterfaceInfo{},
				Type:    policyrules.PolicyTypeEgress,
				Rules:   nil,
			}
			tcObj, err := generator.GenerateFromPolicyRuleSet(rs)

			ensureCallAndQdisc(tcObj, err)
			Expect(tcObj.Filters).To(BeEmpty())
		})

		It("generates objects with default drop filter if PolicyRuleSet with zero rules", func() {
			rs := policyrules.PolicyRuleSet{
				IfcInfo: policyrules.InterfaceInfo{},
				Type:    policyrules.PolicyTypeEgress,
				Rules:   make([]policyrules.Rule, 0),
			}
			tcObj, err := generator.GenerateFromPolicyRuleSet(rs)

			ensureCallAndQdisc(tcObj, err)
			Expect(tcObj.Filters).To(HaveLen(1))
			Expect(tcObj.Filters[0].Equals(defaultDropFliter)).To(BeTrue())
		})

		It("fails to generate objects if PolicyRuleSet is Ingress", func() {
			rs := policyrules.PolicyRuleSet{
				IfcInfo: policyrules.InterfaceInfo{},
				Type:    policyrules.PolicyTypeIngress,
				Rules:   make([]policyrules.Rule, 0),
			}
			_, err := generator.GenerateFromPolicyRuleSet(rs)
			Expect(err).To(HaveOccurred())
		})

		Context("GenerateFromPolicyRuleSet() With policy Rules", func() {
			var rs policyrules.PolicyRuleSet
			var expectedFilters tc.FilterSet
			var actualFilters tc.FilterSet
			var ips = []*net.IPNet{
				ipnetFromStr("192.168.1.2/32"),
				ipnetFromStr("10.100.1.1/24"),
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
				expectedFilters = tc.NewFilterSetImpl()
				actualFilters = tc.NewFilterSetImpl()
			})

			It("generates tc objects for pass rule with IP", func() {
				rules := []policyrules.Rule{{
					IPCidrs: ips,
					Action:  policyrules.PolicyActionPass,
				}}
				rs.Rules = rules

				tcObj, err := generator.GenerateFromPolicyRuleSet(rs)
				ensureCallAndQdisc(tcObj, err)
				for i := range tcObj.Filters {
					actualFilters.Add(tcObj.Filters[i])
				}

				expectedFilters.Add(defaultDropFliter)
				for _, ip := range ips {
					expectedFilters.Add(
						types.NewFlowerFilterBuilder().
							WithPriority(tc.PrioPass).
							WithProtocol(types.FilterProtocolIP).
							WithMatchKeyDstIP(ip.String()).
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

				tcObj, err := generator.GenerateFromPolicyRuleSet(rs)
				ensureCallAndQdisc(tcObj, err)
				for i := range tcObj.Filters {
					actualFilters.Add(tcObj.Filters[i])
				}

				expectedFilters.Add(defaultDropFliter)
				for _, port := range ports {
					expectedFilters.Add(
						types.NewFlowerFilterBuilder().
							WithPriority(tc.PrioPass).
							WithProtocol(types.FilterProtocolIP).
							WithMatchKeyIPProto(string(port.Protocol)).
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

				tcObj, err := generator.GenerateFromPolicyRuleSet(rs)
				ensureCallAndQdisc(tcObj, err)
				for i := range tcObj.Filters {
					actualFilters.Add(tcObj.Filters[i])
				}

				expectedFilters.Add(defaultDropFliter)
				for _, ip := range ips {
					for _, port := range ports {
						expectedFilters.Add(
							types.NewFlowerFilterBuilder().
								WithPriority(tc.PrioPass).
								WithProtocol(types.FilterProtocolIP).
								WithMatchKeyDstIP(ip.String()).
								WithMatchKeyIPProto(string(port.Protocol)).
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

				tcObj, err := generator.GenerateFromPolicyRuleSet(rs)
				ensureCallAndQdisc(tcObj, err)
				for i := range tcObj.Filters {
					actualFilters.Add(tcObj.Filters[i])
				}

				expectedFilters.Add(defaultDropFliter)
				expectedFilters.Add(types.NewFlowerFilterBuilder().
					WithPriority(tc.PrioPass).
					WithProtocol(types.FilterProtocolIP).
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

				tcObj, err := generator.GenerateFromPolicyRuleSet(rs)
				ensureCallAndQdisc(tcObj, err)
				for i := range tcObj.Filters {
					actualFilters.Add(tcObj.Filters[i])
				}

				expectedFilters.Add(defaultDropFliter)
				for _, ip := range ips {
					expectedFilters.Add(
						types.NewFlowerFilterBuilder().
							WithPriority(tc.PrioDrop).
							WithProtocol(types.FilterProtocolIP).
							WithMatchKeyDstIP(ip.String()).
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

				tcObj, err := generator.GenerateFromPolicyRuleSet(rs)
				ensureCallAndQdisc(tcObj, err)
				for i := range tcObj.Filters {
					actualFilters.Add(tcObj.Filters[i])
				}

				expectedFilters.Add(defaultDropFliter)
				for _, port := range ports {
					expectedFilters.Add(
						types.NewFlowerFilterBuilder().
							WithPriority(tc.PrioDrop).
							WithProtocol(types.FilterProtocolIP).
							WithMatchKeyIPProto(string(port.Protocol)).
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

				tcObj, err := generator.GenerateFromPolicyRuleSet(rs)
				ensureCallAndQdisc(tcObj, err)
				for i := range tcObj.Filters {
					actualFilters.Add(tcObj.Filters[i])
				}

				expectedFilters.Add(defaultDropFliter)
				for _, ip := range ips {
					for _, port := range ports {
						expectedFilters.Add(
							types.NewFlowerFilterBuilder().
								WithPriority(tc.PrioDrop).
								WithProtocol(types.FilterProtocolIP).
								WithMatchKeyDstIP(ip.String()).
								WithMatchKeyIPProto(string(port.Protocol)).
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

				tcObj, err := generator.GenerateFromPolicyRuleSet(rs)
				ensureCallAndQdisc(tcObj, err)
				for i := range tcObj.Filters {
					actualFilters.Add(tcObj.Filters[i])
				}

				expectedFilters.Add(defaultDropFliter)
				expectedFilters.Add(types.NewFlowerFilterBuilder().
					WithPriority(tc.PrioDrop).
					WithProtocol(types.FilterProtocolIP).
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
						IPCidrs: []*net.IPNet{ips[1]},
						Ports:   []policyrules.Port{ports[1]},
						Action:  policyrules.PolicyActionDrop,
					},
				}
				rs.Rules = rules

				tcObj, err := generator.GenerateFromPolicyRuleSet(rs)
				ensureCallAndQdisc(tcObj, err)
				for i := range tcObj.Filters {
					actualFilters.Add(tcObj.Filters[i])
				}

				expectedFilters.Add(defaultDropFliter)
				expectedFilters.Add(
					types.NewFlowerFilterBuilder().
						WithPriority(tc.PrioPass).
						WithProtocol(types.FilterProtocolIP).
						WithMatchKeyDstIP(ips[0].String()).
						WithMatchKeyIPProto(string(ports[0].Protocol)).
						WithMatchKeyDstPort(ports[0].Number).
						WithAction(types.NewGenericActionBuiler().WithPass().Build()).
						Build())
				expectedFilters.Add(
					types.NewFlowerFilterBuilder().
						WithPriority(tc.PrioDrop).
						WithProtocol(types.FilterProtocolIP).
						WithMatchKeyDstIP(ips[1].String()).
						WithMatchKeyIPProto(string(ports[1].Protocol)).
						WithMatchKeyDstPort(ports[1].Number).
						WithAction(types.NewGenericActionBuiler().WithDrop().Build()).
						Build())

				filtersEqual(actualFilters, expectedFilters)
			})
		})
	})
})
