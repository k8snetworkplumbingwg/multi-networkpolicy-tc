package tc_test

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	klog "k8s.io/klog/v2"

	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc"
	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc/generator"
	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc/types"
	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/utils"
)

func getLastModifiedTime(path string) time.Time {
	fInfo, err := os.Lstat(path)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return fInfo.ModTime()
}

var _ = Describe("Actuator file writer tests", Ordered, func() {
	var tempDir string
	var logger klog.Logger
	var actuator tc.Actuator
	ingressQdisc := types.NewIngressQDiscBuilder().Build()

	BeforeAll(func() {
		// init logger
		fs := flag.NewFlagSet("test-flag-set", flag.PanicOnError)
		klog.InitFlags(fs)
		Expect(fs.Set("v", "8")).ToNot(HaveOccurred())
		logger = klog.NewKlogr().WithName("actuator-file-writer-test")
		DeferCleanup(klog.Flush)
		By("Logger initialized")

		// create temp dir
		tempDir = GinkgoT().TempDir()
		By(fmt.Sprintf("Generated temp dir for test: %s", tempDir))
	})

	Context("Actuator file writer with bad path", func() {
		It("fails to actuate on non existent path", func() {
			nonExistentPath := filepath.Join(tempDir, "does", "not", "exist")
			actuator = tc.NewActuatorFileWriterImpl(nonExistentPath, logger)
			objs := &generator.Objects{
				QDisc: ingressQdisc,
			}
			err := actuator.Actuate(objs)
			Expect(err).To(HaveOccurred())
		})

		It("fails to actuate on invalid path", func() {
			invalidPath := ""
			actuator = tc.NewActuatorFileWriterImpl(invalidPath, logger)
			objs := &generator.Objects{
				QDisc: ingressQdisc,
			}
			err := actuator.Actuate(objs)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Actuator file writer with valid path", func() {
		var tmpFilePath string
		objs := &generator.Objects{
			QDisc: ingressQdisc,
			Filters: []types.Filter{
				types.NewFlowerFilterBuilder().WithProtocol(types.FilterProtocolIPv4).WithPriority(100).Build(),
			},
		}
		expectedFileContent := `qdisc: ingress
filters:
protocol ip pref 100 flower
`

		BeforeEach(func() {
			tmpFilePath = filepath.Join(tempDir, "test-file")
			exist, err := utils.PathExists(tmpFilePath)
			Expect(err).ToNot(HaveOccurred())
			Expect(exist).To(BeFalse())
			actuator = tc.NewActuatorFileWriterImpl(tmpFilePath, logger)
		})

		AfterEach(func() {
			exist, err := utils.PathExists(tmpFilePath)
			Expect(err).ToNot(HaveOccurred())
			if exist {
				Expect(os.Remove(tmpFilePath)).ToNot(HaveOccurred())
			}
		})

		It("Writes objects to file when file does not exist", func() {
			err := actuator.Actuate(objs)
			Expect(err).ToNot(HaveOccurred())

			content, err := os.ReadFile(tmpFilePath)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(content)).To(BeEquivalentTo(expectedFileContent))
		})

		It("updates objects in file when file exist", func() {
			err := actuator.Actuate(objs)
			Expect(err).ToNot(HaveOccurred())

			objs.Filters = append(
				objs.Filters,
				types.NewFlowerFilterBuilder().WithProtocol(types.FilterProtocolAll).WithPriority(200).Build())

			err = actuator.Actuate(objs)
			Expect(err).ToNot(HaveOccurred())

			content, err := os.ReadFile(tmpFilePath)
			Expect(err).ToNot(HaveOccurred())
			expectedFileContent = `qdisc: ingress
filters:
protocol ip pref 100 flower
protocol all pref 200 flower
`
			Expect(string(content)).To(BeEquivalentTo(expectedFileContent))
		})

		It("does not update file if same objects provided", func() {
			err := actuator.Actuate(objs)
			Expect(err).ToNot(HaveOccurred())

			firstModified := getLastModifiedTime(tmpFilePath)

			err = actuator.Actuate(objs)
			Expect(err).ToNot(HaveOccurred())

			lastModified := getLastModifiedTime(tmpFilePath)

			Expect(firstModified.Equal(lastModified)).To(BeTrue())
		})
	})
})
