package types

const (
	// Action type
	ActionTypeGeneric = "gact"

	// Generic control actions
	ActionGenericPass = "pass"
	ActionGenericDrop = "drop"
)

// Action is an interface which represents a TC action
type Action interface {
	CmdLineGenerator
	// Type returns the action type
	Type() string
	// Spec returns Action Specification
	Spec() map[string]string
	// Equals compares this Action with other, returns true if they are equal or false otherwise
	Equals(other Action) bool
}

// NewGenericAction creates a new GenericAction
func NewGenericAction(controlAction string) *GenericAction {
	return &GenericAction{controlAction: controlAction}
}

// GenericAction is a struct representing TC generic action (gact)
type GenericAction struct {
	controlAction string
}

// Type implements Action interface, it returns the type of the action
func (a *GenericAction) Type() string {
	return ActionTypeGeneric
}

// Spec implements Action interface, it returns the specification of the action
func (a *GenericAction) Spec() map[string]string {
	m := make(map[string]string)
	m["control_action"] = a.controlAction
	return m
}

// Equals implements Action interface, it returns true if this and other Action are equal
func (a *GenericAction) Equals(other Action) bool {
	otherGenericAction, ok := other.(*GenericAction)
	if !ok {
		return false
	}
	if a.controlAction != otherGenericAction.controlAction {
		return false
	}
	return true
}

// GenCmdLineArgs implements CmdLineGenerator interface
func (a *GenericAction) GenCmdLineArgs() []string {
	return []string{"action", ActionTypeGeneric, a.controlAction}
}

// Builer

// NewGenericActionBuiler creates a new GenericActionBuilder
func NewGenericActionBuiler() *GenericActionBuilder {
	return &GenericActionBuilder{}
}

// GenericActionBuilder is a GenericAction builer
type GenericActionBuilder struct {
	genericAction GenericAction
}

// WithDrop adds ActionGenericDrop control action to GenericActionBuilder
func (gb *GenericActionBuilder) WithDrop() *GenericActionBuilder {
	gb.genericAction.controlAction = ActionGenericDrop
	return gb
}

// WithPass adds ActionGenericPass control action to GenericActionBuilder
func (gb *GenericActionBuilder) WithPass() *GenericActionBuilder {
	gb.genericAction.controlAction = ActionGenericPass
	return gb
}

// Build builds and returns a new GenericAction instance
func (gb *GenericActionBuilder) Build() *GenericAction {
	return NewGenericAction(gb.genericAction.controlAction)
}
