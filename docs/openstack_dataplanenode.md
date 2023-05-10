
### Custom Resources

* [OpenStackDataPlaneNode](#openstackdataplanenode)

### Sub Resources

* [AnsibleEESpec](#ansibleeespec)
* [DeployStrategySection](#deploystrategysection)
* [NetworkConfigSection](#networkconfigsection)
* [NetworksSection](#networkssection)
* [NodeSection](#nodesection)
* [OpenStackDataPlaneNodeList](#openstackdataplanenodelist)
* [OpenStackDataPlaneNodeSpec](#openstackdataplanenodespec)

#### AnsibleEESpec

AnsibleEESpec is a specification of the ansible EE attributes

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| networkAttachments | NetworkAttachments is a list of NetworkAttachment resource names to pass to the ansibleee resource which allows to connect the ansibleee runner to the given network | []string | true |
| openStackAnsibleEERunnerImage | OpenStackAnsibleEERunnerImage image to use as the ansibleEE runner image | string | true |
| ansibleTags | AnsibleTags for ansible execution | string | false |
| ansibleLimit | AnsibleLimit for ansible execution | string | false |
| ansibleSkipTags | AnsibleSkipTags for ansible execution | string | false |
| deployIdentifier | DeployIdentifier is a randomly created string created for new deploy it can be used to override a current deploy on ansibleee objects | string | true |
| extraMounts | ExtraMounts containing files which can be mounted into an Ansible Execution Pod | []storage.VolMounts | true |
| env | Env is a list containing the environment variables to pass to the pod | []corev1.EnvVar | false |

[Back to Custom Resources](#custom-resources)

#### DeployStrategySection

DeployStrategySection for fields controlling the deployment

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| deploy | Deploy boolean to trigger ansible execution | bool | true |
| ansibleTags | AnsibleTags for ansible execution | string | false |
| ansibleLimit | AnsibleLimit for ansible execution | string | false |
| ansibleSkipTags | AnsibleSkipTags for ansible execution | string | false |
| deployIdentifier | DeployIdentifier is a generated UUID set as input on OpenStackAnsibleEE resources so that the OpenStackAnsibleEE controller can determine job input uniqueness. It is generated on each new deploy request (when DeployStrategy.Deploy is changed to true). | string | false |

[Back to Custom Resources](#custom-resources)

#### NetworkConfigSection

NetworkConfigSection is a specification of the Network configuration details

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| template | Template - Contains a Ansible j2 nic config template to use when applying node network configuration | string | false |

[Back to Custom Resources](#custom-resources)

#### NetworksSection

NetworksSection is a specification of the network attributes

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| network | Network - Network name to configure | string | false |
| fixedIP | FixedIP - Specific IP address to use for this network | string | false |

[Back to Custom Resources](#custom-resources)

#### NodeSection

NodeSection is a specification of the node attributes

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| networkConfig | NetworkConfig - Network configuration details. Contains os-net-config related properties. | [NetworkConfigSection](#networkconfigsection) | false |
| networks | Networks - Instance networks | [][NetworksSection](#networkssection) | false |
| managementNetwork | ManagementNetwork - Name of network to use for management (SSH/Ansible) | string | false |
| ansibleUser | AnsibleUser SSH user for Ansible connection | string | false |
| ansiblePort | AnsiblePort SSH port for Ansible connection | int | false |
| ansibleVars | AnsibleVars for configuring ansible | string | false |
| ansibleSSHPrivateKeySecret | AnsibleSSHPrivateKeySecret Private SSH Key secret containing private SSH key for connecting to node. Must be of the form: Secret.data.ssh-privatekey: <base64 encoded private key contents> https://kubernetes.io/docs/concepts/configuration/secret/#ssh-authentication-secrets | string | true |
| extraMounts | ExtraMounts containing files which can be mounted into an Ansible Execution Pod | []storage.VolMounts | true |
| userData | UserData  node specific user-data | *corev1.SecretReference | false |
| networkData | NetworkData  node specific network-data | *corev1.SecretReference | false |

[Back to Custom Resources](#custom-resources)

#### OpenStackDataPlaneNode

OpenStackDataPlaneNode is the Schema for the openstackdataplanenodes API

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| metadata |  | metav1.ObjectMeta | false |
| spec |  | [OpenStackDataPlaneNodeSpec](#openstackdataplanenodespec) | false |
| status |  | OpenStackDataPlaneStatus | false |

[Back to Custom Resources](#custom-resources)

#### OpenStackDataPlaneNodeList

OpenStackDataPlaneNodeList contains a list of OpenStackDataPlaneNode

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| metadata |  | metav1.ListMeta | false |
| items |  | [][OpenStackDataPlaneNode](#openstackdataplanenode) | true |

[Back to Custom Resources](#custom-resources)

#### OpenStackDataPlaneNodeSpec

OpenStackDataPlaneNodeSpec defines the desired state of OpenStackDataPlaneNode

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| hostName | HostName - node name | string | false |
| node | Node - node attributes specific to this node | [NodeSection](#nodesection) | false |
| role | Role - role name for this node | string | false |
| ansibleHost | AnsibleHost SSH host for Ansible connection | string | false |
| env | Env is a list containing the environment variables to pass to the pod | []corev1.EnvVar | false |
| deployStrategy | DeployStrategy section to control how the node is deployed | [DeployStrategySection](#deploystrategysection) | false |
| networkAttachments | NetworkAttachments is a list of NetworkAttachment resource names to pass to the ansibleee resource which allows to connect the ansibleee runner to the given network | []string | true |
| openStackAnsibleEERunnerImage | OpenStackAnsibleEERunnerImage image to use as the ansibleEE runner image | string | true |

[Back to Custom Resources](#custom-resources)
