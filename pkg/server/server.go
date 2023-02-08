package server

//nolint:lll
import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	multiv1beta1 "github.com/k8snetworkplumbingwg/multi-networkpolicy/pkg/apis/k8s.cni.cncf.io/v1beta1"
	multiclient "github.com/k8snetworkplumbingwg/multi-networkpolicy/pkg/client/clientset/versioned"
	multiinformer "github.com/k8snetworkplumbingwg/multi-networkpolicy/pkg/client/informers/externalversions"
	multilisterv1beta1 "github.com/k8snetworkplumbingwg/multi-networkpolicy/pkg/client/listers/k8s.cni.cncf.io/v1beta1"
	netdefv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	netdefclient "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/client/clientset/versioned"
	netdefinformerv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/client/informers/externalversions"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/util/async"
	"k8s.io/utils/exec"

	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/controllers"
	netwrappers "github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/net"
	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/policyrules"
	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc"
	cmdlinedriver "github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc/driver/cmdline"
	netlinkdriver "github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc/driver/netlink"
	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc/generator"
	multiutils "github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/utils"
)

const (
	// PolicyNetworkAnnotation is annotation for multiNetworkPolicy,
	// to specify which networks(i.e. net-attach-def) are the targets
	// of the policy
	PolicyNetworkAnnotation = "k8s.v1.cni.cncf.io/policy-for"
)

// Server structure defines data for server
type Server struct {
	// object change trackers
	podChanges    *controllers.PodChangeTracker
	policyChanges *controllers.PolicyChangeTracker
	netdefChanges *controllers.NetDefChangeTracker
	nsChanges     *controllers.NamespaceChangeTracker
	// maps to store the state of the cluster for various object
	podMap       controllers.PodMap
	policyMap    controllers.PolicyMap
	namespaceMap controllers.NamespaceMap
	// clients to access k8s API
	Client              clientset.Interface
	NetworkPolicyClient multiclient.Interface
	NetDefClient        netdefclient.Interface
	// listers
	podLister    corelisters.PodLister
	policyLister multilisterv1beta1.MultiNetworkPolicyLister
	// other fields
	Hostname         string
	Broadcaster      record.EventBroadcaster
	Recorder         record.EventRecorder
	Options          *Options
	ConfigSyncPeriod time.Duration
	NodeRef          *v1.ObjectReference

	mu           sync.Mutex // protects the following fields
	podSynced    bool
	policySynced bool
	netdefSynced bool
	nsSynced     bool
	// initialized is used to determine if pod & policy & netdef & ns has synced in a lockless manner
	// by using atomic operations to read/write its value.
	initialized int32
	// Channel used to signal podConfig to start running by closing the channel
	startPodConfig       chan struct{}
	startPodConfigClosed bool

	syncRunner *async.BoundedFrequencyRunner

	policyRuleRenderer      policyrules.Renderer
	tcRuleGenerator         generator.Generator
	sriovnetProvider        netwrappers.SriovnetProvider
	netlinkProvider         netwrappers.NetlinkProvider
	createActuatorFromRepFn func(string) (tc.Actuator, error)
}

func (s *Server) RunPodConfig(ctx context.Context) {
	klog.Infof("Starting pod config")
	informerFactory := informers.NewSharedInformerFactoryWithOptions(s.Client, s.ConfigSyncPeriod)
	s.podLister = informerFactory.Core().V1().Pods().Lister()

	podConfig := controllers.NewPodConfig(informerFactory.Core().V1().Pods(), s.ConfigSyncPeriod)
	podConfig.RegisterEventHandler(s)
	go podConfig.Run(ctx.Done())
	informerFactory.Start(ctx.Done())
}

