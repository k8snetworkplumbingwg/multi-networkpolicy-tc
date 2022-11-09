package netlink_test

import (
	"errors"
	"net"
	"reflect"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netlink/nl"
	"golang.org/x/sys/unix"
	"k8s.io/klog/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/net/mocks"
	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc"
	netlinkdriver "github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc/driver/netlink"
	tctypes "github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc/types"
	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/utils"
)

type fakeLink struct {
	netlink.LinkAttrs
}

func (f *fakeLink) Attrs() *netlink.LinkAttrs {
	return &f.LinkAttrs
}

func (f *fakeLink) Type() string {
	return "fakeLink"
}

var _ = Describe("TC Netlink driver tests", func() {
	var fLink = &fakeLink{netlink.LinkAttrs{Name: "fakeNetDev", Index: 1}}
	var tcNetlink tc.TC
	var log = klog.NewKlogr().WithName("tc-driver-netlink-test")
	var netlinkProviderMock *mocks.NetlinkProvider
	var testError = errors.New("test error!")
	ipToIpNet := func(ip string) *net.IPNet { ipn, _ := utils.IPToIPNet(ip); return ipn }

	ingressQdisc := tctypes.NewIngressQDiscBuilder().Build()
	nlIngressQdisc := &netlink.Ingress{
		QdiscAttrs: netlink.QdiscAttrs{
			LinkIndex: fLink.Attrs().Index,
			Parent:    netlink.HANDLE_INGRESS,
		},
	}

	chain := tctypes.NewChainBuilder().WithChain(44).Build()
	nlChain := netlink.Chain{
		Chain:  44,
		Parent: netlink.HANDLE_INGRESS,
	}

	filter := tctypes.NewFlowerFilterBuilder().
		WithProtocol(tctypes.FilterProtocolIPv4).
		WithPriority(20).
		WithChain(10).
		WithHandle(0xff).
		WithAction(tctypes.NewGenericActionBuiler().WithPass().Build()).
		WithMatchKeyDstIP(ipToIpNet("192.168.10.0/24")).
		WithMatchKeyIPProto(tctypes.FlowerIPProtoTCP).
		WithMatchKeyDstPort(4000).
		Build()
	nlFilter := &netlink.Flower{
		FilterAttrs: netlink.FilterAttrs{
			LinkIndex: fLink.Attrs().Index,
			Handle:    *filter.Handle,
			Parent:    netlink.HANDLE_INGRESS,
			Chain:     filter.Chain,
			Priority:  *filter.Priority,
			Protocol:  unix.ETH_P_IP,
		},
		DestIP:     net.ParseIP("192.168.10.0").To4(),
		DestIPMask: net.IPMask{0xff, 0xff, 0xff, 0x00},
		EthType:    unix.ETH_P_IP,
		IPProto:    func() *nl.IPProto { p := nl.IPPROTO_TCP; return &p }(),
		DestPort:   *filter.Flower.DstPort,
		Actions: []netlink.Action{&netlink.GenericAction{
			ActionAttrs: netlink.ActionAttrs{
				Action: netlink.TC_ACT_OK,
			},
		}},
	}

	BeforeEach(func() {
		netlinkProviderMock = &mocks.NetlinkProvider{}
		tcNetlink = netlinkdriver.NewTcNetlinkImpl(fLink, log, netlinkProviderMock)
	})

	Context("Qdisc Add", func() {
		It("Fails when netlink call fails", func() {
			netlinkProviderMock.On("QdiscAdd", mock.Anything).Return(testError)
			err := tcNetlink.QDiscAdd(ingressQdisc)
			Expect(err).To(HaveOccurred())
		})

		It("succeeds when netlink call succeeds", func() {
			netlinkProviderMock.On("QdiscAdd", nlIngressQdisc).Return(nil)
			err := tcNetlink.QDiscAdd(ingressQdisc)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("Qdisc Del", func() {
		It("Fails when netlink call fails", func() {
			netlinkProviderMock.On("QdiscDel", mock.Anything).Return(testError)
			err := tcNetlink.QDiscDel(ingressQdisc)
			Expect(err).To(HaveOccurred())
		})

		It("succeeds when netlink call succeeds", func() {
			netlinkProviderMock.On("QdiscDel", nlIngressQdisc).Return(nil)
			err := tcNetlink.QDiscDel(ingressQdisc)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("Qdisc List", func() {
		It("Fails when netlink call fails", func() {
			netlinkProviderMock.On("QdiscList", mock.Anything).Return(nil, testError)
			_, err := tcNetlink.QDiscList()
			Expect(err).To(HaveOccurred())
		})

		It("succeeds when netlink call succeeds", func() {
			netlinkProviderMock.On("QdiscList", fLink).Return([]netlink.Qdisc{nlIngressQdisc}, nil)
			qds, err := tcNetlink.QDiscList()
			Expect(err).ToNot(HaveOccurred())
			Expect(qds).To(HaveLen(1))
			Expect(qds[0].Type()).To(Equal(tctypes.QDiscIngressType))
		})
	})

	Context("Chain Add", func() {
		It("Fails when netlink call fails", func() {
			netlinkProviderMock.On("ChainAdd", mock.Anything, mock.Anything).Return(testError)
			err := tcNetlink.ChainAdd(ingressQdisc, chain)
			Expect(err).To(HaveOccurred())
		})

		It("succeeds when netlink call succeeds", func() {
			netlinkProviderMock.On("ChainAdd", fLink, nlChain).Return(nil)
			err := tcNetlink.ChainAdd(ingressQdisc, chain)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("Chain Del", func() {
		It("Fails when netlink call fails", func() {
			netlinkProviderMock.On("ChainDel", mock.Anything, mock.Anything).Return(testError)
			err := tcNetlink.ChainDel(ingressQdisc, chain)
			Expect(err).To(HaveOccurred())
		})

		It("succeeds when netlink call succeeds", func() {
			netlinkProviderMock.On("ChainDel", fLink, nlChain).Return(nil)
			err := tcNetlink.ChainDel(ingressQdisc, chain)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("Chain List", func() {
		It("Fails when netlink call fails", func() {
			netlinkProviderMock.On("ChainList", mock.Anything, mock.Anything).
				Return(nil, testError)
			_, err := tcNetlink.ChainList(ingressQdisc)
			Expect(err).To(HaveOccurred())
		})

		It("succeeds when netlink call succeeds", func() {
			netlinkProviderMock.On("ChainList", fLink, uint32(netlink.HANDLE_INGRESS)).
				Return([]netlink.Chain{nlChain}, nil)
			qds, err := tcNetlink.ChainList(ingressQdisc)
			Expect(err).ToNot(HaveOccurred())
			Expect(qds).To(HaveLen(1))
			Expect(qds[0].Attrs().Chain).To(Equal(chain.Chain))
		})
	})

	Context("Filter Add", func() {
		It("Fails when netlink call fails", func() {
			netlinkProviderMock.On("FilterAdd", mock.Anything).Return(testError)
			err := tcNetlink.FilterAdd(ingressQdisc, filter)
			Expect(err).To(HaveOccurred())
		})

		It("succeeds when netlink call succeeds", func() {
			netlinkProviderMock.On("FilterAdd", mock.MatchedBy(func(f netlink.Filter) bool {
				flower, ok := f.(*netlink.Flower)
				if !ok {
					return false
				}
				return reflect.DeepEqual(flower, nlFilter)
			})).Return(nil)
			err := tcNetlink.FilterAdd(ingressQdisc, filter)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("Filter Del", func() {
		It("Fails when netlink call fails", func() {
			netlinkProviderMock.On("FilterDel", mock.Anything).Return(testError)
			err := tcNetlink.FilterDel(ingressQdisc, filter.Attrs())
			Expect(err).To(HaveOccurred())
		})

		It("succeeds when netlink call succeeds", func() {
			nlFilterForDel := &netlink.Flower{FilterAttrs: *nlFilter.Attrs()}
			netlinkProviderMock.On("FilterDel", nlFilterForDel).Return(nil)
			err := tcNetlink.FilterDel(ingressQdisc, filter.Attrs())
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("Filter List", func() {
		It("Fails when netlink call fails", func() {
			netlinkProviderMock.On("FilterList", mock.Anything, mock.Anything).
				Return(nil, testError)
			_, err := tcNetlink.FilterList(ingressQdisc)
			Expect(err).To(HaveOccurred())
		})

		It("succeeds when netlink call succeeds", func() {
			netlinkProviderMock.On("FilterList", fLink, uint32(netlink.HANDLE_INGRESS)).
				Return([]netlink.Filter{nlFilter}, nil)
			fl, err := tcNetlink.FilterList(ingressQdisc)
			Expect(err).ToNot(HaveOccurred())
			Expect(fl).To(HaveLen(1))
			Expect(fl[0].Equals(filter)).To(BeTrue())
		})
	})
})
