package types_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc/types"
)

var _ = Describe("Filter tests", func() {
	passAction := types.NewGenericActionBuiler().WithPass().Build()
	testFilterIPv4 := types.NewFlowerFilterBuilder().
		WithProtocol(types.FilterProtocolIPv4).
		WithPriority(100).
		WithChain(0).
		WithHandle(1).
		WithMatchKeyDstIP("10.10.10.10/24").
		WithMatchKeyIPProto(types.FlowerIPProtoTCP).
		WithMatchKeyDstPort(6666).
		WithAction(passAction).
		Build()
	testFilterIPv6 := types.NewFlowerFilterBuilder().
		WithProtocol(types.FilterProtocolIPv6).
		WithPriority(100).
		WithChain(0).
		WithHandle(1).
		WithMatchKeyDstIP("2001::/112").
		WithMatchKeyIPProto(types.FlowerIPProtoTCP).
		WithMatchKeyDstPort(6666).
		WithAction(passAction).
		Build()
	testFilterVlanIPv4 := types.NewFlowerFilterBuilder().
		WithProtocol(types.FilterProtocol8021Q).
		WithPriority(100).
		WithChain(0).
		WithHandle(1).
		WithMatchKeyVlanEthType(types.FlowerVlanEthTypeIPv4).
		WithMatchKeyDstIP("10.10.10.10/24").
		WithMatchKeyIPProto(types.FlowerIPProtoTCP).
		WithMatchKeyDstPort(6666).
		WithAction(passAction).
		Build()
	testFilterVlanIPv6 := types.NewFlowerFilterBuilder().
		WithProtocol(types.FilterProtocol8021Q).
		WithPriority(100).
		WithChain(0).
		WithHandle(1).
		WithMatchKeyVlanEthType(types.FlowerVlanEthTypeIPv6).
		WithMatchKeyDstIP("2001::/112").
		WithMatchKeyIPProto(types.FlowerIPProtoTCP).
		WithMatchKeyDstPort(6666).
		WithAction(passAction).
		Build()

	Describe("Creational", func() {
		Context("FlowerFilterBuilder", func() {
			It("Builds FlowerFilter with correct attributes", func() {
				Expect(testFilterIPv4.Protocol).To(Equal(types.FilterProtocolIPv4))
				Expect(*testFilterIPv4.Priority).To(BeEquivalentTo(100))
				Expect(*testFilterIPv4.Chain).To(BeEquivalentTo(0))
				Expect(*testFilterIPv4.Handle).To(BeEquivalentTo(1))
				Expect(testFilterIPv4.Flower).ToNot(BeNil())
				Expect(*testFilterIPv4.Flower.DstIP).To(Equal("10.10.10.10/24"))
				Expect(*testFilterIPv4.Flower.IPProto).To(Equal(types.FlowerIPProtoTCP))
				Expect(*testFilterIPv4.Flower.DstPort).To(BeEquivalentTo(6666))
				Expect(testFilterIPv4.Actions).To(BeEquivalentTo([]types.Action{passAction}))
			})

			It("Builds FlowerFilter with correct attributes for Vlan Protocol", func() {
				Expect(testFilterVlanIPv4.Protocol).To(Equal(types.FilterProtocol8021Q))
				Expect(testFilterVlanIPv4.Flower).ToNot(BeNil())
				Expect(*testFilterVlanIPv4.Flower.VlanEthType).To(Equal(types.FlowerVlanEthTypeIPv4))
			})
		})
	})

	Describe("Filter Interface", func() {
		Context("Attrs()", func() {
			It("returns expected attrs", func() {
				Expect(testFilterIPv4.Attrs().Protocol).To(Equal(types.FilterProtocolIPv4))
				Expect(*testFilterIPv4.Attrs().Priority).To(BeEquivalentTo(100))
				Expect(*testFilterIPv4.Attrs().Chain).To(BeEquivalentTo(0))
				Expect(*testFilterIPv4.Attrs().Handle).To(BeEquivalentTo(1))
			})
		})

		Context("Equals()", func() {
			// Note(adrianc): Tests below can be made much more exhaustive
			It("returns true if filters are equal", func() {
				other := types.NewFlowerFilterBuilder().
					WithProtocol(types.FilterProtocolIPv4).
					WithPriority(100).
					WithChain(0).
					WithHandle(1).
					WithMatchKeyDstIP("10.10.10.10/24").
					WithMatchKeyIPProto(types.FlowerIPProtoTCP).
					WithMatchKeyDstPort(6666).
					WithAction(passAction).
					Build()
				Expect(testFilterIPv4.Equals(other)).To(BeTrue())
			})

			It("returns true if filters are equal with and without default chain", func() {
				other := types.NewFlowerFilterBuilder().
					WithProtocol(types.FilterProtocolIPv4).
					WithPriority(100).
					WithHandle(1).
					WithMatchKeyDstIP("10.10.10.10/24").
					WithMatchKeyIPProto(types.FlowerIPProtoTCP).
					WithMatchKeyDstPort(6666).
					WithAction(passAction).
					Build()
				Expect(testFilterIPv4.Equals(other)).To(BeTrue())
			})

			It("returns false if filters are not equal", func() {
				other := types.NewFlowerFilterBuilder().
					WithProtocol(types.FilterProtocolIPv4).
					WithPriority(200).
					WithHandle(1).
					Build()
				Expect(testFilterIPv4.Equals(other)).To(BeFalse())
			})

			It("returns false if filters are not equal - different protocol", func() {
				Expect(testFilterIPv4.Equals(testFilterVlanIPv4)).To(BeFalse())
			})

			It("returns false if filters are not equal - vlan protocol, different attributes", func() {
				Expect(testFilterVlanIPv4.Equals(testFilterVlanIPv6)).To(BeFalse())
			})

			It("returns false if filters are not equal - vlan protocol, different eth type", func() {
				filter1 := types.NewFlowerFilterBuilder().
					WithProtocol(types.FilterProtocol8021Q).
					WithMatchKeyVlanEthType(types.FlowerVlanEthTypeIPv4).
					WithAction(passAction).
					Build()
				filter2 := types.NewFlowerFilterBuilder().
					WithProtocol(types.FilterProtocol8021Q).
					WithMatchKeyVlanEthType(types.FlowerVlanEthTypeIPv6).
					WithAction(passAction).
					Build()
				Expect(filter1.Equals(filter2)).To(BeFalse())
			})
		})

		Context("CmdLineGenerator", func() {
			It("generates expected command line args - ipv4", func() {
				expectedArgs := []string{
					"protocol", "ip", "handle", "1", "chain", "0", "pref", "100", "flower",
					"ip_proto", "tcp", "dst_ip", "10.10.10.10/24", "dst_port", "6666", "action", "gact", "pass"}
				Expect(testFilterIPv4.GenCmdLineArgs()).To(Equal(expectedArgs))
			})

			It("generates expected command line args - ipv6", func() {
				expectedArgs := []string{
					"protocol", "ipv6", "handle", "1", "chain", "0", "pref", "100", "flower",
					"ip_proto", "tcp", "dst_ip", "2001::/112", "dst_port", "6666", "action", "gact", "pass"}
				Expect(testFilterIPv6.GenCmdLineArgs()).To(Equal(expectedArgs))
			})

			It("generates expected command line args - vlan ipv4", func() {
				expectedArgs := []string{
					"protocol", "802.1q", "handle", "1", "chain", "0", "pref", "100", "flower",
					"vlan_ethtype", "ip", "ip_proto", "tcp", "dst_ip", "10.10.10.10/24", "dst_port", "6666",
					"action", "gact", "pass"}
				Expect(testFilterVlanIPv4.GenCmdLineArgs()).To(Equal(expectedArgs))
			})

			It("generates expected command line args - vlan ipv6", func() {
				expectedArgs := []string{
					"protocol", "802.1q", "handle", "1", "chain", "0", "pref", "100", "flower",
					"vlan_ethtype", "ipv6", "ip_proto", "tcp", "dst_ip", "2001::/112", "dst_port", "6666",
					"action", "gact", "pass"}
				Expect(testFilterVlanIPv6.GenCmdLineArgs()).To(Equal(expectedArgs))
			})
		})
	})
})
