package types

const (
	QDiscIngressType QDiscType = "ingress"
)

// QDiscType is the type of qdisc
type QDiscType string

// QDiscAttr holds QDisc object attributes
type QDiscAttr struct {
	Handle *uint32
	Parent *uint32
}

// QDisc is an interface which represents a TC qdisc object
type QDisc interface {
	// Attrs returns QDiscAttr for a qdisc
	Attrs() *QDiscAttr
	// Type returns the QDisc type
	Type() QDiscType

	// Driver Specific related Interfaces
	CmdLineGenerator
}

// GenericQDisc is a generic qdisc of an arbitrary type
type GenericQDisc struct {
	QDiscAttr
	QdiscType QDiscType
}

// Attrs implements QDisc interface
func (g *GenericQDisc) Attrs() *QDiscAttr {
	return &g.QDiscAttr
}

// Type implements QDisc interface
func (g *GenericQDisc) Type() QDiscType {
	return g.QdiscType
}

// GenCmdLineArgs implements CmdLineGenerator interface
func (g *GenericQDisc) GenCmdLineArgs() []string {
	// for now we can just use qdisc type without attrs (parent, handle)
	return []string{string(g.QdiscType)}
}

// NewIngressQdisc creates a new Ingress QDisc object
func NewIngressQdisc() *GenericQDisc {
	return &GenericQDisc{
		QDiscAttr: QDiscAttr{
			Handle: nil,
			Parent: nil,
		},
		QdiscType: QDiscIngressType,
	}
}
