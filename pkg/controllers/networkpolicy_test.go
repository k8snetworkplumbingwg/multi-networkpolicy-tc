package controllers_test

import (
	"context"
	"sync"
	"time"

	multiv1beta1 "github.com/k8snetworkplumbingwg/multi-networkpolicy/pkg/apis/k8s.cni.cncf.io/v1beta1"
	multifake "github.com/k8snetworkplumbingwg/multi-networkpolicy/pkg/client/clientset/versioned/fake"
	multiinformerv1beta1 "github.com/k8snetworkplumbingwg/multi-networkpolicy/pkg/client/informers/externalversions"
	"k8s.io/client-go/tools/cache"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Mellanox/multi-networkpolicy-tc/pkg/controllers"
	"github.com/Mellanox/multi-networkpolicy-tc/pkg/controllers/testutil"
)

type FakeNetworkPolicyConfigStub struct {
	CounterAdd    int
	CounterUpdate int
	CounterDelete int
	CounterSynced int
}

func (f *FakeNetworkPolicyConfigStub) OnPolicyAdd(_ *multiv1beta1.MultiNetworkPolicy) {
	f.CounterAdd++
}

func (f *FakeNetworkPolicyConfigStub) OnPolicyUpdate(_, _ *multiv1beta1.MultiNetworkPolicy) {
	f.CounterUpdate++
}

func (f *FakeNetworkPolicyConfigStub) OnPolicyDelete(_ *multiv1beta1.MultiNetworkPolicy) {
	f.CounterDelete++
}

func (f *FakeNetworkPolicyConfigStub) OnPolicySynced() {
	f.CounterSynced++
}

