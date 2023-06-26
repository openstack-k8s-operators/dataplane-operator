# Interacting with Ansible

When a dataplane service is executed during a role deployment, a corresponding
[OpenStackAnsibleEE](https://openstack-k8s-operators.github.io/openstack-ansibleee-operator/openstack_ansibleee/)
resource is created. The OpenStackAnsibleEE resource is the associated ansible
execution with the service.

OpenStackAnsibleEE resources are reconciled by
[openstack-ansibleee-operator](https://github.com/openstack-k8s-operators/openstack-ansibleee-operator).
During reconciliation a
[Job](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/job-v1/)
resource is created which in turn creates a
[Pod](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/) resource. The pod is started with an [Ansible Execution Environment](https://docs.ansible.com/automation-controller/latest/html/userguide/execution_environments.html) image, and runs [ansible-runner](https://ansible.readthedocs.io/projects/runner/en/stable/).

## Retrieving and inspecting OpenStackAnsibleEE resources

During (or after) a deployment the instances of OpenStackAnsibleEE can be
retrieved from the API.

    oc get openstackansibleee

Sample output when the default list of services:

	NAME                                                  NETWORKATTACHMENTS   STATUS   MESSAGE
	dataplane-deployment-configure-network-edpm-compute                        True     AnsibleExecutionJob complete
	dataplane-deployment-configure-os-edpm-compute                             True     AnsibleExecutionJob complete
	dataplane-deployment-install-os-edpm-compute                               True     AnsibleExecutionJob complete
	dataplane-deployment-run-os-edpm-compute                                   True     AnsibleExecutionJob complete
	dataplane-deployment-validate-network-edpm-compute                         True     AnsibleExecutionJob complete

Querying for pods with the OpenStackAnsibleEE label

	oc get pods -l app=openstackansibleee

Sample output:

	dataplane-deployment-configure-network-edpm-compute-j6r4l   0/1     Completed           0          3m36s
	dataplane-deployment-validate-network-edpm-compute-6g7n9    0/1     Pending             0          0s
	dataplane-deployment-validate-network-edpm-compute-6g7n9    0/1     ContainerCreating   0          11s
	dataplane-deployment-validate-network-edpm-compute-6g7n9    1/1     Running             0          13s

Querying for jobs, shows the corresponding job for each OpenStackAnsibleEE resource:

	oc get jobs -l app=openstackansibleee

Sample output:

	NAME                                                  COMPLETIONS   DURATION   AGE
	dataplane-deployment-configure-network-edpm-compute   1/1           8s         2m51s
	dataplane-deployment-configure-os-edpm-compute        1/1           8s         2m27s
	dataplane-deployment-install-os-edpm-compute          1/1           8s         2m35s
	dataplane-deployment-run-os-edpm-compute              1/1           8s         2m19s
	dataplane-deployment-validate-network-edpm-compute    1/1           8s         2m43s

Using the job name, the corresponding pod can be retrieved:

	oc get pods | grep dataplane-deployment-configure-network-edpm-compute

Sample output:

	dataplane-deployment-configure-network-edpm-compute-2hshp   0/1     Completed            0                5m45s

Using the job name, the ansible logs can be retrieved:

    oc logs job.batch/dataplane-deployment-configure-network-edpm-compute

## Controlling the Ansible execution

The deployStrategy field on OpenStackDataPlane, OpenStackDataPlaneRole, and
OpenStackDataPlaneNode has support for specifying the ansible [tags,
skip-tags](https://docs.ansible.com/ansible/latest/playbook_guide/playbooks_tags.html#selecting-or-skipping-tags-when-you-run-a-playbook),
and
[limit](https://docs.ansible.com/ansible/latest/inventory_guide/intro_patterns.html#patterns-and-ad-hoc-commands)

The fields in deployStrategy that correspond to these options are:

    ansibleTags
    ansibleSkipTags
    ansibleLimit

The syntax for these fields match the syntax that ansible accepts on the
command line for `ansible-playbook` and `ansible-runner` for each of these
fields.

Example usage of these fields:

    apiVersion: dataplane.openstack.org/v1beta1
    kind: OpenStackDataPlane
    metadata:
      name: openstack-edpm
    spec:
      deployStrategy:
          deploy: false
          ansibleTags: containers
          ansibleSkipTags: packages
          ansibleLimit: compute1*,compute2*

The above example translates to an ansible command with the following
arguments:

    --tags containers --skip-tags packages --limit compute1*,compute2*
