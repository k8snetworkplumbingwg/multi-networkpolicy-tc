package types

import (
	"fmt"
	"strconv"
)

const (
	// ChainDefaultParent is the default parent of a chain which is the ingress qdisc
	ChainDefaultParent uint32 = 0xfffffff1
	// ChainDefaultChain is the default chain number
	ChainDefaultChain uint16 = 0
)

// Chain is an interface which represents a TC chain
type Chain interface {
	// Attrs returns chain attributes
	Attrs() *ChainAttrs

	// Driver Specific related Interfaces
	CmdLineGenerator
}

// ChainAttrs are the attributes of a Chain
type ChainAttrs struct {
	Parent *uint32
	Chain  *uint16
}

// ChainImpl is a concrete implementation of Chain
type ChainImpl struct {
	ChainAttrs
}

// Attrs implements Chain interface
func (c *ChainImpl) Attrs() *ChainAttrs {
	return &c.ChainAttrs
}

// GenCmdLineArgs implements CmdLineGenerator interface
func (c *ChainImpl) GenCmdLineArgs() []string {
	args := []string{}

	if c.Parent != nil {
		parent := fmt.Sprintf("%x:%x", uint16(*c.Parent>>16), uint16(*c.Parent))
		args = append(args, "parent", parent)
	}

	if c.Chain != nil {
		args = append(args, "chain", strconv.FormatUint(uint64(*c.Chain), 10))
	}
	return args
}

func NewChainImpl(parent *uint32, chain *uint16) *ChainImpl {
	return &ChainImpl{ChainAttrs{
		Parent: parent,
		Chain:  chain,
	}}
}

// builder

// NewChainBuilder creates a new ChainBuilder
func NewChainBuilder() *ChainBuilder {
	return &ChainBuilder{}
}

// ChainBuilder is a Chain builder
type ChainBuilder struct {
	chain ChainImpl
}

// WithParent adds Chain Parent to ChainBuilder
func (cb *ChainBuilder) WithParent(parent uint32) *ChainBuilder {
	cb.chain.Parent = &parent
	return cb
}

// WithChain adds Chain chain number to ChainBuilder
func (cb *ChainBuilder) WithChain(chain uint16) *ChainBuilder {
	cb.chain.Chain = &chain
	return cb
}

// Build builds and returns a new Chain instance
// Note: calling Build() multiple times will not return a completely
// new object on each call. that is, pointer/slice/map types will not be deep copied.
// to create several objects, different builders should be used.
func (cb *ChainBuilder) Build() *ChainImpl {
	if cb.chain.Chain == nil {
		defChain := ChainDefaultChain
		cb.chain.Chain = &defChain

	}
	return NewChainImpl(cb.chain.Parent, cb.chain.Chain)
}
