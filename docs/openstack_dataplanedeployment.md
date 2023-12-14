
### Custom Resources

* [OpenStackDataPlaneDeployment](#openstackdataplanedeployment)

### Sub Resources

* [AnsibleEESpec](#ansibleeespec)
* [AnsibleOpts](#ansibleopts)
* [NodeSection](#nodesection)
* [NodeTemplate](#nodetemplate)
* [OpenStackDataPlaneDeploymentList](#openstackdataplanedeploymentlist)
* [OpenStackDataPlaneDeploymentSpec](#openstackdataplanedeploymentspec)
* [OpenStackDataPlaneDeploymentStatus](#openstackdataplanedeploymentstatus)

#### AnsibleEESpec

AnsibleEESpec is a specification of the ansible EE attributes

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| networkAttachments | NetworkAttachments is a list of NetworkAttachment resource names to pass to the ansibleee resource which allows to connect the ansibleee runner to the given network | []string | true |
| openStackAnsibleEERunnerImage | OpenStackAnsibleEERunnerImage image to use as the ansibleEE runner image | string | false |
| ansibleTags | AnsibleTags for ansible execution | string | false |
| ansibleLimit | AnsibleLimit for ansible execution | string | false |
| ansibleSkipTags | AnsibleSkipTags for ansible execution | string | false |
| extraMounts | ExtraMounts containing files which can be mounted into an Ansible Execution Pod | []storage.VolMounts | false |
| env | Env is a list containing the environment variables to pass to the pod | []corev1.EnvVar | false |
| dnsConfig | DNSConfig for setting dnsservers | *corev1.PodDNSConfig | false |

[Back to Custom Resources](#custom-resources)

#### AnsibleOpts

AnsibleOpts defines a logical grouping of Ansible related configuration options.

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| ansibleUser | AnsibleUser SSH user for Ansible connection | string | true |
| ansibleHost | AnsibleHost SSH host for Ansible connection | string | false |
| ansiblePort | AnsiblePort SSH port for Ansible connection | int | false |
| ansibleVars | AnsibleVars for configuring ansible | map[string]json.RawMessage | false |

[Back to Custom Resources](#custom-resources)

#### NodeSection

NodeSection defines the top level attributes inherited by nodes in the CR.

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| hostName | HostName - node name | string | false |
| networks | Networks - Instance networks | []infranetworkv1.IPSetNetwork | false |
| managementNetwork | ManagementNetwork - Name of network to use for management (SSH/Ansible) | string | false |
| ansible | Ansible is the group of Ansible related configuration options. | [AnsibleOpts](#ansibleopts) | false |
| extraMounts | ExtraMounts containing files which can be mounted into an Ansible Execution Pod | []storage.VolMounts | false |
| userData | UserData  node specific user-data | *corev1.SecretReference | false |
| networkData | NetworkData  node specific network-data | *corev1.SecretReference | false |

[Back to Custom Resources](#custom-resources)

#### NodeTemplate

NodeTemplate is a specification of the node attributes that override top level attributes.

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| ansibleSSHPrivateKeySecret | AnsibleSSHPrivateKeySecret Name of a private SSH key secret containing private SSH key for connecting to node. The named secret must be of the form: Secret.data.ssh-privatekey: <base64 encoded private key contents> <https://kubernetes.io/docs/concepts/configuration/secret/#ssh-authentication-secrets> | string | true |
| networks | Networks - Instance networks | []infranetworkv1.IPSetNetwork | false |
| managementNetwork | ManagementNetwork - Name of network to use for management (SSH/Ansible) | string | true |
| ansible | Ansible is the group of Ansible related configuration options. | [AnsibleOpts](#ansibleopts) | false |
| extraMounts | ExtraMounts containing files which can be mounted into an Ansible Execution Pod | []storage.VolMounts | false |
| userData | UserData  node specific user-data | *corev1.SecretReference | false |
| networkData | NetworkData  node specific network-data | *corev1.SecretReference | false |

[Back to Custom Resources](#custom-resources)

#### OpenStackDataPlaneDeployment

OpenStackDataPlaneDeployment is the Schema for the openstackdataplanedeployments API

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| metadata |  | metav1.ObjectMeta | false |
| spec |  | [OpenStackDataPlaneDeploymentSpec](#openstackdataplanedeploymentspec) | false |
| status |  | [OpenStackDataPlaneDeploymentStatus](#openstackdataplanedeploymentstatus) | false |

[Back to Custom Resources](#custom-resources)

#### OpenStackDataPlaneDeploymentList

OpenStackDataPlaneDeploymentList contains a list of OpenStackDataPlaneDeployment

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| metadata |  | metav1.ListMeta | false |
| items |  | [][OpenStackDataPlaneDeployment](#openstackdataplanedeployment) | true |

[Back to Custom Resources](#custom-resources)

#### OpenStackDataPlaneDeploymentSpec

OpenStackDataPlaneDeploymentSpec defines the desired state of OpenStackDataPlaneDeployment

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| nodeSets | NodeSets is the list of NodeSets deployed | []string | true |
| ansibleTags | AnsibleTags for ansible execution | string | false |
| ansibleLimit | AnsibleLimit for ansible execution | string | false |
| ansibleSkipTags | AnsibleSkipTags for ansible execution | string | false |
| servicesOverride | ServicesOverride list | []string | true |

[Back to Custom Resources](#custom-resources)

#### OpenStackDataPlaneDeploymentStatus

OpenStackDataPlaneDeploymentStatus defines the observed state of OpenStackDataPlaneDeployment

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| conditions | Conditions | condition.Conditions | false |
| nodeSetConditions | NodeSetConditions | map[string]condition.Conditions | false |
| deployed | Deployed | bool | false |

[Back to Custom Resources](#custom-resources)
