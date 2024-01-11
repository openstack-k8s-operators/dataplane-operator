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
	ValidateNetworkLabel = "validate-network"

	// InstallOSLabel for InstallOS OpenStackAnsibleEE
	InstallOSLabel = "install-os"

	// ConfigureOSLabel for ConfigureOS OpenStackAnsibleEE
	ConfigureOSLabel = "configure-os"

	// RunOSLabel for RunOS OpenStackAnsibleEE
	RunOSLabel = "run-os"

	// InstallOpenStackLabel for InstallOpenStack OpenStackAnsibleEE
	InstallOpenStackLabel = "install-openstack"

	// ConfigureOpenStackLabel for ConfigureOpenStack OpenStackAnsibleEE
	ConfigureOpenStackLabel = "configure-openstack"

	// RunOpenStackLabel for RunOpenStack OpenStackAnsibleEE
	RunOpenStackLabel = "run-openstack"

	// InstallCertsLabel for InstallCerts OpenStackAnsibleEE
	InstallCertsLabel = "install-certs"

	// NicConfigTemplateFile is the custom nic config file we use when user provided network config templates are provided.
	NicConfigTemplateFile = "/runner/network/nic-config-template"

	// ConfigPaths base path for volume mounts in OpenStackAnsibleEE pod
	ConfigPaths = "/var/lib/openstack/configs"

	// CertPaths base path for cert volume mount in OpenStackAnsibleEE pod
	CertPaths = "/var/lib/openstack/certs"

	// CACertPaths base path for CA cert volume mount in OpenStackAnsibleEE pod
	CACertPaths = "/var/lib/openstack/cacerts"
)
