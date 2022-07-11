package tc

// Actuator is an interface that applies specified TC Objects on netdev
type Actuator interface {
	// Actuate applies TC object in TCObjects on NetDev provided in TCObjects
	Actuate(objects *TCObjects) error
}
