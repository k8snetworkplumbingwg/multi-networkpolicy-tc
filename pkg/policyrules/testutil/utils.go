package testutil

import (
	"strings"

	"github.com/google/uuid"
	multiv1beta1 "github.com/k8snetworkplumbingwg/multi-networkpolicy/pkg/apis/k8s.cni.cncf.io/v1beta1"

	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/controllers"
)

func ToPtr[T any](v T) *T {
	return &v
}

// Builders

// PodInfoBuiler is a PodInfo Builder for testing purposes
type PodInfoBuiler struct {
	pi *controllers.PodInfo
}

func NewPodInfoBuiler() *PodInfoBuiler {
	return &PodInfoBuiler{pi: &controllers.PodInfo{}}
}

func (b *PodInfoBuiler) WithName(n string) *PodInfoBuiler {
	b.pi.Name = n
	return b
}

func (b *PodInfoBuiler) WithNamespace(ns string) *PodInfoBuiler {
	b.pi.Namespace = ns
	return b
}

// WithLabels accepts list of "<key>="<val>" formatted strings, overrides labels set in preceding call
func (b *PodInfoBuiler) WithLabels(kvs ...string) *PodInfoBuiler {
	b.pi.Labels = make(map[string]string)

	for i := range kvs {
		splitted := strings.Split(kvs[i], "=")
		b.pi.Labels[splitted[0]] = splitted[1]
	}
	return b
}

func (b *PodInfoBuiler) WithInterface(netAttachName string,
	deviceID string, interfaceName string, interfaceType string, ips []string) *PodInfoBuiler {
	ii := controllers.InterfaceInfo{
		NetattachName: netAttachName,
		DeviceID:      deviceID,
		InterfaceName: interfaceName,
		InterfaceType: interfaceType,
		IPs:           ips,
	}
	b.pi.Interfaces = append(b.pi.Interfaces, ii)
	return b
}

func (b *PodInfoBuiler) ResetInterfaces() *PodInfoBuiler {
	b.pi.Interfaces = nil
	return b
}

func (b *PodInfoBuiler) Build() *controllers.PodInfo {
	b.pi.UID = uuid.New().String()
	return b.pi
}

// NamespaceInfoBuilder is a NamespaceInfo Builder for testing purposes
type NamespaceInfoBuilder struct {
	ni *controllers.NamespaceInfo
}

func NewNamespaceInfoBuilder() *NamespaceInfoBuilder {
	return &NamespaceInfoBuilder{ni: &controllers.NamespaceInfo{}}
}

func (b *NamespaceInfoBuilder) WithName(n string) *NamespaceInfoBuilder {
	b.ni.Name = n
	return b
}

// WithLabels accepts list of "<key>="<val>" formatted strings, overrides labels set in preceding call
func (b *NamespaceInfoBuilder) WithLabels(kvs ...string) *NamespaceInfoBuilder {
	b.ni.Labels = make(map[string]string)

	for i := range kvs {
		splitted := strings.Split(kvs[i], "=")
		b.ni.Labels[splitted[0]] = splitted[1]
	}
	return b
}

func (b *NamespaceInfoBuilder) Build() *controllers.NamespaceInfo {
	return b.ni
}

// PolicyInfoBuilder is a PolicyInfo Builder for testing purposes
type PolicyInfoBuilder struct {
	pi *controllers.PolicyInfo
}

func NewPolicyInfoBuilder() *PolicyInfoBuilder {
	return &PolicyInfoBuilder{pi: &controllers.PolicyInfo{}}
}

func (b *PolicyInfoBuilder) WithNetworks(nets ...string) *PolicyInfoBuilder {
	b.pi.PolicyNetworks = append(b.pi.PolicyNetworks, nets...)
	return b
}

func (b *PolicyInfoBuilder) WithPolicy(p *multiv1beta1.MultiNetworkPolicy) *PolicyInfoBuilder {
	b.pi.Policy = p
	return b
}

func (b *PolicyInfoBuilder) Build() *controllers.PolicyInfo {
	return b.pi
}
