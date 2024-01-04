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
	"encoding/json"

	infranetworkv1 "github.com/openstack-k8s-operators/infra-operator/apis/network/v1beta1"
	"github.com/openstack-k8s-operators/lib-common/modules/storage"
	corev1 "k8s.io/api/core/v1"
)

// AnsibleOpts defines a logical grouping of Ansible related configuration options.
type AnsibleOpts struct {
	// AnsibleUser SSH user for Ansible connection
	// +kubebuilder:validation:Optional
	AnsibleUser string `json:"ansibleUser"`

	// AnsibleHost SSH host for Ansible connection
	// +kubebuilder:validation:Optional
	AnsibleHost string `json:"ansibleHost,omitempty"`

	// AnsiblePort SSH port for Ansible connection
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	AnsiblePort int `json:"ansiblePort,omitempty"`

	// AnsibleVars for configuring ansible
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	AnsibleVars map[string]json.RawMessage `json:"ansibleVars,omitempty"`
}

// NodeSection defines the top level attributes inherited by nodes in the CR.
type NodeSection struct {
	// HostName - node name
	// +kubebuilder:validation:Optional
	HostName string `json:"hostName,omitempty"`

	// Networks - Instance networks
	// +kubebuilder:validation:Optional
	Networks []infranetworkv1.IPSetNetwork `json:"networks,omitempty"`

	// ManagementNetwork - Name of network to use for management (SSH/Ansible)
	// +kubebuilder:validation:Optional
	ManagementNetwork string `json:"managementNetwork,omitempty"`

	// Ansible is the group of Ansible related configuration options.
	// +kubebuilder:validation:Optional
	Ansible AnsibleOpts `json:"ansible,omitempty"`

	// ExtraMounts containing files which can be mounted into an Ansible Execution Pod
	// +kubebuilder:validation:Optional
	ExtraMounts []storage.VolMounts `json:"extraMounts,omitempty"`

	// UserData  node specific user-data
	// +kubebuilder:validation:Optional
	UserData *corev1.SecretReference `json:"userData,omitempty"`

	// NetworkData  node specific network-data
	// +kubebuilder:validation:Optional
	NetworkData *corev1.SecretReference `json:"networkData,omitempty"`
}

// NodeTemplate is a specification of the node attributes that override top level attributes.
type NodeTemplate struct {
	// AnsibleSSHPrivateKeySecret Name of a private SSH key secret containing
	// private SSH key for connecting to node.
	// The named secret must be of the form:
	// Secret.data.ssh-privatekey: <base64 encoded private key contents>
	// <https://kubernetes.io/docs/concepts/configuration/secret/#ssh-authentication-secrets>
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret"}
	AnsibleSSHPrivateKeySecret string `json:"ansibleSSHPrivateKeySecret"`

	// Networks - Instance networks
	// +kubebuilder:validation:Optional
	Networks []infranetworkv1.IPSetNetwork `json:"networks,omitempty"`

	// ManagementNetwork - Name of network to use for management (SSH/Ansible)
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=ctlplane
	ManagementNetwork string `json:"managementNetwork"`

	// Ansible is the group of Ansible related configuration options.
	// +kubebuilder:validation:Optional
	Ansible AnsibleOpts `json:"ansible,omitempty"`

	// ExtraMounts containing files which can be mounted into an Ansible Execution Pod
	// +kubebuilder:validation:Optional
	ExtraMounts []storage.VolMounts `json:"extraMounts,omitempty"`

	// UserData  node specific user-data
	// +kubebuilder:validation:Optional
	UserData *corev1.SecretReference `json:"userData,omitempty"`

	// NetworkData  node specific network-data
	// +kubebuilder:validation:Optional
	NetworkData *corev1.SecretReference `json:"networkData,omitempty"`
}

// AnsibleEESpec is a specification of the ansible EE attributes
type AnsibleEESpec struct {
	// NetworkAttachments is a list of NetworkAttachment resource names to pass to the ansibleee resource
	// which allows to connect the ansibleee runner to the given network
	NetworkAttachments []string `json:"networkAttachments"`
	// OpenStackAnsibleEERunnerImage image to use as the ansibleEE runner image
	OpenStackAnsibleEERunnerImage string `json:"openStackAnsibleEERunnerImage,omitempty"`
	// AnsibleTags for ansible execution
	AnsibleTags string `json:"ansibleTags,omitempty"`
	// AnsibleLimit for ansible execution
	AnsibleLimit string `json:"ansibleLimit,omitempty"`
	// AnsibleSkipTags for ansible execution
	AnsibleSkipTags string `json:"ansibleSkipTags,omitempty"`
	// ExtraMounts containing files which can be mounted into an Ansible Execution Pod
	ExtraMounts []storage.VolMounts `json:"extraMounts,omitempty"`
	// Env is a list containing the environment variables to pass to the pod
	Env []corev1.EnvVar `json:"env,omitempty"`
	// DNSConfig for setting dnsservers
	DNSConfig *corev1.PodDNSConfig `json:"dnsConfig,omitempty"`
}
