
### Custom Resources

* [OpenStackDataPlaneService](#openstackdataplaneservice)

### Sub Resources

* [AnsibleEESpec](#ansibleeespec)
* [AnsibleOpts](#ansibleopts)
* [NodeSection](#nodesection)
* [NodeTemplate](#nodetemplate)
* [KubeService](#kubeservice)
* [OpenStackDataPlaneServiceList](#openstackdataplaneservicelist)
* [OpenStackDataPlaneServiceSpec](#openstackdataplaneservicespec)
* [OpenStackDataPlaneServiceStatus](#openstackdataplaneservicestatus)

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

#### KubeService

KubeService represents a Kubernetes Service. It is called like this to avoid the extreme overloading of the Service term in this context

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| name | Name of the Service will have in kubernetes | string | true |
| port | Port is the port of the service | int | true |
| protocol | Protocol is the protocol used to connect to the endpoint | string | false |
| network | Network is the network that will be used to connect to the endpoint | infranetworkv1.NetNameStr | false |

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
| label | Label to use for service. Must follow DNS952 subdomain conventions. Since we are using it to generate the pod name, we need to keep it short. | string | false |
| services | Services to create to expose possible external services in computes | [][KubeService](#kubeservice) | false |
| play | Play is an inline playbook contents that ansible will run on execution. If both Play and Roles are specified, Play takes precedence | string | false |
| playbook | Playbook is a path to the playbook that ansible will run on this execution | string | false |
| configMaps | ConfigMaps list of ConfigMap names to mount as ExtraMounts for the OpenStackAnsibleEE | []string | false |
| secrets | Secrets list of Secret names to mount as ExtraMounts for the OpenStackAnsibleEE | []string | false |
| openStackAnsibleEERunnerImage | OpenStackAnsibleEERunnerImage image to use as the ansibleEE runner image | string | false |
| tlsCertsEnabled | TLSCertsEnabled - Whether the nodes have TLS certs | bool | true |
| issuers | Issuers - Issuers to issue TLS Certificates | map[string]string | false |
| caCerts | CACerts - Secret containing the CA certificate chain | string | false |

[Back to Custom Resources](#custom-resources)

#### OpenStackDataPlaneServiceStatus

OpenStackDataPlaneServiceStatus defines the observed state of OpenStackDataPlaneService

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| conditions | Conditions | condition.Conditions | false |

[Back to Custom Resources](#custom-resources)
