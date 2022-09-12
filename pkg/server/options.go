package server

import (
	"flag"

	"github.com/spf13/pflag"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"

	netwrappers "github.com/Mellanox/multi-networkpolicy-tc/pkg/net"
	"github.com/Mellanox/multi-networkpolicy-tc/pkg/policyrules"
	"github.com/Mellanox/multi-networkpolicy-tc/pkg/tc"
)

// Options stores option for the command
type Options struct {
	// kubeconfig is the path to a KubeConfig file.
	Kubeconfig string
	// KConfig points to a k8s API config, takes precedence over Kubeconfig
	KConfig *rest.Config
	// master is used to override the kubeconfig's URL to the apiserver
	master           string
	hostnameOverride string
	networkPlugins   []string
	podRulesPath     string

	// used for testing purposes, leave empty otherwise
	createActuatorForRep func(string) tc.Actuator
	policyRuleRenderer   policyrules.Renderer
	tcRuleGenerator      tc.Generator
	sriovnetProvider     netwrappers.SriovnetProvider
}

// AddFlags adds command line flags into command
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	klog.InitFlags(nil)
	fs.SortFlags = false
	fs.StringVar(&o.Kubeconfig, "kubeconfig", o.Kubeconfig,
		"Path to kubeconfig file with authorization information (master location is set by the master flag).")
	fs.StringVar(&o.master, "master", o.master,
		"The address of the Kubernetes API server (overrides any value in kubeconfig)")
	fs.StringVar(&o.hostnameOverride, "hostname-override", o.hostnameOverride,
		"If non-empty, will use this string as identification instead of the actual hostname.")
	fs.StringSliceVar(&o.networkPlugins, "network-plugins", []string{"accelerated-bridge"},
		"List of network plugins to be be considered for network policies.")
	fs.StringVar(&o.podRulesPath, "pod-rules-path", o.podRulesPath,
		"If non-empty, will use this path to store pod's rules for troubleshooting.")
	fs.AddGoFlagSet(flag.CommandLine)
}

// NewOptions initializes Options
func NewOptions() *Options {
	return &Options{}
}
