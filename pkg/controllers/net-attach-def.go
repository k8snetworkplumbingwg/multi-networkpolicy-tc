package controllers

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"

	cnitypes "github.com/containernetworking/cni/pkg/types"
	netdefv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	netdefinformerv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/client/informers/externalversions/k8s.cni.cncf.io/v1"
	netdefutils "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/utils"

	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

// NetDefHandler is an abstract interface of objects which receive
// notifications about net-attach-def object changes.
type NetDefHandler interface {
	// OnNetDefAdd is called whenever creation of new object
	// is observed.
	OnNetDefAdd(net *netdefv1.NetworkAttachmentDefinition)
	// OnNetDefUpdate is called whenever modification of an existing
	// object is observed.
	OnNetDefUpdate(oldNet, net *netdefv1.NetworkAttachmentDefinition)
	// OnNetDefDelete is called whenever deletion of an existing
	// object is observed.
	OnNetDefDelete(net *netdefv1.NetworkAttachmentDefinition)
	// OnNetDefSynced is called once all the initial event handlers were
	// called and the state is fully propagated to local cache.
	OnNetDefSynced()
}

// NetDefConfig registers event handlers for NetworkAttachmentDefinitionInformer
type NetDefConfig struct {
	listerSynced  cache.InformerSynced
	eventHandlers []NetDefHandler
}

// NewNetDefConfig creates a new instance of NetDefConfig
func NewNetDefConfig(netdefInformer netdefinformerv1.NetworkAttachmentDefinitionInformer, resyncPeriod time.Duration) *NetDefConfig {
	result := &NetDefConfig{
		listerSynced: netdefInformer.Informer().HasSynced,
	}

	netdefInformer.Informer().AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    result.handleAddNetDef,
			UpdateFunc: result.handleUpdateNetDef,
			DeleteFunc: result.handleDeleteNetDef,
		}, resyncPeriod,
	)

	return result
}

// RegisterEventHandler registers a handler which is called on every netdef change.
func (c *NetDefConfig) RegisterEventHandler(handler NetDefHandler) {
	c.eventHandlers = append(c.eventHandlers, handler)
}

// Run waits for cache synced and invokes handlers after syncing.
func (c *NetDefConfig) Run(stopCh <-chan struct{}) {
	klog.Info("Starting net-attach-def config controller")

	if !cache.WaitForNamedCacheSync("net-attach-def config", stopCh, c.listerSynced) {
		return
	}

	for i := range c.eventHandlers {
		klog.V(4).Infof("Calling handler.OnPolicySynced()")
		c.eventHandlers[i].OnNetDefSynced()
	}
}

// handleAddNetDef calls registered event handlers OnNetDefAdd
func (c *NetDefConfig) handleAddNetDef(obj interface{}) {
	netdef, ok := obj.(*netdefv1.NetworkAttachmentDefinition)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
		return
	}

	for i := range c.eventHandlers {
		klog.V(4).Infof("Calling handler.OnPolicyAdd")
		c.eventHandlers[i].OnNetDefAdd(netdef)
	}
}

// handleUpdateNetDef calls registered event handlers OnNetDefUpdate
func (c *NetDefConfig) handleUpdateNetDef(oldObj, newObj interface{}) {
	oldNetDef, ok := oldObj.(*netdefv1.NetworkAttachmentDefinition)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", oldObj))
		return
	}
	netdef, ok := newObj.(*netdefv1.NetworkAttachmentDefinition)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", newObj))
		return
	}
	for i := range c.eventHandlers {
		klog.V(4).Infof("Calling handler.OnPolicyUpdate")
		c.eventHandlers[i].OnNetDefUpdate(oldNetDef, netdef)
	}
}

// handleDeleteNetDef calls registered event handlers OnNetDefDelete
func (c *NetDefConfig) handleDeleteNetDef(obj interface{}) {
	netdef, ok := obj.(*netdefv1.NetworkAttachmentDefinition)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
		}
		if netdef, ok = tombstone.Obj.(*netdefv1.NetworkAttachmentDefinition); !ok {
			utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
			return
		}
	}
	for i := range c.eventHandlers {
		klog.V(4).Infof("Calling handler.OnNetDefDelete")
		c.eventHandlers[i].OnNetDefDelete(netdef)
	}
}

// NetDefInfo contains information about NetworkAttachmentDefinition.
type NetDefInfo struct {
	Netdef     *netdefv1.NetworkAttachmentDefinition
	PluginType string
}

// Name returns NetworkAttachmentDefinition name
func (info *NetDefInfo) Name() string {
	return info.Netdef.ObjectMeta.Name
}

// NetDefMap maps NetworkAttachmentDefinition namespaced name to NetDefInfo
type NetDefMap map[types.NamespacedName]NetDefInfo

// Update updates NetDefMap base on the given changes
func (n *NetDefMap) Update(changes *NetDefChangeTracker) {
	if n != nil {
		n.apply(changes)
	}
}

