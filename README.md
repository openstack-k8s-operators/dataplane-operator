# dataplane-operator
The dataplane-operator automates the deployment of an OpenStack dataplane.

---
**NOTE**

The OpenStackDataPlane CRD/controller may eventually be moved to
[openstack-operator](https://github.com/openstack-k8s-operator/openstack-operator).
The move is so that openstack-operator, as the umbrella operator, remains the
single managing operator and UX for a complete OpenStack deployment
(control plane and data plane).

---

## DataPlane, Roles, and Nodes

The dataplane-operator exposes the concepts of data plane role and nodes. These
are represented as CRD's within the operator:

* [OpenStackDataPlane](https://github.com/openstack-k8s-operators/dataplane-operator/blob/main/config/crd/bases/dataplane.openstack.org_openstackdataplanes.yaml)
* [OpenStackDataPlaneRole](https://github.com/openstack-k8s-operators/dataplane-operator/blob/main/config/crd/bases/dataplane.openstack.org_openstackdataplaneroles.yaml)
* [OpenStackDataPlaneNode](https://github.com/openstack-k8s-operators/dataplane-operator/blob/main/config/crd/bases/dataplane.openstack.org_openstackdataplanenodes.yaml)

The OpenStackDataPlane CRD provides for a logical grouping of roles, that
altogether make up an OpenStack dataplane.

Roles are grouped into the dataplane with the `DataPlane` field set on the
role. Roles are also labeled with the dataplane they belong to:

`openstackdataplane=<dataplane-name>`

The dataplane does not maintain a list of roles. The source of dataplane and
role grouping is stored in the `DataPlane` field of the role.

The OpenStackDataPlaneRole CRD provides for a logical grouping of nodes of a
similar type within a role. Similarities within a role are defined by the
user, and could be of a small scope (ansible port), or a large scope (same
network config, nova config, provisioning config, etc). The properties that all
nodes in a role share is set in the NodeTemplate field of the role's Spec.

A role also provides for an inheritance model of node attributes. Properties
from the `NodeTemplate` on the role will automatically be inherited by the
nodes on that role. Nodes can also set their own properties within their `Node`
field, which will override the inherited values from the role.

Similarly to the dataplane and role grouping, the grouping of nodes within a
role is done with the `Role` field on the node.  Node's in a given role will
have the `Role` field set to their role name.  Nodes do not have to be in a
role, in which case their `Role` field will be empty. Roles do not maintain the
list of nodes in that role. This ensures that the single source of node and
role association is stored only in the `Role` field on the node. Nodes are also
labeled with their role using `openstackdataplanerole=<role-name>` to allow for
easy querying and filtering on nodes in a role.

## Deployment

The deployment handling in `dataplane-operator` is implemented within the
deployment package at `pkg/deployment`. This allows for both a node and role to
be deployed in the same way. A node uses an ansible inventory containing just
that single node when it triggers a deployment. A role uses an ansible
inventory containing all the nodes in that role when it triggers a deployment.
This allows for deploying just a single node at a time, or an entire role for
bulk deployment.

## Getting Started
You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### Running on the cluster
1. Install Instances of Custom Resources:

```sh
kubectl apply -f config/samples/
```

2. Build and push your image to the location specified by `IMG`:

```sh
make docker-build docker-push IMG=<some-registry>/dataplane-operator:tag
```

3. Deploy the controller to the cluster with the image specified by `IMG`:

```sh
make deploy IMG=<some-registry>/dataplane-operator:tag
```

### Uninstall CRDs
To delete the CRDs from the cluster:

```sh
make uninstall
```

### Undeploy controller
UnDeploy the controller to the cluster:

```sh
make undeploy
```

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

### How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/)
which provides a reconcile function responsible for synchronizing resources untile the desired state is reached on the cluster

### Test It Out
1. Install the CRDs into the cluster:

```sh
make install
```

2. Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

```sh
make run
```

**NOTE:** You can also run this in one step by running: `make install run`

### Modifying the API definitions
If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
