package policyrules_test

import (
	"fmt"
	"net"
	"reflect"

	multiv1beta1 "github.com/k8snetworkplumbingwg/multi-networkpolicy/pkg/apis/k8s.cni.cncf.io/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/controllers"
	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/policyrules"
	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/policyrules/testutil"
)

func checkInterfaceInfos(rules []policyrules.PolicyRuleSet, podInterfaceInfos []controllers.InterfaceInfo) {
	for _, rule := range rules {
		// find pod InterfaceInfo that matches rule InterfaceInfo
		found := false
		for _, podInterfaceInfo := range podInterfaceInfos {
			if podInterfaceInfo.InterfaceName == rule.IfcInfo.InterfaceName {
				found = true
				ExpectWithOffset(1, rule.IfcInfo.DeviceID).To(BeEquivalentTo(podInterfaceInfo.DeviceID))
				ExpectWithOffset(1, rule.IfcInfo.InterfaceName).To(BeEquivalentTo(podInterfaceInfo.InterfaceName))
				ExpectWithOffset(1, rule.IfcInfo.Network).To(BeEquivalentTo(podInterfaceInfo.NetattachName))
				// Ensure same IPs
				ExpectWithOffset(1, rule.IfcInfo.IPs).To(HaveLen(len(podInterfaceInfo.IPs)))
				for k := range rule.IfcInfo.IPs {
					ExpectWithOffset(1, rule.IfcInfo.IPs[k].String()).To(Equal(podInterfaceInfo.IPs[k]))
				}
			}
		}
		ExpectWithOffset(1, found).To(BeTrue())
	}
}

