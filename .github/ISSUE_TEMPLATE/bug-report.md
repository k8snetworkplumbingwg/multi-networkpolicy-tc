---
name: Bug Report
about: Report a bug with multi-networkpolicy-tc

---
<!-- Please use this template while reporting a bug and provide as much relevant info as possible. Doing so give us the best chance to find a prompt resolution to your issue -->

### What happened?

### What did you expect to happen?

### What are the minimal steps needed to reproduce the bug?

### Anything else we need to know?

### Component Versions
Please fill in the below table with the version numbers of components used.

| Component              | Version              |
|------------------------|----------------------|
| Multi-networkpolicy-tc | <Input Version Here> |
| accelerated-bridge CNI | <Input Version Here> |
| Kubernetes             | <Input Version Here> | 
| OS                     | <Input Version Here> |
| Kernel                 | <Input Version Here> |

### Kubernetes output

#### CNI configuration for secondary network (NetworkAttachmentDefinition)

#### Workload Pod Spec

#### MultiNetworkPolicy CRs in cluster

##### multi-networkpolicy-tc logs (use `kubectl logs $PODNAME`)

### System output
#### IP link output:
- `ip link show`

#### bridge command output:
- `bridge link show`
- `bridge vlan show`

#### tc command output (on affected interface)
- `tc filter show dev <vf-rep> ingress`
