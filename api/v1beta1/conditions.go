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

	// ConfigureNetworkReadyCondition Status=True condition indicates if the
	// network configuration is finished and successful.
	ConfigureNetworkReadyCondition condition.Type = "ConfigureNetworkReady"

	// ConfigureNetworkReadyMessage ready
	ConfigureNetworkReadyMessage = "ConfigureNetwork ready"

	// ConfigureNetworkReadyWaitingMessage not yet ready
	ConfigureNetworkReadyWaitingMessage = "ConfigureNetwork not yet ready"

	// ValidateNetworkReadyCondition Status=True condition indicates if the
	// network validation is finished and successful.
	ValidateNetworkReadyCondition condition.Type = "ValidateNetworkReady"

	// ValidateNetworkReadyMessage ready
	ValidateNetworkReadyMessage = "ValidateNetwork ready"

	// ValidateNetworkReadyWaitingMessage not yet ready
	ValidateNetworkReadyWaitingMessage = "ValidateNetwork not yet ready"
)
