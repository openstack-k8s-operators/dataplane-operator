# DataPlane Design

The dataplane-operator prepares nodes with enough operating system and other
configuration to make them suitable for hosting OpenStack workloads.

It uses the
[openstack-baremetal-operator](https://github.com/openstack-k8s-operators/openstack-baremetal-operator)
to optionally provision baremetal. Ansible is used to deploy and configure
software on the nodes.

## DataPlane Design and Resources

The dataplane-operator exposes the concepts of dataplanes, roles, nodes, and
services as CRD's:

* [OpenStackDataPlane](https://github.com/openstack-k8s-operators/dataplane-operator/blob/main/config/crd/bases/dataplane.openstack.org_openstackdataplanes.yaml)
* [OpenStackDataPlaneRole](https://github.com/openstack-k8s-operators/dataplane-operator/blob/main/config/crd/bases/dataplane.openstack.org_openstackdataplaneroles.yaml)
* [OpenStackDataPlaneNode](https://github.com/openstack-k8s-operators/dataplane-operator/blob/main/config/crd/bases/dataplane.openstack.org_openstackdataplanenodes.yaml)
* [OpenStackDataPlaneService](https://github.com/openstack-k8s-operators/dataplane-operator/blob/main/config/crd/bases/dataplane.openstack.org_openstackdataplaneservices.yaml)

Each node in a dataplane is represented by a corresponding
OpenStackDataPlaneNode resource. A node must already be baremetal provisioned
when it's OpenStackDataPlaneNode resource is used to manage the node. The node
can either be pre-provisioned outside the environment by external tooling, or
it may be provisioned by it's associated role.

Each role in a dataplane is represented by a corresponding
OpenStackDataPlaneRole resource.  The OpenStackDataPlaneRole CRD provides for a
logical grouping of nodes of a similar type within a role. Similarities within
a role are defined by the user, and could be of a small scope (ansible port),
or a large scope (same network config, nova config, provisioning config, etc).
The properties that all nodes in a role share is set in the NodeTemplate field
of the role's Spec.

OpenStackDataPlaneRole implements a baremetal provisioning interface to
provision the nodes in the role. This interface can be used to provision the
initial OS on a set of nodes.

A role also provides for an inheritance model of node attributes. Properties
from the `NodeTemplate` on the role will automatically be inherited by the
nodes on that role. Nodes can also set their own properties within their `Node`
field, which will override the inherited values from the role. See
[inheritance](inheritance.md) for more in depth details about how inheritance
works between roles and nodes.

The OpenStackDataPlane CRD provides a way to group all roles and nodes together
into a single resource that represents the entire dataplane. While roles and
nodes can be managed individually, OpenStackDataPlane provides an API and
convenience to manage them all together and at the same time. When using
OpenStackDataPlane, it is the
[Owner](https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/)
of its associated roles and nodes. When using OpenStackDataPlane, do not manage
roles and nodes outside of the OpenStackDataPlane resource. Roles and
nodes that are owned by the dataplane will eventually be reconciled back to
the state as stored by the OpenStackDataPlane resource, potentially undoing
changes done outside of the OpenStackDataPlane resource.

OpenStackDataPlaneService is a CRD for representing an Ansible based service to
orchestrate across the nodes. A composable service interface is provided that
allows for customizing which services are run on which roles and nodes, and for
defining custom services.

## Deployment

The deployment handling in `dataplane-operator` is implemented within the
deployment package at `pkg/deployment`. This allows for both a node and role to
be deployed in the same way. A node uses an ansible inventory containing just
that single node when it triggers a deployment. A role uses an ansible
inventory containing all the nodes in that role when it triggers a deployment.
This allows for deploying just a single node at a time, or an entire role for
bulk deployment.
