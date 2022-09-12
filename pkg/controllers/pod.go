package controllers

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	multiv1beta1 "github.com/k8snetworkplumbingwg/multi-networkpolicy/pkg/apis/k8s.cni.cncf.io/v1beta1"
	netdefv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	netdefutils "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/utils"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	multiutils "github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/utils"
)

// PodHandler is an abstract interface of objects which receive
// notifications about pod object changes.
type PodHandler interface {
	// OnPodAdd is called whenever creation of new pod object
	// is observed.
	OnPodAdd(pod *v1.Pod)
	// OnPodUpdate is called whenever modification of an existing
	// pod object is observed.
	OnPodUpdate(oldPod, pod *v1.Pod)
	// OnPodDelete is called whenever deletion of an existing pod
	// object is observed.
	OnPodDelete(pod *v1.Pod)
	// OnPodSynced is called once all the initial event handlers were
	// called and the state is fully propagated to local cache.
	OnPodSynced()
}

// PodConfig registers event handlers for PodInformer
type PodConfig struct {
	listerSynced  cache.InformerSynced
	eventHandlers []PodHandler
}

// NewPodConfig creates a new PodConfig.
func NewPodConfig(podInformer coreinformers.PodInformer, resyncPeriod time.Duration) *PodConfig {
	result := &PodConfig{
		listerSynced: podInformer.Informer().HasSynced,
	}

	podInformer.Informer().AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    result.handleAddPod,
			UpdateFunc: result.handleUpdatePod,
			DeleteFunc: result.handleDeletePod,
		},
		resyncPeriod,
	)
	return result
}

// RegisterEventHandler registers a handler which is called on every pod change.
func (c *PodConfig) RegisterEventHandler(handler PodHandler) {
	c.eventHandlers = append(c.eventHandlers, handler)
}

// Run waits for cache synced and invokes handlers after syncing.
func (c *PodConfig) Run(stopCh <-chan struct{}) {
	klog.Info("Starting pod config controller")

	if !cache.WaitForNamedCacheSync("pod config", stopCh, c.listerSynced) {
		return
	}

	for i := range c.eventHandlers {
		klog.V(9).Infof("Calling handler.OnPodSynced()")
		c.eventHandlers[i].OnPodSynced()
	}
}

// handleAddPod calls registered event handlers OnPodAdd
func (c *PodConfig) handleAddPod(obj interface{}) {
	pod, ok := obj.(*v1.Pod)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
		return
	}

	for i := range c.eventHandlers {
		klog.V(9).Infof("Calling handler.OnPodAdd")
		c.eventHandlers[i].OnPodAdd(pod)
	}
}

// handleUpdatePod calls registered event handlers OnPodUpdate
func (c *PodConfig) handleUpdatePod(oldObj, newObj interface{}) {
	oldPod, ok := oldObj.(*v1.Pod)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", oldObj))
		return
	}
	pod, ok := newObj.(*v1.Pod)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", newObj))
		return
	}
	for i := range c.eventHandlers {
		klog.V(9).Infof("Calling handler.OnPodUpdate")
		c.eventHandlers[i].OnPodUpdate(oldPod, pod)
	}
}

// handleDeletePod calls registered event handlers OnPodDelete
func (c *PodConfig) handleDeletePod(obj interface{}) {
	pod, ok := obj.(*v1.Pod)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
		}
		if pod, ok = tombstone.Obj.(*v1.Pod); !ok {
			utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
			return
		}
	}
	for i := range c.eventHandlers {
		klog.V(9).Infof("Calling handler.OnPodDelete")
		c.eventHandlers[i].OnPodDelete(pod)
	}
}

// InterfaceInfo contains information that defines a Pod Interface.
type InterfaceInfo struct {
	NetattachName string
	DeviceID      string
	InterfaceName string
	InterfaceType string
	IPs           []string
}

// CheckPolicyNetwork checks whether given interface is target or not,
// based on policyNetworks
func (info *InterfaceInfo) CheckPolicyNetwork(policyNetworks []string) bool {
	for _, policyNetworkName := range policyNetworks {
		if policyNetworkName == info.NetattachName {
			return true
		}
	}
	return false
}

// PodInfo contains information that defines a pod.
type PodInfo struct {
	UID           string
	Name          string
	Labels        map[string]string
	Namespace     string
	NetworkStatus []netdefv1.NetworkStatus
	NodeName      string
	Interfaces    []InterfaceInfo
}

// CheckPolicyNetwork checks whether given pod is target or not,
// based on policyNetworks
func (info *PodInfo) CheckPolicyNetwork(policyNetworks []string) bool {
	for _, intf := range info.Interfaces {
		if intf.CheckPolicyNetwork(policyNetworks) {
			return true
		}
	}
	return false
}

