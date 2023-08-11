/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
)

const (
	// DataPlaneNodeReadyMessage ready
	DataPlaneNodeReadyMessage = "DataPlaneNode ready"

	// DataPlaneNodeErrorMessage error
	DataPlaneNodeErrorMessage = "DataPlaneNode error occurred %s"

	// DataPlaneRoleReadyMessage ready
	DataPlaneRoleReadyMessage = "DataPlaneRole ready"

	// DataPlaneRoleErrorMessage error
	DataPlaneRoleErrorMessage = "DataPlaneRole error occurred %s"

	// DataPlaneReadyMessage ready
	DataPlaneReadyMessage = "DataPlane ready"

	// DataPlaneErrorMessage error
	DataPlaneErrorMessage = "DataPlane error occurred %s"

	// ServiceReadyCondition Status=True condition indicates if the
	// service is finished and successful.
	ServiceReadyCondition string = "%s service ready"

	// ServiceReadyMessage ready
	ServiceReadyMessage = "%s service ready"

	// ServiceReadyWaitingMessage not yet ready
	ServiceReadyWaitingMessage = "%s service not yet ready"

	// ServiceErrorMessage error
	ServiceErrorMessage = "Service error occurred %s"

	// SetupReadyCondition - Overall setup condition
	SetupReadyCondition condition.Type = "SetupReady"

	// NovaComputeReadyCondition Status=True condition indicates nova-compute
	// has been deployed and is ready
	NovaComputeReadyCondition condition.Type = "NovaComputeReady"

	// NovaComputeReadyMessage ready
	NovaComputeReadyMessage = "NovaComputeReady ready"

	// NovaComputeReadyWaitingMessage not yet ready
	NovaComputeReadyWaitingMessage = "NovaComputeReady not yet ready"

	// NovaComputeErrorMessage error
	NovaComputeErrorMessage = "NovaCompute error occurred"

	// RoleBareMetalProvisionReadyCondition Status=True condition indicates
	// all baremetal nodes provisioned for the Role.
	RoleBareMetalProvisionReadyCondition condition.Type = "RoleBaremetalProvisionReady"

	// RoleBaremetalProvisionReadyMessage ready
	RoleBaremetalProvisionReadyMessage = "RoleBaremetalProvisionReady ready"

	// RoleBaremetalProvisionReadyWaitingMessage not yet ready
	RoleBaremetalProvisionReadyWaitingMessage = "RoleBaremetalProvisionReady not yet ready"

	// RoleBaremetalProvisionErrorMessage error
	RoleBaremetalProvisionErrorMessage = "RoleBaremetalProvisionReady error occurred"

	// RoleIPReservationReadyCondition Status=True condition indicates
	// IPSets reserved for all nodes in a Role.
	RoleIPReservationReadyCondition condition.Type = "RoleIPReservationReady"

	// RoleIPReservationReadyMessage ready
	RoleIPReservationReadyMessage = "RoleIPReservationReady ready"

	// RoleIPReservationReadyWaitingMessage not yet ready
	RoleIPReservationReadyWaitingMessage = "RoleIPReservationReady not yet ready"

	// RoleIPReservationReadyErrorMessage error
	RoleIPReservationReadyErrorMessage = "RoleIPReservationReady error occurred"

	// RoleDNSDataReadyCondition Status=True condition indicates
	// DNSData created for the Role.
	RoleDNSDataReadyCondition condition.Type = "RoleDNSDataReady"

	// RoleDNSDataReadyMessage ready
	RoleDNSDataReadyMessage = "RoleDNSDataReady ready"

	// RoleDNSDataReadyWaitingMessage not yet ready
	RoleDNSDataReadyWaitingMessage = "RoleDNSDataReady not yet ready"

	// RoleDNSDataReadyErrorMessage error
	RoleDNSDataReadyErrorMessage = "RoleDNSDataReady error occurred"

	// InputReadyWaitingMessage not yet ready
	InputReadyWaitingMessage = "Waiting for input %s, not yet ready"
)
