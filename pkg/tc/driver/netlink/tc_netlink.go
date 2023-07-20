//nolint:prealloc
package netlink

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
	klog "k8s.io/klog/v2"

	multinet "github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/net"
	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc/types"
)

// NewTcNetlinkImpl creates a new instance of TcNetlinkImpl
func NewTcNetlinkImpl(linkDev netlink.Link, log klog.Logger, netlinkIfc multinet.NetlinkProvider) *TcNetlinkImpl {
	return &TcNetlinkImpl{
		link:       linkDev,
		netlinkIfc: netlinkIfc,
		log:        log,
	}
}

// TcNetlinkImpl is a concrete implementation of TC interface utilizing netlink lib
type TcNetlinkImpl struct {
	link       netlink.Link
	netlinkIfc multinet.NetlinkProvider
	log        klog.Logger
}

// QDiscAdd implements TC interface
func (t *TcNetlinkImpl) QDiscAdd(qdisc types.QDisc) error {
	t.log.V(10).Info("QDiscAdd()")

	if qdisc.Type() != types.QDiscIngressType {
		return fmt.Errorf("unsupported qdisc type: %s", qdisc.Type())
	}

	return t.netlinkIfc.QdiscAdd(qdiscToNlQdisc(qdisc, t.link.Attrs().Index))
}

// QDiscDel implements TC interface
func (t *TcNetlinkImpl) QDiscDel(qdisc types.QDisc) error {
	t.log.V(10).Info("QDiscDel()")

	if qdisc.Type() != types.QDiscIngressType {
		return fmt.Errorf("unsupported qdisc type: %s", qdisc.Type())
	}

	return t.netlinkIfc.QdiscDel(qdiscToNlQdisc(qdisc, t.link.Attrs().Index))
}

// QDiscList implements TC interface
func (t *TcNetlinkImpl) QDiscList() ([]types.QDisc, error) {
	t.log.V(10).Info("QDiscList()")

	nlQdiscs, err := t.netlinkIfc.QdiscList(t.link)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list qdiscs")
	}

	qdiscs := []types.QDisc{}
	for _, nlQdisc := range nlQdiscs {
		if nlQdisc.Type() != string(types.QDiscIngressType) {
			// skip non ingress qdiscs
			continue
		}

		qdiscs = append(qdiscs, nlQdiscToQdisc(nlQdisc))
	}
	return qdiscs, nil
}

// FilterAdd implements TC interface
func (t *TcNetlinkImpl) FilterAdd(qdisc types.QDisc, filter types.Filter) error {
	t.log.V(10).Info("FilterAdd()")

	if filter.Attrs().Kind != types.FilterKindFlower {
		return fmt.Errorf("unsupported filter kind")
	}

	if qdisc.Type() != types.QDiscIngressType {
		return fmt.Errorf("unsupported qdisc type")
	}

	flowerFilter, ok := filter.(*types.FlowerFilter)
	if !ok {
		return fmt.Errorf("unexpected filter")
	}

	nlFlower := flowerFilterToNlFlowerFilter(
		flowerFilter, netlink.HANDLE_INGRESS, t.link.Attrs().Index)

	return t.netlinkIfc.FilterAdd(nlFlower)
}

// FilterDel implements TC interface
func (t *TcNetlinkImpl) FilterDel(qdisc types.QDisc, filterAttr *types.FilterAttrs) error {
	t.log.V(10).Info("FilterDel()")

	if filterAttr.Kind != types.FilterKindFlower {
		return fmt.Errorf("unsupported filter kind")
	}

	if qdisc.Type() != types.QDiscIngressType {
		return fmt.Errorf("unsupported qdisc type")
	}

	flowerFilter := &types.FlowerFilter{FilterAttrs: *filterAttr}

	nlFlower := flowerFilterToNlFlowerFilter(flowerFilter, netlink.HANDLE_INGRESS, t.link.Attrs().Index)

	return t.netlinkIfc.FilterDel(nlFlower)
}

// FilterList implements TC interface
func (t *TcNetlinkImpl) FilterList(qdisc types.QDisc) ([]types.Filter, error) {
	t.log.V(10).Info("FilterList()")

	if qdisc.Type() != types.QDiscIngressType {
		return nil, fmt.Errorf("unsupported qdisc type")
	}

	nlFilters, err := t.netlinkIfc.FilterList(t.link, netlink.HANDLE_INGRESS)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list filters")
	}

	var filters []types.Filter
	for _, nlFilter := range nlFilters {
		if nlFilter.Type() != "flower" {
			continue
		}

		nlFlowerFilter, ok := nlFilter.(*netlink.Flower)
		if !ok {
			continue
		}

		filters = append(filters, nlFlowerFilterToFlowerFilter(nlFlowerFilter))
	}
	return filters, nil
}

// ChainAdd implements TC interface
func (t *TcNetlinkImpl) ChainAdd(qdisc types.QDisc, chain types.Chain) error {
	t.log.V(10).Info("ChainAdd()")

	if qdisc.Type() != types.QDiscIngressType {
		return fmt.Errorf("unsupported qdisc type")
	}

	return t.netlinkIfc.ChainAdd(t.link, chainToNlChain(chain, netlink.HANDLE_INGRESS))
}

// ChainDel implements TC interface
func (t *TcNetlinkImpl) ChainDel(qdisc types.QDisc, chain types.Chain) error {
	t.log.V(10).Info("ChainDel()")

	if qdisc.Type() != types.QDiscIngressType {
		return fmt.Errorf("unsupported qdisc type")
	}

	return t.netlinkIfc.ChainDel(t.link, chainToNlChain(chain, netlink.HANDLE_INGRESS))
}

// ChainList implements TC interface
func (t *TcNetlinkImpl) ChainList(qdisc types.QDisc) ([]types.Chain, error) {
	t.log.V(10).Info("ChainList()")

	if qdisc.Type() != types.QDiscIngressType {
		return nil, fmt.Errorf("unsupported qdisc type")
	}

	nlChains, err := t.netlinkIfc.ChainList(t.link, netlink.HANDLE_INGRESS)

	if err != nil {
		return nil, errors.Wrap(err, "failed to list chains")
	}

	var chains []types.Chain
	for idx := range nlChains {
		chains = append(chains, nlChainToChain(&nlChains[idx]))
	}

	return chains, nil
}
