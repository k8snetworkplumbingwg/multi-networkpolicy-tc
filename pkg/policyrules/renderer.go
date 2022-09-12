package policyrules

import (
	"fmt"
	"net"
	"strconv"

	multiv1beta1 "github.com/k8snetworkplumbingwg/multi-networkpolicy/pkg/apis/k8s.cni.cncf.io/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"

	"github.com/Mellanox/multi-networkpolicy-tc/pkg/controllers"
	multiutils "github.com/Mellanox/multi-networkpolicy-tc/pkg/utils"
)

// Renderer is an interface used to render PolicyRuleSet for a Pod Network
type Renderer interface {
	// RenderEgress renders PolicyRuleSet for Egress Kubernetes multinetwork policy.
	// target - is the target pod for which PolicyRuleSets are generated
	// currentPolicies - is the current state of MultiNetworkPolicies in the cluster
	// currentPods - is the current state of Pods in the cluster
	// currentNamespaces - is the current state of Namespaces in the cluster
	RenderEgress(target *controllers.PodInfo,
		currentPolicies controllers.PolicyMap,
		currentPods controllers.PodMap,
		currentNamespaces controllers.NamespaceMap) ([]PolicyRuleSet, error)
	// RenderIngress renders PolicyRuleSet for Ingress Kubernetes multinetwork policy
	// target - is the target pod for which PolicyRuleSets are generated
	// currentPolicies - is the current state of MultiNetworkPolicies in the cluster
	// currentPods - is the current state of Pods in the cluster
	// currentNamespaces - is the current state of Namespaces in the cluster
	RenderIngress(target *controllers.PodInfo,
		currentPolicies controllers.PolicyMap,
		currentPods controllers.PodMap,
		currentNamespaces controllers.NamespaceMap) ([]PolicyRuleSet, error)
}

// RendererImpl implements Renderer Interface
type RendererImpl struct {
	log klog.Logger
}

// NewRendererImpl creates a new instance of Renderer implementation
func NewRendererImpl(log klog.Logger) *RendererImpl {
	return &RendererImpl{log: log}
}

// RenderEgress implements Renderer Interface
func (r *RendererImpl) RenderEgress(target *controllers.PodInfo,
	currentPolicies controllers.PolicyMap,
	currentPods controllers.PodMap,
	currentNamespaces controllers.NamespaceMap) ([]PolicyRuleSet, error) {
	r.log.V(5).Info("Rendering Egress")
	policyRulesMap := make(map[string]PolicyRuleSet)

	podNamespacedName := types.NamespacedName{
		Namespace: target.Namespace,
		Name:      target.Name,
	}

	for _, policy := range currentPolicies {
		policyNamespacedName := types.NamespacedName{
			Namespace: policy.Policy.Namespace,
			Name:      policy.Policy.Name,
		}
		// check if policy applies for pod
		match, err := target.PolicyAppliesForPod(policy.Policy)
		if err != nil {
			r.log.Error(err, "failed to check if policy applies for pod. skipping",
				"policy", policyNamespacedName)
			continue
		}
		if !match {
			r.log.V(8).Info("policy does not apply for pod, skipping",
				"pod", podNamespacedName, "policy", policyNamespacedName)
			continue
		}
		r.log.V(8).Info("policy match for pod.",
			"policy-name", policyNamespacedName, "pod-name", podNamespacedName)

		// check if policy applies for interface
		for _, ifc := range target.Interfaces {
			if policy.AppliesForNetwork(ifc.NetattachName) {
				r.log.V(8).Info("policy match pod interface. rendering egress for policy",
					"pod-interface", ifc.InterfaceName, "network-name", ifc.NetattachName)
				// render rules for interface
				ifcRuleSet := r.renderEgressForInterface(ifc, policy, currentPods, currentNamespaces)
				existingRuleSetForIfc, ok := policyRulesMap[ifcRuleSet.IfcInfo.GetUID()]
				if ok {
					existingRuleSetForIfc.Rules = append(existingRuleSetForIfc.Rules, ifcRuleSet.Rules...)
					policyRulesMap[ifcRuleSet.IfcInfo.GetUID()] = existingRuleSetForIfc
				} else {
					policyRulesMap[ifcRuleSet.IfcInfo.GetUID()] = ifcRuleSet
				}
			} else {
				r.log.V(8).Info("policy does not match pod interface. skipping",
					"pod-interface", ifc.InterfaceName, "network-name", ifc.NetattachName)
			}
		}
	}

	// iterate over target interfaces and append empty rule set if no policy applied
	for _, ifc := range target.Interfaces {
		emptyPolicyRuleSet := PolicyRuleSet{
			IfcInfo: InterfaceInfo{
				Network:       ifc.NetattachName,
				InterfaceName: ifc.InterfaceName,
				IPs:           multiutils.IPsFromStrings(ifc.IPs),
				DeviceID:      ifc.DeviceID,
			},
			Type:  PolicyTypeEgress,
			Rules: nil,
		}
		_, ok := policyRulesMap[emptyPolicyRuleSet.IfcInfo.GetUID()]
		if !ok {
			policyRulesMap[emptyPolicyRuleSet.IfcInfo.GetUID()] = emptyPolicyRuleSet
		}
	}

	// append rule sets and return
	policyRules := make([]PolicyRuleSet, 0, len(policyRulesMap))
	for _, ruleSet := range policyRulesMap {
		policyRules = append(policyRules, ruleSet)
	}

	return policyRules, nil
}

