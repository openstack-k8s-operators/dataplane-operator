# Composable services

Composable services with dataplane-operator provide a way for users to
customize services that are deployed on dataplane nodes. It is possible to
"compose" a set of services such that the dataplane deployment can be
customized in whichever ways are needed.

Composing services can take different forms. The interfaces in
dataplane-operator allow for:

* Enabling/disabling services
* Ordering services
* Developing custom services

For the purposes of the interfaces in dataplane-operator, a service is an
ansible execution that manages a software deployment (installation,
configuration, execution, etc) on dataplane nodes. The ansible content that
makes up each service is defined by the service itself. Each service is a
resource instance of the
[`OpenStackDataPlaneService`](openstack_dataplaneservice.md) CRD.

## dataplane-operator provided services

dataplane-operator provides a default list of services that will be deployed on
dataplane nodes. The services list is set on the
[`OpenStackDataPlaneNodeSet`](openstack_dataplanenodeset.md#openstackdataplanenodesetspec) CRD.

The default list of services as they will appear on the `services` field on an
`OpenStackDataPlaneNodeSet` spec is:

    services:
      - configure-network
      - validate-network
      - install-os
      - configure-os
      - run-os
      - libvirt
      - nova

If the `services` field is ommitted from the `OpenStackDataPlaneNodeSet` spec,
then the above list will be used.

The default list of services are reconciled during `NodeSet` reconciliation if the
service is in the NodeSets' service list.

## dataplane-operator provided optional services

Not all services which ship with the dataplane-operator are enabled by
default. Additional optional services are documented here.

### ceph-client

Include this service to configure EDPM nodes as clients of a
Ceph server.  Usually this service is included after `install-os`
and before `configure-os`. This service requires the data plane CR to
have an `extraMounts` entry whose `extraVolType` is Ceph in order to
access Ceph secrets. For more information see the
[Ceph documentation](https://github.com/openstack-k8s-operators/docs/blob/main/ceph.md).

    services:
      - ceph-client

### ceph-hci-pre

Include this service to prepare EDPM nodes to host Ceph in an HCI
configuration. For more information see the
[HCI documenation](https://github.com/openstack-k8s-operators/docs/blob/main/hci.md).

    services:
      - ceph-hci-pre

---
**NOTE**

Do not create a custom service with the same name as one of the default
services. The default service will overwrite the custom service with the same
name during `NodeSet` reconciliation.

---

## Interacting with the `OpenStackDataPlaneService` API

The list of services available to be used by the dataplane can be seen by
getting the list of `OpenStackDataPlaneService` resources.

    oc get openstackdataplaneservice

If no custom services have been defined, the default avaiable services are
returned.

    NAME                AGE
    configure-network   20h
    configure-os        8d
    install-os          8d
    repo-setup          8d
    run-os              8d
    validate-network    8d
    libvirt             8d
    nova                8d

A service can be examined in more detail by looking at the YAML representation
of the resource.

    oc get openstackdataplaneservice configure-network -o yaml

In the `spec` output of the `configure-network` service, the ansible content
that is used by the service is shown. While the content is very similar to the
exact ansible syntax that will be used when the service executes ansible, it
may not always be proper ansible syntax. The API for defining the ansible
content of a service matches that of the
['OpenStackAnsibleEE'](https://openstack-k8s-operators.github.io/openstack-ansibleee-operator/openstack_ansibleee/)
CRD from
[openstack-ansible-operator](https://github.com/openstack-k8s-operators/openstack-ansibleee-operator).

The 'role' field on `OpenStackDataPlaneService` is the same API as the `role`
field on `OpenStackAnsibleEE'.

The 'play' field on `OpenStackDataPlaneService` is free form string content
that will be passed directly as playbook content when ansible executes.

Either `role` or `play` can define ansible content for a service, but both can
not be used in the same service.

## Composing services

This example will walk through developing and using a custom service.

### Developing a custom service

To create custom service, create a resource of kind
[`OpenStackDataPlaneService`](openstack_dataplaneservice.md). User either the
'play' or 'role' field in spec to specify custom ansible content. These fields
are fully documented in the spec of the
['OpenStackAnsibleEE'](https://openstack-k8s-operators.github.io/openstack-ansibleee-operator/openstack_ansibleee/)
CRD from
[openstack-ansible-operator](https://github.com/openstack-k8s-operators/openstack-ansibleee-operator).

This example shows using the `play` field. Create a `hello-world.yaml` file
with the following contents:

    apiVersion: dataplane.openstack.org/v1beta1
    kind: OpenStackDataPlaneService
    metadata:
      name: hello-world
    spec:
      label: dataplane-deployment-hello-world
      openstackAnsibleEEImage: quay.io/openstack-k8s-operators/openstack-ansibleee-runner:latest
      play: |
        hosts: all
        tasks:
          - name: Hello World!
            shell: "echo Hello World!"
            register: output
          - name: Show output
            debug:
              msg: "{{ output.stdout }}"

Note that the `play` field is a string, and not YAML. However, it should be
proper ansible playbook syntax when parsed as YAML.

Create the service:

    oc apply -f hello-world.yaml

#### Customizing the ansible-runner image used by a service

The `openstackAnsibleEEImage` field is the container image used by the
ansible-runner execution environment to execute ansible. The default image is
built with the content from
[edpm-ansible](https://github.com/openstack-k8s-operators/edpm-ansible).

In some cases, it may be necessary to customize the image used by the
ansible-runner execution environment in order to add additional ansible content
that might be needed (such as ansible roles or modules).

##### Building a new custom ansible-runner image

Write a `Containerfile` that adds the needed custom content to the default
image:

    FROM quay.io/openstack-k8s-operators/openstack-ansibleee-runner:latest

    COPY my_custom_role /usr/share/ansible/roles/my_custom_role

Build and push the image to a container registry:

    podman build -t quay.io/example_user/my_custom_image:latest .
    podman push quay.io/example_user/my_custom_role:latest

In the `OpenStackDataPlaneService` YAML, specify the custom image for the
`openstackAnsibleEEImage` field:

    apiVersion: dataplane.openstack.org/v1beta1
    kind: OpenStackDataPlaneService
    metadata:
      name: hello-world
    spec:
      label: dataplane-deployment-hello-world
      openstackAnsibleEEImage: quay.io/example_user/my_custom_role:latest
      ...

##### Using ExtraMounts

The `ExtraMounts` field in the
[`NodeSection`](https://openstack-k8s-operators.github.io/dataplane-operator/openstack_dataplanenodeset/#nodesection)
field can be used to mount custom content into the ansible-runner image. In
some cases, this is a simpler method to customize the image than having to
build an entirely new image.

### Enabling a custom service

To add a custom service to be executed as part of an `OpenStackDataPlaneNodeSet`
deployment, add the service name to the `services` field list on the `NodeSet`. Add
the service name in the order that it should be executed relative to the other
services. This example shows adding the `hello-world` service as the first
service to execute for the `edpm-compute` `NodeSet`.

    apiVersion: dataplane.openstack.org/v1beta1
    kind: OpenStackDataPlaneNodeSet
    metadata:
      name: openstack-edpm
    spec:
      services:
        - hello-world
        - configure-network
        - validate-network
        - install-os
        - configure-os
        - run-os
      nodeTemplate:
        nodes:
          edpm-compute:
            ansible:
              ansibleHost: 172.20.12.67
              ansibleSSHPrivateKeySecret: dataplane-ansible-ssh-private-key-secret
              ansibleUser: root
              ansibleVars:
                ansible_ssh_transfer_method: scp
                ctlplane_ip: 172.20.12.67
                external_ip: 172.20.12.76
                fqdn_internal_api: edpm-compute-1.example.com
                internal_api_ip: 172.17.0.101
                storage_ip: 172.18.0.101
                tenant_ip: 172.10.0.101
            hostName: edpm-compute-0
            networkConfig: {}
            nova:
              cellName: cell1
              deploy: true
              novaInstance: nova

When customizing the services list, the default list of services must be
reproduced and then customized if the intent is to still deploy those services.
If just the `hello-world` service was listed in the list, then that is the only
service that would be deployed.
