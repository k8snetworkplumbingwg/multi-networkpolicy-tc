package types_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Mellanox/multi-networkpolicy-tc/pkg/tc/types"
)

var _ = Describe("QDisc tests", func() {
	parent := uint32(0xfffffff1)
	handle := uint32(1)

	assertQdisc := func(q *types.GenericQDisc) {
		ExpectWithOffset(1, q).ToNot(BeNil())
		ExpectWithOffset(1, *q.Parent).To(Equal(parent))
		ExpectWithOffset(1, *q.Handle).To(Equal(handle))
		ExpectWithOffset(1, q.QdiscType).To(Equal(types.QDiscIngressType))
	}

	Describe("Creational", func() {
		Context("NewGenericQDisc", func() {
			It("Creates a new GenericQDisc", func() {
				attr := &types.QDiscAttrs{
					Parent: &parent,
					Handle: &handle,
				}
				q := types.NewGenericQdisc(attr, types.QDiscIngressType)

				assertQdisc(q)
			})
		})

		Context("IngressQDiscBuilder", func() {
			It("Builds Ingress Qdisc with correct attributes", func() {
				q := types.NewIngressQDiscBuilder().WithParent(parent).WithHandle(handle).Build()
				assertQdisc(q)
			})
		})
	})

	Describe("QDisc Interface", func() {
		q := types.NewIngressQDiscBuilder().WithParent(parent).WithHandle(handle).Build()

		Context("Attrs()", func() {
			It("returns expected attrs", func() {
				Expect(*q.Attrs().Parent).To(Equal(parent))
				Expect(*q.Attrs().Handle).To(Equal(handle))
			})
		})

		Context("Type()", func() {
			It("returns expected attrs", func() {
				Expect(q.Type()).To(Equal(types.QDiscIngressType))
			})
		})

		Context("CmdLineGenerator", func() {
			It("generates expected command line args", func() {
				expectedArgs := []string{"ingress"}
				Expect(q.GenCmdLineArgs()).To(Equal(expectedArgs))
			})
		})
	})
})
