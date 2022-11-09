package net

import (
	"github.com/vishvananda/netlink"
)

// NetlinkProvider is a wrapper interface over vishvananda/netlink lib
type NetlinkProvider interface {
	// LinkByName returns Link by netdev name
	LinkByName(name string) (netlink.Link, error)

	// QdiscAdd adds qdisc
	QdiscAdd(qdisc netlink.Qdisc) error
	// QdiscDel deletes qdisc
	QdiscDel(qdisc netlink.Qdisc) error
	// QdiscList lists Qdiscs for link
	QdiscList(link netlink.Link) ([]netlink.Qdisc, error)

	// FilterAdd adds filter
	FilterAdd(filter netlink.Filter) error
	// FilterDel deletes filter
	FilterDel(filter netlink.Filter) error
	// FilterList lists Filters
	FilterList(link netlink.Link, parent uint32) ([]netlink.Filter, error)

	// ChainAdd adds chain
	ChainAdd(link netlink.Link, chain netlink.Chain) error
	// ChainDel deletes chain
	ChainDel(link netlink.Link, chain netlink.Chain) error
	// ChainList lists chains
	ChainList(link netlink.Link, parent uint32) ([]netlink.Chain, error)
}

// NewNetlinkProviderImpl creates a new NetlinkProviderImpl
func NewNetlinkProviderImpl() *NetlinkProviderImpl {
	return &NetlinkProviderImpl{}
}

type NetlinkProviderImpl struct{}

// LinkByName implements NetlinkProvider interface
func (n NetlinkProviderImpl) LinkByName(name string) (netlink.Link, error) {
	return netlink.LinkByName(name)
}

// QdiscAdd implements NetlinkProvider interface
func (n NetlinkProviderImpl) QdiscAdd(qdisc netlink.Qdisc) error {
	return netlink.QdiscAdd(qdisc)
}

// QdiscDel implements NetlinkProvider interface
func (n NetlinkProviderImpl) QdiscDel(qdisc netlink.Qdisc) error {
	return netlink.QdiscDel(qdisc)
}

// QdiscList implements NetlinkProvider interface
func (n NetlinkProviderImpl) QdiscList(link netlink.Link) ([]netlink.Qdisc, error) {
	return netlink.QdiscList(link)
}

// FilterAdd implements NetlinkProvider interface
func (n NetlinkProviderImpl) FilterAdd(filter netlink.Filter) error {
	return netlink.FilterAdd(filter)
}

// FilterDel implements NetlinkProvider interface
func (n NetlinkProviderImpl) FilterDel(filter netlink.Filter) error {
	return netlink.FilterDel(filter)
}

// FilterList implements NetlinkProvider interface
func (n NetlinkProviderImpl) FilterList(link netlink.Link, parent uint32) ([]netlink.Filter, error) {
	return netlink.FilterList(link, parent)
}

// ChainAdd implements NetlinkProvider interface
func (n NetlinkProviderImpl) ChainAdd(link netlink.Link, chain netlink.Chain) error {
	return netlink.ChainAdd(link, chain)
}

// ChainDel implements NetlinkProvider interface
func (n NetlinkProviderImpl) ChainDel(link netlink.Link, chain netlink.Chain) error {
	return netlink.ChainDel(link, chain)
}

// ChainList implements NetlinkProvider interface
func (n NetlinkProviderImpl) ChainList(link netlink.Link, parent uint32) ([]netlink.Chain, error) {
	return netlink.ChainList(link, parent)
}