// Run starts Server, runs until provided context is done
func (s *Server) Run(ctx context.Context) {
	if s.Broadcaster != nil {
		s.Broadcaster.StartRecordingToSink(
			&v1core.EventSinkImpl{Interface: s.Client.CoreV1().Events("")})
	}
	go func() {
		<-ctx.Done()
		s.Broadcaster.Shutdown()
	}()

	informerFactory := informers.NewSharedInformerFactoryWithOptions(s.Client, s.ConfigSyncPeriod)
	nsConfig := controllers.NewNamespaceConfig(informerFactory.Core().V1().Namespaces(), s.ConfigSyncPeriod)
	nsConfig.RegisterEventHandler(s)
	go nsConfig.Run(ctx.Done())
	informerFactory.Start(ctx.Done())

	go func() {
		select {
		case <-s.startPodConfig:
			s.RunPodConfig(ctx)
		case <-ctx.Done():
		}
	}()

	policyInformerFactory := multiinformer.NewSharedInformerFactoryWithOptions(
		s.NetworkPolicyClient, s.ConfigSyncPeriod)
	s.policyLister = policyInformerFactory.K8sCniCncfIo().V1beta1().MultiNetworkPolicies().Lister()

	policyConfig := controllers.NewNetworkPolicyConfig(
		policyInformerFactory.K8sCniCncfIo().V1beta1().MultiNetworkPolicies(), s.ConfigSyncPeriod)
	policyConfig.RegisterEventHandler(s)
	go policyConfig.Run(ctx.Done())
	policyInformerFactory.Start(ctx.Done())

	netdefInformarFactory := netdefinformerv1.NewSharedInformerFactoryWithOptions(
		s.NetDefClient, s.ConfigSyncPeriod)
	netdefConfig := controllers.NewNetDefConfig(
		netdefInformarFactory.K8sCniCncfIo().V1().NetworkAttachmentDefinitions(), s.ConfigSyncPeriod)
	netdefConfig.RegisterEventHandler(s)
	go netdefConfig.Run(ctx.Done())
	netdefInformarFactory.Start(ctx.Done())

	// start sync loop
	go s.SyncLoop(ctx)

	s.birthCry()

	// wait on Context
	<-ctx.Done()
}

// setInitialized sets s.initialized to 1 if value is true. the set operation is atomic
func (s *Server) setInitialized(value bool) {
	var initialized int32
	if value {
		initialized = 1
	}
	atomic.StoreInt32(&s.initialized, initialized)
}

// isInitialized checks if all relevant k8s resources have been synced to their latest state
func (s *Server) isInitialized() bool {
	return atomic.LoadInt32(&s.initialized) > 0
}

// birthCry send start event to node object where this server is running
func (s *Server) birthCry() {
	klog.Infof("Started multi-networkpolicy-tc")
	s.Recorder.Eventf(s.NodeRef, v1.EventTypeNormal, "Started", "Started multi-networkpolicy-tc.")
}

// SyncLoop Waits on Server.Initialized then starts Server.syncRunner.Loop() until context is Done
func (s *Server) SyncLoop(ctx context.Context) {
	klog.Infof("SyncLoop waiting for server initialization")
	_ = wait.PollUntilWithContext(ctx, time.Millisecond*500, func(_ context.Context) (done bool, err error) {
		return s.isInitialized(), nil
	})
	klog.Infof("starting sync runner")
	s.Sync()
	s.syncRunner.Loop(ctx.Done())
}

