package types

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("utils tests", func() {
	Describe("compare", func() {
		x := uint32(0x1)
		y := uint32(0x1)
		z := uint32(0x3)
		nVal := uint32(0x3)

		DescribeTable("check for expected output", func(first, second, nilval *uint32, expected bool) {
			out := compare(first, second, nilval)
			Expect(out).To(Equal(expected))
		},
			Entry("returns true if same pointer no nilVal", &x, &x, nil, true),
			Entry("returns true if same value no nilVal", &x, &y, nil, true),
			Entry("returns true if same pointer with nilVal", &x, &x, &nVal, true),
			Entry("returns true if same value with nilVal", &x, &y, &nVal, true),
			Entry("returns true if nilVal equals first val", &x, nil, &y, true),
			Entry("returns true if nilVal equals second val", nil, &y, &x, true),
			Entry("returns true if both are nil, no nilVal", nil, nil, nil, true),
			Entry("returns true if both are nil, with nilVal", nil, nil, &x, true),
			Entry("returns false if different values", &x, &z, nil, false),
			Entry("returns false if one is nil the other is not without nilVal", &x, nil, nil, false),
			Entry("returns false if one is nil the other is not without nilVal variation", nil, &x, nil, false),
			Entry("returns false if one is nil the other is not with nilVal", &x, nil, &z, false),
			Entry("returns false if one is nil the other is not without nilVal variation", nil, &x, &z, false),
		)
	})
})
