package tc

import (
	"github.com/pkg/errors"
	"k8s.io/klog/v2"

	"github.com/Mellanox/multi-networkpolicy-tc/pkg/tc/types"
)

// NewActuatorTCImpl creates a new ActuatorTCImpl
func NewActuatorTCImpl(tcIfc TC, log klog.Logger) *ActuatorTCImpl {
	return &ActuatorTCImpl{tcApi: tcIfc, log: log}
}

// ActuatorTCImpl is an implementation of Actuator interface using provided TC interface to apply TC objects
type ActuatorTCImpl struct {
	tcApi TC
	log   klog.Logger
}

// Actuate is an implementation of Actuator interface. it applies TCObjects on the representor
// Note: it assumes all filters are in Chain 0
func (a *ActuatorTCImpl) Actuate(objects *TCObjects) error {
	if objects.QDisc == nil && len(objects.Filters) > 0 {
		return errors.New("Qdisc cannot be nil if Filters are provided")
	}

	// list qdiscs
	currentQDiscs, err := a.tcApi.QDiscList()
	if err != nil {
		return errors.Wrap(err, "failed to list qdiscs")
	}

	var ingressQDiscExist bool
	for _, q := range currentQDiscs {
		if q.Type() == types.QDiscIngressType {
			ingressQDiscExist = true
			break
		}
	}

	if objects.QDisc == nil {
		// delete ingress qdisc if exist
		if ingressQDiscExist {
			return a.tcApi.QDiscDel(types.NewIngressQdisc())
		}
		return nil
	}

	if len(objects.Filters) == 0 {
		// delete filters in chain 0 if exist
		chains, err := a.tcApi.ChainList(types.NewIngressQdisc())
		if err != nil {
			return err
		}

		for _, c := range chains {
			if *c.Attrs().Chain == 0 {
				return a.tcApi.ChainDel(objects.QDisc, types.NewChainBuilder().WithChain(0).Build())
			}
		}
		return nil
	}

	// add ingress qdisc if needed
	if !ingressQDiscExist {
		if err = a.tcApi.QDiscAdd(objects.QDisc); err != nil {
			return err
		}
	}

	// get existing filters
	existing, err := a.tcApi.FilterList(objects.QDisc)
	if err != nil {
		return err
	}

	// create filter sets
	existingFilterSet := NewFilterSetImpl()
	newFilterSet := NewFilterSetImpl()

	for _, f := range existing {
		existingFilterSet.Add(f)
	}
	for _, f := range objects.Filters {
		newFilterSet.Add(f)
	}

	if existingFilterSet.Equals(newFilterSet) {
		// same filters nothing to do
		return nil
	}

	//remove un-needed filters and add new ones
	toRemove := existingFilterSet.Difference(newFilterSet).List()
	toAdd := newFilterSet.Difference(existingFilterSet).List()

	for _, f := range toRemove {
		err := a.tcApi.FilterDel(objects.QDisc, f.Attrs())
		if err != nil {
			return err
		}
	}

	for _, f := range toAdd {
		err := a.tcApi.FilterAdd(objects.QDisc, f)
		if err != nil {
			return err
		}
	}

	return nil
}
