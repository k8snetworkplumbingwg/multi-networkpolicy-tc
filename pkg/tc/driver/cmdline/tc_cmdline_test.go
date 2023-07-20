package cmdline_test

import (
	"net"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	klog "k8s.io/klog/v2"
	"k8s.io/utils/exec"

	testingexec "k8s.io/utils/exec/testing"

	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc"
	driver "github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc/driver/cmdline"
	tctypes "github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc/types"
	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/utils"
)

const (
	fakeNetDev = "fake"
)

// fakeExecHelper is a wrapper around testingexec.FakeExec which provides some
// utility functionality to aid in testing
type fakeExecHelper struct {
	testingexec.FakeExec
}

// AddFakeCmd adds a new testingexec.FakeCommandAction to fakeExecHelper.CommandScript
// that creates a new *testingexec.FakeCmd with the called arguments to Command()
func (feh *fakeExecHelper) AddFakeCmd() *testingexec.FakeCmd {
	fakeCmd := &testingexec.FakeCmd{}
	var actionForQdiscAdd testingexec.FakeCommandAction = func(cmd string, args ...string) exec.Cmd {
		return testingexec.InitFakeCmd(fakeCmd, cmd, args...)
	}
	feh.CommandScript = append(feh.CommandScript, actionForQdiscAdd)
	return fakeCmd
}

func newFakeAction(stdout, stderr []byte, err error) testingexec.FakeAction {
	return func() ([]byte, []byte, error) {
		return stdout, stderr, err
	}
}

