package types_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc/types"
)

var _ = Describe("Chain tests", func() {
	parent := uint32(0xfffffff1)
	chain := uint16(0)

	assertChain := func(q *types.ChainImpl) {
		ExpectWithOffset(1, q).ToNot(BeNil())
		ExpectWithOffset(1, *q.Parent).To(Equal(parent))
		ExpectWithOffset(1, *q.Chain).To(Equal(chain))
	}

	Describe("Creational", func() {
		Context("NewChainImpl", func() {
			It("Creates a new ChainImpl", func() {
				c := types.NewChainImpl(&parent, &chain)

				assertChain(c)
			})
		})

		Context("ChainBuilder", func() {
			It("Builds Chain with correct attributes", func() {
				c := types.NewChainBuilder().WithParent(parent).WithChain(chain).Build()

				assertChain(c)
			})
		})
	})

	Describe("Chain Interface", func() {
		c := types.NewChainBuilder().WithParent(parent).WithChain(chain).Build()

		Context("Attrs()", func() {
			It("returns expected attrs", func() {
				Expect(*c.Attrs().Parent).To(Equal(parent))
				Expect(*c.Attrs().Chain).To(Equal(chain))
			})
		})

		Context("CmdLineGenerator", func() {
			It("generates expected command line args", func() {
				expectedArgs := []string{"parent", "ffff:fff1", "chain", "0"}
				Expect(c.GenCmdLineArgs()).To(Equal(expectedArgs))
			})
		})
	})
})
