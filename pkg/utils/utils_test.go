package utils_test

import (
	"net"
	"os"

	multiv1beta1 "github.com/k8snetworkplumbingwg/multi-networkpolicy/pkg/apis/k8s.cni.cncf.io/v1beta1"
	netdefv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Mellanox/multi-networkpolicy-tc/pkg/utils"
)

var _ = Describe("utils test", func() {
	Context("CheckNodeNameIdentical()", func() {
		It("check node name identical with domain", func() {
			h1 := "my-host.some-lab.some-site"
			h2 := "my-host.some-lab-alt-name.some-site-alt-name"
			Expect(utils.CheckNodeNameIdentical(h1, h2)).To(BeTrue())
		})
		It("check node name identical with domain on first arg", func() {
			h1 := "my-host.some-lab.some-site"
			h2 := "my-host"
			Expect(utils.CheckNodeNameIdentical(h1, h2)).To(BeTrue())
		})
		It("check node name identical with domain on second arg", func() {
			h1 := "my-host"
			h2 := "my-host.some-lab.some-site"
			Expect(utils.CheckNodeNameIdentical(h1, h2)).To(BeTrue())
		})
		It("check node name identical with same domain", func() {
			h1 := "my-host.some-lab.some-site"
			h2 := "my-host.some-lab.some-site"
			Expect(utils.CheckNodeNameIdentical(h1, h2)).To(BeTrue())
		})
		It("check node name identical without domain", func() {
			h1 := "my-host"
			h2 := "my-host"
			Expect(utils.CheckNodeNameIdentical(h1, h2)).To(BeTrue())
		})
		It("check node name not identical with domain", func() {
			h1 := "my-host.some-lab.some-site"
			h2 := "my-other-host.some-lab.some-site"
			Expect(utils.CheckNodeNameIdentical(h1, h2)).To(BeFalse())
		})
		It("check node name not identical without domain", func() {
			h1 := "my-host"
			h2 := "my-other-host"
			Expect(utils.CheckNodeNameIdentical(h1, h2)).To(BeFalse())
		})
	})

	Context("GetHostname()", func() {
		It("Gets hostname no override", func() {
			host, err := os.Hostname()
			Expect(err).ToNot(HaveOccurred())
			Expect(utils.GetHostname("")).To(Equal(host))
		})
		It("Gets hostname with override", func() {
			dummyHost := "my-dummy-host"
			Expect(utils.GetHostname(dummyHost)).To(Equal(dummyHost))
		})
	})

	Context("IsMultiNetworkpolicyTarget()", func() {
		createPodFn := func(phase v1.PodPhase, hostNetwork bool) *v1.Pod {
			return &v1.Pod{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{},
				Spec: v1.PodSpec{
					HostNetwork: hostNetwork,
				},
				Status: v1.PodStatus{
					Phase: phase,
				},
			}
		}

		It("returns true if pod is running and is not host network", func() {
			pod := createPodFn(v1.PodRunning, false)
			Expect(utils.IsMultiNetworkpolicyTarget(pod)).To(BeTrue())
		})
		It("returns false if pod is not running", func() {
			pod := createPodFn(v1.PodPending, false)
			Expect(utils.IsMultiNetworkpolicyTarget(pod)).To(BeFalse())
			pod = createPodFn(v1.PodFailed, false)
			Expect(utils.IsMultiNetworkpolicyTarget(pod)).To(BeFalse())
			pod = createPodFn(v1.PodSucceeded, false)
			Expect(utils.IsMultiNetworkpolicyTarget(pod)).To(BeFalse())
		})
		It("returns false if pod is host network", func() {
			pod := createPodFn(v1.PodRunning, true)
			Expect(utils.IsMultiNetworkpolicyTarget(pod)).To(BeFalse())
		})
	})

	Context("NetworkListFromPolicy()", func() {
		createPolicyFn := func(name string, namespace string, policyForAnnot *string) *multiv1beta1.MultiNetworkPolicy {
			policy := &multiv1beta1.MultiNetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
			}
			if policyForAnnot != nil {
				policy.Annotations = map[string]string{utils.PolicyNetworkAnnotation: *policyForAnnot}
			}

			return policy
		}

		It("returns namespaced network with policy namespace if namespace not provided for Network", func() {
			annot := "accel-net1"
			p := createPolicyFn("my-policy", "my-ns", &annot)
			nets := utils.NetworkListFromPolicy(p)
			Expect(nets).To(HaveLen(1))
			Expect(nets[0]).To(Equal("my-ns/accel-net1"))
		})
		It("returns namespaced network with given namespace if namespace provided for Network", func() {
			annot := "default/accel-net1"
			p := createPolicyFn("my-policy", "my-ns", &annot)
			nets := utils.NetworkListFromPolicy(p)
			Expect(nets).To(HaveLen(1))
			Expect(nets[0]).To(Equal(annot))
		})
		It("returns namespaced networks", func() {
			annot := "default/accel-net1, accel-net2"
			p := createPolicyFn("my-policy", "my-ns", &annot)
			nets := utils.NetworkListFromPolicy(p)
			Expect(nets).To(HaveLen(2))
			Expect(nets[0]).To(Equal("default/accel-net1"))
			Expect(nets[1]).To(Equal("my-ns/accel-net2"))
		})
		It("returns no networks if no network annotation", func() {
			p := createPolicyFn("my-policy", "my-ns", nil)
			nets := utils.NetworkListFromPolicy(p)
			Expect(nets).To(BeEmpty())
		})
		It("returns no networks if empty network annotation", func() {
			annot := ""
			p := createPolicyFn("my-policy", "my-ns", &annot)
			nets := utils.NetworkListFromPolicy(p)
			Expect(nets).To(BeEmpty())
		})
		It("returns no networks if network annotation has only spaces", func() {
			annot := "   "
			p := createPolicyFn("my-policy", "my-ns", &annot)
			nets := utils.NetworkListFromPolicy(p)
			Expect(nets).To(BeEmpty())
		})
	})

	Context("GetDeviceIDFromNetworkStatus()", func() {
		It("returns device ID from device information field for PCI device type", func() {
			status := netdefv1.NetworkStatus{
				DeviceInfo: &netdefv1.DeviceInfo{
					Type:    netdefv1.DeviceInfoTypePCI,
					Version: "1.0.0",
					Pci: &netdefv1.PciDevice{
						PciAddress: "0000:03:01.1",
					},
				},
			}
			devid, err := utils.GetDeviceIDFromNetworkStatus(status)
			Expect(err).ToNot(HaveOccurred())
			Expect(devid).To(Equal("0000:03:01.1"))
		})
		It("returns error if no DeviceInfo in NetworkStatus", func() {
			status := netdefv1.NetworkStatus{
				DeviceInfo: nil,
			}
			_, err := utils.GetDeviceIDFromNetworkStatus(status)
			Expect(err).To(HaveOccurred())
		})
		It("returns error if DeviceInfo is not of type PCI", func() {
			status := netdefv1.NetworkStatus{
				DeviceInfo: &netdefv1.DeviceInfo{
					Type:    netdefv1.DeviceInfoTypeMemif,
					Version: "1.0.0",
					Memif:   &netdefv1.MemifDevice{},
				},
			}
			_, err := utils.GetDeviceIDFromNetworkStatus(status)
			Expect(err).To(HaveOccurred())
		})
		It("returns error if DeviceInfo is of type PCI but Pci is nil", func() {
			status := netdefv1.NetworkStatus{
				DeviceInfo: &netdefv1.DeviceInfo{
					Type:    netdefv1.DeviceInfoTypePCI,
					Version: "1.0.0",
					Pci:     nil,
				},
			}
			_, err := utils.GetDeviceIDFromNetworkStatus(status)
			Expect(err).To(HaveOccurred())
		})
		It("returns error if DeviceInfo is of type PCI but Pci.PciAddress is empty", func() {
			status := netdefv1.NetworkStatus{
				DeviceInfo: &netdefv1.DeviceInfo{
					Type:    netdefv1.DeviceInfoTypePCI,
					Version: "1.0.0",
					Pci: &netdefv1.PciDevice{
						PciAddress: "",
					},
				},
			}
			_, err := utils.GetDeviceIDFromNetworkStatus(status)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("IPsFromStrings()", func() {
		It("Successfully parses IPv4 IPs", func() {
			ips := []string{"10.10.1.1", "10.10.1.2"}
			parsedIPs := utils.IPsFromStrings(ips)
			Expect(parsedIPs).To(HaveLen(2))
			for i := range parsedIPs {
				Expect(parsedIPs[i].String()).To(Equal(ips[i]))
			}
		})
		It("Successfully parses IPv6 IPs", func() {
			ips := []string{"2001:db8:85a3:abfa:afba:8a2e:370:3333", "2001:db8:85a3:abfa:afba:8a2e:370:4444"}
			parsedIPs := utils.IPsFromStrings(ips)
			for i := range parsedIPs {
				Expect(parsedIPs[i].String()).To(Equal(ips[i]))
			}
		})
		It("returns empty list if no IPs", func() {
			ips := []string{}
			parsedIPs := utils.IPsFromStrings(ips)
			Expect(parsedIPs).To(BeEmpty())
		})
		It("returns list with nil as IPs if they are in bad format", func() {
			ips := []string{"", " ", "invalid"}
			parsedIPs := utils.IPsFromStrings(ips)
			var nilIP net.IP = nil
			Expect(parsedIPs).To(HaveLen(3))
			for i := range parsedIPs {
				Expect(parsedIPs[i]).To(Equal(nilIP))
			}
		})
	})

	Context("IsIPv4()", func() {
		It("returns true for IPv4 IP", func() {
			ip := net.ParseIP("10.10.1.1")
			Expect(utils.IsIPv4(ip)).To(BeTrue())
		})
		It("returns false for IPv6 IP", func() {
			ip := net.ParseIP("2001:0db8:85a3:0000:0000:8a2e:0370:3333")
			Expect(utils.IsIPv4(ip)).To(BeFalse())
		})
		It("returns false for nil IP", func() {
			var ip net.IP
			Expect(utils.IsIPv4(ip)).To(BeFalse())
		})
	})
})
