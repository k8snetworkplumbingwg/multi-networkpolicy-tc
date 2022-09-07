package server

import (
	"context"
	"flag"
	"path/filepath"
	"testing"

	multiclient "github.com/k8snetworkplumbingwg/multi-networkpolicy/pkg/client/clientset/versioned"
	netdefclient "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/client/clientset/versioned"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	testEnv      *envtest.Environment
	kconfig      *rest.Config
	nodeName     = "my-test-node"
	kClient      *clientset.Clientset
	netDefClient *netdefclient.Clientset
	policyClient *multiclient.Clientset
)

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "server")
}

var _ = BeforeSuite(func() {
	By("bootstrapping test environment")
	var err error

	By("Initializing logger")
	fs := flag.NewFlagSet("test-flag-set", flag.PanicOnError)
	klog.InitFlags(fs)
	Expect(fs.Set("v", "6")).ToNot(HaveOccurred())

	By("setup envtest")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "deploy", "crds")},
		ErrorIfCRDPathMissing: true,
		CRDInstallOptions: envtest.CRDInstallOptions{
			CleanUpAfterUse: true,
		},
	}

	kconfig, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(kconfig).NotTo(BeNil())

	By("create clientsets")
	kClient, err = clientset.NewForConfig(kconfig)
	Expect(err).ToNot(HaveOccurred())

	netDefClient, err = netdefclient.NewForConfig(kconfig)
	Expect(err).ToNot(HaveOccurred())

	policyClient, err = multiclient.NewForConfig(kconfig)
	Expect(err).ToNot(HaveOccurred())

	By("create test node object")
	// create node object
	node := v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
		},
		Spec:   v1.NodeSpec{},
		Status: v1.NodeStatus{},
	}

	_, err = kClient.CoreV1().Nodes().Create(context.Background(), &node, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	By("removing test node object")
	Expect(kClient.CoreV1().Nodes().Delete(context.Background(), nodeName, metav1.DeleteOptions{})).ToNot(HaveOccurred())
	By("stopping test env")
	Expect(testEnv.Stop()).ToNot(HaveOccurred())
	klog.Flush()
})