var _ = Describe("networkpolicy config", func() {
	configSync := 15 * time.Minute
	var wg sync.WaitGroup
	var stopCtx context.Context
	var stopFunc context.CancelFunc
	var fakeClient *multifake.Clientset
	var informerFactory multiinformerv1beta1.SharedInformerFactory
	var stub *FakeNetworkPolicyConfigStub
	var netPolConfig *controllers.NetworkPolicyConfig
	var mnp *multiv1beta1.MultiNetworkPolicy

	BeforeEach(func() {
		wg = sync.WaitGroup{}
		stopCtx, stopFunc = context.WithCancel(context.Background())
		fakeClient = multifake.NewSimpleClientset()
		informerFactory = multiinformerv1beta1.NewSharedInformerFactory(fakeClient, configSync)
		multiNetInformer := informerFactory.K8sCniCncfIo().V1beta1().MultiNetworkPolicies()
		netPolConfig = controllers.NewNetworkPolicyConfig(multiNetInformer, configSync)
		stub = &FakeNetworkPolicyConfigStub{}
		mnp = testutil.NewNetworkPolicy("testns1", "test1")

		netPolConfig.RegisterEventHandler(stub)
		informerFactory.Start(stopCtx.Done())

		wg.Add(1)
		go func() {
			netPolConfig.Run(stopCtx.Done())
			wg.Done()
		}()

		cacheSyncCtx, cfn := context.WithTimeout(context.Background(), 1*time.Second)
		defer cfn()
		Expect(cache.WaitForCacheSync(cacheSyncCtx.Done(), multiNetInformer.Informer().HasSynced)).To(BeTrue())
	})

	AfterEach(func() {
		stopFunc()
		wg.Wait()
	})

	It("check sync handler", func() {
		Eventually(&stub.CounterSynced).Should(HaveValue(Equal(1)))
		Eventually(&stub.CounterAdd).Should(HaveValue(Equal(0)))
		Eventually(&stub.CounterUpdate).Should(HaveValue(Equal(0)))
		Eventually(&stub.CounterDelete).Should(HaveValue(Equal(0)))
	})

	It("check add handler", func() {
		_, err := fakeClient.K8sCniCncfIoV1beta1().MultiNetworkPolicies(mnp.Namespace).Create(
			context.Background(), mnp, metav1.CreateOptions{})

		Expect(err).ToNot(HaveOccurred())
		Eventually(&stub.CounterAdd).Should(HaveValue(Equal(1)))
		Eventually(&stub.CounterUpdate).Should(HaveValue(Equal(0)))
		Eventually(&stub.CounterDelete).Should(HaveValue(Equal(0)))
	})

	It("check update handler", func() {
		p, err := fakeClient.K8sCniCncfIoV1beta1().MultiNetworkPolicies(mnp.Namespace).Create(
			context.Background(), mnp, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		p.Labels = map[string]string{"my": "label"}
		_, err = fakeClient.K8sCniCncfIoV1beta1().MultiNetworkPolicies(mnp.Namespace).Update(
			context.Background(), p, metav1.UpdateOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(&stub.CounterAdd).Should(HaveValue(Equal(1)))
		Eventually(&stub.CounterUpdate).Should(HaveValue(Equal(1)))
		Eventually(&stub.CounterDelete).Should(HaveValue(Equal(0)))
	})

	It("check delete handler", func() {
		p, err := fakeClient.K8sCniCncfIoV1beta1().MultiNetworkPolicies(mnp.Namespace).Create(
			context.Background(), mnp, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		err = fakeClient.K8sCniCncfIoV1beta1().MultiNetworkPolicies(mnp.Namespace).Delete(
			context.Background(), p.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(&stub.CounterAdd).Should(HaveValue(Equal(1)))
		Eventually(&stub.CounterUpdate).Should(HaveValue(Equal(0)))
		Eventually(&stub.CounterDelete).Should(HaveValue(Equal(1)))
	})
})

var _ = Describe("networkpolicy controller", func() {
	var policyChanges *controllers.PolicyChangeTracker
	var policyMap controllers.PolicyMap
	var policy1, policy2 *multiv1beta1.MultiNetworkPolicy

	BeforeEach(func() {
		policyChanges = controllers.NewPolicyChangeTracker()
		policyMap = make(controllers.PolicyMap)
		policy1 = testutil.NewNetworkPolicy("testns1", "test1")
		policy2 = testutil.NewNetworkPolicy("testns2", "test2")
	})

	nsName := func(np *multiv1beta1.MultiNetworkPolicy) types.NamespacedName {
		return types.NamespacedName{Namespace: np.Namespace, Name: np.Name}
	}

	checkPolicyMapWithPolicy := func(policy *multiv1beta1.MultiNetworkPolicy) {
		policyTest, ok := policyMap[nsName(policy)]
		ExpectWithOffset(1, ok).To(BeTrue())
		ExpectWithOffset(1, policyTest.Name()).To(Equal(policy.Name))
		ExpectWithOffset(1, policyTest.Namespace()).To(Equal(policy.Namespace))
		ExpectWithOffset(1, policyTest.Policy).To(BeEquivalentTo(policy))
	}

	It("invalid Update case both nil - NetDefChangeTracker", func() {
		Expect(policyChanges.Update(nil, nil)).To(BeFalse())
	})

	It("invalid Update case - NetDefMap", func() {
		policyMap.Update(nil)
		Expect(policyMap).To(BeEmpty())
	})

	It("empty update - NetDefMap", func() {
		policyMap.Update(policyChanges)
		Expect(policyMap).To(BeEmpty())
	})

	It("Add policy and verify", func() {
		Expect(policyChanges.Update(nil, policy1)).To(BeTrue())
		Expect(policyChanges.Update(nil, policy2)).To(BeTrue())

		policyMap.Update(policyChanges)
		Expect(policyMap).To(HaveLen(2))
		checkPolicyMapWithPolicy(policy1)
		checkPolicyMapWithPolicy(policy2)
	})

	It("Add policy then delete it and verify", func() {
		Expect(policyChanges.Update(nil, policy1)).To(BeTrue())
		Expect(policyChanges.Update(nil, policy2)).To(BeTrue())
		Expect(policyChanges.Update(policy1, nil)).To(BeTrue())

		policyMap.Update(policyChanges)
		Expect(policyMap).To(HaveLen(1))
		checkPolicyMapWithPolicy(policy2)
	})

	It("Add policy then update it and verify", func() {
		Expect(policyChanges.Update(nil, policy1)).To(BeTrue())
		updatedPolicy := testutil.NewNetworkPolicy("testns1", "test1")
		updatedPolicy.Spec.PolicyTypes = []multiv1beta1.MultiPolicyType{multiv1beta1.PolicyTypeEgress}
		Expect(policyChanges.Update(policy1, updatedPolicy)).To(BeTrue())

		policyMap.Update(policyChanges)
		Expect(policyMap).To(HaveLen(1))
		checkPolicyMapWithPolicy(updatedPolicy)
	})
})
