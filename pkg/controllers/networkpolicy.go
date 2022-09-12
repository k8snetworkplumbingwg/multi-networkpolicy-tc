package controllers

//nolint:lll
import (
	"fmt"
	"reflect"
	"sync"
	"time"

	multiv1beta1 "github.com/k8snetworkplumbingwg/multi-networkpolicy/pkg/apis/k8s.cni.cncf.io/v1beta1"
	multiinformerv1beta1 "github.com/k8snetworkplumbingwg/multi-networkpolicy/pkg/client/informers/externalversions/k8s.cni.cncf.io/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	multiutils "github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/utils"
)

// NetworkPolicyHandler is an abstract interface of objects which receive
// notifications about policy object changes.
type NetworkPolicyHandler interface {
	// OnPolicyAdd is called whenever creation of new policy object
	// is observed.
	OnPolicyAdd(policy *multiv1beta1.MultiNetworkPolicy)
	// OnPolicyUpdate is called whenever modification of an existing
	// policy object is observed.
	OnPolicyUpdate(oldPolicy, policy *multiv1beta1.MultiNetworkPolicy)
	// OnPolicyDelete is called whenever deletion of an existing policy
	// object is observed.
	OnPolicyDelete(policy *multiv1beta1.MultiNetworkPolicy)
	// OnPolicySynced is called once all the initial event handlers were
	// called and the state is fully propagated to local cache.
	OnPolicySynced()
}

// NetworkPolicyConfig registers event handlers for MultiNetworkPolicy
type NetworkPolicyConfig struct {
	listerSynced  cache.InformerSynced
	eventHandlers []NetworkPolicyHandler
}

// NewNetworkPolicyConfig creates a new NetworkPolicyConfig .
func NewNetworkPolicyConfig(policyInformer multiinformerv1beta1.MultiNetworkPolicyInformer,
	resyncPeriod time.Duration) *NetworkPolicyConfig {
	result := &NetworkPolicyConfig{
		listerSynced: policyInformer.Informer().HasSynced,
	}

	policyInformer.Informer().AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    result.handleAddPolicy,
			UpdateFunc: result.handleUpdatePolicy,
			DeleteFunc: result.handleDeletePolicy,
		}, resyncPeriod,
	)

	return result
}

// RegisterEventHandler registers a handler which is called on every policy change.
func (c *NetworkPolicyConfig) RegisterEventHandler(handler NetworkPolicyHandler) {
	c.eventHandlers = append(c.eventHandlers, handler)
}

// Run waits for cache synced and invokes handlers after syncing.
func (c *NetworkPolicyConfig) Run(stopCh <-chan struct{}) {
	klog.Info("Starting policy config controller")

	if !cache.WaitForNamedCacheSync("policy config", stopCh, c.listerSynced) {
		return
	}

	for i := range c.eventHandlers {
		klog.V(4).Infof("Calling handler.OnPolicySynced()")
		c.eventHandlers[i].OnPolicySynced()
	}
}

// handleAddPolicy calls registered event handlers OnPolicyAdd
func (c *NetworkPolicyConfig) handleAddPolicy(obj interface{}) {
	policy, ok := obj.(*multiv1beta1.MultiNetworkPolicy)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
		return
	}

	for i := range c.eventHandlers {
		klog.V(4).Infof("Calling handler.OnPolicyAdd")
		c.eventHandlers[i].OnPolicyAdd(policy)
	}
}

// handleUpdatePolicy calls registered event handlers OnPolicyUpdate
func (c *NetworkPolicyConfig) handleUpdatePolicy(oldObj, newObj interface{}) {
	oldPolicy, ok := oldObj.(*multiv1beta1.MultiNetworkPolicy)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", oldObj))
		return
	}
	policy, ok := newObj.(*multiv1beta1.MultiNetworkPolicy)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", newObj))
		return
	}
	for i := range c.eventHandlers {
		klog.V(4).Infof("Calling handler.OnPolicyUpdate")
		c.eventHandlers[i].OnPolicyUpdate(oldPolicy, policy)
	}
}

// handleDeletePolicy calls registered event handlers OnPolicyDelete
func (c *NetworkPolicyConfig) handleDeletePolicy(obj interface{}) {
	policy, ok := obj.(*multiv1beta1.MultiNetworkPolicy)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
		}
		if policy, ok = tombstone.Obj.(*multiv1beta1.MultiNetworkPolicy); !ok {
			utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
			return
		}
	}
	for i := range c.eventHandlers {
		klog.V(4).Infof("Calling handler.OnPolicyDelete")
		c.eventHandlers[i].OnPolicyDelete(policy)
	}
}

