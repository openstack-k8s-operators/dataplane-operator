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

package deployment

const (

	// CtlPlaneNetwork - default CtlPlane Network Name in NetConfig
	CtlPlaneNetwork = "CtlPlane"

	// ValidateNetworkLabel for ValidateNetwork OpenStackAnsibleEE
	ValidateNetworkLabel = "dataplane-deployment-validate-network"

	// InstallOSLabel for InstallOS OpenStackAnsibleEE
	InstallOSLabel = "dataplane-deployment-install-os"

	// ConfigureOSLabel for ConfigureOS OpenStackAnsibleEE
	ConfigureOSLabel = "dataplane-deployment-configure-os"

	// RunOSLabel for RunOS OpenStackAnsibleEE
	RunOSLabel = "dataplane-deployment-run-os"

	// ConfigureCephClientLabel for ConfigureCephClient OpenStackAnsibleEE
	ConfigureCephClientLabel = "dataplane-deployment-configure-ceph-client"

	// InstallOpenStackLabel for InstallOpenStack OpenStackAnsibleEE
	InstallOpenStackLabel = "dataplane-deployment-install-openstack"

	// ConfigureOpenStackLabel for ConfigureOpenStack OpenStackAnsibleEE
	ConfigureOpenStackLabel = "dataplane-deployment-configure-openstack"

	// RunOpenStackLabel for RunOpenStack OpenStackAnsibleEE
	RunOpenStackLabel = "dataplane-deployment-run-openstack"

	// NicConfigTemplateFile is the custom nic config file we use when user provided network config templates are provided.
	NicConfigTemplateFile = "/runner/network/nic-config-template"

	// ConfigPaths base path for volume mounts in OpenStackAnsibleEE pod
	ConfigPaths = "/var/lib/openstack/configs"
)
