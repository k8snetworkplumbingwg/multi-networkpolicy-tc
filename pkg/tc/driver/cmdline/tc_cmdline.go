//nolint:prealloc
package cmdline

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/klog/v2"
	"k8s.io/utils/exec"

	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc/types"
	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/utils"
)

// NewTcCmdLineImpl creates a new instance of TcCmdLineImpl
func NewTcCmdLineImpl(dev string, log klog.Logger, executor exec.Interface) *TcCmdLineImpl {
	return &TcCmdLineImpl{
		netDev:   dev,
		log:      log,
		executor: executor,
		cmdline:  "tc",
		options:  []string{"-json"},
	}
}

// TcCmdLineImpl is a concrete implementation of TC interface utilizing TC command line
type TcCmdLineImpl struct {
	netDev   string
	log      klog.Logger
	executor exec.Interface

	cmdline string
	options []string
}

// execTcCmdNoOutput executes tc command with args, returning error if occurred
func (t *TcCmdLineImpl) execTcCmdNoOutput(args []string) error {
	finalArgs := append(t.options, args...)
	t.log.V(10).Info("executing", "cmd", "tc", "args", finalArgs)
	cmd := t.executor.Command("tc", finalArgs...)
	err := cmd.Run()
	t.log.V(10).Info("exec result", "err", err)
	return err
}

// execTcCmd executes tc command with args, returning stdout output and error
func (t *TcCmdLineImpl) execTcCmd(args []string) ([]byte, error) {
	finalArgs := append(t.options, args...)
	t.log.V(10).Info("executing", "cmd", "tc", "args", finalArgs)
	cmd := t.executor.Command("tc", finalArgs...)
	out, err := cmd.Output()
	t.log.V(10).Info("exec result", "err", err, "out", out)
	return out, err
}

// QDiscAdd implements TC interface
func (t *TcCmdLineImpl) QDiscAdd(qdisc types.QDisc) error {
	args := []string{"qdisc", "add", "dev", t.netDev}
	args = append(args, qdisc.GenCmdLineArgs()...)
	return t.execTcCmdNoOutput(args)
}

// QDiscDel implements TC interface
func (t *TcCmdLineImpl) QDiscDel(qdisc types.QDisc) error {
	args := []string{"qdisc", "del", "dev", t.netDev}
	args = append(args, qdisc.GenCmdLineArgs()...)
	return t.execTcCmdNoOutput(args)
}

// QDiscList implements TC interface
func (t *TcCmdLineImpl) QDiscList() ([]types.QDisc, error) {
	args := []string{"qdisc", "list", "dev", t.netDev}
	out, err := t.execTcCmd(args)
	if err != nil {
		return nil, err
	}
	// parse output and return objects
	var cQdiscs []cQDisc
	err = json.Unmarshal(out, &cQdiscs)
	if err != nil {
		return nil, err
	}

	var objs []types.QDisc
	for _, q := range cQdiscs {
		if q.Kind != string(types.QDiscIngressType) {
			// skip non-ingress qdiscs
			continue
		}
		handle, err := parseMajorMinor(q.Handle)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to parse qdisc Handle")
		}
		parent, err := parseMajorMinor(q.Parent)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to parse qdisc Parent")
		}
		qdisc := types.NewIngressQDiscBuilder().WithParent(parent).WithHandle(handle).Build()
		objs = append(objs, qdisc)
	}
	return objs, nil
}

// FilterAdd implements TC interface
func (t *TcCmdLineImpl) FilterAdd(qdisc types.QDisc, filter types.Filter) error {
	args := []string{"filter", "add", "dev", t.netDev}
	args = append(args, qdisc.GenCmdLineArgs()...)
	args = append(args, filter.GenCmdLineArgs()...)
	return t.execTcCmdNoOutput(args)
}

// FilterDel implements TC interface
func (t *TcCmdLineImpl) FilterDel(qdisc types.QDisc, filterAttr *types.FilterAttrs) error {
	args := []string{"filter", "del", "dev", t.netDev}
	args = append(args, qdisc.GenCmdLineArgs()...)
	args = append(args, filterAttr.GenCmdLineArgs()...)
	return t.execTcCmdNoOutput(args)
}

