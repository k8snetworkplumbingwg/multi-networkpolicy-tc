package tc_test

import (
	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"net"

	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc"
	tctypes "github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc/types"
)

var _ = Describe("FilterSetImpl tests", func() {
	var filterSet tc.FilterSet
	ipToIpNet := func(ip string) *net.IPNet { ipn, _ := utils.IPToIPNet(ip); return ipn }

	BeforeEach(func() {
		filterSet = tc.NewFilterSetImpl()
	})

	Context("FilterSet.Add()", func() {
		It("Adds filter to FilterSet", func() {
			filters := []tctypes.Filter{
				tctypes.NewFlowerFilterBuilder().
					WithProtocol(tctypes.FilterProtocolIPv4).
					WithAction(
						tctypes.NewGenericActionBuiler().
							WithDrop().
							Build()).
					Build(),
				tctypes.NewFlowerFilterBuilder().
					WithProtocol(tctypes.FilterProtocolIPv4).
					WithAction(
						tctypes.NewGenericActionBuiler().
							WithPass().
							Build()).
					Build(),
			}
			for i := range filters {
				filterSet.Add(filters[i])
			}

			filterList := filterSet.List()
			Expect(filterList).To(HaveLen(2))
			Expect(filterList).To(ContainElements(filters))
		})

		It("Does not add an already existing filter to FilterSet", func() {
			filter := tctypes.NewFlowerFilterBuilder().
				WithProtocol(tctypes.FilterProtocolIPv4).
				WithAction(
					tctypes.NewGenericActionBuiler().WithDrop().Build()).
				Build()

			filterSet.Add(filter)
			filterSet.Add(filter)

			filterList := filterSet.List()
			Expect(filterList).To(HaveLen(1))
			Expect(filterList).To(ContainElement(filter))
		})
	})

	Context("FilterSet.Remove()", func() {
		It("removes filter from set if exists", func() {
			filter := tctypes.NewFlowerFilterBuilder().
				WithProtocol(tctypes.FilterProtocolIPv4).
				WithAction(
					tctypes.NewGenericActionBuiler().
						WithDrop().
						Build()).
				Build()
			filterSet.Add(filter)
			Expect(filterSet.Len()).To(Equal(1))
			filterSet.Remove(filter)
			Expect(filterSet.Len()).To(Equal(0))
		})

		It("does not remove filter from set if does not exist", func() {
			filterToAdd := tctypes.NewFlowerFilterBuilder().
				WithProtocol(tctypes.FilterProtocolIPv4).
				WithAction(
					tctypes.NewGenericActionBuiler().
						WithDrop().
						Build()).
				Build()
			filterToRemove := tctypes.NewFlowerFilterBuilder().
				WithProtocol(tctypes.FilterProtocolAll).
				WithAction(
					tctypes.NewGenericActionBuiler().
						WithDrop().
						Build()).
				Build()
			filterSet.Add(filterToAdd)
			Expect(filterSet.Len()).To(Equal(1))
			filterSet.Remove(filterToRemove)
			Expect(filterSet.Len()).To(Equal(1))
			Expect(filterSet.List()).To(ContainElement(filterToAdd))
		})
	})

	Context("FilterSet.Has()", func() {
		var filter tctypes.Filter

		BeforeEach(func() {
			filter = tctypes.NewFlowerFilterBuilder().
				WithProtocol(tctypes.FilterProtocolIPv4).
				WithMatchKeyDstIP(ipToIpNet("192.168.1.2")).
				Build()
			filterSet.Add(filter)
		})

		It("returns true if Filter in set", func() {
			Expect(filterSet.Has(filter)).To(BeTrue())
		})

		It("returns false if Filter no in set", func() {
			otherFilter := tctypes.NewFlowerFilterBuilder().
				WithProtocol(tctypes.FilterProtocolIPv4).
				WithMatchKeyDstIP(ipToIpNet("192.168.2.2")).
				Build()
			Expect(filterSet.Has(otherFilter)).To(BeFalse())
		})
	})

	Context("FilterSet.Len()", func() {
		It("returns zero if no Filters", func() {
			Expect(filterSet.Len()).To(BeZero())
		})

		It("returns number of filters in set", func() {
			filters := []tctypes.Filter{
				tctypes.NewFlowerFilterBuilder().
					WithProtocol(tctypes.FilterProtocolIPv4).
					Build(),
				tctypes.NewFlowerFilterBuilder().
					WithProtocol(tctypes.FilterProtocolAll).
					Build(),
			}
			for i := range filters {
				filterSet.Add(filters[i])
			}
			Expect(filterSet.Len()).To(Equal(2))
		})
	})

	Context("FilterSet.In()", func() {
		var this tc.FilterSet
		var other tc.FilterSet
		filters := []tctypes.Filter{
			tctypes.NewFlowerFilterBuilder().
				WithProtocol(tctypes.FilterProtocolIPv4).
				WithAction(
					tctypes.NewGenericActionBuiler().
						WithDrop().
						Build()).
				Build(),
			tctypes.NewFlowerFilterBuilder().
				WithProtocol(tctypes.FilterProtocolIPv4).
				WithAction(
					tctypes.NewGenericActionBuiler().
						WithPass().
						Build()).
				Build(),
			tctypes.NewFlowerFilterBuilder().
				WithProtocol(tctypes.FilterProtocolIPv4).
				WithMatchKeyDstIP(ipToIpNet("192.168.1.1")).
				WithAction(
					tctypes.NewGenericActionBuiler().
						WithPass().
						Build()).
				Build(),
		}

		BeforeEach(func() {
			this = filterSet
			other = tc.NewFilterSetImpl()
		})

		It("returns true if this set in other", func() {
			for i := range filters {
				other.Add(filters[i])
			}
			for i := 0; i < len(filters)-1; i++ {
				this.Add(filters[i])
			}
			Expect(this.In(other)).To(BeTrue())
		})

		It("returns true if sets are equal", func() {
			for i := range filters {
				this.Add(filters[i])
				other.Add(filters[i])
			}
			Expect(this.In(other)).To(BeTrue())
		})

		It("returns true if both sets empty", func() {
			Expect(this.In(other)).To(BeTrue())
		})

		It("returns true if this set is empty", func() {
			other.Add(filters[0])
			Expect(this.In(other)).To(BeTrue())
		})

		It("returns false if this set not in other, completely disjoint", func() {
			this.Add(filters[0])
			other.Add(filters[1])
			Expect(this.In(other)).To(BeFalse())
		})

		It("returns false if this set not in other, partially disjoint different set len", func() {
			for i := range filters {
				this.Add(filters[i])
			}
			for i := 0; i < len(filters)-1; i++ {
				other.Add(filters[i])
			}
			Expect(this.In(other)).To(BeFalse())
		})

		It("returns false if this set not in other, partially disjoint same set len", func() {
			for i := 1; i < len(filters); i++ {
				this.Add(filters[i])
			}
			for i := 0; i < len(filters)-1; i++ {
				other.Add(filters[i])
			}
			Expect(this.In(other)).To(BeFalse())
		})

		It("returns false if other is empty but this is not", func() {
			for i := 0; i < len(filters); i++ {
				this.Add(filters[i])
			}
			Expect(this.In(other)).To(BeFalse())
		})
	})

	Context("FilterSet.Intersect()", func() {
		var this tc.FilterSet
		var other tc.FilterSet
		filters := []tctypes.Filter{
			tctypes.NewFlowerFilterBuilder().
				WithProtocol(tctypes.FilterProtocolIPv4).
				WithAction(
					tctypes.NewGenericActionBuiler().
						WithDrop().
						Build()).
				Build(),
			tctypes.NewFlowerFilterBuilder().
				WithProtocol(tctypes.FilterProtocolIPv4).
				WithAction(
					tctypes.NewGenericActionBuiler().
						WithPass().
						Build()).
				Build(),
			tctypes.NewFlowerFilterBuilder().
				WithProtocol(tctypes.FilterProtocolIPv4).
				WithMatchKeyDstIP(ipToIpNet("192.168.1.1")).
				WithAction(
					tctypes.NewGenericActionBuiler().
						WithPass().
						Build()).
				Build(),
			tctypes.NewFlowerFilterBuilder().
				WithProtocol(tctypes.FilterProtocolAll).
				WithAction(
					tctypes.NewGenericActionBuiler().
						WithDrop().
						Build()).
				Build(),
		}

		BeforeEach(func() {
			this = filterSet
			other = tc.NewFilterSetImpl()
		})

		It("returns this if this is in other", func() {
			for i := range filters {
				other.Add(filters[i])
			}
			for i := 0; i < len(filters)-1; i++ {
				this.Add(filters[i])
			}
			common := this.Intersect(other)
			Expect(common.Len()).To(Equal(len(filters) - 1))
			Expect(common.Equals(this)).To(BeTrue())
		})

		It("returns filter set with common items from this and other", func() {
			this.Add(filters[0])
			this.Add(filters[1])
			this.Add(filters[2])
			other.Add(filters[1])
			other.Add(filters[2])
			other.Add(filters[3])

			common := this.Intersect(other)
			Expect(common.Len()).To(Equal(2))
			Expect(common.Has(filters[1])).To(BeTrue())
			Expect(common.Has(filters[2])).To(BeTrue())
		})

		It("returns empty filter set if this and other are disjoint", func() {
			this.Add(filters[0])
			this.Add(filters[1])
			other.Add(filters[2])
			other.Add(filters[3])

			common := this.Intersect(other)
			Expect(common.Len()).To(BeZero())
		})

		It("returns empty filter set if this is empty", func() {
			for i := range filters {
				other.Add(filters[i])
			}

			common := this.Intersect(other)
			Expect(common.Len()).To(BeZero())
		})

		It("returns empty filter set if other is empty", func() {
			for i := range filters {
				this.Add(filters[i])
			}

			common := this.Intersect(other)
			Expect(common.Len()).To(BeZero())
		})
	})

	Context("FilterSet.Difference()", func() {
		var this tc.FilterSet
		var other tc.FilterSet
		filters := []tctypes.Filter{
			tctypes.NewFlowerFilterBuilder().
				WithProtocol(tctypes.FilterProtocolIPv4).
				WithAction(
					tctypes.NewGenericActionBuiler().
						WithDrop().
						Build()).
				Build(),
			tctypes.NewFlowerFilterBuilder().
				WithProtocol(tctypes.FilterProtocolIPv4).
				WithAction(
					tctypes.NewGenericActionBuiler().
						WithPass().
						Build()).
				Build(),
			tctypes.NewFlowerFilterBuilder().
				WithProtocol(tctypes.FilterProtocolIPv4).
				WithMatchKeyDstIP(ipToIpNet("192.168.1.1")).
				WithAction(
					tctypes.NewGenericActionBuiler().
						WithPass().
						Build()).
				Build(),
			tctypes.NewFlowerFilterBuilder().
				WithProtocol(tctypes.FilterProtocolAll).
				WithAction(
					tctypes.NewGenericActionBuiler().
						WithDrop().
						Build()).
				Build(),
		}

		BeforeEach(func() {
			this = filterSet
			other = tc.NewFilterSetImpl()
		})

		It("returns set with filters that this filter has and not in other", func() {
			for i := range filters {
				this.Add(filters[i])
			}
			other.Add(filters[0])
			other.Add(filters[1])

			diff := this.Difference(other)

			expected := tc.NewFilterSetImpl()
			expected.Add(filters[2])
			expected.Add(filters[3])

			Expect(diff.Equals(expected)).To(BeTrue())
		})

		It("returns empty set if this and other sets are equal", func() {
			for i := range filters {
				this.Add(filters[i])
				other.Add(filters[i])
			}

			diff := this.Difference(other)
			Expect(diff.Len()).To(BeZero())
		})

		It("returns empty set if this in other", func() {
			for i := range filters {
				if i%2 == 0 {
					this.Add(filters[i])
				}
				other.Add(filters[i])
			}

			diff := this.Difference(other)
			Expect(diff.Len()).To(BeZero())
		})

		It("returns empty set if this is empty", func() {
			for i := range filters {
				other.Add(filters[i])
			}

			Expect(this.Difference(other).Len()).To(BeZero())
		})

		It("returns empty if both are empty", func() {
			Expect(this.Difference(other).Len()).To(BeZero())
		})

		It("returns this if other is empty", func() {
			for i := range filters {
				this.Add(filters[i])
			}

			diff := this.Difference(other)
			Expect(diff.Equals(this)).To(BeTrue())
		})
	})

	Context("FilterSet.Equals()", func() {
		var this tc.FilterSet
		var other tc.FilterSet
		filters := []tctypes.Filter{
			tctypes.NewFlowerFilterBuilder().
				WithProtocol(tctypes.FilterProtocolIPv4).
				WithAction(
					tctypes.NewGenericActionBuiler().
						WithDrop().
						Build()).
				Build(),
			tctypes.NewFlowerFilterBuilder().
				WithProtocol(tctypes.FilterProtocolIPv4).
				WithAction(
					tctypes.NewGenericActionBuiler().
						WithPass().
						Build()).
				Build(),
			tctypes.NewFlowerFilterBuilder().
				WithProtocol(tctypes.FilterProtocolIPv4).
				WithMatchKeyDstIP(ipToIpNet("192.168.1.1")).
				WithAction(
					tctypes.NewGenericActionBuiler().
						WithPass().
						Build()).
				Build(),
		}

		BeforeEach(func() {
			this = filterSet
			other = tc.NewFilterSetImpl()
		})

		It("returns true if both sets are empty", func() {
			Expect(this.Equals(other)).To(BeTrue())
		})

		It("returns true if both sets have the same filters", func() {
			for i := range filters {
				this.Add(filters[i])
				other.Add(filters[i])
			}

			Expect(this.Equals(other)).To(BeTrue())
		})

		It("returns false if sets have different number of filters", func() {
			for i := range filters {
				if i%2 == 0 {
					this.Add(filters[i])
				}
				other.Add(filters[i])
			}

			Expect(this.Equals(other)).To(BeFalse())
		})

		It("returns false if sets have different filters but same number of elements", func() {
			this.Add(filters[0])
			this.Add(filters[1])
			other.Add(filters[1])
			other.Add(filters[2])
		})
	})

	Context("FilterSet.List()", func() {
		It("returns zero if empty", func() {
			Expect(filterSet.Len()).To(BeZero())
		})

		It("returns the number of filters in set", func() {
			filterSet.Add(tctypes.NewFlowerFilterBuilder().WithProtocol(tctypes.FilterProtocolIPv4).Build())
			filterSet.Add(tctypes.NewFlowerFilterBuilder().WithProtocol(tctypes.FilterProtocolAll).Build())

			Expect(filterSet.Len()).To(Equal(2))
		})
	})
})
