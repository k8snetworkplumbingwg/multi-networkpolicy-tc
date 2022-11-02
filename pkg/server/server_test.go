package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/k8snetworkplumbingwg/multi-networkpolicy/pkg/apis/k8s.cni.cncf.io/v1beta1"
	netdefv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/controllers/testutil"
	netmocks "github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/net/mocks"
	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/policyrules"
	policymocks "github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/policyrules/mocks"
	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc"
	generatorMocks "github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc/generator/mocks"
	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/tc/mocks"
)

func createTestPodAndSetRunning(ctx context.Context, testPod *v1.Pod) {
	p, err := kClient.
		CoreV1().
		Pods("default").
		Create(ctx, testPod, metav1.CreateOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	p.Status.Phase = v1.PodRunning
	_, err = kClient.CoreV1().Pods("default").UpdateStatus(ctx, p, metav1.UpdateOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
}

func lenOfCalls(m *mock.Mock) func() int {
	return func() int {
		return len(m.Calls)
	}
}

var _ = Describe("Server test", func() {
	var testServer *Server
	var runCtx context.Context
	var cFunc context.CancelFunc
	var wg sync.WaitGroup
	var mockActuator *mocks.Actuator
	var mockRenderer *policymocks.Renderer
	var mockRuleGenerator *generatorMocks.Generator
	var mockSriovnetProvider *netmocks.SriovnetProvider

	BeforeEach(func() {
		var err error
		mockActuator = &mocks.Actuator{}
		mockRenderer = &policymocks.Renderer{}
		mockRuleGenerator = &generatorMocks.Generator{}
		mockSriovnetProvider = &netmocks.SriovnetProvider{}

		o := &Options{
			KConfig:              kconfig,
			hostnameOverride:     nodeName,
			networkPlugins:       []string{"accelerated-bridge"},
			podRulesPath:         "",
			createActuatorForRep: func(string) tc.Actuator { return mockActuator },
			policyRuleRenderer:   mockRenderer,
			tcRuleGenerator:      mockRuleGenerator,
			sriovnetProvider:     mockSriovnetProvider,
		}

		testServer, err = NewServer(o)
		Expect(err).ToNot(HaveOccurred())

		runCtx, cFunc = context.WithCancel(context.Background())
		wg = sync.WaitGroup{}
		wg.Add(1)
		go func() {
			testServer.Run(runCtx)
			wg.Done()
		}()
	})

	AfterEach(func() {
		cFunc()
		wg.Wait()
	})

	Context("Basic", func() {
		It("Starts successfully", func() {
			Eventually(func(g Gomega) {
				events, err := kClient.CoreV1().Events("").
					List(context.Background(), metav1.ListOptions{FieldSelector: fmt.Sprintf("involvedObject.name=%s", nodeName)})
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(events.Items).To(HaveLen(1))
				g.Expect(events.Items[0].Reason).To(Equal("Started"))
			}).Should(Succeed())
		})
	})

	Context("Sync pod on node", func() {
		var podOnOtherNode, podOnNodeNoNet, podOnNode *v1.Pod
		var policy *v1beta1.MultiNetworkPolicy
		var network *netdefv1.NetworkAttachmentDefinition

		BeforeEach(func() {
			mockRenderer.On("RenderEgress", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return([]policyrules.PolicyRuleSet{{}}, nil)
			mockSriovnetProvider.On("GetVfIndexByPciAddress", mock.Anything).
				Return(1, nil)
			mockSriovnetProvider.On("GetUplinkRepresentor", mock.Anything).
				Return("enp3s0f0", nil)
			mockSriovnetProvider.On("GetVfRepresentor", mock.Anything, mock.Anything).
				Return("eth5", nil)
			mockRuleGenerator.On("GenerateFromPolicyRuleSet", mock.Anything).
				Return(nil, nil)
			mockActuator.On("Actuate", mock.Anything).
				Return(nil)

			network = testutil.NewNetDef("default", "accel-net", testutil.NewCNIConfig(
				"accel-net", "accelerated-bridge"))

			policy = testutil.NewNetworkPolicy("default", "policy")

			podOnNode = testutil.NewFakePodWithNetAnnotation(
				"default", "target-pod", "default/accel-net",
				testutil.NewFakeNetworkStatus("default", "accel-net"))
			podOnNode.Spec.NodeName = nodeName

			podOnOtherNode = testutil.NewFakePodWithNetAnnotation(
				"default", "pod-other-node", "default/accel-net",
				testutil.NewFakeNetworkStatus("default", "accel-net"))
			podOnOtherNode.Spec.NodeName = "other-node"

			podOnNodeNoNet = testutil.NewFakePodWithNetAnnotation(
				"default", "pod-on-node-no-net", "default/accel-net",
				`	[{
		"name": "",
		"interface": "eth0",
		"ips": [
			"10.244.1.4"
		],
		"mac": "aa:e1:20:71:15:01",
		"default": true,
		"dns": {}
		}]`)
			podOnNodeNoNet.Spec.NodeName = nodeName

			var err error
			ctx := context.Background()

			// create network
			_, err = netDefClient.K8sCniCncfIoV1().
				NetworkAttachmentDefinitions("default").
				Create(ctx, network, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// create policy
			_, err = policyClient.
				K8sCniCncfIoV1beta1().
				MultiNetworkPolicies("default").
				Create(ctx, policy, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			//create pods
			createTestPodAndSetRunning(ctx, podOnOtherNode)
			createTestPodAndSetRunning(ctx, podOnNodeNoNet)
			createTestPodAndSetRunning(ctx, podOnNode)
		})

		AfterEach(func() {
			var err error
			ctx := context.Background()
			// delete pods
			err = kClient.
				CoreV1().
				Pods("default").
				Delete(ctx, podOnOtherNode.ObjectMeta.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			err = kClient.
				CoreV1().
				Pods("default").
				Delete(ctx, podOnNodeNoNet.ObjectMeta.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			err = kClient.
				CoreV1().
				Pods("default").
				Delete(ctx, podOnNode.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			//delete policy
			err = policyClient.
				K8sCniCncfIoV1beta1().
				MultiNetworkPolicies("default").
				Delete(ctx, policy.ObjectMeta.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			//delete network
			err = netDefClient.K8sCniCncfIoV1().
				NetworkAttachmentDefinitions("default").
				Delete(ctx, network.ObjectMeta.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("Syncs TC rules successfully", func() {
			// we define success when all mocks were called more than once
			Eventually(lenOfCalls(&mockRenderer.Mock)).
				WithTimeout(5 * time.Second).
				Should(BeNumerically(">=", 1))
			Eventually(lenOfCalls(&mockSriovnetProvider.Mock)).
				WithTimeout(5 * time.Second).
				Should(BeNumerically(">=", 1))
			Eventually(lenOfCalls(&mockRuleGenerator.Mock)).
				WithTimeout(5 * time.Second).
				Should(BeNumerically(">=", 1))
			Eventually(lenOfCalls(&mockActuator.Mock)).
				WithTimeout(5 * time.Second).
				Should(BeNumerically(">=", 1))
		})
	})
})
