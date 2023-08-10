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
  ansibleHost: 192.168.122.19
  hostName: openstackdataplanenode-sample-2.localdomain
  node:
    networks:
    - fixedIP: 192.168.122.19
      network: ctlplane
  role: openstackdataplanerole-sample
```

Because of inheritance, redundant information did not need to be
provided to both nodes. Only the information which differed per node,
e.g. `ansibleHost`, had to be specified. Furthermore, the redundant
information is not seen in the two nodes' CRs. I.e. we do not see the
following from the `nodeTemplate` in node 1 and 2 above.

```yaml
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

Almost any top level property in a node overrides the whole property
in a role. E.g. if the role `nodeTemplate` had a list like the
following:

```
    foo:
      - bar: baz
```

and a node had a list like the following:

```
    foo:
      - qux: quux
```

Then the node will only have `{"qux": "quux"}` in its list. I.e. there
would not be any merging and the node would not also have `{"bar":
"baz"}` in its list. The one exception to this rule is the
`AnsibleVars` property. If the following CRs are created:

```yaml
kind: OpenStackDataPlaneRole
metadata:
  name: edpm-compute
spec:
  nodeTemplate:
    ansibleVars:
      edpm_network_config_interface_name: eth0
      edpm_chrony_ntp_servers:
        - clock.redhat.com
        - clock2.redhat.com
  ...
```

```yaml
kind: OpenStackDataPlaneNode
metadata:
  name: edpm-compute-0
spec:
  role: edpm-compute
  node:
    ansibleVars:
      tenant_ip: 192.168.24.100
  ...
```

The `ConfigMap` containing the inventory would have the following
vars:

```yaml
      edpm_network_config_interface_name: eth0
      tenant_ip: 192.168.24.100
      edpm_chrony_ntp_servers:
        - clock.redhat.com
        - clock2.redhat.com
```

Only top level keys are compared for merging. E.g. if `edpm-compute-0`
had a `edpm_chrony_ntp_servers` list with `clock3.redhat.com`, then
the resultant inventory for the node would not have three NTP servers;
it would only have `clock3.redhat.com`. I.e. there is no "deep
merging".

It's also possible to create a node directly outside of a role CR
and define its role. If the following CR is created:

```yaml
---
apiVersion: dataplane.openstack.org/v1beta1
kind: OpenStackDataPlaneNode
metadata:
  name: openstackdataplanenode-sample-3-from
spec:
  role: openstackdataplanerole-sample
  hostName: openstackdataplanenode-sample-3.localdomain
  ansibleHost: 192.168.122.20
  node:
    networks:
      - network: ctlplane
        fixedIP: 192.168.122.20
```

After the above CR is created, the node
openstackdataplanenode-sample-3-from may then be inspected further
using a command like
`oc get OpenStackDataPlaneNode openstackdataplanenode-sample-3-from -o
yaml` which should show that it inherited values from the role
`nodeTemplate`. In cases like these, the `dataPlaneNodes` list will
not reflect all of the nodes within the role.
