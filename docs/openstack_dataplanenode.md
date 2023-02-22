
### Custom Resources

* [OpenStackDataPlaneNode](#openstackdataplanenode)

### Sub Resources

* [NetworkConfigSection](#networkconfigsection)
* [NetworksSection](#networkssection)
* [NodeSection](#nodesection)
* [OpenStackDataPlaneNodeList](#openstackdataplanenodelist)
* [OpenStackDataPlaneNodeSpec](#openstackdataplanenodespec)
* [OpenStackDataPlaneNodeStatus](#openstackdataplanenodestatus)

#### NetworkConfigSection

NetworkConfigSection is a specification of the Network configuration details

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| template | Template - ansible j2 nic config template to use when applying node network configuration | string | false |

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
| managed | Managed - Whether the node is actually provisioned (True) or should be treated as preprovisioned (False) | bool | false |
| managementNetwork | ManagementNetwork - Name of network to use for management (SSH/Ansible) | string | false |
| ansibleUser | AnsibleUser SSH user for Ansible connection | string | false |
| ansiblePort | AnsiblePort SSH port for Ansible connection | int | false |
| ansibleVars | AnsibleVars for configuring ansible | string | false |
| ansibleSSHPrivateKeySecret | AnsibleSSHPrivateKeySecret Private SSH Key secret containing private SSH key for connecting to node. Must be of the form: Secret.data.ssh_private_key: <base64 encoded private key contents> | string | true |

[Back to Custom Resources](#custom-resources)

#### OpenStackDataPlaneNode

OpenStackDataPlaneNode is the Schema for the openstackdataplanenodes API

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| metadata |  | metav1.ObjectMeta | false |
| spec |  | [OpenStackDataPlaneNodeSpec](#openstackdataplanenodespec) | false |
| status |  | [OpenStackDataPlaneNodeStatus](#openstackdataplanenodestatus) | false |

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
| deploy | Deploy boolean to trigger ansible execution | bool | true |
| nodeFrom | NodeFrom - Existing node name to reference. Can only be used if Node is empty. | string | false |

[Back to Custom Resources](#custom-resources)

#### OpenStackDataPlaneNodeStatus

OpenStackDataPlaneNodeStatus defines the observed state of OpenStackDataPlaneNode

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| conditions | Conditions | condition.Conditions | false |
| deployed | Deployed | bool | false |

[Back to Custom Resources](#custom-resources)
