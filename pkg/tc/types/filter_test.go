package types_test

import (
	"net"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc/types"
	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/utils"
)

var _ = Describe("Filter tests", func() {
	ipToIpNet := func(ip string) *net.IPNet { ipn, _ := utils.IPToIPNet(ip); return ipn }
	passAction := types.NewGenericActionBuiler().WithPass().Build()
	testFilterIPv4 := types.NewFlowerFilterBuilder().
		WithProtocol(types.FilterProtocolIPv4).
		WithPriority(100).
		WithChain(0).
		WithHandle(1).
		WithMatchKeyDstIP(ipToIpNet("10.10.10.0/24")).
		WithMatchKeyIPProto(types.FlowerIPProtoTCP).
		WithMatchKeyDstPort(6666).
		WithAction(passAction).
		Build()
	testFilterIPv6 := types.NewFlowerFilterBuilder().
		WithProtocol(types.FilterProtocolIPv6).
		WithPriority(100).
		WithChain(0).
		WithHandle(1).
		WithMatchKeyDstIP(ipToIpNet("2001::/112")).
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
		WithMatchKeyDstIP(ipToIpNet("10.10.10.0/24")).
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
		WithMatchKeyDstIP(ipToIpNet("2001::/112")).
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
				Expect(testFilterIPv4.Flower.DstIP.String()).To(Equal("10.10.10.0/24"))
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
					WithMatchKeyDstIP(ipToIpNet("10.10.10.0/24")).
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
					WithMatchKeyDstIP(ipToIpNet("10.10.10.0/24")).
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

			It("retuns true for filters with/without /32 mask for ipv4 dest IP", func() {
				filter1 := types.NewFlowerFilterBuilder().
					WithProtocol(types.FilterProtocolIPv4).
					WithMatchKeyDstIP(ipToIpNet("192.168.10.11/32")).
					Build()
				filter2 := types.NewFlowerFilterBuilder().
					WithProtocol(types.FilterProtocolIPv4).
					WithMatchKeyDstIP(ipToIpNet("192.168.10.11")).
					Build()
				Expect(filter1.Equals(filter2)).To(BeTrue())
			})

			It("retuns true for filters with/without /128 mask for ipv6 dest IP", func() {
				filter1 := types.NewFlowerFilterBuilder().
					WithProtocol(types.FilterProtocolIPv6).
					WithMatchKeyDstIP(ipToIpNet("2001:0db8:3c4d:0015:0000:d234::3eee:12be/128")).
					Build()
				filter2 := types.NewFlowerFilterBuilder().
					WithProtocol(types.FilterProtocolIPv6).
					WithMatchKeyDstIP(ipToIpNet("2001:0db8:3c4d:0015:0000:d234::3eee:12be")).
					Build()
				Expect(filter1.Equals(filter2)).To(BeTrue())
			})

			It("returns false for filters with different IPv4 addresses", func() {
				filter1 := types.NewFlowerFilterBuilder().
					WithProtocol(types.FilterProtocolIPv4).
					WithMatchKeyDstIP(ipToIpNet("192.168.10.11")).
					Build()
				filter2 := types.NewFlowerFilterBuilder().
					WithProtocol(types.FilterProtocolIPv4).
					WithMatchKeyDstIP(ipToIpNet("192.168.10.12")).
					Build()
				Expect(filter1.Equals(filter2)).To(BeFalse())
			})

			It("returns false for filters with/without IP addresses", func() {
				filter1 := types.NewFlowerFilterBuilder().
					WithProtocol(types.FilterProtocolIPv4).
					WithMatchKeyDstIP(ipToIpNet("192.168.10.11")).
					Build()
				filter2 := types.NewFlowerFilterBuilder().
					WithProtocol(types.FilterProtocolIPv4).
					Build()
				Expect(filter1.Equals(filter2)).To(BeFalse())
			})

			It("returns false for filters with different IPv6 addresses", func() {
				filter1 := types.NewFlowerFilterBuilder().
					WithProtocol(types.FilterProtocolIPv6).
					WithMatchKeyDstIP(ipToIpNet("2001::ff00")).
					Build()
				filter2 := types.NewFlowerFilterBuilder().
					WithProtocol(types.FilterProtocolIPv6).
					WithMatchKeyDstIP(ipToIpNet("2001::abcd")).
					Build()
				Expect(filter1.Equals(filter2)).To(BeFalse())
			})

			It("returns false for filters with different IPv4 masks", func() {
				ip := net.IP{0x10, 0x20, 0x30, 0x2}
				msk1 := net.IPMask{0xff, 0xff, 0xff, 0xf0}
				msk2 := net.IPMask{0xff, 0xff, 0xff, 0x00}
				filter1 := types.NewFlowerFilterBuilder().
					WithProtocol(types.FilterProtocolIPv4).
					WithMatchKeyDstIP(&net.IPNet{IP: ip, Mask: msk1}).
					Build()
				filter2 := types.NewFlowerFilterBuilder().
					WithProtocol(types.FilterProtocolIPv4).
					WithMatchKeyDstIP(&net.IPNet{IP: ip, Mask: msk2}).
					Build()
				Expect(filter1.Equals(filter2)).To(BeFalse())
			})

			It("returns false for filters with different IPv6 masks", func() {
				ip := net.IP{0x10, 0x20, 0x30, 0x2, 0x10, 0x20, 0x30, 0x2, 0x10, 0x20,
					0x30, 0x2, 0x10, 0x20, 0x30, 0x2}
				msk1 := net.IPMask{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x00}
				msk2 := net.IPMask{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
					0xff, 0xff, 0xff, 0xff, 0xff, 0x00, 0x00}

				filter1 := types.NewFlowerFilterBuilder().
					WithProtocol(types.FilterProtocolIPv6).
					WithMatchKeyDstIP(&net.IPNet{IP: ip, Mask: msk1}).
					Build()
				filter2 := types.NewFlowerFilterBuilder().
					WithProtocol(types.FilterProtocolIPv6).
					WithMatchKeyDstIP(&net.IPNet{IP: ip, Mask: msk2}).
					Build()
				Expect(filter1.Equals(filter2)).To(BeFalse())
			})
		})

		Context("CmdLineGenerator", func() {
			It("generates expected command line args - ipv4", func() {
				expectedArgs := []string{
					"protocol", "ip", "handle", "1", "chain", "0", "pref", "100", "flower",
					"ip_proto", "tcp", "dst_ip", "10.10.10.0/24", "dst_port", "6666", "action", "gact", "pass"}
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
					"vlan_ethtype", "ip", "ip_proto", "tcp", "dst_ip", "10.10.10.0/24", "dst_port", "6666",
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