// apply applies changes to NetDefChangeTracker
func (n *NetDefMap) apply(changes *NetDefChangeTracker) {
	if n == nil || changes == nil {
		return
	}

	changes.lock.Lock()
	defer changes.lock.Unlock()
	for _, change := range changes.items {
		n.unmerge(change.previous)
		n.merge(change.current)
	}
	// clear changes after applying them to ServiceMap.
	changes.items = make(map[types.NamespacedName]*netdefChange)
}

// merge merges changes into NetDefMap
func (n *NetDefMap) merge(other NetDefMap) {
	if n == nil {
		return
	}

	for netDefName, info := range other {
		(*n)[netDefName] = info
	}
}

// unmerge deletes entries in other from NetDefMap
func (n *NetDefMap) unmerge(other NetDefMap) {
	if n == nil {
		return
	}

	for netDefName := range other {
		delete(*n, netDefName)
	}
}

// netdefChange represents a change in NetworkAttachmentDefinition represented via NetDefMap
type netdefChange struct {
	previous NetDefMap
	current  NetDefMap
}

// NetDefChangeTracker carries state about uncommitted changes to an arbitrary number of
// NetworkAttachmentDefinition keyed by their namespace and name
type NetDefChangeTracker struct {
	// lock protects items.
	lock sync.Mutex
	// items maps a service to its netdefChange.
	items     map[types.NamespacedName]*netdefChange
	netdefMap NetDefMap
}

// String returns string representation of the changes currently held in NetDefChangeTracker
func (ndt *NetDefChangeTracker) String() string {
	return fmt.Sprintf("netdefChange: %v", ndt.items)
}

// GetPluginType returns the CNI plugin name for the given (secondary) network represented by its namespaced name
func (ndt *NetDefChangeTracker) GetPluginType(name types.NamespacedName) string {
	ndt.netdefMap.Update(ndt)
	if cur, ok := ndt.netdefMap[name]; ok {
		return cur.PluginType
	}
	return ""
}

// newNetDefInfo creates a new instance of NetDefInfo
func (ndt *NetDefChangeTracker) newNetDefInfo(netdef *netdefv1.NetworkAttachmentDefinition) (*NetDefInfo, error) {
	confBytes, err := netdefutils.GetCNIConfig(netdef, "/etc/cni/multus/net.d")
	if err != nil {
		return nil, err
	}

	netconfList := &cnitypes.NetConfList{}
	if err := json.Unmarshal(confBytes, netconfList); err != nil {
		return nil, err
	}

	var info *NetDefInfo
	if len(netconfList.Plugins) == 0 {
		netconf := &cnitypes.NetConf{}
		if err := json.Unmarshal(confBytes, netconf); err != nil {
			return nil, err
		}

		info = &NetDefInfo{
			Netdef:     netdef,
			PluginType: netconf.Type,
		}
	} else {
		info = &NetDefInfo{
			Netdef:     netdef,
			PluginType: netconfList.Plugins[0].Type,
		}
	}
	return info, nil
}

// netDefToNetDefMap creates NetDefMap from NetworkAttachmentDefinition.
// Note(adrianc): it is basically a map with single entry.
func (ndt *NetDefChangeTracker) netDefToNetDefMap(netdef *netdefv1.NetworkAttachmentDefinition) NetDefMap {
	if netdef == nil {
		return nil
	}
	netdefMap := make(NetDefMap)
	netdefInfo, err := ndt.newNetDefInfo(netdef)
	if err != nil {
		klog.Errorf("err: %v\n", err)
		return nil
	}
	// TODO: need to revisit (why we need map?, just netdefInfo might be ok?)
	netdefMap[types.NamespacedName{Namespace: netdef.Namespace, Name: netdef.Name}] = *netdefInfo
	return netdefMap
}

// Update handles an update of a given NetworkAttachmentDefinition
func (ndt *NetDefChangeTracker) Update(previous, current *netdefv1.NetworkAttachmentDefinition) bool {
	netdef := current

	if netdef == nil {
		netdef = previous
	}
	if netdef == nil {
		return false
	}

	namespacedName := types.NamespacedName{Namespace: netdef.Namespace, Name: netdef.Name}

	ndt.lock.Lock()
	defer ndt.lock.Unlock()

	change, exists := ndt.items[namespacedName]
	if !exists {
		change = &netdefChange{}
		prevNetDefMap := ndt.netDefToNetDefMap(previous)
		change.previous = prevNetDefMap
		ndt.items[namespacedName] = change
	}

	curNetDefMap := ndt.netDefToNetDefMap(current)
	change.current = curNetDefMap
	if reflect.DeepEqual(change.previous, change.current) {
		delete(ndt.items, namespacedName)
	}

	return true
}

// NewNetDefChangeTracker creates a new instance of NetDefChangeTracker
func NewNetDefChangeTracker() *NetDefChangeTracker {
	return &NetDefChangeTracker{
		items:     make(map[types.NamespacedName]*netdefChange),
		netdefMap: make(NetDefMap),
	}
}