// PolicyAppliesForPod returns true if provided policy is applicable to the provided pod
// by checking if the pod and policy share the same namespace and the pod matches the policy's pod selector
// Note: it does not mean it applies to any networks of that pod
func (info *PodInfo) PolicyAppliesForPod(policy *multiv1beta1.MultiNetworkPolicy) (bool, error) {
	if policy.Namespace != info.Namespace {
		return false, nil
	}
	if policy.Spec.PodSelector.Size() != 0 {
		policyMap, err := metav1.LabelSelectorAsMap(&policy.Spec.PodSelector)
		if err != nil {
			return false, fmt.Errorf("bad label selector for policy [%s]: %w",
				types.NamespacedName{Namespace: policy.Namespace, Name: policy.Name}.String(), err)
		}
		policyPodSelector := labels.Set(policyMap).AsSelectorPreValidated()
		if !policyPodSelector.Matches(labels.Set(info.Labels)) {
			return false, nil
		}
	}
	return true, nil
}

// String returns a string representation of PodInfo
func (info *PodInfo) String() string {
	return fmt.Sprintf("pod:%s", info.Name)
}

// podChange represents a change in Pod represented via PodMap
type podChange struct {
	previous PodMap
	current  PodMap
}

// PodChangeTracker carries state about uncommitted changes to an arbitrary number of
// Pods keyed by their namespace and name
type PodChangeTracker struct {
	// lock protects items.
	lock           sync.Mutex
	networkPlugins []string
	netdefChanges  *NetDefChangeTracker
	// items maps a service to its podChange.
	items map[types.NamespacedName]*podChange
}

// String returns a string representation of PodChangeTracker
func (pct *PodChangeTracker) String() string {
	return fmt.Sprintf("podChange: %v", pct.items)
}

// newPodInfo creates a new instance of PodInfo
func (pct *PodChangeTracker) newPodInfo(pod *v1.Pod) *PodInfo {
	// TODO(adrianc): we should rework PodChangeTracker to only track pods that have secondary network with CNI
	// that is supported.
	var statuses []netdefv1.NetworkStatus
	var netifs []InterfaceInfo
	// get network information only if the pod is ready
	podNamespacedName := types.NamespacedName{
		Namespace: pod.Namespace,
		Name:      pod.Name,
	}.String()
	klog.V(8).Infof("pod:%s pod-node-name:%s", podNamespacedName, pod.Spec.NodeName)
	if multiutils.IsMultiNetworkpolicyTarget(pod) {
		networks, err := netdefutils.ParsePodNetworkAnnotation(pod)
		if err != nil {
			if _, ok := err.(*netdefv1.NoK8sNetworkError); !ok {
				klog.Errorf("failed to get pod network annotation: %v", err)
			}
		}
		// parse networkStatus
		statuses, err = netdefutils.GetNetworkStatus(pod)
		if err != nil {
			klog.V(8).Infof("unable to get network status for pod %s. %v", podNamespacedName, err)
		}
		klog.V(1).Infof("creating podInfo for pod:%s", podNamespacedName)

		// netdefname -> plugin name map
		networkPlugins := make(map[types.NamespacedName]string)
		if networks == nil {
			klog.V(8).Infof("%s: NO NET", podNamespacedName)
		} else {
			klog.V(8).Infof("%s: net: %v", podNamespacedName, networks)
		}
		for _, n := range networks {
			namespace := pod.Namespace
			if n.Namespace != "" {
				namespace = n.Namespace
			}
			namespacedName := types.NamespacedName{Namespace: namespace, Name: n.Name}
			klog.V(8).Infof("networkPlugins[%s], %v", namespacedName, pct.netdefChanges.GetPluginType(namespacedName))
			networkPlugins[namespacedName] = pct.netdefChanges.GetPluginType(namespacedName)
		}
		klog.V(8).Infof("netdef->pluginMap: %v", networkPlugins)

		// match it with
		for _, s := range statuses {
			klog.V(8).Infof("processing network status: %+v", s)
			var netNamespace, netName string
			slashItems := strings.Split(s.Name, "/")
			if len(slashItems) == 2 {
				netNamespace = strings.TrimSpace(slashItems[0])
				netName = slashItems[1]
			} else {
				netNamespace = pod.ObjectMeta.Namespace
				netName = s.Name
			}
			namespacedName := types.NamespacedName{Namespace: netNamespace, Name: netName}

			for _, pluginName := range pct.networkPlugins {
				if networkPlugins[namespacedName] == pluginName {
					deviceID, err := multiutils.GetDeviceIDFromNetworkStatus(s)
					if err != nil {
						klog.ErrorS(err, "failed to get device ID for pod interface",
							"pod", podNamespacedName, "network", namespacedName)
						continue
					}
					netifs = append(netifs, InterfaceInfo{
						NetattachName: s.Name,
						InterfaceName: s.Interface,
						DeviceID:      deviceID,
						InterfaceType: networkPlugins[namespacedName],
						IPs:           s.IPs,
					})
				}
			}
		}

		klog.V(6).Infof("Pod:%s netIF:%+v", podNamespacedName, netifs)
	} else {
		klog.V(1).Infof("pod:%s, pod-node-name:%s, not ready",
			podNamespacedName, pod.Spec.NodeName)
	}
	info := &PodInfo{
		UID:           string(pod.UID),
		Name:          pod.ObjectMeta.Name,
		Labels:        pod.Labels,
		Namespace:     pod.ObjectMeta.Namespace,
		NetworkStatus: statuses,
		NodeName:      pod.Spec.NodeName,
		Interfaces:    netifs,
	}
	return info
}

