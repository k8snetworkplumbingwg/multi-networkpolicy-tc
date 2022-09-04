package testutil

import (
	"fmt"

	multiv1beta1 "github.com/k8snetworkplumbingwg/multi-networkpolicy/pkg/apis/k8s.cni.cncf.io/v1beta1"
	netdefv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewNamespace(name string, labels map[string]string) *v1.Namespace {
	return &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}
}

func NewNetDef(namespace, name, cniConfig string) *netdefv1.NetworkAttachmentDefinition {
	return &netdefv1.NetworkAttachmentDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: netdefv1.NetworkAttachmentDefinitionSpec{
			Config: cniConfig,
		},
	}
}

func NewCNIConfig(cniName, cniType string) string {
	cniConfigTemp := `
	{
		"name": "%s",
		"type": "%s"
	}`
	return fmt.Sprintf(cniConfigTemp, cniName, cniType)
}

func NewCNIConfigList(cniName, cniType string) string {
	cniConfigTemp := `
	{
		"name": "%s",
		"plugins": [ 
			{
				"type": "%s"
			}]
	}`
	return fmt.Sprintf(cniConfigTemp, cniName, cniType)
}

func NewNetworkPolicy(namespace, name string) *multiv1beta1.MultiNetworkPolicy {
	return &multiv1beta1.MultiNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
}

func NewFakePodWithNetAnnotation(namespace, name, networks, status string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			UID:       "testUID",
			Annotations: map[string]string{
				"k8s.v1.cni.cncf.io/networks": networks,
				netdefv1.NetworkStatusAnnot:   status,
			},
		},
		Spec: v1.PodSpec{
			NodeName: "nodeName",
			Containers: []v1.Container{
				{Name: "ctr1", Image: "image"},
			},
		},
		Status: v1.PodStatus{
			Phase: v1.PodRunning,
		},
	}
}

func NewFakeNetworkStatus(netns, netname string) string {
	baseStr := `
	[
		{
            "name": "",
            "interface": "eth0",
            "ips": [
                "10.244.1.4"
            ],
            "mac": "aa:e1:20:71:15:01",
            "default": true,
            "dns": {}
        },{
            "name": "%s/%s",
            "interface": "net1",
            "ips": [
                "10.1.1.101"
            ],
            "mac": "42:90:65:12:3e:bf",
            "dns": {},
			"device-info": {
				"type": "pci",
				"version": "1.0.0",
				"pci": {
					"pci-address": "0000:03:00.2"
				}
			}
		},{
			"name": "some-other-network",
			"interface": "net2",
			"ips": [
           		"20.1.1.101"
			],
			"mac": "42:90:65:12:3e:bf",
			"dns": {},
			"device-info": {
				"type": "pci",
				"version": "1.0.0",
				"pci": {
					"pci-address": "0000:03:00.3"
				}
			}
        }
]
`
	return fmt.Sprintf(baseStr, netns, netname)
}

func NewFakePod(namespace, name string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			UID:       "testUID",
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{Name: "ctr1", Image: "image"},
			},
		},
		Status: v1.PodStatus{
			Phase: v1.PodRunning,
		},
	}
}
