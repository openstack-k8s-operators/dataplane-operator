
### Custom Resources

* [OpenStackDataPlaneNodeSet](#openstackdataplanenodeset)

### Sub Resources

* [AnsibleEESpec](#ansibleeespec)
* [AnsibleOpts](#ansibleopts)
* [DeployStrategySection](#deploystrategysection)
* [NetworkConfigSection](#networkconfigsection)
* [NodeSection](#nodesection)
* [NodeTemplate](#nodetemplate)
* [OpenStackDataPlaneNodeSetList](#openstackdataplanenodesetlist)
* [OpenStackDataPlaneNodeSetSpec](#openstackdataplanenodesetspec)
* [OpenStackDataPlaneNodeSetStatus](#openstackdataplanenodesetstatus)

#### AnsibleEESpec

AnsibleEESpec is a specification of the ansible EE attributes

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| networkAttachments | NetworkAttachments is a list of NetworkAttachment resource names to pass to the ansibleee resource which allows to connect the ansibleee runner to the given network | []string | false |
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
| ansibleUser | AnsibleUser SSH user for Ansible connection | string | false |
| ansibleHost | AnsibleHost SSH host for Ansible connection | string | false |
| ansiblePort | AnsiblePort SSH port for Ansible connection | int | false |
| ansibleVars | AnsibleVars for configuring ansible | map[string]json.RawMessage | false |

[Back to Custom Resources](#custom-resources)

#### DeployStrategySection

DeployStrategySection for fields controlling the deployment

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| deploy | Deploy boolean to trigger ansible execution | bool | true |
| ansibleTags | AnsibleTags for ansible execution | string | false |
| ansibleLimit | AnsibleLimit for ansible execution | string | false |
| ansibleSkipTags | AnsibleSkipTags for ansible execution | string | false |

[Back to Custom Resources](#custom-resources)

#### NetworkConfigSection

NetworkConfigSection is a specification of the Network configuration details

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| template | Template - Contains a Ansible j2 nic config template to use when applying node network configuration | string | false |

[Back to Custom Resources](#custom-resources)

#### NodeSection

NodeSection defines the top level attributes inherited by nodes in the CR.

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| hostName | HostName - node name | string | false |
| networkConfig | NetworkConfig - Network configuration details. Contains os-net-config related properties. | [NetworkConfigSection](#networkconfigsection) | true |
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
| nodes | Nodes - Map of Node Names and node specific data. Values here override defaults in the upper level section. | map[string][NodeSection](#nodesection) | true |
| networkConfig | NetworkConfig - Network configuration details. Contains os-net-config related properties. | [NetworkConfigSection](#networkconfigsection) | false |
| networks | Networks - Instance networks | []infranetworkv1.IPSetNetwork | false |
| managementNetwork | ManagementNetwork - Name of network to use for management (SSH/Ansible) | string | false |
| ansible | Ansible is the group of Ansible related configuration options. | [AnsibleOpts](#ansibleopts) | false |
| extraMounts | ExtraMounts containing files which can be mounted into an Ansible Execution Pod | []storage.VolMounts | false |
| userData | UserData  node specific user-data | *corev1.SecretReference | false |
| networkData | NetworkData  node specific network-data | *corev1.SecretReference | false |

[Back to Custom Resources](#custom-resources)

#### OpenStackDataPlaneNodeSet

OpenStackDataPlaneNodeSet is the Schema for the openstackdataplanenodesets API

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| metadata |  | metav1.ObjectMeta | false |
| spec |  | [OpenStackDataPlaneNodeSetSpec](#openstackdataplanenodesetspec) | false |
| status |  | [OpenStackDataPlaneNodeSetStatus](#openstackdataplanenodesetstatus) | false |

[Back to Custom Resources](#custom-resources)

#### OpenStackDataPlaneNodeSetList

OpenStackDataPlaneNodeSetList contains a list of OpenStackDataPlaneNodeSets

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| metadata |  | metav1.ListMeta | false |
| items |  | [][OpenStackDataPlaneNodeSet](#openstackdataplanenodeset) | true |

[Back to Custom Resources](#custom-resources)

#### OpenStackDataPlaneNodeSetSpec

OpenStackDataPlaneNodeSetSpec defines the desired state of OpenStackDataPlaneNodeSet

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| baremetalSetTemplate | BaremetalSetTemplate Template for BaremetalSet for the NodeSet | baremetalv1.OpenStackBaremetalSetSpec | false |
| nodeTemplate | NodeTemplate - node attributes specific to nodes defined by this resource. These attributes can be overriden at the individual node level, else take their defaults from valus in this section. | [NodeTemplate](#nodetemplate) | true |
| preProvisioned | \n\nPreProvisioned - Whether the nodes are actually pre-provisioned (True) or should be preprovisioned (False) | bool | false |
| env | Env is a list containing the environment variables to pass to the pod | []corev1.EnvVar | false |
| deployStrategy | DeployStrategy section to control how the node is deployed | [DeployStrategySection](#deploystrategysection) | false |
| networkAttachments | NetworkAttachments is a list of NetworkAttachment resource names to pass to the ansibleee resource which allows to connect the ansibleee runner to the given network | []string | false |
| services | Services list | []string | true |

[Back to Custom Resources](#custom-resources)

#### OpenStackDataPlaneNodeSetStatus

OpenStackDataPlaneNodeSetStatus defines the observed state of OpenStackDataPlaneNodeSet

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| conditions | Conditions | condition.Conditions | false |
| deployed | Deployed | bool | false |

[Back to Custom Resources](#custom-resources)
