# multi-networkpolicy-tc
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](http://www.apache.org/licenses/LICENSE-2.0)
[![Build](https://github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/actions/workflows/build.yml/badge.svg)](https://github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/actions/workflows/build.yml)
[![Test](https://github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/actions/workflows/test.yml/badge.svg)](https://github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/k8snetworkplumbingwg/multi-networkpolicy-tc)](https://goreportcard.com/report/github.com/k8snetworkplumbingwg/multi-networkpolicy-tc)
[![Coverage Status](https://coveralls.io/repos/github/k8snetworkplumbingwg/multi-networkpolicy-tc/badge.svg?branch=main)](https://coveralls.io/github/k8snetworkplumbingwg/multi-networkpolicy-tc?branch=main)


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
$ git clone https://github.com/k8snetworkplumbingwg/multi-networkpolicy-tc
$ cd multi-networkpolicy-tc
$ kubectl create -f deploy/crds/multi-net-crd.yaml
customresourcedefinition.apiextensions.k8s.io/multi-networkpolicies.k8s.cni.cncf.io created
```

Deploy multi-networkpolicy-tc into Kubernetes.

```
$ git clone https://github.com/k8snetworkplumbingwg/multi-networkpolicy-tc
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

## Configuration reference

The following configuration flags are supported by `multi-networkpolicy-tc`:
```
      --kubeconfig string                Path to kubeconfig file with authorization information (the master location is set by the master flag).
      --master string                    The address of the Kubernetes API server (overrides any value in kubeconfig)
      --hostname-override string         If non-empty, will use this string as identification instead of the actual hostname.
      --network-plugins strings          List of network plugins to be be considered for network policies. (default [accelerated-bridge])
      --pod-rules-path string            If non-empty, will use this path to store pod's rules for troubleshooting.
      --add_dir_header                   If true, adds the file directory to the header of the log messages
      --alsologtostderr                  log to standard error as well as files (no effect when -logtostderr=true)
      --log_backtrace_at traceLocation   when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                   If non-empty, write log files in this directory (no effect when -logtostderr=true)
      --log_file string                  If non-empty, use this log file (no effect when -logtostderr=true)
      --log_file_max_size uint           Defines the maximum size a log file can grow to (no effect when -logtostderr=true). Unit is megabytes. If the value is 0, the maximum file size is unlimited. (default 1800)
      --logtostderr                      log to standard error instead of files (default true)
      --one_output                       If true, only write logs to their native severity level (vs also writing to each lower severity level; no effect when -logtostderr=true)
      --skip_headers                     If true, avoid header prefixes in the log messages
      --skip_log_headers                 If true, avoid headers when opening log files (no effect when -logtostderr=true)
      --stderrthreshold severity         logs at or above this threshold go to stderr when writing to files and stderr (no effect when -logtostderr=true or -alsologtostderr=false) (default 2)
  -v, --v Level                          number for the log level verbosity
      --vmodule moduleSpec               comma-separated list of pattern=N settings for file-filtered logging
      --log-flush-frequency duration     Maximum number of seconds between log flushes (default 5s)
  -h, --help                             help for multi-networkpolicy-tc
```

## Limitations

As this project is under active development, there are several limitations which are planned to be addressed
in the future.

- MultiNetworkPolicy Ingress rules are not supported. Ingress policy will not be enforced

## Contributing

To report a bug or request a feature, open an issue in this repository.
to contribute to the project please refer to `CONTRIBUTING.md` doc