// renderEgressForInterface renders egress policyRuleSet for given interface and given policy
func (r *RendererImpl) renderEgressForInterface(targetInterface controllers.InterfaceInfo,
	policy controllers.PolicyInfo,
	currentPods controllers.PodMap,
	currentNamespaces controllers.NamespaceMap) PolicyRuleSet {
	policyRuleSet := PolicyRuleSet{
		IfcInfo: InterfaceInfo{
			Network:       targetInterface.NetattachName,
			InterfaceName: targetInterface.InterfaceName,
			IPs:           multiutils.IPsFromStrings(targetInterface.IPs),
			DeviceID:      targetInterface.DeviceID,
		},
		Type:  PolicyTypeEgress,
		Rules: []Rule{},
	}

	// iterate over to fields
	for _, egressPolicyRule := range policy.Policy.Spec.Egress {
		ports := r.getPorts(egressPolicyRule.Ports)
		for _, peer := range egressPolicyRule.To {
			// Note(adrianc): an all nil MultiNetworkPolicyPeer is skipped as it assumes to be invalid
			// Note(adrianc): this generated a Rule per peer, this can be made more compact by consolidating all IPs
			// per group of ports taking into account a separate rule is needed for all IPBlock except field.
			if peer.IPBlock != nil {
				// handle IPBlock
				rules := r.renderRulesWithIPBlock(peer.IPBlock, ports)
				if len(rules) > 0 {
					policyRuleSet.Rules = append(policyRuleSet.Rules, rules...)
				}
			} else if peer.PodSelector != nil || peer.NamespaceSelector != nil {
				// handle pod/ns selectors
				rules := r.renderRulesWithSelectors(peer.PodSelector, peer.NamespaceSelector, ports, currentPods,
					currentNamespaces, targetInterface.NetattachName, policy.Namespace())
				if len(rules) > 0 {
					policyRuleSet.Rules = append(policyRuleSet.Rules, rules...)
				}
			}
		}

		// Note(adrianc): Handle special cases.
		//  1. len(egressPolicyRule.To) == 0 && len(egressPolicyRule.Ports) == 0 - allow traffic to all IPs
		//  2. len(egressPolicyRule.To) == 0 &&  len(egressPolicyRule.Ports) > 0 - allow traffic on these ports to all IPs
		if len(egressPolicyRule.To) == 0 {
			policyRuleSet.Rules = append(policyRuleSet.Rules, Rule{
				Ports:  ports,
				Action: PolicyActionPass,
			})
		}
	}
	return policyRuleSet
}

