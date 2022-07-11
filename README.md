# multi-networkpolicy-tc
[multi-networkpolicy](https://github.com/k8snetworkplumbingwg/multi-networkpolicy) implementation
using [Linux Traffic Control (TC)](https://tldp.org/HOWTO/Traffic-Control-HOWTO/intro.html)

## Description

Kubernetes provides [Network Policies](https://kubernetes.io/docs/concepts/services-networking/network-policies/)
for network security.
MultiNetworkPolicy defines an API similar to Kubernetes built-in NetworkPolicy API for secondary kubernetes networks
defined via [NetworkAttachmentDefinition CRD](https://github.com/k8snetworkplumbingwg/multi-net-spec).
multi-networkpolicy-tc implements MultiNetworkPolicy API using Linux TC, providing network
security for net-attach-def networks.

## Supported CNIs

multi-networkpolicy-tc is intended to be used with networks provided via [accelerated bridge cni](https://github.com/k8snetworkplumbingwg/accelerated-bridge-cni).
it is currently not compatible with other CNIs however support may be extended for additional CNIs.

multi-networkpolicy-tc relies on the fact that a Pod has an SRIOV VF allocated for the network with a corresponding VF representor netdev
which follows the kernel [switchdev model](https://www.kernel.org/doc/html/latest/networking/switchdev.html).

given a MultiNetworkPolicy it generates and programs TC rules to enforce the policy.
for more information refer to `docs/tc-rule-pipeline.md`.

## Prerequisites

- Linux kernel 5.17.9 or newer
- NIC supporting switchdev and TC hardware offload such as:
  - Nvidia Mellanox ConnectX-6Dx

## Quickstart

### Build

This project uses go modules for dependency management and requires Go 1.18 to build.

to build binary run:
```shell
$ make build
```
Binary executable is located under `build` folder

### Install

Install MultiNetworkPolicy CRD into Kubernetes.

```
$ git clone https://github.com/Mellanox/multi-networkpolicy-tc
$ cd multi-networkpolicy-tc
$ kubectl create -f deploy/multi-net-crd.yaml
customresourcedefinition.apiextensions.k8s.io/multi-networkpolicies.k8s.cni.cncf.io created
```

Deploy multi-networkpolicy-tc into Kubernetes.

```
$ git clone https://github.com/Mellanox/multi-networkpolicy-tc
$ cd multi-networkpolicy-tc
$ kubectl create -f deploy/deploy.yml
clusterrole.rbac.authorization.k8s.io/multi-networkpolicy created
clusterrolebinding.rbac.authorization.k8s.io/multi-networkpolicy created
serviceaccount/multi-networkpolicy created
daemonset.apps/multi-networkpolicy-ds-amd64 created
```

## multi-network-policy-tc DaemonSet

multi-network-policy-tc runs as a daemonset on each node.
`multi-networkpolicy-tc` watches MultiNetworkPolicy object and creates TC rules on VF representor to filters packets
 to/from interface, based on MultiNetworkPolicy.

## Limitations

As this project is under active development, there are several limitations which are planned to be addressed
in the near future.

- MultiNetworkPolicy Ingress rules are not supported. Ingress policy will not be enforced
- VLAN tagged traffic is not supported network policy will not be enforced
- QinQ traffic is not supported network policy will not be enforced
- IPV6 traffic is not supported network policy will not be enforced

## Contributing

To report a bug or request a feature, open an issue in this repository.
to contribute to the project please refer to `CONTRIBUTING.md` doc
