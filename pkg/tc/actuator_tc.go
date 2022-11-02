package tc

import (
	"github.com/pkg/errors"
	"k8s.io/klog/v2"

	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc/generator"
	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc/types"
)

// NewActuatorTCImpl creates a new ActuatorTCImpl
func NewActuatorTCImpl(tcIfc TC, log klog.Logger) *ActuatorTCImpl {
	return &ActuatorTCImpl{tcAPI: tcIfc, log: log}
}

// ActuatorTCImpl is an implementation of Actuator interface using provided TC interface to apply TC objects
type ActuatorTCImpl struct {
	tcAPI TC
	log   klog.Logger
}

// Actuate is an implementation of Actuator interface. it applies Objects on the representor
// Note: it assumes all filters are in Chain 0
func (a *ActuatorTCImpl) Actuate(objects *generator.Objects) error {
	if objects.QDisc == nil && len(objects.Filters) > 0 {
		return errors.New("Qdisc cannot be nil if Filters are provided")
	}

	// list qdiscs
	currentQDiscs, err := a.tcAPI.QDiscList()
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
			return a.tcAPI.QDiscDel(types.NewIngressQDiscBuilder().Build())
		}
		return nil
	}

	if len(objects.Filters) == 0 {
		// delete filters in chain 0 if exist
		chains, err := a.tcAPI.ChainList(types.NewIngressQDiscBuilder().Build())
		if err != nil {
			return err
		}

		for _, c := range chains {
			if *c.Attrs().Chain == 0 {
				return a.tcAPI.ChainDel(objects.QDisc, types.NewChainBuilder().WithChain(0).Build())
			}
		}
		return nil
	}

	// add ingress qdisc if needed
	if !ingressQDiscExist {
		if err = a.tcAPI.QDiscAdd(objects.QDisc); err != nil {
			return err
		}
	}

	// get existing filters
	existing, err := a.tcAPI.FilterList(objects.QDisc)
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

	// remove un-needed filters and add new ones
	toRemove := existingFilterSet.Difference(newFilterSet).List()
	toAdd := newFilterSet.Difference(existingFilterSet).List()

	for _, f := range toRemove {
		err := a.tcAPI.FilterDel(objects.QDisc, f.Attrs())
		if err != nil {
			return err
		}
	}

	for _, f := range toAdd {
		err := a.tcAPI.FilterAdd(objects.QDisc, f)
		if err != nil {
			return err
		}
	}

	return nil
}