// FilterList implements TC interface
func (t *TcCmdLineImpl) FilterList(qdisc types.QDisc) ([]types.Filter, error) {
	args := []string{"filter", "list", "dev", t.netDev}
	args = append(args, qdisc.GenCmdLineArgs()...)
	out, err := t.execTcCmd(args)
	if err != nil {
		return nil, err
	}
	// parse output and return objects
	var cFilters []cFilter
	err = json.Unmarshal(out, &cFilters)
	if err != nil {
		return nil, err
	}

	var objs []types.Filter
	for _, f := range cFilters {
		// skip filters with no Options
		if f.Options == nil {
			continue
		}
		if f.Kind != string(types.FilterKindFlower) {
			return nil, fmt.Errorf("unexpected filter Kind: %s", f.Kind)
		}

		fb := types.NewFlowerFilterBuilder().
			WithChain(f.Chain).
			WithProtocol(sToFilterProtocol(f.Protocol)).
			WithPriority(f.Priority).
			WithHandle(f.Options.Handle)

		if f.Options.Keys.VlanEthType != nil {
			fb.WithMatchKeyVlanEthType(sToFlowerVlanEthType(*f.Options.Keys.VlanEthType))
		}
		if f.Options.Keys.IPProto != nil {
			fb.WithMatchKeyIPProto(sToFlowerIPProto(*f.Options.Keys.IPProto))
		}
		if f.Options.Keys.DstIP != nil {
			ipn, err := utils.IPToIPNet(*f.Options.Keys.DstIP)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to parse dest IP: %s", *f.Options.Keys.DstIP)
			}
			fb.WithMatchKeyDstIP(ipn)
		}
		if f.Options.Keys.DstPort != nil {
			fb.WithMatchKeyDstPort(*f.Options.Keys.DstPort)
		}

		for _, a := range f.Options.Actions {
			// TODO(adrianc): sort first by Order, ATM only one action is expected
			if a.Kind != string(types.ActionTypeGeneric) {
				return nil, fmt.Errorf("unexpected action: %s", a.Kind)
			}
			act := types.NewGenericAction(types.ActionGenericType(a.ControlAction.Type))
			fb.WithAction(act)
		}
		objs = append(objs, fb.Build())
	}
	return objs, nil
}

// ChainAdd implements TC interface
func (t *TcCmdLineImpl) ChainAdd(qdisc types.QDisc, chain types.Chain) error {
	args := []string{"chain", "add", "dev", t.netDev}
	args = append(args, qdisc.GenCmdLineArgs()...)
	args = append(args, chain.GenCmdLineArgs()...)
	return t.execTcCmdNoOutput(args)
}

// ChainDel implements TC interface
func (t *TcCmdLineImpl) ChainDel(qdisc types.QDisc, chain types.Chain) error {
	args := []string{"chain", "del", "dev", t.netDev}
	args = append(args, qdisc.GenCmdLineArgs()...)
	args = append(args, chain.GenCmdLineArgs()...)
	return t.execTcCmdNoOutput(args)
}

// ChainList implements TC interface
func (t *TcCmdLineImpl) ChainList(qdisc types.QDisc) ([]types.Chain, error) {
	args := []string{"chain", "list", "dev", t.netDev}
	args = append(args, qdisc.GenCmdLineArgs()...)
	out, err := t.execTcCmd(args)
	if err != nil {
		return nil, err
	}
	// parse output and return objects
	var cChains []cChain
	err = json.Unmarshal(out, &cChains)
	if err != nil {
		return nil, err
	}

	var objs []types.Chain
	for _, c := range cChains {
		parent, err := parseMajorMinor(c.Parent)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to parse Chain Parent")
		}
		objs = append(objs, types.NewChainBuilder().WithChain(c.Chain).WithParent(parent).Build())
	}
	return objs, nil
}

// parseMajorMinor parses TC string Handle and Parent. for a given format the following output is expected as depicted
// below.
//
//	"abcd" -> int32(0xabcd)
//	"abcdef01" -> int32(0xabcdef01)
//	"abcd:" -> int32(0xabcd)
//	"abcd:ef01" -> int32(0xabcdef01)
func parseMajorMinor(mm string) (uint32, error) {
	parsedMm := strings.Split(mm, ":")

	switch len(parsedMm) {
	case 1:
		p, err := strconv.ParseUint(parsedMm[0], 16, 32)
		return uint32(p), err
	case 2:
		major, err := strconv.ParseUint(parsedMm[0], 16, 32)
		if err != nil {
			return 0, err
		}
		if len(parsedMm[1]) > 0 {
			// we have minor
			minor, err := strconv.ParseUint(parsedMm[1], 16, 32)
			if err != nil {
				return 0, err
			}
			return ((uint32(major) & 0xffff) << 16) | (uint32(minor) & 0xffff), nil
		}
		return uint32(major), nil
	default:
		return 0, fmt.Errorf("failed to parse MajorMinor string: %s", mm)
	}
}
