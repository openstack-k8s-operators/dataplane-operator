/*
Copyright 2024.

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

package util

import (
	"context"
	"errors"

	dataplanev1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	openstackv1 "github.com/openstack-k8s-operators/openstack-operator/apis/core/v1beta1"
)

// GetVersion
func GetVersion(ctx context.Context, helper *helper.Helper) (*openstackv1.OpenStackVersion, error) {
	client := helper.GetClient()
	log := helper.GetLogger()
	var version *openstackv1.OpenStackVersion
	versions := &openstackv1.OpenStackVersionList{}
	if err := client.List(ctx, versions); err != nil {
		log.Error(err, "Unable to retrieve OpenStackVersions %w")
		return nil, err
	}
	if len(versions.Items) > 1 {
		errorMsg := "Found multiple OpenStackVersions when at most 1 should exist"
		err := errors.New(errorMsg)
		log.Error(err, errorMsg)
		return nil, err
	} else if len(versions.Items) == 1 {
		version = &versions.Items[0]
	}

	return version, nil
}

// GetContainerImages
func GetContainerImages(version *openstackv1.OpenStackVersion) dataplanev1.ContainerImages {

	var containerImages dataplanev1.ContainerImages

	// Set the containerImages variable for the container images If there is an
	// OpenStackVersion, use the value from there, else use the default value.
	if version != nil {
		containerImages.FrrImage = version.Status.ContainerImages.FrrImage
		containerImages.IscsiDImage = version.Status.ContainerImages.IscsiDImage
		containerImages.LogrotateCrondImage = version.Status.ContainerImages.LogrotateCrondImage
		containerImages.NeutronMetadataAgentImage = version.Status.ContainerImages.NeutronMetadataAgentImage
		containerImages.NovaComputeImage = version.Status.ContainerImages.NovaComputeImage
		containerImages.OvnControllerImage = version.Status.ContainerImages.OvnControllerImage
		containerImages.OvnBgpAgentImage = version.Status.ContainerImages.OvnBgpAgentImage
		containerImages.TelemetryCeilometerComputeImage = version.Status.ContainerImages.TelemetryCeilometerComputeImage
		containerImages.TelemetryCeilometerIpmiImage = version.Status.ContainerImages.TelemetryCeilometerIpmiImage
		containerImages.TelemetryNodeExporterImage = version.Status.ContainerImages.TelemetryNodeExporterImage
	} else {
		containerImages.FrrImage = dataplanev1.ContainerImageDefaults.FrrImage
		containerImages.IscsiDImage = dataplanev1.ContainerImageDefaults.IscsiDImage
		containerImages.LogrotateCrondImage = dataplanev1.ContainerImageDefaults.LogrotateCrondImage
		containerImages.NeutronMetadataAgentImage = dataplanev1.ContainerImageDefaults.NeutronMetadataAgentImage
		containerImages.NovaComputeImage = dataplanev1.ContainerImageDefaults.NovaComputeImage
		containerImages.OvnControllerImage = dataplanev1.ContainerImageDefaults.OvnControllerImage
		containerImages.OvnBgpAgentImage = dataplanev1.ContainerImageDefaults.OvnBgpAgentImage
		containerImages.TelemetryCeilometerComputeImage = dataplanev1.ContainerImageDefaults.TelemetryCeilometerComputeImage
		containerImages.TelemetryCeilometerIpmiImage = dataplanev1.ContainerImageDefaults.TelemetryCeilometerIpmiImage
		containerImages.TelemetryNodeExporterImage = dataplanev1.ContainerImageDefaults.TelemetryNodeExporterImage
	}

	return containerImages
}