// NewPodChangeTracker creates a new instance of PodChangeTracker
func NewPodChangeTracker(networkPlugins []string, ndt *NetDefChangeTracker) *PodChangeTracker {
	return &PodChangeTracker{
		items:          make(map[types.NamespacedName]*podChange),
		networkPlugins: networkPlugins,
		netdefChanges:  ndt,
	}
}

// podToPodMap creates PodMap from Pod.
// Note(adrianc): it is basically a map with single entry.
func (pct *PodChangeTracker) podToPodMap(pod *v1.Pod) PodMap {
	if pod == nil {
		return nil
	}

	podMap := make(PodMap)
	podinfo := pct.newPodInfo(pod)
	podMap[types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}] = *podinfo

	return podMap
}

// Update handles an update of a given Pod
func (pct *PodChangeTracker) Update(previous, current *v1.Pod) bool {
	pod := current

	if pod == nil {
		pod = previous
	}
	if pod == nil {
		return false
	}
	namespacedName := types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}

	pct.lock.Lock()
	defer pct.lock.Unlock()

	change, exists := pct.items[namespacedName]
	if !exists {
		change = &podChange{}
		prevPodMap := pct.podToPodMap(previous)
		change.previous = prevPodMap
		pct.items[namespacedName] = change
	}
	curPodMap := pct.podToPodMap(current)
	change.current = curPodMap
	if reflect.DeepEqual(change.previous, change.current) {
		delete(pct.items, namespacedName)
	}
	return true
}

// PodMap maps Pod namespaced name to PodInfo
type PodMap map[types.NamespacedName]PodInfo

// Update updates podMap base on the given changes
func (pm *PodMap) Update(changes *PodChangeTracker) {
	if pm != nil {
		pm.apply(changes)
	}
}

// apply changes to PodMap
func (pm *PodMap) apply(changes *PodChangeTracker) {
	if pm == nil || changes == nil {
		return
	}

	changes.lock.Lock()
	defer changes.lock.Unlock()
	for _, change := range changes.items {
		pm.unmerge(change.previous)
		pm.merge(change.current)
	}
	// clear changes after applying them to ServiceMap.
	changes.items = make(map[types.NamespacedName]*podChange)
}

// merge changes into PodMap
func (pm *PodMap) merge(other PodMap) {
	if pm == nil {
		return
	}

	for podName, info := range other {
		(*pm)[podName] = info
	}
}

// unmerge deletes entries in other from PodMap
func (pm *PodMap) unmerge(other PodMap) {
	if pm == nil {
		return
	}

	for podName := range other {
		delete(*pm, podName)
	}
}

// GetPodInfo returns PodInfo identified by namespace and name from PodMap
func (pm *PodMap) GetPodInfo(namespace, name string) (*PodInfo, error) {
	if pm == nil {
		return nil, fmt.Errorf("nil PodMap")
	}
	namespacedName := types.NamespacedName{Namespace: namespace, Name: name}

	podInfo, ok := (*pm)[namespacedName]
	if ok {
		return &podInfo, nil
	}

	return nil, fmt.Errorf("not found")
}

// List lists all PodInfo in PodMap, returns error if PodMap is nil
func (pm *PodMap) List() ([]PodInfo, error) {
	if pm == nil {
		return nil, fmt.Errorf("nil PodMap")
	}

	lst := make([]PodInfo, 0, len(*pm))
	for key := range *pm {
		lst = append(lst, (*pm)[key])
	}
	return lst, nil
}