// NewServer creates a new *Server instance
//
//nolint:funlen
func NewServer(o *Options) (*Server, error) {
	var kubeConfig *rest.Config
	var err error

	switch {
	case o.KConfig != nil:
		kubeConfig = o.KConfig
	case o.Kubeconfig != "":
		kubeConfig, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: o.Kubeconfig},
			&clientcmd.ConfigOverrides{ClusterInfo: clientcmdapi.Cluster{Server: o.master}},
		).ClientConfig()
	default:
		klog.Info("Neither kubeconfig file nor master URL was specified. Falling back to in-cluster config.")
		kubeConfig, err = rest.InClusterConfig()
	}

	if err != nil {
		return nil, err
	}

	if o.podRulesPath != "" {
		// create pod rules directory if it does not exist
		if _, err := os.Stat(o.podRulesPath); os.IsNotExist(err) {
			err = os.Mkdir(o.podRulesPath, 0700)
			if err != nil {
				return nil, err
			}
		}
	}

	client, err := clientset.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	networkPolicyClient, err := multiclient.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	netdefClient, err := netdefclient.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	hostname, err := multiutils.GetHostname(o.hostnameOverride)
	if err != nil {
		return nil, err
	}

	eventBroadcaster := record.NewBroadcaster()
	recorder := eventBroadcaster.NewRecorder(
		scheme.Scheme,
		v1.EventSource{Component: "multi-networkpolicy-node", Host: hostname})

	nodeRef := &v1.ObjectReference{
		Kind:      "Node",
		Name:      hostname,
		UID:       types.UID(hostname),
		Namespace: "",
	}

	syncPeriod := 30 * time.Second
	minSyncPeriod := 0 * time.Second
	burstSyncs := 2

	policyChanges := controllers.NewPolicyChangeTracker()
	netdefChanges := controllers.NewNetDefChangeTracker()
	nsChanges := controllers.NewNamespaceChangeTracker()
	podChanges := controllers.NewPodChangeTracker(o.networkPlugins, netdefChanges)

	if o.policyRuleRenderer == nil {
		o.policyRuleRenderer = policyrules.NewRendererImpl(klog.NewKlogr().WithName("policy-rule-renderer"))
	}

	if o.tcRuleGenerator == nil {
		o.tcRuleGenerator = generator.NewSimpleTCGenerator()
	}

	if o.sriovnetProvider == nil {
		o.sriovnetProvider = netwrappers.NewSriovnetProviderImpl()
	}

	if o.netlinkProvider == nil {
		o.netlinkProvider = netwrappers.NewNetlinkProviderImpl()
	}

	server := &Server{
		Options:             o,
		Client:              client,
		Hostname:            hostname,
		NetworkPolicyClient: networkPolicyClient,
		NetDefClient:        netdefClient,
		Broadcaster:         eventBroadcaster,
		Recorder:            recorder,
		ConfigSyncPeriod:    15 * time.Minute,
		NodeRef:             nodeRef,
		policyChanges:       policyChanges,
		podChanges:          podChanges,
		netdefChanges:       netdefChanges,
		nsChanges:           nsChanges,
		podMap:              make(controllers.PodMap),
		policyMap:           make(controllers.PolicyMap),
		namespaceMap:        make(controllers.NamespaceMap),
		startPodConfig:      make(chan struct{}),

		policyRuleRenderer:      o.policyRuleRenderer,
		tcRuleGenerator:         o.tcRuleGenerator,
		sriovnetProvider:        o.sriovnetProvider,
		netlinkProvider:         o.netlinkProvider,
		createActuatorFromRepFn: o.createActuatorForRep,
	}

	if server.createActuatorFromRepFn == nil {
		// use builtin method if unspecified
		server.createActuatorFromRepFn = server.createActuatorForRep
	}

	server.syncRunner = async.NewBoundedFrequencyRunner(
		"sync-runner", server.syncMultiPolicy, minSyncPeriod, syncPeriod, burstSyncs)

	return server, nil
}

// Sync requests to Run syncRunner
func (s *Server) Sync() {
	klog.V(5).Infof("Sync Requested")
	s.syncRunner.Run()
}

// AllExceptPodsSynced return true if all informers except Pod have synced caches
func (s *Server) AllExceptPodsSynced() bool {
	return s.policySynced && s.netdefSynced && s.nsSynced
}

// AllSynced return true if all informers caches synced
func (s *Server) AllSynced() bool {
	return s.policySynced && s.netdefSynced && s.nsSynced && s.podSynced
}

// OnPodAdd Event handler for Pod
func (s *Server) OnPodAdd(pod *v1.Pod) {
	klog.V(5).InfoS("OnPodAdd", "namespace", pod.Namespace, "name", pod.Name)
	if s.podChanges.Update(nil, pod) && s.podSynced {
		s.Sync()
	}
}

// OnPodUpdate Event handler for Pod
func (s *Server) OnPodUpdate(oldPod, pod *v1.Pod) {
	klog.V(5).InfoS("OnPodUpdate", "namespace", oldPod.Namespace, "name", oldPod.Name)
	if s.podChanges.Update(oldPod, pod) && s.podSynced {
		s.Sync()
	}
}

// OnPodDelete Event handler for Pod
func (s *Server) OnPodDelete(pod *v1.Pod) {
	klog.V(5).InfoS("OnPodDelete", "namespace", pod.Namespace, "name", pod.Name)
	if s.podChanges.Update(pod, nil) && s.podSynced {
		s.Sync()
	}
}

