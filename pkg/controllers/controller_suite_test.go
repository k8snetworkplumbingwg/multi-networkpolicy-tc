package controllers

import (
	"flag"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/klog/v2"
)

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "controllers")
}

var _ = BeforeSuite(func() {
	fs := flag.NewFlagSet("test-flag-set", flag.PanicOnError)
	klog.InitFlags(fs)
	Expect(fs.Set("v", "8")).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	klog.Flush()
})
