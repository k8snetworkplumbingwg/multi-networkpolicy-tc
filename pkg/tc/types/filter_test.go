package types_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Mellanox/multi-networkpolicy-tc/pkg/tc/types"
)

var _ = Describe("Filter tests", func() {
	passAction := types.NewGenericActionBuiler().WithPass().Build()
	f := types.NewFlowerFilterBuilder().
		WithProtocol(types.FilterProtocolIP).
		WithPriority(100).
		WithChain(0).
		WithHandle(1).
		WithMatchKeyDstIP("10.10.10.10/24").
		WithMatchKeyIPProto("tcp").
		WithMatchKeyDstPort(6666).
		WithAction(passAction).
		Build()

	Describe("Creational", func() {
		Context("FlowerFilterBuilder", func() {
			It("Builds FlowerFilter with correct attributes", func() {
				Expect(f.Protocol).To(Equal(types.FilterProtocolIP))
				Expect(*f.Priority).To(BeEquivalentTo(100))
				Expect(*f.Chain).To(BeEquivalentTo(0))
				Expect(*f.Handle).To(BeEquivalentTo(1))
				Expect(f.Flower).ToNot(BeNil())
				Expect(*f.Flower.DstIP).To(Equal("10.10.10.10/24"))
				Expect(*f.Flower.IPProto).To(Equal("tcp"))
				Expect(*f.Flower.DstPort).To(BeEquivalentTo(6666))
				Expect(f.Actions).To(BeEquivalentTo([]types.Action{passAction}))
			})
		})
	})

	Describe("Filter Interface", func() {
		Context("Attrs()", func() {
			It("returns expected attrs", func() {
				Expect(f.Attrs().Protocol).To(Equal(types.FilterProtocolIP))
				Expect(*f.Attrs().Priority).To(BeEquivalentTo(100))
				Expect(*f.Attrs().Chain).To(BeEquivalentTo(0))
				Expect(*f.Attrs().Handle).To(BeEquivalentTo(1))
			})
		})

		Context("Equals()", func() {
			// Note(adrianc): Tests below can be made much more exhaustive
			It("returns true if filters are equal", func() {
				other := types.NewFlowerFilterBuilder().
					WithProtocol(types.FilterProtocolIP).
					WithPriority(100).
					WithChain(0).
					WithHandle(1).
					WithMatchKeyDstIP("10.10.10.10/24").
					WithMatchKeyIPProto("tcp").
					WithMatchKeyDstPort(6666).
					WithAction(passAction).
					Build()
				Expect(f.Equals(other)).To(BeTrue())
			})

			It("returns true if filters are equal with and without default chain", func() {
				other := types.NewFlowerFilterBuilder().
					WithProtocol(types.FilterProtocolIP).
					WithPriority(100).
					WithHandle(1).
					WithMatchKeyDstIP("10.10.10.10/24").
					WithMatchKeyIPProto("tcp").
					WithMatchKeyDstPort(6666).
					WithAction(passAction).
					Build()
				Expect(f.Equals(other)).To(BeTrue())
			})

			It("returns false if filters are not equal", func() {
				other := types.NewFlowerFilterBuilder().
					WithProtocol(types.FilterProtocolIP).
					WithPriority(200).
					WithHandle(1).
					Build()
				Expect(f.Equals(other)).To(BeFalse())
			})
		})

		Context("CmdLineGenerator", func() {
			It("generates expected command line args", func() {
				expectedArgs := []string{
					"protocol", "ip", "handle", "1", "chain", "0", "pref", "100", "flower",
					"ip_proto", "tcp", "dst_ip", "10.10.10.10/24", "dst_port", "6666", "action", "gact", "pass"}
				Expect(f.GenCmdLineArgs()).To(Equal(expectedArgs))
			})
		})
	})
})
