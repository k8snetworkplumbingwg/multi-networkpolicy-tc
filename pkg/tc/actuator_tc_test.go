package tc_test

import (
	"flag"
	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc/generator"
	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/utils"
	"net"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/pkg/errors"
	klog "k8s.io/klog/v2"

	"github.com/stretchr/testify/mock"

	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc"
	tctypes "github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc/types"

	tcmocks "github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc/mocks"
)

func ingressQdiscMatch() func(q tctypes.QDisc) bool {
	return func(q tctypes.QDisc) bool {
		return q.Type() == tctypes.QDiscIngressType
	}
}

func filterMatch(filter tctypes.Filter) func(f tctypes.Filter) bool {
	return func(f tctypes.Filter) bool {
		return filter.Equals(f)
	}
}

func filterAttrMatch(filterAttr *tctypes.FilterAttrs) func(f *tctypes.FilterAttrs) bool {
	return func(f *tctypes.FilterAttrs) bool {
		return filterAttr.Equals(f)
	}
}

func chainMatch(chain uint32) func(c tctypes.Chain) bool {
	// Note(adrianc): ATM we are not needed to match on parent. it may change...
	return func(c tctypes.Chain) bool {
		return chain == *c.Attrs().Chain
	}
}

var _ = Describe("Actuator TC tests", func() {
	var actuator tc.Actuator
	var tcMock *tcmocks.TC
	var logger klog.Logger
	ingressQdisc := tctypes.NewIngressQDiscBuilder().Build()

	BeforeEach(func() {
		// init logger
		fs := flag.NewFlagSet("test-flag-set", flag.PanicOnError)
		klog.InitFlags(fs)
		Expect(fs.Set("v", "8")).ToNot(HaveOccurred())
		logger = klog.NewKlogr().WithName("actuator-tc-test")
		DeferCleanup(klog.Flush)
		By("Logger initialized")

		tcMock = tcmocks.NewTC(GinkgoT())
		actuator = tc.NewActuatorTCImpl(tcMock, logger)
	})

	Context("Actuate Qdisc Only", func() {
		var tcObj *generator.Objects

		BeforeEach(func() {
			tcObj = &generator.Objects{}
		})

		It("fails if listing qdisc fails", func() {
			tcObj.QDisc = ingressQdisc

			tcMock.On("QDiscList").Return(nil, errors.New("test error!"))

			err := actuator.Actuate(tcObj)
			Expect(err).To(HaveOccurred())
		})

		It("fails if delete qdisc fails", func() {
			tcMock.On("QDiscList").Return([]tctypes.QDisc{ingressQdisc}, nil)
			tcMock.On("QDiscDel", mock.Anything).Return(errors.New("test error!"))

			err := actuator.Actuate(tcObj)
			Expect(err).To(HaveOccurred())
		})

		When("Objects does not contain Qdisc", func() {
			It("deletes ingress Qdisc when exists", func() {
				tcMock.On("QDiscList").Return([]tctypes.QDisc{ingressQdisc}, nil)
				tcMock.On("QDiscDel", mock.MatchedBy(ingressQdiscMatch())).Return(nil)

				err := actuator.Actuate(tcObj)
				Expect(err).ToNot(HaveOccurred())
			})

			It("does nothing if ingress Qdisc does not exists", func() {
				tcMock.On("QDiscList").Return([]tctypes.QDisc{}, nil)

				err := actuator.Actuate(tcObj)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		When("Objects contain ingress Qdisc", func() {
			It("does nothing if ingress Qdisc exist without chain 0", func() {
				tcObj.QDisc = ingressQdisc

				tcMock.On("QDiscList").Return([]tctypes.QDisc{}, nil)
				tcMock.On("ChainList", mock.Anything).Return([]tctypes.Chain{
					tctypes.NewChainBuilder().WithParent(0xfffffff1).WithChain(1).Build()}, nil)

				err := actuator.Actuate(tcObj)
				Expect(err).ToNot(HaveOccurred())
			})

			It("deletes chain 0 on ingress qdisc when exists", func() {
				tcObj.QDisc = ingressQdisc

				tcMock.On("QDiscList").Return([]tctypes.QDisc{}, nil)
				tcMock.On("ChainList", mock.Anything).Return([]tctypes.Chain{
					tctypes.NewChainBuilder().WithParent(0xfffffff1).WithChain(0).Build()}, nil)
				tcMock.On("ChainDel",
					mock.MatchedBy(ingressQdiscMatch()),
					mock.MatchedBy(chainMatch(0))).
					Return(nil)

				err := actuator.Actuate(tcObj)
				Expect(err).ToNot(HaveOccurred())
			})

			It("does nothing if ingress Qdisc exists, chain 0 does not exist", func() {
				tcObj.QDisc = ingressQdisc

				tcMock.On("QDiscList").Return([]tctypes.QDisc{ingressQdisc}, nil)
				tcMock.On("ChainList", mock.Anything).Return([]tctypes.Chain{}, nil)

				err := actuator.Actuate(tcObj)
				Expect(err).ToNot(HaveOccurred())
			})

			It("fails if delete chain fails", func() {
				tcObj.QDisc = ingressQdisc

				tcMock.On("QDiscList").Return([]tctypes.QDisc{}, nil)
				tcMock.On("ChainList", mock.Anything).Return([]tctypes.Chain{
					tctypes.NewChainBuilder().WithParent(0xfffffff1).WithChain(0).Build()}, nil)
				tcMock.On("ChainDel", mock.Anything, mock.Anything).Return(errors.New("test error!"))

				err := actuator.Actuate(tcObj)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Context("Actuate with filters", func() {
		ipToIpNet := func(ip string) *net.IPNet { ipn, _ := utils.IPToIPNet(ip); return ipn }
		neededFilters := []tctypes.Filter{
			tctypes.NewFlowerFilterBuilder().
				WithProtocol(tctypes.FilterProtocolIPv4).
				WithPriority(100).
				WithMatchKeyDstIP(ipToIpNet("10.100.0.0/24")).
				WithAction(tctypes.NewGenericActionBuiler().WithDrop().Build()).
				Build(),
			tctypes.NewFlowerFilterBuilder().
				WithProtocol(tctypes.FilterProtocolIPv4).
				WithPriority(200).
				WithMatchKeyDstIP(ipToIpNet("10.100.0.0/16")).
				WithAction(tctypes.NewGenericActionBuiler().WithPass().Build()).
				Build(),
		}
		existingFilters := []tctypes.Filter{
			tctypes.NewFlowerFilterBuilder().
				WithProtocol(tctypes.FilterProtocolIPv4).
				WithPriority(100).
				WithMatchKeyDstIP(ipToIpNet("10.100.1.0/24")).
				WithAction(tctypes.NewGenericActionBuiler().WithDrop().Build()).
				Build(),
			tctypes.NewFlowerFilterBuilder().
				WithProtocol(tctypes.FilterProtocolIPv4).
				WithPriority(200).
				WithMatchKeyDstIP(ipToIpNet("10.100.0.0/16")).
				WithAction(tctypes.NewGenericActionBuiler().WithPass().Build()).
				Build(),
		}
		var tcObj *generator.Objects

		BeforeEach(func() {
			tcObj = &generator.Objects{
				QDisc:   ingressQdisc,
				Filters: neededFilters,
			}
		})

		When("no ingress qdisc in Objects", func() {
			It("fails", func() {
				tcObj.QDisc = nil

				err := actuator.Actuate(tcObj)
				Expect(err).To(HaveOccurred())
			})
		})

		When("filters provided in Objects, no filters set on ingress qdisc", func() {
			BeforeEach(func() {
				tcMock.On("QDiscList").Return([]tctypes.QDisc{ingressQdisc}, nil)
			})

			It("adds them to qdisc", func() {
				tcMock.On("FilterList", mock.MatchedBy(ingressQdiscMatch())).Return([]tctypes.Filter{}, nil)
				for i := range neededFilters {
					tcMock.On(
						"FilterAdd",
						mock.MatchedBy(ingressQdiscMatch()),
						mock.MatchedBy(filterMatch(neededFilters[i]))).
						Return(nil)
				}

				err := actuator.Actuate(tcObj)
				Expect(err).ToNot(HaveOccurred())
			})

			It("fails if listing filter on qdisc fails", func() {
				tcMock.On("FilterList", mock.Anything).
					Return(nil, errors.New("test error!"))

				err := actuator.Actuate(tcObj)
				Expect(err).To(HaveOccurred())
			})

			It("fails if adding filter to qdisc fails", func() {
				tcMock.On("FilterList", mock.MatchedBy(ingressQdiscMatch())).Return([]tctypes.Filter{}, nil)
				tcMock.On("FilterAdd", mock.Anything, mock.Anything).Return(errors.New("test error!"))

				err := actuator.Actuate(tcObj)
				Expect(err).To(HaveOccurred())
			})
		})

		When("filters provided in Objects, and filters set on ingress qdisc", func() {
			BeforeEach(func() {
				tcMock.On("QDiscList").Return([]tctypes.QDisc{ingressQdisc}, nil)
				tcMock.On("FilterList", mock.MatchedBy(ingressQdiscMatch())).Return(existingFilters, nil)
			})

			It("removes un-needed filters and adds needed filters", func() {
				tcMock.On(
					"FilterDel",
					mock.MatchedBy(ingressQdiscMatch()),
					mock.MatchedBy(filterAttrMatch(existingFilters[0].Attrs()))).
					Return(nil)
				tcMock.On(
					"FilterAdd",
					mock.MatchedBy(ingressQdiscMatch()),
					mock.MatchedBy(filterMatch(neededFilters[0]))).
					Return(nil)

				err := actuator.Actuate(tcObj)
				Expect(err).ToNot(HaveOccurred())
			})

			It("fails if removing filter from qdisc fails", func() {
				tcMock.On(
					"FilterDel",
					mock.MatchedBy(ingressQdiscMatch()),
					mock.MatchedBy(filterAttrMatch(existingFilters[0].Attrs()))).
					Return(errors.New("test error!"))
				err := actuator.Actuate(tcObj)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