// renderRulesWithSelectors renders rules for pod/ns Peers
func (r *RendererImpl) renderRulesWithSelectors(podSel *metav1.LabelSelector,
	nsSel *metav1.LabelSelector,
	ports []Port,
	currentPods controllers.PodMap,
	currentNamespaces controllers.NamespaceMap,
	networkName string,
	policyNamespace string) []Rule {
	rules := []Rule{}

	if podSel == nil && nsSel == nil {
		// invalid input
		return rules
	}

	// select all pods matching label
	var matchingPods []controllers.PodInfo

	podSelectorMap, err := metav1.LabelSelectorAsMap(podSel)
	if err != nil {
		r.log.Error(err, "failed to convert pod selector to map")
		return rules
	}
	podLabelSelector := labels.Set(podSelectorMap).AsSelectorPreValidated()
	if podLabelSelector.Empty() {
		// matchall selector, so just take the full current list of podsInfo objects
		matchingPods, _ = currentPods.List()
	} else {
		// filter pods according to pod selector
		currentPodsList, _ := currentPods.List()
		for _, podInfo := range currentPodsList {
			// we can run this in parallel if needed
			if podLabelSelector.Matches(labels.Set(podInfo.Labels)) {
				matchingPods = append(matchingPods, podInfo)
			}
		}
	}

	// filter according to ns
	var matchingPodsAndNs []controllers.PodInfo
	if nsSel == nil {
		// filter pods according to policy namespace
		for _, podInfo := range matchingPods {
			if podInfo.Namespace == policyNamespace {
				matchingPodsAndNs = append(matchingPodsAndNs, podInfo)
			}
		}
	} else {
		// filter pods according to namespace that matches selector. empty selector matches all namespaces
		nsSelectorMap, err := metav1.LabelSelectorAsMap(nsSel)
		if err != nil {
			r.log.Error(err, "failed to convert namespace selector to map")
			return rules
		}
		nsLabelSelector := labels.Set(nsSelectorMap).AsSelectorPreValidated()

		if nsLabelSelector.Empty() {
			// matchall selector, so just take matchingPods
			matchingPodsAndNs = matchingPods
		} else {
			// we need to filter according to NS labels
			for _, podInfo := range matchingPods {
				nsInfo, err := currentNamespaces.GetNamespaceInfo(podInfo.Namespace)
				if err != nil {
					r.log.Error(err, "failed to get namespace from map", "ns", podInfo.Namespace)
					continue
				}
				if nsLabelSelector.Matches(labels.Set(nsInfo.Labels)) {
					matchingPodsAndNs = append(matchingPodsAndNs, podInfo)
				}
			}
		}
	}

	// 	collect IPs for network
	var ipCidrs []*net.IPNet
	for _, podInfo := range matchingPodsAndNs {
		for _, ifc := range podInfo.Interfaces {
			if ifc.NetattachName == networkName {
				ips := multiutils.IPsFromStrings(ifc.IPs)
				for _, ip := range ips {
					if ip != nil {
						var mask net.IPMask
						if multiutils.IsIPv4(ip) {
							mask = net.CIDRMask(net.IPv4len<<3, net.IPv4len<<3)
						} else {
							mask = net.CIDRMask(net.IPv6len<<3, net.IPv6len<<3)
						}
						ipCidrs = append(ipCidrs, &net.IPNet{IP: ip, Mask: mask})
					} else {
						r.log.Error(fmt.Errorf("failed to parse IPs for pod interface"), "", "ips", ifc.IPs)
						continue
					}
				}
			}
		}
	}
	// add Rule with these IPs
	if len(ipCidrs) > 0 || len(ports) > 0 {
		rules = append(rules, Rule{
			IPCidrs: ipCidrs,
			Ports:   ports,
			Action:  PolicyActionPass,
		})
	}
	return rules
}

// renderRulesWithIPBlock renders Rules for IPBlock peer with CIDR and Except
func (r *RendererImpl) renderRulesWithIPBlock(ipBlock *multiv1beta1.IPBlock, ports []Port) []Rule {
	var rules []Rule

	// handle cidr
	_, ipnet, err := net.ParseCIDR(ipBlock.CIDR)
	if err != nil {
		r.log.Error(err, "failed to parse ipBlock CIDR", "CIDR", ipBlock.CIDR)
		return []Rule{}
	}
	rules = append(rules, Rule{
		IPCidrs: []*net.IPNet{ipnet},
		Ports:   ports,
		Action:  PolicyActionPass,
	})

	// hanlde Except field
	exceptIPs := []*net.IPNet{}
	for _, exceptIPCidr := range ipBlock.Except {
		_, ipnet, err := net.ParseCIDR(exceptIPCidr)
		if err != nil {
			r.log.Error(err, "failed to parse ipBlock except CIDR, skipping", "CIDR", exceptIPCidr)
			continue
		}
		exceptIPs = append(exceptIPs, ipnet)
	}
	if len(exceptIPs) > 0 {
		rules = append(rules, Rule{
			IPCidrs: exceptIPs,
			Ports:   ports,
			Action:  PolicyActionDrop,
		})
	}
	return rules
}

// getPorts parses []MutliNetworkPolicyPort and returns []Port
func (r *RendererImpl) getPorts(ports []multiv1beta1.MultiNetworkPolicyPort) []Port {
	policyPorts := make([]Port, 0, len(ports))
	for _, p := range ports {
		policyPort := Port{}
		// handle port number
		portAsUint, err := strconv.ParseUint(p.Port.String(), 0, 16)
		if err != nil {
			r.log.Error(err, "Failed to convert port to unit", "port", p.Port.String())
			continue // move to next port
		}
		policyPort.Number = uint16(portAsUint)

		// hanlde protocol
		policyPort.Protocol = ProtocolTCP
		if p.Protocol != nil {
			switch *p.Protocol {
			case corev1.ProtocolTCP:
				break
			case corev1.ProtocolUDP:
				policyPort.Protocol = ProtocolUDP
			default:
				r.log.Error(fmt.Errorf("unsupported protocol"), "", "protocol", p.Protocol)
				continue // move to next port
			}
		}
		policyPorts = append(policyPorts, policyPort)
	}
	return policyPorts
}

// RenderIngress implements Renderer Interface
func (r *RendererImpl) RenderIngress(target *controllers.PodInfo, currentPolicies controllers.PolicyMap,
	currentPods controllers.PodMap, currentNamespaces controllers.NamespaceMap) ([]PolicyRuleSet, error) {
	// TODO implement me
	klog.Infof("RenderIngress() not Implemented")
	return []PolicyRuleSet{}, fmt.Errorf("not implemented")
}
