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
	// DataPlaneNodeReadyCondition Status=True condition indicates
	// DataPlaneNode is ready.
	DataPlaneNodeReadyCondition condition.Type = "DataPlaneNodeReady"

	// DataPlaneNodeReadyMessage ready
	DataPlaneNodeReadyMessage = "DataPlaneNode ready"

	// DataPlaneNodeReadyWaitingMessage ready
	DataPlaneNodeReadyWaitingMessage = "DataPlaneNode not yet ready"

	// DataPlaneNodeErrorMessage error
	DataPlaneNodeErrorMessage = "DataPlaneNode error occurred %s"

	// ConfigureNetworkReadyCondition Status=True condition indicates if the
	// network configuration is finished and successful.
	ConfigureNetworkReadyCondition condition.Type = "ConfigureNetworkReady"

	// ConfigureNetworkReadyMessage ready
	ConfigureNetworkReadyMessage = "ConfigureNetwork ready"

	// ConfigureNetworkReadyWaitingMessage not yet ready
	ConfigureNetworkReadyWaitingMessage = "ConfigureNetwork not yet ready"

	// ConfigureNetworkErrorMessage error
	ConfigureNetworkErrorMessage = "ConfigureNetwork error occurred %s"

	// ValidateNetworkReadyCondition Status=True condition indicates if the
	// network validation is finished and successful.
	ValidateNetworkReadyCondition condition.Type = "ValidateNetworkReady"

	// ValidateNetworkReadyMessage ready
	ValidateNetworkReadyMessage = "ValidateNetwork ready"

	// ValidateNetworkReadyWaitingMessage not yet ready
	ValidateNetworkReadyWaitingMessage = "ValidateNetwork not yet ready"

	// ValidateNetworkErrorMessage error
	ValidateNetworkErrorMessage = "ValidateNetwork error occurred %s"

	// InstallOSReadyCondition Status=True condition indicates if the
	// OS configuration is finished and successful.
	InstallOSReadyCondition condition.Type = "InstallOSReady"

	// InstallOSReadyMessage ready
	InstallOSReadyMessage = "InstallOS ready"

	// InstallOSReadyWaitingMessage not yet ready
	InstallOSReadyWaitingMessage = "InstallOS not yet ready"

	// InstallOSErrorMessage error
	InstallOSErrorMessage = "InstallOS error occurred %s"

	// ConfigureOSReadyCondition Status=True condition indicates if the
	// OS configuration is finished and successful.
	ConfigureOSReadyCondition condition.Type = "ConfigureOSReady"

	// ConfigureOSReadyMessage ready
	ConfigureOSReadyMessage = "ConfigureOS ready"

	// ConfigureOSReadyWaitingMessage not yet ready
	ConfigureOSReadyWaitingMessage = "ConfigureOS not yet ready"

	// ConfigureOSErrorMessage error
	ConfigureOSErrorMessage = "ConfigureOS error occurred %s"

	// RunOSReadyCondition Status=True condition indicates if the
	// OS configuration is finished and successful.
	RunOSReadyCondition condition.Type = "RunOSReady"

	// RunOSReadyMessage ready
	RunOSReadyMessage = "RunOS ready"

	// RunOSReadyWaitingMessage not yet ready
	RunOSReadyWaitingMessage = "RunOS not yet ready"

	// RunOSErrorMessage error
	RunOSErrorMessage = "RunOS error occurred %s"

	// InstallOpenStackReadyCondition Status=True condition indicates if the
	// OpenStack configuration is finished and successful.
	InstallOpenStackReadyCondition condition.Type = "InstallOpenStackReady"

	// InstallOpenStackReadyMessage ready
	InstallOpenStackReadyMessage = "InstallOpenStack ready"

	// InstallOpenStackReadyWaitingMessage not yet ready
	InstallOpenStackReadyWaitingMessage = "InstallOpenStack not yet ready"

	// InstallOpenStackErrorMessage error
	InstallOpenStackErrorMessage = "InstallOpenStack error occurred %s"

	// ConfigureOpenStackReadyCondition Status=True condition indicates if the
	// OpenStack configuration is finished and successful.
	ConfigureOpenStackReadyCondition condition.Type = "ConfigureOpenStackReady"

	// ConfigureOpenStackReadyMessage ready
	ConfigureOpenStackReadyMessage = "ConfigureOpenStack ready"

	// ConfigureOpenStackReadyWaitingMessage not yet ready
	ConfigureOpenStackReadyWaitingMessage = "ConfigureOpenStack not yet ready"

	// ConfigureOpenStackErrorMessage error
	ConfigureOpenStackErrorMessage = "ConfigureOpenStack error occurred %s"

	// RunOpenStackReadyCondition Status=True condition indicates if the
	// OpenStack configuration is finished and successful.
	RunOpenStackReadyCondition condition.Type = "RunOpenStackReady"

	// RunOpenStackReadyMessage ready
	RunOpenStackReadyMessage = "RunOpenStack ready"

	// RunOpenStackReadyWaitingMessage not yet ready
	RunOpenStackReadyWaitingMessage = "RunOpenStack not yet ready"

	// RunOpenStackErrorMessage error
	RunOpenStackErrorMessage = "RunOpenStack error occurred %s"
)