// PolicyInfo contains information that defines a policy.
type PolicyInfo struct {
	PolicyNetworks []string
	Policy         *multiv1beta1.MultiNetworkPolicy
}

// Name returns MultiNetworkPolicy name
func (info *PolicyInfo) Name() string {
	return info.Policy.ObjectMeta.Name
}

// Namespace returns MultiNetworkPolicy namespace
func (info *PolicyInfo) Namespace() string {
	return info.Policy.ObjectMeta.Namespace
}

// AppliesForNetwork returns true if Policy applies for the provided network
func (info *PolicyInfo) AppliesForNetwork(networkName string) bool {
	for _, policyNetName := range info.PolicyNetworks {
		if policyNetName == networkName {
			return true
		}
	}
	return false
}

// PolicyMap maps MultiNetworkPolicy namespaced name to PolicyInfo
type PolicyMap map[types.NamespacedName]PolicyInfo

// Update updates PolicyMap base on the given changes
func (pm *PolicyMap) Update(changes *PolicyChangeTracker) {
	if pm != nil {
		pm.apply(changes)
	}
}

// apply applies changes to PolicyMap
func (pm *PolicyMap) apply(changes *PolicyChangeTracker) {
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
	changes.items = make(map[types.NamespacedName]*policyChange)
}

// merge merges changes into PolicyMap
func (pm *PolicyMap) merge(other PolicyMap) {
	if pm == nil {
		return
	}

	for policyName, info := range other {
		(*pm)[policyName] = info
	}
}

// unmerge deletes entries in other from PolicyMap
func (pm *PolicyMap) unmerge(other PolicyMap) {
	if pm == nil {
		return
	}

	for policyName := range other {
		delete(*pm, policyName)
	}
}

// policyChange represents a change in MultiNetworkPolicy represented via PolicyMap
type policyChange struct {
	previous PolicyMap
	current  PolicyMap
}

// PolicyChangeTracker carries state about uncommitted changes to an arbitrary number of
// MultiNetworkPolicies keyed by their namespace and name
type PolicyChangeTracker struct {
	// lock protects items.
	lock sync.Mutex
	// items maps a service to its serviceChange.
	items map[types.NamespacedName]*policyChange
}

// String returns a string representation of PolicyChangeTracker changes
func (pct *PolicyChangeTracker) String() string {
	return fmt.Sprintf("policyChange: %v", pct.items)
}

// newPolicyInfo creates a new instance of PolicyInfo
func (pct *PolicyChangeTracker) newPolicyInfo(policy *multiv1beta1.MultiNetworkPolicy) *PolicyInfo {
	info := &PolicyInfo{
		PolicyNetworks: multiutils.NetworkListFromPolicy(policy),
		Policy:         policy,
	}
	return info
}

// policyToPolicyMap creates PolicyMap from MultiNetworkPolicy.
// Note(adrianc): it is basically a map with single entry.
func (pct *PolicyChangeTracker) policyToPolicyMap(policy *multiv1beta1.MultiNetworkPolicy) PolicyMap {
	if policy == nil {
		return nil
	}

	policyMap := make(PolicyMap)
	policyInfo := pct.newPolicyInfo(policy)
	policyMap[types.NamespacedName{Namespace: policy.Namespace, Name: policy.Name}] = *policyInfo

	return policyMap
}

// Update handles an update of a given MultiNetworkPolicy
func (pct *PolicyChangeTracker) Update(previous, current *multiv1beta1.MultiNetworkPolicy) bool {
	policy := current

	if policy == nil {
		policy = previous
	}
	if policy == nil {
		return false
	}

	namespacedName := types.NamespacedName{Namespace: policy.Namespace, Name: policy.Name}

	pct.lock.Lock()
	defer pct.lock.Unlock()

	change, exists := pct.items[namespacedName]
	if !exists {
		change = &policyChange{}
		prevPolicyMap := pct.policyToPolicyMap(previous)
		change.previous = prevPolicyMap
		pct.items[namespacedName] = change
	}

	curPolicyMap := pct.policyToPolicyMap(current)
	change.current = curPolicyMap
	if reflect.DeepEqual(change.previous, change.current) {
		delete(pct.items, namespacedName)
	}

	return true
}

// NewPolicyChangeTracker creates a new instance of PolicyChangeTracker
func NewPolicyChangeTracker() *PolicyChangeTracker {
	return &PolicyChangeTracker{
		items: make(map[types.NamespacedName]*policyChange),
	}
}
