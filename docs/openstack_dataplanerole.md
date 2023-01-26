
### Custom Resources

* [OpenStackDataPlaneRole](#openstackdataplanerole)

### Sub Resources

* [DataPlaneNodeSection](#dataplanenodesection)
* [OpenStackDataPlaneRoleList](#openstackdataplanerolelist)
* [OpenStackDataPlaneRoleSpec](#openstackdataplanerolespec)

#### DataPlaneNodeSection

DataPlaneNodeSection is a specification of the data plane node attributes

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| node | Node - node attributes specific to this node | NodeSection | false |
| nodeFrom | NodeFrom - Existing node name to reference. Can only be used if Node is empty. | string | false |

[Back to Custom Resources](#custom-resources)

#### OpenStackDataPlaneRole

OpenStackDataPlaneRole is the Schema for the openstackdataplaneroles API

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| metadata |  | metav1.ObjectMeta | false |
| spec |  | [OpenStackDataPlaneRoleSpec](#openstackdataplanerolespec) | false |
| status |  | [OpenStackDataPlaneRoleStatus](#openstackdataplanerolestatus) | false |

[Back to Custom Resources](#custom-resources)

#### OpenStackDataPlaneRoleList

OpenStackDataPlaneRoleList contains a list of OpenStackDataPlaneRole

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| metadata |  | metav1.ListMeta | false |
| items |  | [][OpenStackDataPlaneRole](#openstackdataplanerole) | true |

[Back to Custom Resources](#custom-resources)

#### OpenStackDataPlaneRoleSpec

OpenStackDataPlaneRoleSpec defines the desired state of OpenStackDataPlaneRole

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| dataPlaneNodes | DataPlaneNodes - List of nodes | [][DataPlaneNodeSection](#dataplanenodesection) | false |
| nodeTemplate | NodeTemplate - node attributes specific to this roles | NodeSection | false |

[Back to Custom Resources](#custom-resources)
