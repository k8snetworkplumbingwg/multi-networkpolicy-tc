package testutil

import (
	multiv1beta1 "github.com/k8snetworkplumbingwg/multi-networkpolicy/pkg/apis/k8s.cni.cncf.io/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	TargetNamespace = "target"
	SourceNamespace = "source"
)

var (
	PolicyDefaultAllow = multiv1beta1.MultiNetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MultiNetworkPolicy",
			APIVersion: "k8s.cni.cncf.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ipblock-policy-allow",
			Namespace: TargetNamespace,
		},
		Spec: multiv1beta1.MultiNetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []multiv1beta1.MultiPolicyType{multiv1beta1.PolicyTypeEgress},
			Ingress:     nil,
			Egress: []multiv1beta1.MultiNetworkPolicyEgressRule{
				{
					Ports: nil,
					To:    nil,
				},
			},
		},
	}

	PolicyDefaultDeny = multiv1beta1.MultiNetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MultiNetworkPolicy",
			APIVersion: "k8s.cni.cncf.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ipblock-policy-allow",
			Namespace: TargetNamespace,
		},
		Spec: multiv1beta1.MultiNetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []multiv1beta1.MultiPolicyType{multiv1beta1.PolicyTypeEgress},
			Ingress:     nil,
			Egress:      nil,
		},
	}

	PolicyIPBlockNoPorts = multiv1beta1.MultiNetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MultiNetworkPolicy",
			APIVersion: "k8s.cni.cncf.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ipblock-policy",
			Namespace: TargetNamespace,
		},
		Spec: multiv1beta1.MultiNetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "target"},
			},
			PolicyTypes: []multiv1beta1.MultiPolicyType{multiv1beta1.PolicyTypeEgress},
			Ingress:     nil,
			Egress: []multiv1beta1.MultiNetworkPolicyEgressRule{
				{
					Ports: nil,
					To: []multiv1beta1.MultiNetworkPolicyPeer{
						{
							IPBlock: &multiv1beta1.IPBlock{
								CIDR:   "10.17.0.0/16",
								Except: []string{"10.17.0.0/24"},
							},
						},
					},
				},
			},
		},
	}

	PolicyIPBlockWithPorts = multiv1beta1.MultiNetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MultiNetworkPolicy",
			APIVersion: "k8s.cni.cncf.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ipblock-policy",
			Namespace: TargetNamespace,
		},
		Spec: multiv1beta1.MultiNetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []multiv1beta1.MultiPolicyType{multiv1beta1.PolicyTypeEgress},
			Ingress:     nil,
			Egress: []multiv1beta1.MultiNetworkPolicyEgressRule{
				{
					Ports: []multiv1beta1.MultiNetworkPolicyPort{
						{
							Protocol: ToPtr(v1.ProtocolTCP),
							Port:     ToPtr(intstr.FromInt(6666)),
						},
						{
							Protocol: ToPtr(v1.ProtocolUDP),
							Port:     ToPtr(intstr.FromInt(7777)),
						},
						{
							Port: ToPtr(intstr.FromInt(8888)),
						},
					},
					To: []multiv1beta1.MultiNetworkPolicyPeer{
						{
							IPBlock: &multiv1beta1.IPBlock{
								CIDR:   "10.17.0.0/16",
								Except: []string{"10.17.0.0/24"},
							},
						},
					},
				},
			},
		},
	}

	PolicyIPBlockWithMultipeRules = multiv1beta1.MultiNetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MultiNetworkPolicy",
			APIVersion: "k8s.cni.cncf.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ipblock-policy",
			Namespace: TargetNamespace,
		},
		Spec: multiv1beta1.MultiNetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "target"},
			},
			PolicyTypes: []multiv1beta1.MultiPolicyType{multiv1beta1.PolicyTypeEgress},
			Ingress:     nil,
			Egress: []multiv1beta1.MultiNetworkPolicyEgressRule{
				{
					Ports: nil,
					To: []multiv1beta1.MultiNetworkPolicyPeer{
						{
							IPBlock: &multiv1beta1.IPBlock{
								CIDR:   "10.17.0.0/16",
								Except: []string{"10.17.0.0/24", "10.17.1.0/24"},
							},
						},
					},
				},
				{
					Ports: []multiv1beta1.MultiNetworkPolicyPort{
						{
							Protocol: ToPtr(v1.ProtocolTCP),
							Port:     ToPtr(intstr.FromInt(6666)),
						},
					},
					To: []multiv1beta1.MultiNetworkPolicyPeer{
						{
							IPBlock: &multiv1beta1.IPBlock{
								CIDR:   "20.17.0.0/16",
								Except: []string{"20.17.0.0/24", "20.17.1.0/24"},
							},
						},
					},
				},
			},
		},
	}

	PolicyIPBlockWithMultipePeers = multiv1beta1.MultiNetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MultiNetworkPolicy",
			APIVersion: "k8s.cni.cncf.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ipblock-policy",
			Namespace: TargetNamespace,
		},
		Spec: multiv1beta1.MultiNetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "target"},
			},
			PolicyTypes: []multiv1beta1.MultiPolicyType{multiv1beta1.PolicyTypeEgress},
			Ingress:     nil,
			Egress: []multiv1beta1.MultiNetworkPolicyEgressRule{
				{
					Ports: []multiv1beta1.MultiNetworkPolicyPort{
						{
							Protocol: ToPtr(v1.ProtocolTCP),
							Port:     ToPtr(intstr.FromInt(6666)),
						},
					},
					To: []multiv1beta1.MultiNetworkPolicyPeer{
						{
							IPBlock: &multiv1beta1.IPBlock{
								CIDR:   "10.17.0.0/16",
								Except: []string{"10.17.0.0/24"},
							},
						},
						{
							IPBlock: &multiv1beta1.IPBlock{
								CIDR:   "20.17.0.0/16",
								Except: []string{"20.17.0.0/24"},
							},
						},
					},
				},
			},
		},
	}

	PolicySelectorAsSourceNoPorts = multiv1beta1.MultiNetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MultiNetworkPolicy",
			APIVersion: "k8s.cni.cncf.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "selector-policy",
			Namespace: TargetNamespace,
		},
		Spec: multiv1beta1.MultiNetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "target"},
			},
			PolicyTypes: []multiv1beta1.MultiPolicyType{multiv1beta1.PolicyTypeEgress},
			Ingress:     nil,
			Egress: []multiv1beta1.MultiNetworkPolicyEgressRule{
				{
					Ports: nil,
					To: []multiv1beta1.MultiNetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"app": "source"},
							},
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"kubernetes.io/metadata.name": SourceNamespace},
							},
						},
					},
				},
			},
		},
	}

	PolicySelectorAsSourceWithPorts = multiv1beta1.MultiNetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MultiNetworkPolicy",
			APIVersion: "k8s.cni.cncf.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "selector-policy",
			Namespace: TargetNamespace,
		},
		Spec: multiv1beta1.MultiNetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "target"},
			},
			PolicyTypes: []multiv1beta1.MultiPolicyType{multiv1beta1.PolicyTypeEgress},
			Ingress:     nil,
			Egress: []multiv1beta1.MultiNetworkPolicyEgressRule{
				{
					Ports: []multiv1beta1.MultiNetworkPolicyPort{
						{
							Protocol: ToPtr(v1.ProtocolTCP),
							Port:     ToPtr(intstr.FromInt(6666)),
						},
						{
							Protocol: ToPtr(v1.ProtocolUDP),
							Port:     ToPtr(intstr.FromInt(7777)),
						},
						{
							Port: ToPtr(intstr.FromInt(8888)),
						},
					},
					To: []multiv1beta1.MultiNetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{},
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"kubernetes.io/metadata.name": SourceNamespace},
							},
						},
					},
				},
			},
		},
	}

	PolicySelectorAsSourceMultipleRules = multiv1beta1.MultiNetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MultiNetworkPolicy",
			APIVersion: "k8s.cni.cncf.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "selector-policy",
			Namespace: TargetNamespace,
		},
		Spec: multiv1beta1.MultiNetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []multiv1beta1.MultiPolicyType{multiv1beta1.PolicyTypeEgress},
			Ingress:     nil,
			Egress: []multiv1beta1.MultiNetworkPolicyEgressRule{
				{
					Ports: nil,
					To: []multiv1beta1.MultiNetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"app": "source-1"},
							},
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"kubernetes.io/metadata.name": SourceNamespace},
							},
						},
					},
				},
				{
					Ports: []multiv1beta1.MultiNetworkPolicyPort{
						{
							Protocol: ToPtr(v1.ProtocolTCP),
							Port:     ToPtr(intstr.FromInt(6666)),
						},
					},
					To: []multiv1beta1.MultiNetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"app": "source-2"},
							},
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"kubernetes.io/metadata.name": SourceNamespace},
							},
						},
					},
				},
			},
		},
	}

	PolicySelectorAsSourceMultiplePeers = multiv1beta1.MultiNetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MultiNetworkPolicy",
			APIVersion: "k8s.cni.cncf.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "selector-policy",
			Namespace: TargetNamespace,
		},
		Spec: multiv1beta1.MultiNetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "target"},
			},
			PolicyTypes: []multiv1beta1.MultiPolicyType{multiv1beta1.PolicyTypeEgress},
			Ingress:     nil,
			Egress: []multiv1beta1.MultiNetworkPolicyEgressRule{
				{
					Ports: []multiv1beta1.MultiNetworkPolicyPort{
						{
							Protocol: ToPtr(v1.ProtocolTCP),
							Port:     ToPtr(intstr.FromInt(6666)),
						},
					},
					To: []multiv1beta1.MultiNetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"app": "source-1"},
							},
							NamespaceSelector: &metav1.LabelSelector{},
						},
						{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"app": "source-2"},
							},
							NamespaceSelector: &metav1.LabelSelector{},
						},
					},
				},
			},
		},
	}
)
