
### Custom Resources

* [OpenStackDataPlaneService](#openstackdataplaneservice)

### Sub Resources

* [AnsibleEESpec](#ansibleeespec)
* [DeployStrategySection](#deploystrategysection)
* [NetworkConfigSection](#networkconfigsection)
* [NodeSection](#nodesection)
* [KubeService](#kubeservice)
* [OpenStackDataPlaneServiceList](#openstackdataplaneservicelist)
* [OpenStackDataPlaneServiceSpec](#openstackdataplaneservicespec)
* [OpenStackDataPlaneServiceStatus](#openstackdataplaneservicestatus)

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

NodeSection is a specification of the node attributes

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| networkConfig | NetworkConfig - Network configuration details. Contains os-net-config related properties. | [NetworkConfigSection](#networkconfigsection) | false |
| networks | Networks - Instance networks | []infranetworkv1.IPSetNetwork | false |
| managementNetwork | ManagementNetwork - Name of network to use for management (SSH/Ansible) | string | false |
| ansibleUser | AnsibleUser SSH user for Ansible connection | string | false |
| ansiblePort | AnsiblePort SSH port for Ansible connection | int | false |
| ansibleVars | AnsibleVars for configuring ansible | map[string]json.RawMessage | false |
| ansibleSSHPrivateKeySecret | AnsibleSSHPrivateKeySecret Private SSH Key secret containing private SSH key for connecting to node. Must be of the form: Secret.data.ssh-privatekey: <base64 encoded private key contents> <https://kubernetes.io/docs/concepts/configuration/secret/#ssh-authentication-secrets> | string | false |
| extraMounts | ExtraMounts containing files which can be mounted into an Ansible Execution Pod | []storage.VolMounts | false |
| userData | UserData  node specific user-data | *corev1.SecretReference | false |
| networkData | NetworkData  node specific network-data | *corev1.SecretReference | false |

[Back to Custom Resources](#custom-resources)

#### KubeService

KubeService represents a Kubernetes Service. It is called like this to avoid the extreme overloading of the Service term in this context

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| name | Name of the Service will have in kubernetes | string | true |
| port | Port is the port of the service | int | true |
| protocol | Protocol is the protocol used to connect to the endpoint | string | false |

[Back to Custom Resources](#custom-resources)

#### OpenStackDataPlaneService

OpenStackDataPlaneService is the Schema for the openstackdataplaneservices API

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| metadata |  | metav1.ObjectMeta | false |
| spec |  | [OpenStackDataPlaneServiceSpec](#openstackdataplaneservicespec) | false |
| status |  | [OpenStackDataPlaneServiceStatus](#openstackdataplaneservicestatus) | false |

[Back to Custom Resources](#custom-resources)

#### OpenStackDataPlaneServiceList

OpenStackDataPlaneServiceList contains a list of OpenStackDataPlaneService

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| metadata |  | metav1.ListMeta | false |
| items |  | [][OpenStackDataPlaneService](#openstackdataplaneservice) | true |

[Back to Custom Resources](#custom-resources)

#### OpenStackDataPlaneServiceSpec

OpenStackDataPlaneServiceSpec defines the desired state of OpenStackDataPlaneService

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| label | Label to use for service | string | false |
| services | Services to create to expose possible external services in computes | [][KubeService](#kubeservice) | false |
| play | Play is an inline playbook contents that ansible will run on execution. If both Play and Roles are specified, Play takes precedence | string | false |
| playbook | Playbook is a path to the playbook that ansible will run on this execution | string | false |
| role | Role is the description of an Ansible Role | *ansibleeev1.Role | false |
| configMaps | ConfigMaps list of ConfigMap names to mount as ExtraMounts for the OpenStackAnsibleEE | []string | false |
| secrets | Secrets list of Secret names to mount as ExtraMounts for the OpenStackAnsibleEE | []string | false |
| openStackAnsibleEERunnerImage | OpenStackAnsibleEERunnerImage image to use as the ansibleEE runner image | string | false |

[Back to Custom Resources](#custom-resources)

#### OpenStackDataPlaneServiceStatus

OpenStackDataPlaneServiceStatus defines the observed state of OpenStackDataPlaneService

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| conditions | Conditions | condition.Conditions | false |

[Back to Custom Resources](#custom-resources)
