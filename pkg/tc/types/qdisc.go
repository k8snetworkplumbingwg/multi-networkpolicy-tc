package types

const (
	QDiscIngressType QDiscType = "ingress"
)

// QDiscType is the type of qdisc
type QDiscType string

// QDiscAttrs holds QDisc object attributes
type QDiscAttrs struct {
	Parent *uint32
	Handle *uint32
}

// NewQDiscAttrs creates new QDiscAttrs instance
func NewQDiscAttrs(parent, handle *uint32) *QDiscAttrs {
	return &QDiscAttrs{
		Parent: parent,
		Handle: handle,
	}
}

// QDisc is an interface which represents a TC qdisc object
type QDisc interface {
	// Attrs returns QDiscAttrs for a qdisc
	Attrs() *QDiscAttrs
	// Type returns the QDisc type
	Type() QDiscType

	// Driver Specific related Interfaces
	CmdLineGenerator
}

// GenericQDisc is a generic qdisc of an arbitrary type
type GenericQDisc struct {
	QDiscAttrs
	QdiscType QDiscType
}

// Attrs implements QDisc interface
func (g *GenericQDisc) Attrs() *QDiscAttrs {
	return &g.QDiscAttrs
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

// NewGenericQdisc creates a new Generic QDisc object
func NewGenericQdisc(qDiscAttrs *QDiscAttrs, qType QDiscType) *GenericQDisc {
	return &GenericQDisc{
		QDiscAttrs: *qDiscAttrs,
		QdiscType:  qType,
	}
}

// Builders

// NewQDiscAttrsBuilder returns a new QDiscAttrsBuilder
func NewQDiscAttrsBuilder() *QDiscAttrsBuilder {
	return &QDiscAttrsBuilder{}
}

// QDiscAttrsBuilder is a QDiscAttrs builder
type QDiscAttrsBuilder struct {
	qDiscAttrs QDiscAttrs
}

// WithParent adds Parent to QDiscAttrsBuilder
func (qb *QDiscAttrsBuilder) WithParent(p uint32) *QDiscAttrsBuilder {
	qb.qDiscAttrs.Parent = &p
	return qb
}

// WithHandle adds Handle to QDiscAttrsBuilder
func (qb *QDiscAttrsBuilder) WithHandle(h uint32) *QDiscAttrsBuilder {
	qb.qDiscAttrs.Handle = &h
	return qb
}

// Build builds and returns a new QDiscAttrs instance
// Note: calling Build() multiple times will not return a completely
// new object on each call. that is, pointer/slice/map types will not be deep copied.
// to create several objects, different builders should be used.
func (qb *QDiscAttrsBuilder) Build() *QDiscAttrs {
	return NewQDiscAttrs(qb.qDiscAttrs.Parent, qb.qDiscAttrs.Handle)
}

// NewIngressQDiscBuilder returns a new NewIngressQDiscBuilder
func NewIngressQDiscBuilder() *IngressQDiscBuilder {
	return &IngressQDiscBuilder{qDiscAttrsBuilder: NewQDiscAttrsBuilder(), qDiscType: QDiscIngressType}
}

// IngressQDiscBuilder is an IngressQDisc builder
type IngressQDiscBuilder struct {
	qDiscAttrsBuilder *QDiscAttrsBuilder
	qDiscType         QDiscType
}

// WithParent adds Parent to IngressQDiscBuilder
func (iqb *IngressQDiscBuilder) WithParent(p uint32) *IngressQDiscBuilder {
	iqb.qDiscAttrsBuilder.WithParent(p)
	return iqb
}

// WithHandle adds Handle to IngressQDiscBuilder
func (iqb *IngressQDiscBuilder) WithHandle(h uint32) *IngressQDiscBuilder {
	iqb.qDiscAttrsBuilder.WithHandle(h)
	return iqb
}

// Build builds and returns a new GenericQDisc instance of type QDiscIngressType
// Note: calling Build() multiple times will not return a completely
// new object on each call. that is, pointer/slice/map types will not be deep copied.
// to create several objects, different builders should be used.
func (iqb *IngressQDiscBuilder) Build() *GenericQDisc {
	attrs := iqb.qDiscAttrsBuilder.Build()
	return NewGenericQdisc(attrs, iqb.qDiscType)
}
