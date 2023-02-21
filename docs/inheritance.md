# Inheritance

An `OpenStackDataPlaneNode` inherits any attribute of an
`OpenStackDataPlaneRole` but those attributes may also be overridden
on the node level.

Suppose the following CR is created with `oc create -f
openstackdataplanerole-sample.yaml`:

```yaml
---
apiVersion: dataplane.openstack.org/v1beta1
kind: OpenStackDataPlaneRole
metadata:
  name: openstackdataplanerole-sample
spec:
  dataPlaneNodes:
  - name: openstackdataplanenode-sample-1
    ansibleHost: 192.168.122.18
    hostName: openstackdataplanenode-sample-1.localdomain
    node:
      networks:
      - fixedIP: 192.168.122.18
        network: ctlplane
  - name: openstackdataplanenode-sample-2
    ansibleHost: 192.168.122.19
    hostName: openstackdataplanenode-sample-2.localdomain
    node:
      networks:
      - fixedIP: 192.168.122.19
        network: ctlplane
  nodeTemplate:
    ansiblePort: 22
    ansibleUser: root
    managed: false
    managementNetwork: ctlplane
    networkConfig:
      template: templates/net_config_bridge.j2
```
Then two CRs like the following, representing two nodes, will be
created automatically by the dataplane-operator and there is no
need to `oc create` separate files containing the following.

Node 1
```yaml
---
apiVersion: dataplane.openstack.org/v1beta1
kind: OpenStackDataPlaneNode
metadata:
  name: openstackdataplanenode-sample-1
spec:
  ansibleHost: 192.168.122.18
  hostName: openstackdataplanenode-sample-1.localdomain
  node:
    networks:
    - fixedIP: 192.168.122.18
      network: ctlplane
  role: openstackdataplanerole-sample
```
Node 2
```yaml
---
apiVersion: dataplane.openstack.org/v1beta1
kind: OpenStackDataPlaneNode
metadata:
  name: openstackdataplanenode-sample-2
spec:
  hostName: openstackdataplanenode-sample-2.localdomain
  networks:
  node:
    ansibleHost: 192.168.122.19
    - fixedIP: 192.168.122.19
      network: ctlplane
  role: openstackdataplanerole-sample
```
Because of inheritance, redundant information did not need to be
provided to both nodes. Only the information which differed per node,
e.g. `ansibleHost`, had to be specified. Furthermore, the redundant
information is not seen in the two nodes' CRs. I.e. we do not see the
following from the `nodeTemplate` in node 1 and 2 above.
```
    ansiblePort: 22
    ansibleUser: root
    managed: false
    managementNetwork: ctlplane
    networkConfig:
      template: templates/net_config_bridge.j2
```
However, it's unambiguous that each node has `ansiblePort` 22
because they have `role: openstackdataplanerole-sample`. If the
node is inspected however, port 22 will be set.

The node controller resolves this dynamically by looking at
`role.nodeSpec` and we can assume that the value was inherited.
If we need to override a setting for any single node so that it
doesn't use it's `nodeTemplate`, then we may do so by directly
updating only that node (e.g. with `oc edit`). In that case we'd see
an `ansiblePort` set directly in that node's CR. This allows the user
to change the `nodeTemplate` after creation and once reconciliation is
completed all existing nodes will inherit the new value.
