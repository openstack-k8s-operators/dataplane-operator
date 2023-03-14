
### Custom Resources

* [OpenStackDataPlane](#openstackdataplane)

### Sub Resources

* [OpenStackDataPlaneList](#openstackdataplanelist)
* [OpenStackDataPlaneStatus](#openstackdataplanestatus)

#### OpenStackDataPlane

OpenStackDataPlane is the Schema for the openstackdataplanes API

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| metadata |  | metav1.ObjectMeta | false |
| spec |  | [OpenStackDataPlaneSpec](#openstackdataplanespec) | false |
| status |  | [OpenStackDataPlaneStatus](#openstackdataplanestatus) | false |

[Back to Custom Resources](#custom-resources)

#### OpenStackDataPlaneList

OpenStackDataPlaneList contains a list of OpenStackDataPlane

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| metadata |  | metav1.ListMeta | false |
| items |  | [][OpenStackDataPlane](#openstackdataplane) | true |

[Back to Custom Resources](#custom-resources)

#### OpenStackDataPlaneStatus

OpenStackDataPlaneStatus defines the observed state of OpenStackDataPlaneNode

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| conditions | Conditions | condition.Conditions | false |
| deployed | Deployed | bool | false |

[Back to Custom Resources](#custom-resources)
