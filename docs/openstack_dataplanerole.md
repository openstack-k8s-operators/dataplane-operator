
### Custom Resources

* [OpenStackDataPlaneRole](#openstackdataplanerole)

### Sub Resources

* [OpenStackDataPlaneRoleList](#openstackdataplanerolelist)
* [OpenStackDataPlaneRoleSpec](#openstackdataplanerolespec)

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
| nodeTemplate | NodeTemplate - node attributes specific to this roles | NodeSection | false |

[Back to Custom Resources](#custom-resources)
