package tc

import (
	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc/generator"
)

// Actuator is an interface that applies specified TC Objects on netdev
type Actuator interface {
	// Actuate applies TC object in Objects on NetDev provided in Objects
	Actuate(objects *generator.Objects) error
}
