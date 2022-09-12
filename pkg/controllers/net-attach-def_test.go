package controllers_test

import (
	"context"
	"sync"
	"time"

	netdefv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	netdeffake "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/client/clientset/versioned/fake"
	netdefinformerv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/client/informers/externalversions"
	"k8s.io/client-go/tools/cache"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/controllers"
	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/controllers/testutil"
)

type FakeNetDefConfigStub struct {
	CounterAdd    int
	CounterUpdate int
	CounterDelete int
	CounterSynced int
}

func (f *FakeNetDefConfigStub) OnNetDefAdd(_ *netdefv1.NetworkAttachmentDefinition) {
	f.CounterAdd++
}

func (f *FakeNetDefConfigStub) OnNetDefUpdate(_, _ *netdefv1.NetworkAttachmentDefinition) {
	f.CounterUpdate++
}

func (f *FakeNetDefConfigStub) OnNetDefDelete(_ *netdefv1.NetworkAttachmentDefinition) {
	f.CounterDelete++
}

func (f *FakeNetDefConfigStub) OnNetDefSynced() {
	f.CounterSynced++
}

var _ = Describe("net-attach-def config", func() {
	configSync := 15 * time.Minute
	var wg sync.WaitGroup
	var stopCtx context.Context
	var stopFunc context.CancelFunc
	var fakeClient *netdeffake.Clientset
	var informerFactory netdefinformerv1.SharedInformerFactory
	var stub *FakeNetDefConfigStub
	var netDefConfig *controllers.NetDefConfig
	var nd1 *netdefv1.NetworkAttachmentDefinition

	BeforeEach(func() {
		wg = sync.WaitGroup{}
		stopCtx, stopFunc = context.WithCancel(context.Background())
		fakeClient = netdeffake.NewSimpleClientset()
		informerFactory = netdefinformerv1.NewSharedInformerFactory(fakeClient, configSync)
		netDefInformer := informerFactory.K8sCniCncfIo().V1().NetworkAttachmentDefinitions()
		netDefConfig = controllers.NewNetDefConfig(netDefInformer, configSync)
		stub = &FakeNetDefConfigStub{}
		nd1 = testutil.NewNetDef("testns1", "test1", testutil.NewCNIConfig("cniConfig1", "testType1"))

		netDefConfig.RegisterEventHandler(stub)
		informerFactory.Start(stopCtx.Done())

		wg.Add(1)
		go func() {
			netDefConfig.Run(stopCtx.Done())
			wg.Done()
		}()

		cacheSyncCtx, cfn := context.WithTimeout(context.Background(), 1*time.Second)
		defer cfn()
		Expect(cache.WaitForCacheSync(cacheSyncCtx.Done(), netDefInformer.Informer().HasSynced)).To(BeTrue())
	})

	AfterEach(func() {
		stopFunc()
		wg.Wait()
	})

	It("check sync handler", func() {
		Eventually(&stub.CounterSynced).Should(HaveValue(Equal(1)))
	})

	It("check add handler", func() {
		_, err := fakeClient.K8sCniCncfIoV1().NetworkAttachmentDefinitions(nd1.Namespace).
			Create(context.Background(), nd1, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(&stub.CounterAdd).Should(HaveValue(Equal(1)))
		Eventually(&stub.CounterUpdate).Should(HaveValue(Equal(0)))
		Eventually(&stub.CounterDelete).Should(HaveValue(Equal(0)))
	})

	It("check update handler", func() {
		updatedNd, err := fakeClient.K8sCniCncfIoV1().NetworkAttachmentDefinitions(nd1.Namespace).
			Create(context.Background(), nd1, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		updatedNd.Spec.Config = testutil.NewCNIConfig("cniConfig2", "testType2")
		_, err = fakeClient.K8sCniCncfIoV1().NetworkAttachmentDefinitions(updatedNd.Namespace).
			Update(context.Background(), updatedNd, metav1.UpdateOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(&stub.CounterAdd).Should(HaveValue(Equal(1)))
		Eventually(&stub.CounterUpdate).Should(HaveValue(Equal(1)))
		Eventually(&stub.CounterDelete).Should(HaveValue(Equal(0)))
	})

	It("check delete handler", func() {
		_, err := fakeClient.K8sCniCncfIoV1().NetworkAttachmentDefinitions(nd1.Namespace).
			Create(context.Background(), nd1, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		err = fakeClient.K8sCniCncfIoV1().NetworkAttachmentDefinitions(nd1.Namespace).
			Delete(context.Background(), nd1.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(&stub.CounterAdd).Should(HaveValue(Equal(1)))
		Eventually(&stub.CounterUpdate).Should(HaveValue(Equal(0)))
		Eventually(&stub.CounterDelete).Should(HaveValue(Equal(1)))
	})
})

var _ = Describe("net-attach-def change tracker", func() {
	var ndChanges *controllers.NetDefChangeTracker
	var ndMap controllers.NetDefMap
	var nd1, nd2 *netdefv1.NetworkAttachmentDefinition

	nsName := func(nd *netdefv1.NetworkAttachmentDefinition) types.NamespacedName {
		return types.NamespacedName{Namespace: nd.Namespace, Name: nd.Name}
	}

	checkNetDefMapWithNetDef := func(nd *netdefv1.NetworkAttachmentDefinition, expectedPluginType string) {
		ndTest, ok := ndMap[nsName(nd)]
		ExpectWithOffset(1, ok).To(BeTrue())
		ExpectWithOffset(1, ndTest.Name()).To(Equal(nd.Name))
		ExpectWithOffset(1, ndTest.PluginType).To(Equal(expectedPluginType))
		ExpectWithOffset(1, ndTest.Netdef).To(BeEquivalentTo(nd))
	}

	BeforeEach(func() {
		ndMap = make(controllers.NetDefMap)
		ndChanges = controllers.NewNetDefChangeTracker()
		nd1 = testutil.NewNetDef("testns1", "test1", testutil.NewCNIConfig("cniConfig1", "testType1"))
		nd2 = testutil.NewNetDef("testns2", "test2", testutil.NewCNIConfigList("cniConfig2", "testType2"))
	})

	It("invalid Update case both nil - NetDefChangeTracker", func() {
		Expect(ndChanges.Update(nil, nil)).To(BeFalse())
	})

	It("invalid Update case - NetDefMap", func() {
		ndMap.Update(nil)
		Expect(ndMap).To(BeEmpty())
	})

	It("empty update - NetDefMap", func() {
		ndMap.Update(ndChanges)
		Expect(ndMap).To(BeEmpty())
	})

	It("Add netdef and verify", func() {
		Expect(ndChanges.Update(nil, nd1)).To(BeTrue())
		Expect(ndChanges.Update(nil, nd2)).To(BeTrue())

		ndMap.Update(ndChanges)
		Expect(ndMap).To(HaveLen(2))
		checkNetDefMapWithNetDef(nd1, "testType1")
		checkNetDefMapWithNetDef(nd2, "testType2")
	})

	It("Add netdef then del it and verify", func() {
		Expect(ndChanges.Update(nil, nd1)).To(BeTrue())
		Expect(ndChanges.Update(nil, nd2)).To(BeTrue())
		Expect(ndChanges.Update(nd2, nil)).To(BeTrue())

		ndMap.Update(ndChanges)
		Expect(ndMap).To(HaveLen(1))
		checkNetDefMapWithNetDef(nd1, "testType1")
	})

	It("Add netdef then update it and verify", func() {
		Expect(ndChanges.Update(nil, nd1)).To(BeTrue())
		updatedNd1 := testutil.NewNetDef(nd1.Namespace, nd1.Name, testutil.NewCNIConfigList("cniConfig2", "testType2"))
		Expect(ndChanges.Update(nd1, updatedNd1)).To(BeTrue())

		ndMap.Update(ndChanges)
		Expect(ndMap).To(HaveLen(1))
		checkNetDefMapWithNetDef(updatedNd1, "testType2")
	})

	It("add same netdef as current and previous", func() {
		Expect(ndChanges.Update(nd1, nd1)).To(BeTrue())
		ndMap.Update(ndChanges)
		Expect(ndMap).To(BeEmpty())
	})
})