// ruleEqual checks that this and other Rules are equal
// limitation: it assumes IPCidrs and Ports contain no duplicate entries
func ruleEqual(this, other policyrules.Rule) bool {
	// Note(adrianc): we can probably do something more efficient here. (e.g use maps as sets)
	if this.Action != other.Action {
		return false
	}

	if len(this.IPCidrs) != len(other.IPCidrs) {
		return false
	}

	if len(this.Ports) != len(other.Ports) {
		return false
	}

	for _, ip := range this.IPCidrs {
		match := false
		for _, otherIP := range other.IPCidrs {
			if ip.String() == otherIP.String() {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}

	for _, port := range this.Ports {
		match := false
		for _, otherPort := range other.Ports {
			if reflect.DeepEqual(port, otherPort) {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}

	return true
}

func checkRules(actual, expected []policyrules.Rule) {
	ExpectWithOffset(1, actual).To(HaveLen(len(expected)))
	for _, actualRule := range actual {
		match := false
		for _, expectedRule := range expected {
			if ruleEqual(actualRule, expectedRule) {
				match = true
				break
			}
		}
		ExpectWithOffset(1, match).To(BeTrue(), fmt.Sprintf("actual: %+v, expected: %+v", actual, expected))
	}
}

var _ = Describe("Renderer tests", func() {
	var logger = klog.NewKlogr().WithName("policyrules-renderer-test")
	var renderer policyrules.Renderer
	var target *controllers.PodInfo
	var currentPolicies controllers.PolicyMap
	var currentPods controllers.PodMap
	var currentNamespaces controllers.NamespaceMap

	addPolicy := func(p *multiv1beta1.MultiNetworkPolicy, forNetworks ...string) {
		pInfo := testutil.NewPolicyInfoBuilder().WithPolicy(p).WithNetworks(forNetworks...).Build()
		currentPolicies[types.NamespacedName{
			Namespace: pInfo.Namespace(),
			Name:      pInfo.Name()}] = *pInfo
	}

	addPodInfo := func(pInfos ...*controllers.PodInfo) {
		for _, pi := range pInfos {
			currentPods[types.NamespacedName{Namespace: pi.Namespace, Name: pi.Name}] = *pi
		}
	}

	addNsByName := func(nsNames ...string) {
		for _, name := range nsNames {
			nsInfo := testutil.NewNamespaceInfoBuilder().
				WithName(name).
				WithLabels(fmt.Sprintf("kubernetes.io/metadata.name=%s", name)).
				Build()
			currentNamespaces[name] = *nsInfo
		}
	}

	BeforeEach(func() {
		renderer = policyrules.NewRendererImpl(logger)
		currentPolicies = make(controllers.PolicyMap)
		currentPods = make(controllers.PodMap)
		currentNamespaces = make(controllers.NamespaceMap)
	})

	Describe("RenderIngress", func() {
		It("returns not implemented", func() {
			rules, err := renderer.RenderIngress(target, currentPolicies, currentPods, currentNamespaces)
			Expect(err).To(HaveOccurred())
			Expect(rules).To(BeEmpty())
		})
	})

	Describe("RenderEgress", func() {
		BeforeEach(func() {
			target = testutil.NewPodInfoBuiler().
				WithName("target-pod").
				WithNamespace(testutil.TargetNamespace).
				WithInterface(
					"accel-net",
					"0000:03:00.4",
					"net1",
					"accelerated-bridge",
					[]string{"192.168.1.2"}).
				WithLabels("app=target").
				Build()
		})

		Describe("edge cases", func() {
			Context("no matching policy", func() {
				It("renders empty rule set it policy does not apply to pod", func() {
					target = testutil.NewPodInfoBuiler().
						WithName("target-pod").
						WithNamespace("default").
						WithInterface(
							"accel-net",
							"0000:03:00.4",
							"net1",
							"accelerated-bridge",
							[]string{"192.168.1.2"}).
						WithLabels("app=target").
						Build()
					addPolicy(&testutil.PolicyIPBlockNoPorts, "accel-net")

					ruleSets, err := renderer.RenderEgress(target, currentPolicies, currentPods, currentNamespaces)
					Expect(err).ToNot(HaveOccurred())
					By(fmt.Sprintf("got rule sets: %+v", ruleSets))

					Expect(ruleSets).To(HaveLen(1))
					checkInterfaceInfos(ruleSets, target.Interfaces)
					Expect(ruleSets[0].Type).To(Equal(policyrules.PolicyTypeEgress))
					Expect(ruleSets[0].Rules).To(BeEmpty())
				})
			})

			Context("default deny", func() {
				It("renders default drop rule", func() {
					addPolicy(&testutil.PolicyDefaultDeny, "accel-net")

					ruleSets, err := renderer.RenderEgress(target, currentPolicies, currentPods, currentNamespaces)
					Expect(err).ToNot(HaveOccurred())
					By(fmt.Sprintf("got rule sets: %+v", ruleSets))

					Expect(ruleSets).To(HaveLen(1))
					checkInterfaceInfos(ruleSets, target.Interfaces)
					Expect(ruleSets[0].Type).To(Equal(policyrules.PolicyTypeEgress))
					Expect(ruleSets[0].Rules).To(BeEmpty())
				})
			})

			Context("default allow", func() {
				It("renders default pass rule", func() {
					addPolicy(&testutil.PolicyDefaultAllow, "accel-net")

					ruleSets, err := renderer.RenderEgress(target, currentPolicies, currentPods, currentNamespaces)
					Expect(err).ToNot(HaveOccurred())
					By(fmt.Sprintf("got rule sets: %+v", ruleSets))

					Expect(ruleSets).To(HaveLen(1))
					checkInterfaceInfos(ruleSets, target.Interfaces)

					// Check rules
					expectedRules := []policyrules.Rule{
						{
							IPCidrs: nil,
							Ports:   []policyrules.Port{},
							Action:  policyrules.PolicyActionPass,
						},
					}
					Expect(ruleSets[0].Type).To(Equal(policyrules.PolicyTypeEgress))
					checkRules(ruleSets[0].Rules, expectedRules)
				})
			})
		})

		Describe("IPBlock single policy", func() {
			Context("without ports", func() {
				It("returns correct rules for single pod interface", func() {
					addPolicy(&testutil.PolicyIPBlockNoPorts, "accel-net")

					ruleSets, err := renderer.RenderEgress(target, currentPolicies, currentPods, currentNamespaces)
					Expect(err).ToNot(HaveOccurred())
					By(fmt.Sprintf("got rule sets: %+v", ruleSets))

					Expect(ruleSets).To(HaveLen(1))
					checkInterfaceInfos(ruleSets, target.Interfaces)

					// Check rules
					expectedPolicyRules := []policyrules.Rule{
						{
							IPCidrs: []*net.IPNet{
								{
									IP:   net.IP{10, 17, 0, 0},
									Mask: net.IPMask{255, 255, 0, 0},
								},
							},
							Ports:  []policyrules.Port{},
							Action: policyrules.PolicyActionPass,
						},
						{
							IPCidrs: []*net.IPNet{
								{
									IP:   net.IP{10, 17, 0, 0},
									Mask: net.IPMask{255, 255, 255, 0},
								},
							},
							Ports:  []policyrules.Port{},
							Action: policyrules.PolicyActionDrop,
						},
					}
					Expect(ruleSets[0].Type).To(Equal(policyrules.PolicyTypeEgress))
					checkRules(ruleSets[0].Rules, expectedPolicyRules)
				})
			})

			Context("with ports", func() {
				It("returns correct rules", func() {
					addPolicy(&testutil.PolicyIPBlockWithPorts, "accel-net")

					ruleSets, err := renderer.RenderEgress(target, currentPolicies, currentPods, currentNamespaces)
					Expect(err).ToNot(HaveOccurred())
					Expect(ruleSets).To(HaveLen(1))
					By(fmt.Sprintf("got rule sets: %+v", ruleSets))

					checkInterfaceInfos(ruleSets, target.Interfaces)

					// Check rules
					expectedPorts := []policyrules.Port{
						{
							Protocol: policyrules.ProtocolTCP,
							Number:   6666,
						},
						{
							Protocol: policyrules.ProtocolUDP,
							Number:   7777,
						},
						{
							Protocol: policyrules.ProtocolTCP,
							Number:   8888,
						},
					}
					expectedPolicyRules := []policyrules.Rule{
						{
							IPCidrs: []*net.IPNet{
								{
									IP:   net.IP{10, 17, 0, 0},
									Mask: net.IPMask{255, 255, 0, 0},
								},
							},
							Ports:  expectedPorts,
							Action: policyrules.PolicyActionPass,
						},
						{
							IPCidrs: []*net.IPNet{
								{
									IP:   net.IP{10, 17, 0, 0},
									Mask: net.IPMask{255, 255, 255, 0},
								},
							},
							Ports:  expectedPorts,
							Action: policyrules.PolicyActionDrop,
						},
					}

					Expect(ruleSets[0].Type).To(Equal(policyrules.PolicyTypeEgress))
					checkRules(ruleSets[0].Rules, expectedPolicyRules)
				})
			})

			Context("multiple rules", func() {
				It("returns expected rules", func() {
					addPolicy(&testutil.PolicyIPBlockWithMultipeRules, "accel-net")

					ruleSets, err := renderer.RenderEgress(target, currentPolicies, currentPods, currentNamespaces)
					Expect(err).ToNot(HaveOccurred())
					By(fmt.Sprintf("got rule sets: %+v", ruleSets))

					Expect(ruleSets).To(HaveLen(1))
					checkInterfaceInfos(ruleSets, target.Interfaces)

					// Check rules
					expectedPolicyRules := []policyrules.Rule{
						{
							IPCidrs: []*net.IPNet{
								{
									IP:   net.IP{10, 17, 0, 0},
									Mask: net.IPMask{255, 255, 0, 0},
								},
							},
							Ports:  []policyrules.Port{},
							Action: policyrules.PolicyActionPass,
						},
						{
							IPCidrs: []*net.IPNet{
								{
									IP:   net.IP{10, 17, 0, 0},
									Mask: net.IPMask{255, 255, 255, 0},
								},
								{
									IP:   net.IP{10, 17, 1, 0},
									Mask: net.IPMask{255, 255, 255, 0},
								},
							},
							Ports:  []policyrules.Port{},
							Action: policyrules.PolicyActionDrop,
						},
						{
							IPCidrs: []*net.IPNet{
								{
									IP:   net.IP{20, 17, 0, 0},
									Mask: net.IPMask{255, 255, 0, 0},
								},
							},
							Ports: []policyrules.Port{
								{
									Protocol: policyrules.ProtocolTCP,
									Number:   6666,
								},
							},
							Action: policyrules.PolicyActionPass,
						},
						{
							IPCidrs: []*net.IPNet{
								{
									IP:   net.IP{20, 17, 0, 0},
									Mask: net.IPMask{255, 255, 255, 0},
								},
								{
									IP:   net.IP{20, 17, 1, 0},
									Mask: net.IPMask{255, 255, 255, 0},
								},
							},
							Ports: []policyrules.Port{
								{
									Protocol: policyrules.ProtocolTCP,
									Number:   6666,
								},
							},
							Action: policyrules.PolicyActionDrop,
						},
					}

					Expect(ruleSets[0].Type).To(Equal(policyrules.PolicyTypeEgress))
					checkRules(ruleSets[0].Rules, expectedPolicyRules)
				})
			})

			Context("multiple peers", func() {
				It("returns expected rules", func() {
					addPolicy(&testutil.PolicyIPBlockWithMultipePeers, "accel-net")

					ruleSets, err := renderer.RenderEgress(target, currentPolicies, currentPods, currentNamespaces)
					Expect(err).ToNot(HaveOccurred())
					By(fmt.Sprintf("got rule sets: %+v", ruleSets))

					Expect(ruleSets).To(HaveLen(1))
					checkInterfaceInfos(ruleSets, target.Interfaces)

					// Check rules
					expectedPolicyRules := []policyrules.Rule{
						{
							IPCidrs: []*net.IPNet{
								{
									IP:   net.IP{10, 17, 0, 0},
									Mask: net.IPMask{255, 255, 0, 0},
								},
							},
							Ports: []policyrules.Port{
								{
									Protocol: policyrules.ProtocolTCP,
									Number:   6666,
								},
							},
							Action: policyrules.PolicyActionPass,
						},
						{
							IPCidrs: []*net.IPNet{
								{
									IP:   net.IP{10, 17, 0, 0},
									Mask: net.IPMask{255, 255, 255, 0},
								},
							},
							Ports: []policyrules.Port{
								{
									Protocol: policyrules.ProtocolTCP,
									Number:   6666,
								},
							},
							Action: policyrules.PolicyActionDrop,
						},
						{
							IPCidrs: []*net.IPNet{
								{
									IP:   net.IP{20, 17, 0, 0},
									Mask: net.IPMask{255, 255, 0, 0},
								},
							},
							Ports: []policyrules.Port{
								{
									Protocol: policyrules.ProtocolTCP,
									Number:   6666,
								},
							},
							Action: policyrules.PolicyActionPass,
						},
						{
							IPCidrs: []*net.IPNet{
								{
									IP:   net.IP{20, 17, 0, 0},
									Mask: net.IPMask{255, 255, 255, 0},
								},
							},
							Ports: []policyrules.Port{
								{
									Protocol: policyrules.ProtocolTCP,
									Number:   6666,
								},
							},
							Action: policyrules.PolicyActionDrop,
						},
					}

					Expect(ruleSets[0].Type).To(Equal(policyrules.PolicyTypeEgress))
					checkRules(ruleSets[0].Rules, expectedPolicyRules)
				})
			})
		})

		Describe("Selectors single policy", func() {
			Context("without ports", func() {
				It("returns correct rules for single pod interface", func() {
					addPolicy(&testutil.PolicySelectorAsSourceNoPorts, "accel-net")

					source1 := testutil.NewPodInfoBuiler().
						WithName("source-pod-1").
						WithNamespace(testutil.SourceNamespace).
						WithInterface(
							"accel-net",
							"0000:03:00.5",
							"net1",
							"accelerated-bridge",
							[]string{"192.168.1.3"}).
						WithLabels("app=source").
						Build()

					source2 := testutil.NewPodInfoBuiler().
						WithName("source-pod-2").
						WithNamespace(testutil.SourceNamespace).
						WithInterface(
							"accel-net",
							"0000:03:00.6",
							"net1",
							"accelerated-bridge",
							[]string{"192.168.1.4"}).
						WithLabels("app=not-a-source").
						Build()

					addPodInfo(source1, source2, target)
					addNsByName("target", "source")

					ruleSets, err := renderer.RenderEgress(target, currentPolicies, currentPods, currentNamespaces)
					Expect(err).ToNot(HaveOccurred())
					By(fmt.Sprintf("got rule sets: %+v", ruleSets))

					Expect(ruleSets).To(HaveLen(1))
					checkInterfaceInfos(ruleSets, target.Interfaces)

					// Check rules
					expectedPolicyRules := []policyrules.Rule{
						{
							IPCidrs: []*net.IPNet{
								{
									IP:   net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 192, 168, 1, 3},
									Mask: net.IPMask{255, 255, 255, 255},
								},
							},
							Ports:  []policyrules.Port{},
							Action: policyrules.PolicyActionPass,
						},
					}
					checkRules(ruleSets[0].Rules, expectedPolicyRules)
					Expect(ruleSets[0].Type).To(Equal(policyrules.PolicyTypeEgress))
				})
			})

			Context("with ports", func() {
				It("returns correct rules for single pod interface", func() {
					addPolicy(&testutil.PolicySelectorAsSourceWithPorts, "accel-net")

					source1 := testutil.NewPodInfoBuiler().
						WithName("source-pod-1").
						WithNamespace(testutil.SourceNamespace).
						WithInterface(
							"accel-net",
							"0000:03:00.5",
							"net1",
							"accelerated-bridge",
							[]string{"192.168.1.3"}).
						WithLabels("app=source").
						Build()

					source2 := testutil.NewPodInfoBuiler().
						WithName("source-pod-2").
						WithNamespace(testutil.SourceNamespace).
						WithInterface(
							"accel-net",
							"0000:03:00.6",
							"net1",
							"accelerated-bridge",
							[]string{"192.168.1.4"}).
						WithLabels("app=source").
						Build()

					addPodInfo(source1, source2, target)
					addNsByName("target", "source")

					ruleSets, err := renderer.RenderEgress(target, currentPolicies, currentPods, currentNamespaces)
					Expect(err).ToNot(HaveOccurred())
					By(fmt.Sprintf("got rule sets: %+v", ruleSets))

					Expect(ruleSets).To(HaveLen(1))
					checkInterfaceInfos(ruleSets, target.Interfaces)

					// Check rules
					expectedPolicyRules := []policyrules.Rule{
						{
							IPCidrs: []*net.IPNet{
								{
									IP:   net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 192, 168, 1, 3},
									Mask: net.IPMask{255, 255, 255, 255},
								},
								{
									IP:   net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 192, 168, 1, 4},
									Mask: net.IPMask{255, 255, 255, 255},
								},
							},
							Ports: []policyrules.Port{
								{
									Protocol: policyrules.ProtocolTCP,
									Number:   6666,
								},
								{
									Protocol: policyrules.ProtocolUDP,
									Number:   7777,
								},
								{
									Protocol: policyrules.ProtocolTCP,
									Number:   8888,
								},
							},
							Action: policyrules.PolicyActionPass,
						},
					}
					checkRules(ruleSets[0].Rules, expectedPolicyRules)
					Expect(ruleSets[0].Type).To(Equal(policyrules.PolicyTypeEgress))
				})
			})

			Context("multiple rules", func() {
				It("returns correct rules for single pod interface", func() {
					addPolicy(&testutil.PolicySelectorAsSourceMultipleRules, "accel-net")

					source1 := testutil.NewPodInfoBuiler().
						WithName("source-pod-1").
						WithNamespace(testutil.SourceNamespace).
						WithInterface(
							"accel-net",
							"0000:03:00.5",
							"net1",
							"accelerated-bridge",
							[]string{"192.168.1.3"}).
						WithLabels("app=source-1").
						Build()

					source2 := testutil.NewPodInfoBuiler().
						WithName("source-pod-2").
						WithNamespace(testutil.SourceNamespace).
						WithInterface(
							"accel-net",
							"0000:03:00.6",
							"net1",
							"accelerated-bridge",
							[]string{"192.168.1.4"}).
						WithLabels("app=source-2").
						Build()

					addPodInfo(source1, source2, target)
					addNsByName("target", "source")

					ruleSets, err := renderer.RenderEgress(target, currentPolicies, currentPods, currentNamespaces)
					Expect(err).ToNot(HaveOccurred())
					By(fmt.Sprintf("got rule sets: %+v", ruleSets))

					Expect(ruleSets).To(HaveLen(1))
					checkInterfaceInfos(ruleSets, target.Interfaces)

					// Check rules
					expectedPolicyRules := []policyrules.Rule{
						{
							IPCidrs: []*net.IPNet{
								{
									IP:   net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 192, 168, 1, 3},
									Mask: net.IPMask{255, 255, 255, 255},
								},
							},
							Ports:  []policyrules.Port{},
							Action: policyrules.PolicyActionPass,
						},
						{
							IPCidrs: []*net.IPNet{
								{
									IP:   net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 192, 168, 1, 4},
									Mask: net.IPMask{255, 255, 255, 255},
								},
							},
							Ports: []policyrules.Port{
								{
									Protocol: policyrules.ProtocolTCP,
									Number:   6666,
								},
							},
							Action: policyrules.PolicyActionPass,
						},
					}
					checkRules(ruleSets[0].Rules, expectedPolicyRules)
					Expect(ruleSets[0].Type).To(Equal(policyrules.PolicyTypeEgress))
				})
			})

			Context("multiple peers", func() {
				It("returns correct rules for single pod interface", func() {
					addPolicy(&testutil.PolicySelectorAsSourceMultiplePeers, "accel-net")

					source1 := testutil.NewPodInfoBuiler().
						WithName("source-pod-1").
						WithNamespace(testutil.SourceNamespace).
						WithInterface(
							"accel-net",
							"0000:03:00.5",
							"net1",
							"accelerated-bridge",
							[]string{"192.168.1.3"}).
						WithLabels("app=source-1").
						Build()

					source2 := testutil.NewPodInfoBuiler().
						WithName("source-pod-2").
						WithNamespace(testutil.SourceNamespace).
						WithInterface(
							"accel-net",
							"0000:03:00.6",
							"net1",
							"accelerated-bridge",
							[]string{"192.168.1.4"}).
						WithLabels("app=source-2").
						Build()

					addPodInfo(source1, source2, target)
					addNsByName("target", "source")

					ruleSets, err := renderer.RenderEgress(target, currentPolicies, currentPods, currentNamespaces)
					Expect(err).ToNot(HaveOccurred())
					By(fmt.Sprintf("got rule sets: %+v", ruleSets))

					Expect(ruleSets).To(HaveLen(1))
					checkInterfaceInfos(ruleSets, target.Interfaces)

					// Check rules
					expectedPolicyRules := []policyrules.Rule{
						{
							IPCidrs: []*net.IPNet{
								{
									IP:   net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 192, 168, 1, 3},
									Mask: net.IPMask{255, 255, 255, 255},
								},
							},
							Ports: []policyrules.Port{
								{
									Protocol: policyrules.ProtocolTCP,
									Number:   6666,
								},
							},
							Action: policyrules.PolicyActionPass,
						},
						{
							IPCidrs: []*net.IPNet{
								{
									IP:   net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 192, 168, 1, 4},
									Mask: net.IPMask{255, 255, 255, 255},
								},
							},
							Ports: []policyrules.Port{
								{
									Protocol: policyrules.ProtocolTCP,
									Number:   6666,
								},
							},
							Action: policyrules.PolicyActionPass,
						},
					}
					checkRules(ruleSets[0].Rules, expectedPolicyRules)
					Expect(ruleSets[0].Type).To(Equal(policyrules.PolicyTypeEgress))
				})
			})
		})

		Describe("Multiple", func() {
			Context("multiple policies on same interface", func() {
				It("returns correct rules", func() {
					addPolicy(&testutil.PolicySelectorAsSourceNoPorts, "accel-net")
					addPolicy(&testutil.PolicyIPBlockNoPorts, "accel-net")

					source := testutil.NewPodInfoBuiler().
						WithName("source-pod-1").
						WithNamespace(testutil.SourceNamespace).
						WithInterface(
							"accel-net",
							"0000:03:00.5",
							"net1",
							"accelerated-bridge",
							[]string{"192.168.1.3"}).
						WithLabels("app=source").
						Build()

					addPodInfo(source, target)
					addNsByName("target", "source")

					ruleSets, err := renderer.RenderEgress(target, currentPolicies, currentPods, currentNamespaces)
					Expect(err).ToNot(HaveOccurred())
					By(fmt.Sprintf("got rule sets: %+v", ruleSets))

					Expect(ruleSets).To(HaveLen(1))
					checkInterfaceInfos(ruleSets, target.Interfaces)

					// Check rules
					expectedPolicyRules := []policyrules.Rule{
						{
							IPCidrs: []*net.IPNet{
								{
									IP:   net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 192, 168, 1, 3},
									Mask: net.IPMask{255, 255, 255, 255},
								},
							},
							Ports:  []policyrules.Port{},
							Action: policyrules.PolicyActionPass,
						},
						{
							IPCidrs: []*net.IPNet{
								{
									IP:   net.IP{10, 17, 0, 0},
									Mask: net.IPMask{255, 255, 0, 0},
								},
							},
							Ports:  []policyrules.Port{},
							Action: policyrules.PolicyActionPass,
						},
						{
							IPCidrs: []*net.IPNet{
								{
									IP:   net.IP{10, 17, 0, 0},
									Mask: net.IPMask{255, 255, 255, 0},
								},
							},
							Ports:  []policyrules.Port{},
							Action: policyrules.PolicyActionDrop,
						},
					}
					checkRules(ruleSets[0].Rules, expectedPolicyRules)
					Expect(ruleSets[0].Type).To(Equal(policyrules.PolicyTypeEgress))
				})
			})

			Context("multiple interfaces same network", func() {
				It("returns correct rules per interface", func() {
					addPolicy(&testutil.PolicyIPBlockNoPorts, "accel-net")

					target = testutil.NewPodInfoBuiler().
						WithName("target-pod").
						WithNamespace(testutil.TargetNamespace).
						WithInterface(
							"accel-net",
							"0000:03:00.4",
							"net1",
							"accelerated-bridge",
							[]string{"192.168.1.2"}).
						WithInterface(
							"accel-net",
							"0000:03:00.5",
							"net2",
							"accelerated-bridge",
							[]string{"192.168.1.3"}).
						WithLabels("app=target").
						Build()

					ruleSets, err := renderer.RenderEgress(target, currentPolicies, currentPods, currentNamespaces)
					Expect(err).ToNot(HaveOccurred())
					By(fmt.Sprintf("got rule sets: %+v", ruleSets))

					Expect(ruleSets).To(HaveLen(2))
					checkInterfaceInfos(ruleSets, target.Interfaces)

					// Check rules
					expectedPolicyRules := []policyrules.Rule{
						{
							IPCidrs: []*net.IPNet{
								{
									IP:   net.IP{10, 17, 0, 0},
									Mask: net.IPMask{255, 255, 0, 0},
								},
							},
							Ports:  []policyrules.Port{},
							Action: policyrules.PolicyActionPass,
						},
						{
							IPCidrs: []*net.IPNet{
								{
									IP:   net.IP{10, 17, 0, 0},
									Mask: net.IPMask{255, 255, 255, 0},
								},
							},
							Ports:  []policyrules.Port{},
							Action: policyrules.PolicyActionDrop,
						},
					}

					for i := range ruleSets {
						Expect(ruleSets[i].Type).To(Equal(policyrules.PolicyTypeEgress))
						checkRules(ruleSets[i].Rules, expectedPolicyRules)
					}
				})
			})

			Context("multiple interfaces different network", func() {
				It("returns correct rules per interface", func() {
					addPolicy(&testutil.PolicyIPBlockNoPorts, "accel-net1", "accel-net2")

					target = testutil.NewPodInfoBuiler().
						WithName("target-pod").
						WithNamespace(testutil.TargetNamespace).
						WithInterface(
							"accel-net1",
							"0000:03:00.4",
							"net1",
							"accelerated-bridge",
							[]string{"192.168.1.2"}).
						WithInterface(
							"accel-net2",
							"0000:03:00.5",
							"net2",
							"accelerated-bridge",
							[]string{"192.168.1.3"}).
						WithLabels("app=target").
						Build()

					ruleSets, err := renderer.RenderEgress(target, currentPolicies, currentPods, currentNamespaces)
					Expect(err).ToNot(HaveOccurred())
					By(fmt.Sprintf("got rule sets: %+v", ruleSets))

					Expect(ruleSets).To(HaveLen(2))
					checkInterfaceInfos(ruleSets, target.Interfaces)

					// Check rules
					expectedPolicyRules := []policyrules.Rule{
						{
							IPCidrs: []*net.IPNet{
								{
									IP:   net.IP{10, 17, 0, 0},
									Mask: net.IPMask{255, 255, 0, 0},
								},
							},
							Ports:  []policyrules.Port{},
							Action: policyrules.PolicyActionPass,
						},
						{
							IPCidrs: []*net.IPNet{
								{
									IP:   net.IP{10, 17, 0, 0},
									Mask: net.IPMask{255, 255, 255, 0},
								},
							},
							Ports:  []policyrules.Port{},
							Action: policyrules.PolicyActionDrop,
						},
					}

					for i := range ruleSets {
						Expect(ruleSets[i].Type).To(Equal(policyrules.PolicyTypeEgress))
						checkRules(ruleSets[i].Rules, expectedPolicyRules)
					}
				})
			})
		})
	})
})