var _ = Describe("TC Cmdline driver tests", func() {
	var fakeExec *fakeExecHelper
	var tcCmdLine tc.TC
	var log = klog.NewKlogr().WithName("tc-driver-cmdline-test")
	var testError = errors.New("test error!")
	ipToIpNet := func(ip string) *net.IPNet { ipn, _ := utils.IPToIPNet(ip); return ipn }

	BeforeEach(func() {
		fakeExec = &fakeExecHelper{testingexec.FakeExec{}}
		tcCmdLine = driver.NewTcCmdLineImpl(fakeNetDev, log, fakeExec)
	})

	Context("QDiscAdd", func() {
		var fakeCmd *testingexec.FakeCmd
		qdiscToAdd := tctypes.NewIngressQDiscBuilder().Build()
		expectedCmdArgs := []string{"tc", "-json", "qdisc", "add", "dev", fakeNetDev}
		expectedCmdArgs = append(expectedCmdArgs, qdiscToAdd.GenCmdLineArgs()...)

		BeforeEach(func() {
			fakeCmd = fakeExec.AddFakeCmd()
		})

		It("retuns no error when underlying command passes", func() {
			fakeCmd.RunScript = append(fakeCmd.RunScript, newFakeAction(nil, nil, nil))

			err := tcCmdLine.QDiscAdd(qdiscToAdd)

			Expect(err).ToNot(HaveOccurred())
			Expect(fakeCmd.Argv).To(BeEquivalentTo(expectedCmdArgs))
		})

		It("retuns error when underlying command errors", func() {
			fakeCmd.RunScript = append(fakeCmd.RunScript, newFakeAction(
				nil, nil, testError))

			err := tcCmdLine.QDiscAdd(qdiscToAdd)

			Expect(err).To(HaveOccurred())
		})
	})

	Context("QDiscDel", func() {
		var fakeCmd *testingexec.FakeCmd
		qdiscToDel := tctypes.NewIngressQDiscBuilder().Build()
		expectedCmdArgs := []string{"tc", "-json", "qdisc", "del", "dev", fakeNetDev}
		expectedCmdArgs = append(expectedCmdArgs, qdiscToDel.GenCmdLineArgs()...)

		BeforeEach(func() {
			fakeCmd = fakeExec.AddFakeCmd()
		})

		It("retuns no error when underlying command passes", func() {
			fakeCmd.RunScript = append(fakeCmd.RunScript, newFakeAction(nil, nil, nil))

			err := tcCmdLine.QDiscDel(qdiscToDel)

			Expect(err).ToNot(HaveOccurred())
			Expect(fakeCmd.Argv).To(BeEquivalentTo(expectedCmdArgs))
		})

		It("retuns error when underlying command errors", func() {
			fakeCmd.RunScript = append(fakeCmd.RunScript, newFakeAction(nil, nil, testError))

			err := tcCmdLine.QDiscDel(qdiscToDel)

			Expect(err).To(HaveOccurred())
		})
	})

	Context("QDiscList", func() {
		var fakeCmd *testingexec.FakeCmd
		expectedCmdArgs := []string{"tc", "-json", "qdisc", "list", "dev", fakeNetDev}
		qdiscListOut := `[
		{"kind":"mq","handle":"0:","root":true,"options":{}},
		{"kind":"fq_codel","handle":"0:","parent":":1","options":
			{"limit":10240,"flows":1024,"quantum":1514,"target":4999,"interval":99999,"memory_limit":33554432,"ecn":true,"drop_batch":64}
		},
		{"kind":"ingress","handle":"ffff:","parent":"ffff:fff1","options":{}}
	]`

		BeforeEach(func() {
			fakeCmd = fakeExec.AddFakeCmd()
		})

		It("retuns ingress qdisc only without error when underlying command passes", func() {
			fakeCmd.OutputScript = append(fakeCmd.OutputScript, newFakeAction([]byte(qdiscListOut), nil, nil))
			expectedIngressQdisc := tctypes.NewIngressQDiscBuilder().
				WithParent(0xfffffff1).
				WithHandle(0xffff).
				Build()

			qdiscs, err := tcCmdLine.QDiscList()

			Expect(err).ToNot(HaveOccurred())
			Expect(fakeCmd.Argv).To(BeEquivalentTo(expectedCmdArgs))
			Expect(qdiscs).To(HaveLen(1))
			Expect(qdiscs[0]).To(BeEquivalentTo(expectedIngressQdisc))
		})

		It("retuns error when underlying command errors", func() {
			fakeCmd.OutputScript = append(fakeCmd.OutputScript, newFakeAction(
				nil, nil, testError))

			qdiscs, err := tcCmdLine.QDiscList()

			Expect(err).To(HaveOccurred())
			Expect(qdiscs).To(BeNil())
		})
	})

	Context("ChainAdd", func() {
		var fakeCmd *testingexec.FakeCmd
		ingressQdisc := tctypes.NewIngressQDiscBuilder().Build()
		chainToAdd := tctypes.NewChainBuilder().WithChain(99).Build()
		expectedCmdArgs := []string{"tc", "-json", "chain", "add", "dev", fakeNetDev}
		expectedCmdArgs = append(expectedCmdArgs, ingressQdisc.GenCmdLineArgs()...)
		expectedCmdArgs = append(expectedCmdArgs, chainToAdd.GenCmdLineArgs()...)

		BeforeEach(func() {
			fakeCmd = fakeExec.AddFakeCmd()
		})

		It("retuns no error when underlying command passes", func() {
			fakeCmd.RunScript = append(fakeCmd.RunScript, newFakeAction(nil, nil, nil))

			err := tcCmdLine.ChainAdd(ingressQdisc, chainToAdd)

			Expect(err).ToNot(HaveOccurred())
			Expect(fakeCmd.Argv).To(BeEquivalentTo(expectedCmdArgs))
		})

		It("retuns error when underlying command errors", func() {
			fakeCmd.RunScript = append(fakeCmd.RunScript, newFakeAction(
				nil, nil, testError))

			err := tcCmdLine.ChainAdd(ingressQdisc, chainToAdd)

			Expect(err).To(HaveOccurred())
		})
	})

	Context("ChainDel", func() {
		var fakeCmd *testingexec.FakeCmd
		ingressQdisc := tctypes.NewIngressQDiscBuilder().Build()
		chainToDel := tctypes.NewChainBuilder().WithChain(99).Build()
		expectedCmdArgs := []string{"tc", "-json", "chain", "del", "dev", fakeNetDev}
		expectedCmdArgs = append(expectedCmdArgs, ingressQdisc.GenCmdLineArgs()...)
		expectedCmdArgs = append(expectedCmdArgs, chainToDel.GenCmdLineArgs()...)

		BeforeEach(func() {
			fakeCmd = fakeExec.AddFakeCmd()
		})

		It("retuns no error when underlying command passes", func() {
			fakeCmd.RunScript = append(fakeCmd.RunScript, newFakeAction(nil, nil, nil))

			err := tcCmdLine.ChainDel(ingressQdisc, chainToDel)

			Expect(err).ToNot(HaveOccurred())
			Expect(fakeCmd.Argv).To(BeEquivalentTo(expectedCmdArgs))
		})

		It("retuns error when underlying command errors", func() {
			fakeCmd.RunScript = append(fakeCmd.RunScript, newFakeAction(nil, nil, testError))

			err := tcCmdLine.ChainDel(ingressQdisc, chainToDel)

			Expect(err).To(HaveOccurred())
		})
	})

	Context("ChainList", func() {
		var fakeCmd *testingexec.FakeCmd
		ingressQdisc := tctypes.NewIngressQDiscBuilder().Build()
		expectedCmdArgs := []string{"tc", "-json", "chain", "list", "dev", fakeNetDev}
		expectedCmdArgs = append(expectedCmdArgs, ingressQdisc.GenCmdLineArgs()...)
		chainListOut := `[{"parent": "ffff:", "chain": 99}]`

		BeforeEach(func() {
			fakeCmd = fakeExec.AddFakeCmd()
		})

		It("returns chain without error when underlying command passes", func() {
			fakeCmd.OutputScript = append(fakeCmd.OutputScript, newFakeAction([]byte(chainListOut), nil, nil))
			expectedChain := tctypes.NewChainBuilder().
				WithParent(0xffff).
				WithChain(99).
				Build()

			chains, err := tcCmdLine.ChainList(ingressQdisc)

			Expect(err).ToNot(HaveOccurred())
			Expect(fakeCmd.Argv).To(BeEquivalentTo(expectedCmdArgs))
			Expect(chains).To(HaveLen(1))
			Expect(chains[0]).To(BeEquivalentTo(expectedChain))
		})

		It("retuns error when underlying command errors", func() {
			fakeCmd.OutputScript = append(fakeCmd.OutputScript, newFakeAction(
				nil, nil, testError))

			chains, err := tcCmdLine.ChainList(ingressQdisc)

			Expect(err).To(HaveOccurred())
			Expect(chains).To(BeNil())
		})
	})

	Context("FilterAdd", func() {
		var fakeCmd *testingexec.FakeCmd
		ingressQdisc := tctypes.NewIngressQDiscBuilder().Build()
		filterToAdd := tctypes.NewFlowerFilterBuilder().
			WithProtocol(tctypes.FilterProtocolIPv4).
			WithAction(tctypes.NewGenericActionBuiler().WithPass().Build()).
			WithMatchKeyDstIP(ipToIpNet("10.10.10.2/24")).
			Build()
		expectedCmdArgs := []string{"tc", "-json", "filter", "add", "dev", fakeNetDev}
		expectedCmdArgs = append(expectedCmdArgs, ingressQdisc.GenCmdLineArgs()...)
		expectedCmdArgs = append(expectedCmdArgs, filterToAdd.GenCmdLineArgs()...)

		BeforeEach(func() {
			fakeCmd = fakeExec.AddFakeCmd()
		})

		It("retuns no error when underlying command passes", func() {
			fakeCmd.RunScript = append(fakeCmd.RunScript, newFakeAction(nil, nil, nil))

			err := tcCmdLine.FilterAdd(ingressQdisc, filterToAdd)

			Expect(err).ToNot(HaveOccurred())
			Expect(fakeCmd.Argv).To(BeEquivalentTo(expectedCmdArgs))
		})

		It("retuns error when underlying command errors", func() {
			fakeCmd.RunScript = append(fakeCmd.RunScript, newFakeAction(
				nil, nil, testError))

			err := tcCmdLine.FilterAdd(ingressQdisc, filterToAdd)

			Expect(err).To(HaveOccurred())
		})
	})

	Context("FilterDel", func() {
		var fakeCmd *testingexec.FakeCmd
		ingressQdisc := tctypes.NewIngressQDiscBuilder().Build()
		filterToDel := tctypes.NewFilterAttrsBuilder().
			WithProtocol(tctypes.FilterProtocolIPv4).
			WithPriority(200).
			WithHandle(0x1).
			WithChain(0).
			WithKind(tctypes.FilterKindFlower).
			Build()
		expectedCmdArgs := []string{"tc", "-json", "filter", "del", "dev", fakeNetDev}
		expectedCmdArgs = append(expectedCmdArgs, ingressQdisc.GenCmdLineArgs()...)
		expectedCmdArgs = append(expectedCmdArgs, filterToDel.GenCmdLineArgs()...)

		BeforeEach(func() {
			fakeCmd = fakeExec.AddFakeCmd()
		})

		It("retuns no error when underlying command passes", func() {
			fakeCmd.RunScript = append(fakeCmd.RunScript, newFakeAction(nil, nil, nil))

			err := tcCmdLine.FilterDel(ingressQdisc, filterToDel)

			Expect(err).ToNot(HaveOccurred())
			Expect(fakeCmd.Argv).To(BeEquivalentTo(expectedCmdArgs))
		})

		It("retuns error when underlying command errors", func() {
			fakeCmd.RunScript = append(fakeCmd.RunScript, newFakeAction(nil, nil, testError))

			err := tcCmdLine.FilterDel(ingressQdisc, filterToDel)

			Expect(err).To(HaveOccurred())
		})
	})

	Context("FilterList", func() {
		var fakeCmd *testingexec.FakeCmd
		ingressQdisc := tctypes.NewIngressQDiscBuilder().Build()
		expectedCmdArgs := []string{"tc", "-json", "filter", "list", "dev", fakeNetDev}
		expectedCmdArgs = append(expectedCmdArgs, ingressQdisc.GenCmdLineArgs()...)
		filterListOut := `[
  {
    "protocol": "ip",
    "pref": 200,
    "kind": "flower",
    "chain": 0
  },
  {
    "protocol": "ip",
    "pref": 200,
    "kind": "flower",
    "chain": 0,
    "options": {
      "handle": 1,
      "keys": {
        "eth_type": "ipv4",
		"ip_proto": "tcp",
		"dst_port": 6666,
        "dst_ip": "10.10.10.2/24"
      },
      "in_hw": true,
      "in_hw_count": 1,
      "actions": [
        {
          "order": 1,
          "kind": "gact",
          "control_action": {
            "type": "pass"
          },
          "prob": {
            "random_type": "none",
            "control_action": {
              "type": "pass"
            },
            "val": 0
          },
          "index": 2,
          "ref": 1,
          "bind": 1,
          "used_hw_stats": [
            "delayed"
          ]
        }
      ]
    }
  }
]`

		BeforeEach(func() {
			fakeCmd = fakeExec.AddFakeCmd()
		})

		It("retuns non empty flower filter without error when underlying command passes", func() {
			fakeCmd.OutputScript = append(fakeCmd.OutputScript, newFakeAction([]byte(filterListOut), nil, nil))
			expectedFilter := tctypes.NewFlowerFilterBuilder().
				WithProtocol(tctypes.FilterProtocolIPv4).
				WithPriority(200).
				WithHandle(1).
				WithChain(0).
				WithMatchKeyDstIP(ipToIpNet("10.10.10.2/24")).
				WithMatchKeyDstPort(6666).
				WithMatchKeyIPProto(tctypes.FlowerIPProtoTCP).
				WithAction(tctypes.NewGenericActionBuiler().WithPass().Build()).
				Build()

			filters, err := tcCmdLine.FilterList(ingressQdisc)

			Expect(err).ToNot(HaveOccurred())
			Expect(fakeCmd.Argv).To(BeEquivalentTo(expectedCmdArgs))
			Expect(filters).To(HaveLen(1))
			Expect(filters[0].Equals(expectedFilter)).To(BeTrue())
		})

		It("retuns error when underlying command errors", func() {
			fakeCmd.OutputScript = append(fakeCmd.OutputScript, newFakeAction(
				nil, nil, testError))

			filters, err := tcCmdLine.FilterList(ingressQdisc)

			Expect(err).To(HaveOccurred())
			Expect(filters).To(BeNil())
		})
	})

	Context("filterList with 802.1Q filter", func() {
		var fakeCmd *testingexec.FakeCmd
		ingressQdisc := tctypes.NewIngressQDiscBuilder().Build()
		expectedCmdArgs := []string{"tc", "-json", "filter", "list", "dev", fakeNetDev}
		expectedCmdArgs = append(expectedCmdArgs, ingressQdisc.GenCmdLineArgs()...)
		filterListOut := `[
  {
    "protocol": "802.1Q",
    "pref": 200,
    "kind": "flower",
    "chain": 0,
    "options": {
      "handle": 1,
      "keys": {
		"vlan_ethtype": "ip",
        "eth_type": "ipv4"
      },
      "in_hw": true,
      "in_hw_count": 1
    }
  }
]`

		BeforeEach(func() {
			fakeCmd = fakeExec.AddFakeCmd()
		})

		It("returns expected filter", func() {
			fakeCmd.OutputScript = append(fakeCmd.OutputScript, newFakeAction([]byte(filterListOut), nil, nil))
			expectedFilter := tctypes.NewFlowerFilterBuilder().
				WithProtocol(tctypes.FilterProtocol8021Q).
				WithMatchKeyVlanEthType(tctypes.FlowerVlanEthTypeIPv4).
				WithPriority(200).
				WithHandle(1).
				WithChain(0).
				Build()

			filters, err := tcCmdLine.FilterList(ingressQdisc)

			Expect(err).ToNot(HaveOccurred())
			Expect(fakeCmd.Argv).To(BeEquivalentTo(expectedCmdArgs))
			Expect(filters).To(HaveLen(1))
			Expect(filters[0].Equals(expectedFilter)).To(BeTrue())
		})
	})
})
