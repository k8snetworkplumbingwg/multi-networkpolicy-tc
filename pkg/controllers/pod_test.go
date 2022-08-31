package controllers_test

import (
	"context"
	"fmt"
	"sync"
	"time"

	netdefv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Mellanox/multi-networkpolicy-tc/pkg/controllers"
)

type FakePodConfigStub struct {
	CounterAdd    int
	CounterUpdate int
	CounterDelete int
	CounterSynced int
}

func (f *FakePodConfigStub) OnPodAdd(_ *v1.Pod) {
	f.CounterAdd++
}

func (f *FakePodConfigStub) OnPodUpdate(_, _ *v1.Pod) {
	f.CounterUpdate++
}

func (f *FakePodConfigStub) OnPodDelete(_ *v1.Pod) {
	f.CounterDelete++
}

func (f *FakePodConfigStub) OnPodSynced() {
	f.CounterSynced++
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

var _ = Describe("pod config", func() {
	configSync := 15 * time.Minute
	var wg sync.WaitGroup
	var stopCtx context.Context
	var stopFunc context.CancelFunc
	var fakeClient *fake.Clientset
	var informerFactory informers.SharedInformerFactory
	var stub *FakePodConfigStub
	var podConfig *controllers.PodConfig
	var testPod1 *v1.Pod

	BeforeEach(func() {
		wg = sync.WaitGroup{}
		stopCtx, stopFunc = context.WithCancel(context.Background())
		fakeClient = fake.NewSimpleClientset()
		informerFactory = informers.NewSharedInformerFactory(fakeClient, configSync)
		podInformer := informerFactory.Core().V1().Pods()
		podConfig = controllers.NewPodConfig(podInformer, configSync)
		stub = &FakePodConfigStub{}
		testPod1 = NewFakePod("testns1", "pod1")

		podConfig.RegisterEventHandler(stub)
		informerFactory.Start(stopCtx.Done())

		wg.Add(1)
		go func() {
			podConfig.Run(stopCtx.Done())
			wg.Done()
		}()

		cacheSyncCtx, cfn := context.WithTimeout(context.Background(), 1*time.Second)
		defer cfn()
		Expect(cache.WaitForCacheSync(cacheSyncCtx.Done(), podInformer.Informer().HasSynced)).To(BeTrue())
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
		_, err := fakeClient.CoreV1().Pods(testPod1.Namespace).Create(
			context.Background(), testPod1, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(&stub.CounterAdd).Should(HaveValue(Equal(1)))
		Eventually(&stub.CounterUpdate).Should(HaveValue(Equal(0)))
		Eventually(&stub.CounterDelete).Should(HaveValue(Equal(0)))
	})

	It("check update handler", func() {
		p, err := fakeClient.CoreV1().Pods(testPod1.Namespace).Create(
			context.Background(), testPod1, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		p.Labels = map[string]string{"my": "label"}
		_, err = fakeClient.CoreV1().Pods(p.Namespace).Update(
			context.Background(), p, metav1.UpdateOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(&stub.CounterAdd).Should(HaveValue(Equal(1)))
		Eventually(&stub.CounterUpdate).Should(HaveValue(Equal(1)))
		Eventually(&stub.CounterDelete).Should(HaveValue(Equal(0)))
	})

	It("check delete handler", func() {
		p, err := fakeClient.CoreV1().Pods(testPod1.Namespace).Create(
			context.Background(), testPod1, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		err = fakeClient.CoreV1().Pods(p.Namespace).Delete(
			context.Background(), p.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(&stub.CounterAdd).Should(HaveValue(Equal(1)))
		Eventually(&stub.CounterUpdate).Should(HaveValue(Equal(0)))
		Eventually(&stub.CounterDelete).Should(HaveValue(Equal(1)))
	})
})

var _ = Describe("pod change tracker", func() {
	var ndChanges *controllers.NetDefChangeTracker
	var podChanges *controllers.PodChangeTracker
	var podMap controllers.PodMap

	nsName := func(p *v1.Pod) types.NamespacedName {
		return types.NamespacedName{Namespace: p.Namespace, Name: p.Name}
	}

	checkPodInfo := func(p *v1.Pod, numOfInterfaces int) {
		testPodInfo, ok := podMap[nsName(p)]
		ExpectWithOffset(1, ok).To(BeTrue())
		ExpectWithOffset(1, testPodInfo.Name).To(Equal(p.Name))
		ExpectWithOffset(1, testPodInfo.Namespace).To(Equal(p.Namespace))
		ExpectWithOffset(1, testPodInfo.Interfaces).To(HaveLen(numOfInterfaces))
	}

	BeforeEach(func() {
		ndChanges = controllers.NewNetDefChangeTracker()
		podChanges = controllers.NewPodChangeTracker([]string{"accelerated-bridge"}, ndChanges)
		podMap = make(controllers.PodMap)
	})

	Context("basic cases", func() {
		It("invalid Update case both nil - NetDefChangeTracker", func() {
			Expect(podChanges.Update(nil, nil)).To(BeFalse())
		})

		It("invalid Update case - NetDefMap", func() {
			podMap.Update(nil)
			Expect(podMap).To(BeEmpty())
		})

		It("empty update - NetDefMap", func() {
			podMap.Update(podChanges)
			Expect(podMap).To(BeEmpty())
		})
	})

	Context("basic pods - no secondary network and status", func() {
		var pod1, pod2 *v1.Pod

		BeforeEach(func() {
			pod1 = NewFakePod("testns1", "testpod1")
			pod2 = NewFakePod("testns2", "testpod2")
		})

		It("Add pod and verify", func() {
			Expect(podChanges.Update(nil, pod1)).To(BeTrue())
			podMap.Update(podChanges)
			Expect(podMap).To(HaveLen(1))
			checkPodInfo(pod1, 0)
		})

		It("Add ns then del ns and verify", func() {
			Expect(podChanges.Update(nil, pod1)).To(BeTrue())
			Expect(podChanges.Update(nil, pod2)).To(BeTrue())
			Expect(podChanges.Update(pod1, nil)).To(BeTrue())

			podMap.Update(podChanges)
			Expect(podMap).To(HaveLen(1))
			checkPodInfo(pod2, 0)
		})

		It("Add ns then update ns and verify", func() {
			podWithLables := NewFakePod("testns1", "testpod1")
			podWithLables.Labels = map[string]string{"Some": "Label"}

			Expect(podChanges.Update(nil, pod1)).To(BeTrue())
			Expect(podChanges.Update(pod1, podWithLables)).To(BeTrue())

			podMap.Update(podChanges)
			Expect(podMap).To(HaveLen(1))
			checkPodInfo(podWithLables, 0)
		})
	})

	Context("pods with networks", func() {
		BeforeEach(func() {
			Expect(ndChanges.Update(
				nil, NewNetDef("testns1", "net-attach1", NewCNIConfig(
					"testCNI", "accelerated-bridge")))).To(BeTrue())

		})

		It("Add pod with net-attach annotation and status", func() {
			podWithNeworkAndStatus := NewFakePodWithNetAnnotation("testns1", "testpod1",
				"net-attach1", NewFakeNetworkStatus("testns1", "net-attach1"))
			Expect(podChanges.Update(nil, podWithNeworkAndStatus)).To(BeTrue())
			podMap.Update(podChanges)
			Expect(podMap).To(HaveLen(1))

			checkPodInfo(podWithNeworkAndStatus, 1)

			// Check interface
			pInfo := podMap[nsName(podWithNeworkAndStatus)]
			Expect(pInfo.Interfaces[0].DeviceID).To(Equal("0000:03:00.2"))
			Expect(pInfo.Interfaces[0].InterfaceType).To(Equal("accelerated-bridge"))
			Expect(pInfo.Interfaces[0].InterfaceName).To(Equal("net1"))
			Expect(pInfo.Interfaces[0].IPs).To(BeEquivalentTo([]string{"10.1.1.101"}))
			Expect(pInfo.Interfaces[0].NetattachName).To(Equal("testns1/net-attach1"))
		})

		It("Add pod with net-attach annotation no status", func() {
			podWitoutNeworkStatus := NewFakePodWithNetAnnotation("testns1", "testpod1", "net-attach1", "")
			Expect(podChanges.Update(nil, podWitoutNeworkStatus)).To(BeTrue())
			podMap.Update(podChanges)
			Expect(podMap).To(HaveLen(1))
			checkPodInfo(podWitoutNeworkStatus, 0)

			podWithNeworkAndStatus := NewFakePodWithNetAnnotation("testns1", "testpod1",
				"net-attach1", NewFakeNetworkStatus("testns1", "net-attach1"))
			Expect(podChanges.Update(podWitoutNeworkStatus, podWithNeworkAndStatus)).To(BeTrue())

			podMap.Update(podChanges)
			Expect(podMap).To(HaveLen(1))
			checkPodInfo(podWithNeworkAndStatus, 1)
		})
	})
})