// OnPodSynced Event handler for Pod
func (s *Server) OnPodSynced() {
	klog.Infof("OnPodSynced")
	s.mu.Lock()
	defer s.mu.Unlock()

	s.podSynced = true
	s.setInitialized(s.AllSynced())
}

// OnPolicyAdd Event handler for Policy
func (s *Server) OnPolicyAdd(policy *multiv1beta1.MultiNetworkPolicy) {
	klog.V(5).InfoS("OnPolicyAdd", "namespace", policy.Namespace, "name", policy.Name)
	if s.policyChanges.Update(nil, policy) && s.isInitialized() {
		s.Sync()
	}
}

// OnPolicyUpdate Event handler for Policy
func (s *Server) OnPolicyUpdate(oldPolicy, policy *multiv1beta1.MultiNetworkPolicy) {
	klog.V(5).InfoS("OnPolicyUpdate", "namespace", oldPolicy.Namespace, "name", oldPolicy.Name)
	if s.policyChanges.Update(oldPolicy, policy) && s.isInitialized() {
		s.Sync()
	}
}

// OnPolicyDelete Event handler for Policy
func (s *Server) OnPolicyDelete(policy *multiv1beta1.MultiNetworkPolicy) {
	klog.V(5).InfoS("OnPolicyDelete", "namespace", policy.Namespace, "name", policy.Name)
	if s.policyChanges.Update(policy, nil) && s.isInitialized() {
		s.Sync()
	}
}

// OnPolicySynced Event handler for Policy
func (s *Server) OnPolicySynced() {
	klog.Infof("OnPolicySynced")
	s.mu.Lock()
	defer s.mu.Unlock()

	s.policySynced = true
	s.setInitialized(s.AllSynced())

	if s.AllExceptPodsSynced() {
		if !s.startPodConfigClosed {
			close(s.startPodConfig)
			s.startPodConfigClosed = true
		}
	}
}

// OnNetDefAdd Event handler for NetworkAttachmentDefinition
func (s *Server) OnNetDefAdd(net *netdefv1.NetworkAttachmentDefinition) {
	klog.V(5).InfoS("OnNetDefAdd", "namespace", net.Namespace, "name", net.Name)
	if s.netdefChanges.Update(nil, net) && s.isInitialized() {
		s.Sync()
	}
}

// OnNetDefUpdate Event handler for NetworkAttachmentDefinition
func (s *Server) OnNetDefUpdate(oldNet, net *netdefv1.NetworkAttachmentDefinition) {
	klog.V(5).InfoS("OnNetDefUpdate", "namespace", oldNet.Namespace, "name", oldNet.Name)
	if s.netdefChanges.Update(oldNet, net) && s.isInitialized() {
		s.Sync()
	}
}

// OnNetDefDelete Event handler for NetworkAttachmentDefinition
func (s *Server) OnNetDefDelete(net *netdefv1.NetworkAttachmentDefinition) {
	klog.V(5).InfoS("OnNetDefDelete", "namespace", net.Namespace, "name", net.Name)
	if s.netdefChanges.Update(net, nil) && s.isInitialized() {
		s.Sync()
	}
}

// OnNetDefSynced Event handler for NetworkAttachmentDefinition
func (s *Server) OnNetDefSynced() {
	klog.Infof("OnNetDefSynced")
	s.mu.Lock()
	defer s.mu.Unlock()

	s.netdefSynced = true
	s.setInitialized(s.AllSynced())

	if s.AllExceptPodsSynced() {
		if !s.startPodConfigClosed {
			close(s.startPodConfig)
			s.startPodConfigClosed = true
		}
	}
}

// OnNamespaceAdd Event handler for Namespace
func (s *Server) OnNamespaceAdd(ns *v1.Namespace) {
	klog.V(5).InfoS("OnNamespaceAdd", "namespace", ns.Namespace, "name", ns.Name)
	if s.nsChanges.Update(nil, ns) && s.isInitialized() {
		s.Sync()
	}
}

