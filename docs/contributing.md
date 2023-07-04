# Contributing

## Getting Started

You’ll need a Kubernetes cluster to run against. You can use
[KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run
against a remote cluster.  **Note:** Your controller will automatically use the
current context in your kubeconfig file (i.e. whatever cluster `kubectl
cluster-info` shows).

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

### How it works

This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/)
which provide a reconcile function responsible for synchronizing resources
until the desired state is reached on the cluster

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

## Testing

The tests can be run with the following command:

```bash
make test
```

The `test` target runs the
[EnvTest](https://book.kubebuilder.io/reference/envtest.html) tests with
[Ginkgo](https://onsi.github.io/ginkgo/). These tools will be installed by the
`test` if needed.

`EnvTest` tests are under the
[`tests/functional`](https://github.com/openstack-k8s-operators/dataplane-operator/tree/main/tests/functional)
directory in dataplane-operator.

### Running kuttl tests

The kuttl tests require a running cluster with
[openstack-ansibleee-operator](https://github.com/openstack-k8s-operators/openstack-ansibleee-operator)
running in the cluster.

kuttl tests are under the
['tests/kuttl/tests'](https://github.com/openstack-k8s-operators/dataplane-operator/tree/main/tests/kuttl/tests)
in dataplane-operator.

#### From install_yamls

The kuttl tests are run from the
[install_yamls](https://github.com/openstack-k8s-operators/install_yamls)
repository by the CI jobs.

Running from `install_yamls`:

```sh
cd install_yamls
# Set environment variables if needed to use a specific repo and branch of dataplane-operator
export DATAPLANE_REPO=https://github.com/openstack-k8s-operators/dataplane-operator.git
export DATAPLANE_BRANCH=main
make dataplane_kuttl
```

#### From dataplane-operator

The kuttl tests can also be run directly from the dataplane-operator checkout.
When running from a dataplane-operator checkout, `kubectl-kuttl` must be
installed. The `kubectl-kuttl` command can be installed from
[kuttl releases](https://github.com/kudobuilder/kuttl/releases), or using the
Makefile target `kuttl`:

```sh
make kuttl
```

Then, run the operator from a checkout:

```sh
make run
```

Execute the kuttl tests:

```sh
make kuttl-test
```

Run a single test if desired:

```sh
make kuttl-test KUTTL_ARGS="--test dataplane-deploy-no-nodes"
```

Skip the test resource delete, which will leave the test resources created in the
cluster, and can be useful for debugging failed tests:

```sh
make kuttl-test KUTTL_ARGS="--test dataplane-deploy-no-nodes --skip-delete"
```

## Contributing to documentation

### Rendering documentation locally

Install docs build requirements into virtualenv:

```
python3 -m venv local/docs-venv
source local/docs-venv/bin/activate
pip install -r docs/doc_requirements.txt
```

Serve docs site on localhost:

```
mkdocs serve
```

Click the link it outputs. As you save changes to files modified in your editor,
the browser will automatically show the new content.

### Create or edit diagrams

Create a `puml` file inside `docs/diagrams/src`

```
touch docs/diagrams/src/demo.puml
```

Check the PlantUML syntax here: <https://plantuml.com/deployment-diagram>

Serve docs site on localhost:

```
mkdocs serve
```

Add the yielded `svg` into the desired `.md` file

```
![Diagram demo](diagrams/out/demo.svg "Diagram demo")
```
