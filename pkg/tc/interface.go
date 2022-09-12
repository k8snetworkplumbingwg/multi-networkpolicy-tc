package tc

import (
	tctypes "github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc/types"
)

// TC defines an interface to interact with Linux Traffic Control subsystem
// an implementation should be associated with a specific network interface (netdev).
type TC interface {
	// QDiscAdd adds the specified Qdisc
	QDiscAdd(qdisc tctypes.QDisc) error
	// QDiscDel deletes the specified Qdisc
	QDiscDel(qdisc tctypes.QDisc) error
	// QDiscList lists QDiscs
	QDiscList() ([]tctypes.QDisc, error)

	// FilterAdd adds filter to qdisc
	FilterAdd(qdisc tctypes.QDisc, filter tctypes.Filter) error
	// FilterDel deletes filter identified by filterAttr from qdisc
	FilterDel(qdisc tctypes.QDisc, filterAttr *tctypes.FilterAttrs) error
	// FilterList lists Filters on qdisc
	FilterList(qdisc tctypes.QDisc) ([]tctypes.Filter, error)

	// ChainAdd adds chain to qdiscss
	ChainAdd(qdisc tctypes.QDisc, chain tctypes.Chain) error
	// ChainDel deletes chain from qdisc
	ChainDel(qdisc tctypes.QDisc, chain tctypes.Chain) error
	// ChainList lists chains on qdisc
	ChainList(qdisc tctypes.QDisc) ([]tctypes.Chain, error)
}
