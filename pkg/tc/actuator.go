package tc

// Actuator is an interface that applies specified TC Objects on netdev
type Actuator interface {
	// Actuate applies TC object in Objects on NetDev provided in Objects
	Actuate(objects *Objects) error
}
