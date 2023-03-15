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
	"github.com/openstack-k8s-operators/lib-common/modules/storage"
)

// NodeSection is a specification of the node attributes
type NodeSection struct {

	// +kubebuilder:validation:Optional
	// NetworkConfig - Network configuration details. Contains os-net-config
	// related properties.
	NetworkConfig NetworkConfigSection `json:"networkConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// Networks - Instance networks
	Networks []NetworksSection `json:"networks,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	// Managed - Whether the node is actually provisioned (True) or should be
	// treated as preprovisioned (False)
	Managed bool `json:"managed,omitempty"`

	// +kubebuilder:validation:Optional
	// ManagementNetwork - Name of network to use for management (SSH/Ansible)
	ManagementNetwork string `json:"managementNetwork,omitempty"`

	// +kubebuilder:validation:Optional
	// AnsibleUser SSH user for Ansible connection
	AnsibleUser string `json:"ansibleUser,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	// AnsiblePort SSH port for Ansible connection
	AnsiblePort int `json:"ansiblePort,omitempty"`

	// +kubebuilder:validation:Optional
	// AnsibleVars for configuring ansible
	AnsibleVars string `json:"ansibleVars,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret"}
	// AnsibleSSHPrivateKeySecret Private SSH Key secret containing private SSH
	// key for connecting to node. Must be of the form:
	// Secret.data.ssh-privatekey: <base64 encoded private key contents>
	// https://kubernetes.io/docs/concepts/configuration/secret/#ssh-authentication-secrets
	AnsibleSSHPrivateKeySecret string `json:"ansibleSSHPrivateKeySecret"`

	// ExtraMounts containing files which can be mounted into an Ansible Execution Pod
	// +kubebuilder:validation:Optional
	// +kubebuilder:default={}
	ExtraMounts []storage.VolMounts `json:"extraMounts"`
}

// DeployStrategySection for fields controlling the deployment
type DeployStrategySection struct {

	// +kubebuilder:default=true
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	// Deploy boolean to trigger ansible execution
	Deploy bool `json:"deploy"`

	// +kubebuilder:validation:Optional
	// AnsibleTags for ansible execution
	AnsibleTags string `json:"ansibleTags,omitempty"`
}

// NetworkConfigSection is a specification of the Network configuration details
type NetworkConfigSection struct {

	// +kubebuilder:validation:Optional
	// Template - ansible j2 nic config template to use when applying node
	// network configuration
	Template string `json:"template,omitempty" yaml:"template,omitempty"`
}

// NetworksSection is a specification of the network attributes
type NetworksSection struct {

	// +kubebuilder:validation:Optional
	// Network - Network name to configure
	Network string `json:"network,omitempty" yaml:"network,omitempty"`

	// +kubebuilder:validation:Optional
	// FixedIP - Specific IP address to use for this network
	FixedIP string `json:"fixedIP,omitempty" yaml:"fixedIP,omitempty"`
}