// OnNamespaceUpdate Event handler for Namespace
func (s *Server) OnNamespaceUpdate(oldNamespace, ns *v1.Namespace) {
	klog.V(5).InfoS("OnNamespaceUpdate", "namespace", oldNamespace.Namespace, "name", oldNamespace.Name)
	if s.nsChanges.Update(oldNamespace, ns) && s.isInitialized() {
		s.Sync()
	}
}

// OnNamespaceDelete Event handler for Namespace
func (s *Server) OnNamespaceDelete(ns *v1.Namespace) {
	klog.V(5).InfoS("OnNamespaceDelete", "namespace", ns.Namespace, "name", ns.Name)
	if s.nsChanges.Update(ns, nil) && s.isInitialized() {
		s.Sync()
	}
}

// OnNamespaceSynced Event handler for Namespace
func (s *Server) OnNamespaceSynced() {
	klog.Infof("OnNamespaceSynced")
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nsSynced = true
	s.setInitialized(s.AllSynced())

	if s.AllExceptPodsSynced() {
		if !s.startPodConfigClosed {
			close(s.startPodConfig)
			s.startPodConfigClosed = true
		}
	}
}

// syncMultiPolicy is the main business logic for Server, it syncs TC for pod interfaces to match
// defined MultiNetworkPolicy
func (s *Server) syncMultiPolicy() {
	klog.Infof("syncMultiPolicy")
	now := time.Now()
	defer func() {
		klog.V(4).InfoS("syncMultiPolicy", "execution time", time.Since(now))
	}()

	s.namespaceMap.Update(s.nsChanges)
	s.podMap.Update(s.podChanges)
	s.policyMap.Update(s.policyChanges)

	podsInfo, _ := s.podMap.List()
	podsWithRules := make(map[string]struct{})
	for _, p := range podsInfo {
		podNamespacedName := types.NamespacedName{Namespace: p.Namespace, Name: p.Name}.String()
		// skip pods that are not scheduled on this node
		if !multiutils.CheckNodeNameIdentical(s.Hostname, p.NodeName) {
			klog.V(5).InfoS("pod skipped. not scheduled on this node", "pod", podNamespacedName)
			continue
		}

		// update podMap with latest changes
		s.podMap.Update(s.podChanges)
		// get PodInfo again to have its most updated state
		podInfo, err := s.podMap.GetPodInfo(p.Namespace, p.Name)
		if err != nil {
			klog.Errorf("cannot get %s podInfo: %v", podNamespacedName, err)
			continue
		}

		if len(podInfo.Interfaces) == 0 {
			klog.V(8).InfoS("skipped as pod has no secondary network interfaces", "pod", podNamespacedName)
			continue
		}
		klog.InfoS("syncing policy for", "pod", podNamespacedName)

		rules, err := s.policyRuleRenderer.RenderEgress(podInfo, s.policyMap, s.podMap, s.namespaceMap)
		if err != nil {
			klog.ErrorS(err, "Failed to render egress policy rules. skipping.")
			continue
		}
		podsWithRules[p.UID] = struct{}{}
		klog.V(5).Infof("rules: %+v", rules)

		// convert rules to TC and apply them
		for _, ruleSet := range rules {
			klog.InfoS("processing policy rule set for pod",
				"network", ruleSet.IfcInfo.Network, "interface", ruleSet.IfcInfo.InterfaceName)

			// get VF rep
			rep, err := s.getRepresentor(ruleSet.IfcInfo.DeviceID)
			if err != nil {
				klog.ErrorS(err, "Failed to get VF representor. skipping.", "pci-address",
					ruleSet.IfcInfo.DeviceID)
				continue
			}

			// Generate TC rules for ruleSet
			tcObjs, err := s.tcRuleGenerator.GenerateFromPolicyRuleSet(ruleSet)
			if err != nil {
				klog.ErrorS(err, "Failed to generate tc rules. skipping.")
				continue
			}
			klog.V(5).Infof("tcObjs: %+v", tcObjs)

			// Actuate TC rules
			actuator, err := s.createActuatorFromRepFn(rep)
			if err != nil {
				klog.ErrorS(err, "Failed to create actuator. skipping.")
				continue
			}

			err = actuator.Actuate(tcObjs)
			if err != nil {
				klog.ErrorS(err, "Failed to actuate rules. skipping.")
				continue
			}
			klog.InfoS("rules set applied successfully for pod")

			// optionally save rules to file
			err = s.savePodInterfaceRules(podInfo, ruleSet, tcObjs, rep)
			if err != nil {
				klog.Warningf("failed to save pod interface rules. %v", err)
				continue
			}
		}
	}

	s.deleteStalePodInterfaceRules(podsWithRules)
}

