package types_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Mellanox/multi-networkpolicy-tc/pkg/tc/types"
)

var _ = Describe("Action tests", func() {
	Describe("Creational", func() {
		Context("NewGenericAction", func() {
			It("Creates a new generic action", func() {
				ga := types.NewGenericAction(types.ActionGenericPass)
				Expect(ga).ToNot(BeNil())
			})
		})

		Context("GenericActionBuilder", func() {
			It("Builds GenericAction with correct attributes", func() {
				ga := types.NewGenericActionBuiler().WithPass().Build()
				Expect(ga).ToNot(BeNil())
				Expect(ga.Type()).To(Equal(types.ActionTypeGeneric))
				Expect(ga.Spec()).To(HaveKey("control_action"))
				Expect(ga.Spec()["control_action"]).To(BeEquivalentTo(types.ActionGenericPass))
			})
		})
	})

	Describe("Action Interface", func() {
		ga := types.NewGenericActionBuiler().WithPass().Build()

		Context("Type()", func() {
			It("returns expected type", func() {
				Expect(ga.Type()).To(Equal(types.ActionTypeGeneric))
			})
		})

		Context("Spec()", func() {
			It("returns expected spec", func() {
				expectedSpec := map[string]string{"control_action": "pass"}
				Expect(ga.Spec()).To(Equal(expectedSpec))
			})
		})

		Context("Equals()", func() {
			It("returns true if Actions are equal", func() {
				ga2 := types.NewGenericActionBuiler().WithPass().Build()
				Expect(ga.Equals(ga2)).To(BeTrue())
			})

			It("returns false if Actions are not equal", func() {
				ga2 := types.NewGenericActionBuiler().WithDrop().Build()
				Expect(ga.Equals(ga2)).To(BeFalse())
			})
		})

		Context("CmdLineGenerator", func() {
			It("generates expected command line args", func() {
				expectedArgs := []string{"action", "gact", "pass"}
				Expect(ga.GenCmdLineArgs()).To(Equal(expectedArgs))
			})
		})
	})
})
