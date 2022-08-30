package controllers_test

import (
	"context"
	"sync"
	"time"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Mellanox/multi-networkpolicy-tc/pkg/controllers"
)

type FakeNamespaceConfigStub struct {
	CounterAdd    int
	CounterUpdate int
	CounterDelete int
	CounterSynced int
}

func (f *FakeNamespaceConfigStub) OnNamespaceAdd(_ *v1.Namespace) {
	f.CounterAdd++
}

func (f *FakeNamespaceConfigStub) OnNamespaceUpdate(_, _ *v1.Namespace) {
	f.CounterUpdate++
}

func (f *FakeNamespaceConfigStub) OnNamespaceDelete(_ *v1.Namespace) {
	f.CounterDelete++
}

func (f *FakeNamespaceConfigStub) OnNamespaceSynced() {
	f.CounterSynced++
}

func newTestNamespace(name string, labels map[string]string) *v1.Namespace {
	return &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}
}

var _ = Describe("namespace config", func() {
	configSync := 15 * time.Minute
	var wg sync.WaitGroup
	var stopCtx context.Context
	var stopFunc context.CancelFunc
	var fakeClient *fake.Clientset
	var informerFactory informers.SharedInformerFactory
	var stub *FakeNamespaceConfigStub
	var nsConfig *controllers.NamespaceConfig

	BeforeEach(func() {
		wg = sync.WaitGroup{}
		stopCtx, stopFunc = context.WithCancel(context.Background())
		fakeClient = fake.NewSimpleClientset()
		informerFactory = informers.NewSharedInformerFactory(fakeClient, configSync)
		nsInformer := informerFactory.Core().V1().Namespaces()
		nsConfig = controllers.NewNamespaceConfig(nsInformer, configSync)
		stub = &FakeNamespaceConfigStub{}

		nsConfig.RegisterEventHandler(stub)
		informerFactory.Start(stopCtx.Done())

		wg.Add(1)
		go func() {
			nsConfig.Run(stopCtx.Done())
			wg.Done()
		}()

		cacheSyncCtx, cfn := context.WithTimeout(context.Background(), 1*time.Second)
		defer cfn()
		Expect(cache.WaitForCacheSync(cacheSyncCtx.Done(), nsInformer.Informer().HasSynced)).To(BeTrue())
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
		_, err := fakeClient.CoreV1().Namespaces().Create(
			context.Background(), newTestNamespace("test", nil), metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(&stub.CounterAdd).Should(HaveValue(Equal(1)))
		Eventually(&stub.CounterUpdate).Should(HaveValue(Equal(0)))
		Eventually(&stub.CounterDelete).Should(HaveValue(Equal(0)))

	})

	It("check update handler", func() {
		ns, err := fakeClient.CoreV1().Namespaces().Create(
			context.Background(), newTestNamespace("test", nil), metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		ns.Labels = map[string]string{"my": "label"}
		_, err = fakeClient.CoreV1().Namespaces().Update(
			context.Background(), ns, metav1.UpdateOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(&stub.CounterAdd).Should(HaveValue(Equal(1)))
		Eventually(&stub.CounterUpdate).Should(HaveValue(Equal(1)))
		Eventually(&stub.CounterDelete).Should(HaveValue(Equal(0)))
	})

	It("check delete handler", func() {
		ns, err := fakeClient.CoreV1().Namespaces().Create(
			context.Background(), newTestNamespace("test", nil), metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		err = fakeClient.CoreV1().Namespaces().Delete(context.Background(), ns.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(&stub.CounterAdd).Should(HaveValue(Equal(1)))
		Eventually(&stub.CounterUpdate).Should(HaveValue(Equal(0)))
		Eventually(&stub.CounterDelete).Should(HaveValue(Equal(1)))
	})
})

var _ = Describe("namespace change tracker", func() {
	var nsMap controllers.NamespaceMap
	var nsChanges *controllers.NamespaceChangeTracker
	var ns1, ns2 *v1.Namespace

	checkNsMapWithNS := func(ns *v1.Namespace) {
		nsTest, ok := nsMap[ns.Name]
		ExpectWithOffset(1, ok).To(BeTrue())
		ExpectWithOffset(1, ok).To(BeTrue())
		ExpectWithOffset(1, nsTest.Name).To(Equal(ns.Name))
		ExpectWithOffset(1, nsTest.Labels).To(BeEquivalentTo(ns.Labels))
	}

	BeforeEach(func() {
		nsChanges = controllers.NewNamespaceChangeTracker()
		nsMap = make(controllers.NamespaceMap)
		ns1 = newTestNamespace("test1", map[string]string{"labelName1": "labelValue1"})
		ns2 = newTestNamespace("test2", map[string]string{"labelName2": "labelValue2"})
	})

	It("empty update", func() {
		nsMap.Update(nsChanges)
		Expect(nsMap).To(BeEmpty())
	})

	It("invalid Update case both nil - NamespaceChangeTracker", func() {
		Expect(nsChanges.Update(nil, nil)).To(BeFalse())
	})

	It("invalid Update case - NamespaceMap", func() {
		nsMap.Update(nil)
		Expect(nsMap).To(BeEmpty())
	})

	It("add ns and verify", func() {
		Expect(nsChanges.Update(nil, ns1)).To(BeTrue())

		nsMap.Update(nsChanges)
		Expect(nsMap).To(HaveLen(1))
		checkNsMapWithNS(ns1)
	})

	It("add ns then del ns and verify", func() {
		Expect(nsChanges.Update(nil, ns1)).To(BeTrue())
		Expect(nsChanges.Update(nil, ns2)).To(BeTrue())
		Expect(nsChanges.Update(ns2, nil)).To(BeTrue())

		nsMap.Update(nsChanges)
		Expect(nsMap).To(HaveLen(1))
		checkNsMapWithNS(ns1)
	})

	It("add ns then update ns and verify", func() {
		updatedNs1 := newTestNamespace("test1", map[string]string{"otherLabelName": "otherLabelValue"})
		Expect(nsChanges.Update(nil, ns1)).To(BeTrue())
		Expect(nsChanges.Update(ns1, updatedNs1)).To(BeTrue())

		nsMap.Update(nsChanges)
		Expect(nsMap).To(HaveLen(1))
		checkNsMapWithNS(updatedNs1)
	})

	It("add same ns as current and previous", func() {
		Expect(nsChanges.Update(ns1, ns1)).To(BeTrue())
		nsMap.Update(nsChanges)
		Expect(nsMap).To(BeEmpty())
	})
})
