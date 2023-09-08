# Deploying a DataPlaneNodeSet

Deploying a dataplane consists of creating the OpenStackDataPlaneNodeSet custom resource that
define the layout of the dataplane.

This documentation will cover using each resource individually, as well as
using the OpenStackDataPlaneNodeSet resource to deploy everything in a single
resource.

## Samples

The
[config/samples](https://github.com/openstack-k8s-operators/dataplane-operator/tree/main/config/samples)
directory contains many sample custom resources that illustrate different
dataplane deployments. These can be used as a starting point and customized for
a given deployment, or a resource can be entirely written from no sample.

## Prerequisites

### ControlPlane

A functional dataplane requires a functional controlplane deployed by
[openstack-operator](https://github.com/openstack-k8s-operators/openstack-operator).
This documentation will make use of resources within the controlplane to
correctly customize the dataplane configuration. The docs here do not cover
deploying a controlplane and assume that has already been completed separately.

### Configured namespace (openstack)

The `oc` commands shown in the documentation assume that the `oc` client has
been configured to use the correct namespace for an OpenStack deployment. By
default, the namespace is `openstack`.

### Review the available fields on the dataplane CRD's

Further documentation on each field available in the dataplane CRD's is
available under the Custom Resources documentation section. This deployment
section does not go into full detail about each available field.

## Deploying a dataplane using pre-provisioned nodes

This section documents using pre-provisioned nodes in the dataplane.
Pre-provisioned nodes already have their OS (operating system) installed and
are powered on and booted into the installed OS.

### Create SSH key secret

Pre-provisioned nodes must be pre-configured with an SSH public key in the
`$HOME/.ssh/authorized_keys` file for a user that has password-less sudo
privileges. Ansible will make use of this user and SSH key when it executes.

The private key for the SSH keypair is created as a Secret in the cluster. Set
the environment variables shown with the name of the secret and the path to the
SSH private key. Run the shown command to create the Secret in the cluster.

    # Name of the secret that will be created in the cluster
    SECRET_NAME="dataplane-ansible-ssh-private-key-secret"
    # Path of the SSH private key file. Public key file should also exist at
    # the same path, but with a ".pub" extension.
    SSH_KEY_FILE="ansibleee-ssh-key-id_rsa"

    oc create secret generic ${SECRET_NAME} \
    --save-config \
    --dry-run=client \
    --from-file=authorized_keys=${SSH_KEY_FILE}.pub \
    --from-file=ssh-privatekey=${SSH_KEY_FILE} \
    --from-file=ssh-publickey=${SSH_KEY_FILE}.pub \
    -o yaml | \
    oc apply -f -

Verify the secret was created:

    oc describe secret dataplane-ansible-ssh-private-key-secret

### Create OpenStackDataPlaneNodeSet

This document will cover writing the `YAML` document for an
`OpenStackDataPlaneNodeSet` resource. Once the document is ready, it will be created
with `oc` as the last step.

Start the `YAML` document in an `openstack-edpm.yaml` file and give the
dataplane a name.

    apiVersion: dataplane.openstack.org/v1beta1
    kind: OpenStackDataPlaneNodeSet
    metadata:
      name: openstack-edpm

Begin writing the dataplane spec. Initially, a `deployStrategy` field will be
added to the spec that contains `deploy: false`. This allows for creating
the dataplane resources without triggering an Ansible execution immediately.

    apiVersion: dataplane.openstack.org/v1beta1
    kind: OpenStackDataPlaneNodeSet
    metadata:
      name: openstack-edpm
    spec:
      deployStrategy:
          deploy: false

Add roles to the dataplane. This example uses a single role, but any number
could be added. A single role called `dataplane-role` is added for this
example. Under the role, a `preProvisioned` field is set to `True` since these
nodes are preprovisioned. A `nodeTemplate` field is also started that contains
the fields that will have their values inherited by each node in the role. See
[Inheritance](inheritance.md) for more details about how role and node
inheritance works. Within `nodeTemplate`, the fields shown are documented
inline in the example.

    apiVersion: dataplane.openstack.org/v1beta1
    kind: OpenStackDataPlaneNodeSet
    metadata:
      name: openstack-edpm
    spec:
      deployStrategy:
        deploy: false
      roles:
        edpm-compute:
          preProvisioned: true
          nodeTemplate:

            # User that has the SSH key for access
            ansibleUser: rhel-user
            # Secret name containing SSH key. Use the same secret name as
            # ${SECRET_NAME} that was used to create the secret.
            ansibleSSHPrivateKeySecret: dataplane-ansible-ssh-private-key-secret

            # Ansible variables that configure how the roles from edpm-ansible
            # customize the deployment.
            ansibleVars:

              # edpm_network_config
              # Default nic config template for a EDPM compute node
              # These vars are edpm_network_config role vars
              edpm_network_config_template: templates/single_nic_vlans/single_nic_vlans.j2

              # See config/samples/dataplane_v1beta1_openstackdataplanenodeset.yaml
              # for the other most common ansible varialbes that need to be set.

The list of ansible variables that can be set under `ansibleVars` is extensive.
To understand what variables are available for each service, see the
documentation in the [Create
OpenStackDataPlaneServices](#create-openstackdataplaneservices) section.

Common configurations that can be enabled with `ansibleVars` are also
documented at [Common Configurations](common_configurations.md).

Some of the ansible variables will need to be set based on values from the
controlplane that is already deployed. This set of ansible variables and the
`oc` command that can be used to get their values are shown below.

```console
export EDPM_OVN_METADATA_AGENT_TRANSPORT_URL=$(oc get secret rabbitmq-transport-url-neutron-neutron-transport -o json | jq -r .data.transport_url | base64 -d)
export EDPM_OVN_METADATA_AGENT_SB_CONNECTION=$(oc get ovndbcluster ovndbcluster-sb -o json | jq -r .status.dbAddress)
export EDPM_OVN_METADATA_AGENT_NOVA_METADATA_HOST=$(oc get svc nova-metadata-internal -o json |jq -r '.status.loadBalancer.ingress[0].ip')
export EDPM_OVN_METADATA_AGENT_PROXY_SHARED_SECRET=$(oc get secret osp-secret -o json | jq -r .data.MetadataSecret  | base64 -d)
export EDPM_OVN_METADATA_AGENT_BIND_HOST=127.0.0.1
export EDPM_OVN_DBS=$(oc get ovndbcluster ovndbcluster-sb -o json | jq -r '.status.networkAttachments."openstack/internalapi"')

echo "
edpm_ovn_metadata_agent_DEFAULT_transport_url: ${EDPM_OVN_METADATA_AGENT_TRANSPORT_URL}
edpm_ovn_metadata_agent_metadata_agent_ovn_ovn_sb_connection: ${EDPM_OVN_METADATA_AGENT_SB_CONNECTION}
edpm_ovn_metadata_agent_metadata_agent_DEFAULT_nova_metadata_host: ${EDPM_OVN_METADATA_AGENT_NOVA_METADATA_HOST}
edpm_ovn_metadata_agent_metadata_agent_DEFAULT_metadata_proxy_shared_secret: ${EDPM_OVN_METADATA_AGENT_PROXY_SHARED_SECRET}
edpm_ovn_metadata_agent_DEFAULT_bind_host: ${EDPM_OVN_METADATA_AGENT_BIND_HOST}
edpm_ovn_dbs: ${EDPM_OVN_DBS}
"
```

Add the output to the `ansibleVars` field to configure the values on the
role.

Add nodes to the dataplane. Each node should have its `role` field set to the
name of its role. Since we are using a single role in this example, that role
name will be `edpm-compute`. Each node will also inherit values
from the `nodeTemplate` field on its role into the `node` field on the node.
However, certain fields will need to be overridden given that they are specific
to a node. In this example, `ansibleVars` has the node specific variables.

---
**NOTE**

In the case of `ansibleVars`, the value is merged with that of the value from
the role. This makes it so that the entire value of `ansibleVars` from the role
does not need to be reproduced for each node just to set a few node specific
values.

---

With the nodes and the controlplane specific variables added, the full
`openstack-datplane` `YAML` document looks like the following:

    apiVersion: dataplane.openstack.org/v1beta1
    kind: OpenStackDataPlaneNodeSet
    metadata:
      name: openstack-edpm
    spec:
      deployStrategy:
        deploy: false
      roles:
        edpm-compute:
          preProvisioned: true
          nodeTemplate:

            # User that has the SSH key for access
            ansibleUser: rhel-user
            # Secret name containing SSH key. Use the same secret name as
            # ${SECRET_NAME} that was used to create the secret.
            ansibleSSHPrivateKeySecret: dataplane-ansible-ssh-private-key-secret

            # Ansible variables that configure how the roles from edpm-ansible
            # customize the deployment.
            ansibleVars:

              # edpm_network_config
              # Default nic config template for a EDPM compute node
              # These vars are edpm_network_config role vars
              edpm_network_config_template: templates/single_nic_vlans/single_nic_vlans.j2

              # Variables set with values from the controlplane
              edpm_ovn_metadata_agent_default_transport_url: rabbit://default_user@rabbitmq.openstack.svc:5672
              edpm_ovn_metadata_agent_metadata_agent_ovn_ovn_sb_connection: tcp:10.217.5.121:6642
              edpm_ovn_metadata_agent_metadata_agent_DEFAULT_nova_metadata_host: 127.0.0.1
              edpm_ovn_metadata_agent_metadata_agent_DEFAULT_metadata_proxy_shared_secret: 12345678
              edpm_ovn_metadata_agent_default_bind_host: 127.0.0.1
              edpm_ovn_dbs:
              - 192.168.24.1

              # See config/samples/dataplane_v1beta1_openstackdataplanenodeset.yaml
              # for the other most common ansible varialbes that need to be set.

      nodes:
        edpm-compute-0:
          role: edpm-compute
          hostName: edpm-compute-0
          ansibleHost: 192.168.122.100
          node:
            ansibleVars:
              ctlplane_ip: 192.168.122.100
              internal_api_ip: 172.17.0.100
              storage_ip: 172.18.0.100
              tenant_ip: 172.19.0.100
              fqdn_internal_api: edpm-compute-0.example.com
        edpm-compute-1:
          role: edpm-compute
          hostName: edpm-compute-1
          ansibleHost: 192.168.122.101
          node:
            ansibleVars:
              ctlplane_ip: 192.168.122.101
              internal_api_ip: 172.17.0.101
              storage_ip: 172.18.0.101
              tenant_ip: 172.19.0.101
              fqdn_internal_api: edpm-compute-1.example.com

Create the dataplane using the `oc` command.

    oc create -f openstack-edpm.yaml

Verify that the dataplane nodeset were created.

    oc get openstackdataplanenodeset

The output should be similar to:

```console
$ oc get openstackdataplanenodeset
NAME             STATUS   MESSAGE
openstack-edpm   False    Deployment not started
```

### Understanding OpenStackDataPlaneServices

A dataplane is configured with a set of services that define the Ansible roles
and task files that are executed to complete the deployment. The
dataplane-operator has a default list of services that are deployed by default
(unless the `services` field is overridden). The default services are provided
within the
[config/services](https://github.com/openstack-k8s-operators/dataplane-operator/tree/main/config/services)
directory.

Each service is a custom resource of type
[OpenStackDataPlaneService](openstack_dataplaneservice.md). The services will
be created and updated automatically during OpenStackDataPlaneRole reconciliation.

See [Composable Services](composable_services.md) for further documentation
about services and customizing services.

Verify the services were created.

    oc get openstackdataplaneservice

The output should be similar to:

    NAME                AGE
    download-cache      6d7h
    configure-network   6d7h
    configure-os        6d6h
    install-os          6d6h
    run-os              6d6h
    validate-network    6d6h
    libvirt             6d6h
    nova                6d6h

Each service uses the
[`role`](https://openstack-k8s-operators.github.io/openstack-ansibleee-operator/openstack_ansibleee/#role)
field from the `OpenStackAnsibleEE` CRD provided
by
[openstack-ansibleee-operator](https://github.com/openstack-k8s-operators/openstack-ansibleee-operator)
to define the Ansible roles and task files that are executed as part of that
service.

For example, the list of roles for the `install-os` service can be seen by
describing the resource.

    oc describe openstackdataplaneservice install-os

Any role listed in the `osp.edpm` namespace is provided by the
[edpm-ansible](https://github.com/openstack-k8s-operators/edpm-ansible)
project. Within that project, the ansible variables that can be used to
configure the role are documented.

For example, in the describe output for the `install-os` service, the
`osp.edpm.edpm_sshd` role is seen.

```console
import_role:
  Name:        osp.edpm.edpm_sshd
  tasks_from:  install.yml
Name:          Install edpm_sshd
Tags:
  edpm_sshd
```

The ansible variables that configure the
behavior of the `osp.edpm.edpm_sshd` role are available at
<https://github.com/openstack-k8s-operators/edpm-ansible/blob/main/roles/edpm_sshd/tasks/main.yml>.

---
**NOTE**

If the default provided services are edited, those edits will be lost after any
further role reconciliations.

---

### Deploy the dataplane

With the dataplane resources created, it can be seen from their status message
that they have not yet been deployed. This means no ansible has been executed
to configure any of the services on the nodes. They still need to be deployed.

To deploy the `openstack-edpm` dataplane resource, the
`spec.deployStrategy.deploy` field needs to be set to `True`. This will trigger
the deployment of all the configured services across the nodes. The field can
be set with the following command to start the deployment:

    oc patch openstackdataplanenodeset openstack-edpm  -p='[{"op": "replace", "path": "/spec/deployStrategy/deploy", "value":true}]' --type json

The `oc patch` command sets the `deploy` field to `True`, which starts the
deployment. `oc edit OpenStackDataPlaneNodeSet openstack-edpm` could alternatively be
used to edit the resource directly in an editor to set the field to `True`.

With the deployment started, ansible will be executed to configure the nodes.
When the deployment is complete, the status messages will change to indicate
the deployment is ready.

```console
$ oc get openstackdataplanenodeset
NAME             STATUS   MESSAGE
openstack-edpm   True    DataPlane Ready
```

If the deployment involved adding new compute nodes then after the deployment
is ready those compute nodes need to be mapped in nova. To do that run the
following command:
```console
oc rsh nova-cell0-conductor-0 nova-manage cell_v2 discover_hosts --verbose
```

### Understanding dataplane conditions

Each dataplane resource has a series of conditions within their `status`
subresource that indicate the overall state of the resource, including its
deployment progress.

`OpenStackDataPlaneNodeSet` resource conditions:

```console
$ oc get openstackdataplanenodeset openstack-edpm -o json | jq .status.conditions[].type
"Ready"
"DeploymentReady"
"SetupReady"
```

Each resource has a `Ready`, `DeploymentReady`, and `SetupReady` conditions.
The role and node also have a condition for each service that is being
deployed.

#### Condition Progress

The `Ready` condition reflects the latest condition state that has changed.
Until a deployment has been started and finished successfully, the `Ready`
condition will be `False`. When the deployment succeeds, it will be set to
`True`. A subsequent deployment that is started will set the condition back to
`False` until the deployment succeeds when it will be set back to `True`.

`SetupReady` will be set to `True` once all setup related tasks for a resource
are complete. Setup related tasks include verifying the SSH key secret and
verifying other fields on the resource, as well as creating the Ansible
inventory for each resource.

Each service specific condition will be set to `True` as that service completes
successfully. Looking at the service conditions will indicate which services
have completed their deployment, or in failure cases, which services failed.
