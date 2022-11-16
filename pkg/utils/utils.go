package utils

import (
	"fmt"
	"net"
	"os"
	"strings"

	multiv1beta1 "github.com/k8snetworkplumbingwg/multi-networkpolicy/pkg/apis/k8s.cni.cncf.io/v1beta1"
	netdefv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	v1 "k8s.io/api/core/v1"
)

// PolicyNetworkAnnotation is annotation for multiNetworkPolicy,
// to specify which networks(i.e. net-attach-def) are the targets
// of the policy
const PolicyNetworkAnnotation = "k8s.v1.cni.cncf.io/policy-for"

// CheckNodeNameIdentical checks both strings point a same node
// it just checks hostname without domain
func CheckNodeNameIdentical(s1, s2 string) bool {
	return strings.Split(s1, ".")[0] == strings.Split(s2, ".")[0]
}

// GetHostname returns OS's hostname if 'hostnameOverride' is empty; otherwise, return 'hostnameOverride'.
func GetHostname(hostnameOverride string) (string, error) {
	hostName := hostnameOverride
	if len(hostName) == 0 {
		var err error
		hostName, err = os.Hostname()
		if err != nil {
			return "", fmt.Errorf("couldn't determine hostname: %w", err)
		}
	}

	// Trim whitespaces first to avoid getting an empty hostname
	// For linux, the hostname is read from file /proc/sys/kernel/hostname directly
	hostName = strings.TrimSpace(hostName)
	if len(hostName) == 0 {
		return "", fmt.Errorf("empty hostname is invalid")
	}
	return strings.ToLower(hostName), nil
}

// IsMultiNetworkpolicyTarget checks if pod is in running phase and is not hostNetwork
func IsMultiNetworkpolicyTarget(pod *v1.Pod) bool {
	return pod.Status.Phase == v1.PodRunning && !pod.Spec.HostNetwork
}

// NetworkListFromPolicy returns a list of networks which apply to the provided MultiNetworkPolicy
func NetworkListFromPolicy(policy *multiv1beta1.MultiNetworkPolicy) []string {
	policyNetworksAnnot, ok := policy.GetAnnotations()[PolicyNetworkAnnotation]
	if !ok {
		return []string{}
	}
	policyNetworksAnnot = strings.ReplaceAll(policyNetworksAnnot, " ", "")
	if policyNetworksAnnot == "" {
		return []string{}
	}

	policyNetworks := strings.Split(policyNetworksAnnot, ",")
	for idx, policyNetName := range policyNetworks {
		if policyNetName == "" {
			continue
		}
		// fill namespace
		if !strings.ContainsAny(policyNetName, "/") {
			policyNetworks[idx] = fmt.Sprintf("%s/%s", policy.Namespace, policyNetName)
		}
	}
	return policyNetworks
}

// GetDeviceIDFromNetworkStatus returns the PCI device ID associated with provided NetworkStatus
func GetDeviceIDFromNetworkStatus(status netdefv1.NetworkStatus) (string, error) {
	if status.DeviceInfo == nil {
		return "", fmt.Errorf("device-info field not set in network status")
	}

	if status.DeviceInfo.Type != netdefv1.DeviceInfoTypePCI {
		return "", fmt.Errorf("device info type is not PCI, it is %s", status.DeviceInfo.Type)
	}

	if status.DeviceInfo.Pci == nil {
		return "", fmt.Errorf("unexpected error, device info pci field is empty")
	}

	if status.DeviceInfo.Pci.PciAddress == "" {
		return "", fmt.Errorf("unexpected error, device info pci address is empty")
	}

	return status.DeviceInfo.Pci.PciAddress, nil
}

// IPsFromStrings receives a list of IPs in string format and returns a list of net.IP
// invalid IPs will be nil net.IP
func IPsFromStrings(ips []string) []net.IP {
	netIPs := make([]net.IP, 0, len(ips))
	for _, ip := range ips {
		netIPs = append(netIPs, net.ParseIP(ip))
	}
	return netIPs
}

// IsIPv4 returns true if IP is of type IPV4
func IsIPv4(ip net.IP) bool {
	// Note(adrianc): when Creating net.IP using net package e.g via net.ParseIP() it creates
	// IP with a fixed size of net.IPv6Len, so we cannot rely on length.
	return ip.To4() != nil
}

// PathExists returns true if path exists in the system or false if it doesnt
// in case of error, and error is returned
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// IsMaskFull returns true if provided mask is all 1s (e.g in CIDR notation, for IPv4 mask its a /32 mask,
// and for IPv6 mask it is /128
func IsMaskFull(mask net.IPMask) bool {
	ones, bits := mask.Size()
	return ones == bits
}

// IPToIPNet coverts IP or CIDR formatted string to *net.IPNet.
// if no CIDR notation, then /32 or /128 mask is assumed for ipv4 and ipv6 respectively.
func IPToIPNet(ip string) (*net.IPNet, error) {
	if !strings.Contains(ip, "/") {
		ipp := net.ParseIP(ip)
		if ipp == nil {
			return nil, fmt.Errorf("failed to parse ip: %s", ip)
		}
		if ipp.To4() != nil {
			ip += "/32"
		} else {
			ip += "/128"
		}
	}
	_, ipn, err := net.ParseCIDR(ip)
	return ipn, err
}