// savePodInterfaceRules saves pod interface tc objects to file if podRulesPath option is enabled in server
func (s *Server) savePodInterfaceRules(
	pInfo *controllers.PodInfo, ruleSet policyrules.PolicyRuleSet, tcObj *generator.Objects, rep string) error {
	// skip it if no podRulesPath option
	if s.Options.podRulesPath == "" {
		return nil
	}

	// create directory for pod if not exist
	podRulesPath := filepath.Join(s.Options.podRulesPath, pInfo.UID)
	if _, err := os.Stat(podRulesPath); os.IsNotExist(err) {
		err := os.Mkdir(podRulesPath, 0700)
		if err != nil {
			klog.Errorf("cannot create pod dir (%s): %v", podRulesPath, err)
			return err
		}
	}

	networkNameNoSep := strings.ReplaceAll(ruleSet.IfcInfo.Network, "/", "-")
	fullPath := fmt.Sprintf("%s/%s-%s.rules", podRulesPath, networkNameNoSep, rep)
	klog.V(4).InfoS("saving pod interface rules", "path", fullPath)
	fileActuator := tc.NewActuatorFileWriterImpl(fullPath, klog.NewKlogr().WithName("actuator-file-writer"))
	err := fileActuator.Actuate(tcObj)
	return err
}

// deleteStalePodInterfaceRules deletes stale pod rule folders
func (s *Server) deleteStalePodInterfaceRules(podsWithRules map[string]struct{}) {
	if s.Options.podRulesPath == "" {
		return
	}

	// delete stale pod rule dirs
	klog.V(4).Info("deleting stale pod rules")
	entries, err := os.ReadDir(s.Options.podRulesPath)
	if err != nil {
		klog.Warningf("failed to read pod rules dir(%s). %v", s.Options.podRulesPath, err)
		return
	}

	for _, entry := range entries {
		if _, ok := podsWithRules[entry.Name()]; !ok {
			podRulesPath := filepath.Join(s.Options.podRulesPath, entry.Name())
			klog.V(4).InfoS("deleting pod rules dir", "path", podRulesPath)
			err := os.RemoveAll(podRulesPath)
			if err != nil {
				klog.Warningf("failed to delete pod rules dir. %v", err)
			}
		}
	}
}

// getRepresentor returns Representor netdev for pci address
func (s *Server) getRepresentor(pciAddr string) (string, error) {
	vfIdx, err := s.sriovnetProvider.GetVfIndexByPciAddress(pciAddr)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get VF index for PCI address %s", pciAddr)
	}
	uplink, err := s.sriovnetProvider.GetUplinkRepresentor(pciAddr)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get uplink for PCI address %s", pciAddr)
	}
	vfRep, err := s.sriovnetProvider.GetVfRepresentor(uplink, vfIdx)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get VF representor for uplink: %s vf index: %d", uplink, vfIdx)
	}
	return vfRep, nil
}

// createActuatorForRepFn creates a new tc.Actuator given tc Driver type and representor netdev
func (s *Server) createActuatorForRep(rep string) (tc.Actuator, error) {
	var tcAPI tc.TC

	switch s.Options.tcDriver {
	case "netlink":
		lnk, err := s.netlinkProvider.LinkByName(rep)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get link from rep: %s", rep)
		}
		tcAPI = netlinkdriver.NewTcNetlinkImpl(
			lnk, klog.NewKlogr().WithName("tc-netlink-driver"), s.netlinkProvider)
	case "cmdline":
		tcAPI = cmdlinedriver.NewTcCmdLineImpl(
			rep, klog.NewKlogr().WithName("tc-cmdline-driver"), exec.New())
	default:
		return nil, fmt.Errorf("unknown TC driver: %s", s.Options.tcDriver)
	}

	return tc.NewActuatorTCImpl(tcAPI, klog.NewKlogr().WithName("tc-actuator")), nil
}
