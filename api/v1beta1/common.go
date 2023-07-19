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
	"reflect"

	infranetworkv1 "github.com/openstack-k8s-operators/infra-operator/apis/network/v1beta1"
	"github.com/openstack-k8s-operators/lib-common/modules/storage"
	corev1 "k8s.io/api/core/v1"
)

// NodeSection is a specification of the node attributes
type NodeSection struct {

	// +kubebuilder:validation:Optional
	// HostName - node name
	HostName string `json:"hostName,omitempty"`

	// +kubebuilder:validation:Optional
	// NetworkAttachments is a list of NetworkAttachment resource names to pass to the ansibleee resource
	// which allows to connect the ansibleee runner to the given network
	NetworkAttachments []string `json:"networkAttachments,omitempty"`

	// +kubebuilder:validation:Optional
	// NetworkConfig - Network configuration details. Contains os-net-config
	// related properties.
	NetworkConfig NetworkConfigSection `json:"networkConfig"`

	// +kubebuilder:validation:Optional
	// Networks - Instance networks
	Networks []infranetworkv1.IPSetNetwork `json:"networks,omitempty"`

	// +kubebuilder:validation:Optional
	// ManagementNetwork - Name of network to use for management (SSH/Ansible)
	ManagementNetwork string `json:"managementNetwork,omitempty"`

	// +kubebuilder:validation:Optional
	// AnsibleHost SSH host for Ansible connection
	AnsibleHost string `json:"ansibleHost,omitempty"`

	// +kubebuilder:validation:Optional
	// AnsibleUser SSH user for Ansible connection
	AnsibleUser string `json:"ansibleUser,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	// AnsiblePort SSH port for Ansible connection
	AnsiblePort int `json:"ansiblePort,omitempty"`

	// +kubebuilder:validation:Optional
	// AnsibleVars for configuring ansible
	AnsibleVars map[string]json.RawMessage `json:"ansibleVars,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret"}
	// AnsibleSSHPrivateKeySecret Private SSH Key secret containing private SSH
	// key for connecting to node. Must be of the form:
	// Secret.data.ssh-privatekey: <base64 encoded private key contents>
	// <https://kubernetes.io/docs/concepts/configuration/secret/#ssh-authentication-secrets>
	AnsibleSSHPrivateKeySecret string `json:"ansibleSSHPrivateKeySecret,omitempty"`

	// ExtraMounts containing files which can be mounted into an Ansible Execution Pod
	// +kubebuilder:validation:Optional
	ExtraMounts []storage.VolMounts `json:"extraMounts,omitempty"`

	// UserData  node specific user-data
	// +kubebuilder:validation:Optional
	UserData *corev1.SecretReference `json:"userData,omitempty"`

	// NetworkData  node specific network-data
	// +kubebuilder:validation:Optional
	NetworkData *corev1.SecretReference `json:"networkData,omitempty"`

	// NovaTemplate specifies the parameters for the compute service deployment
	// on the EDPM node. If it is specified both in OpenstackDataPlaneRole and
	// the OpenstackDataPlaneNode for the same EDPM node then the configuration
	// in OpenstackDataPlaneNode will be used and the configuration in the
	// OpenstackDataPlaneRole will be ignored. If this is defined in neither
	// then compute service(s) will not be deployed on the EDPM node.
	// +kubebuilder:validation:Optional
	Nova *NovaTemplate `json:"nova,omitempty"`
}

// NodeTemplate is a specification of the node attributes
type NodeTemplate struct {
	// +kubebuilder:validation:Required
	// Nodes - Map of Node Names and node specific data. Values here override defaults in the
	// upper level section.
	Nodes map[string]NodeSection `json:"nodeNames"`

	// +kubebuilder:validation:Optional
	// HostName - node name
	HostName string `json:"hostName,omitempty"`

	// +kubebuilder:validation:Optional
	// NetworkAttachments is a list of NetworkAttachment resource names to pass to the ansibleee resource
	// which allows to connect the ansibleee runner to the given network
	NetworkAttachments []string `json:"networkAttachments,omitempty"`

	// +kubebuilder:validation:Optional
	// NetworkConfig - Network configuration details. Contains os-net-config
	// related properties.
	NetworkConfig NetworkConfigSection `json:"networkConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// Networks - Instance networks
	Networks []infranetworkv1.IPSetNetwork `json:"networks,omitempty"`

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

	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	// AnsibleVars for configuring ansible
	AnsibleVars map[string]json.RawMessage `json:"ansibleVars,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret"}
	// AnsibleSSHPrivateKeySecret Private SSH Key secret containing private SSH
	// key for connecting to node. Must be of the form:
	// Secret.data.ssh-privatekey: <base64 encoded private key contents>
	// <https://kubernetes.io/docs/concepts/configuration/secret/#ssh-authentication-secrets>
	AnsibleSSHPrivateKeySecret string `json:"ansibleSSHPrivateKeySecret,omitempty"`

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

// DeployStrategySection for fields controlling the deployment
type DeployStrategySection struct {
	// +kubebuilder:default=true
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	// Deploy boolean to trigger ansible execution
	Deploy bool `json:"deploy"`

	// +kubebuilder:validation:Optional
	// AnsibleTags for ansible execution
	AnsibleTags string `json:"ansibleTags,omitempty"`

	// +kubebuilder:validation:Optional
	// AnsibleLimit for ansible execution
	AnsibleLimit string `json:"ansibleLimit,omitempty"`

	// +kubebuilder:validation:Optional
	// AnsibleSkipTags for ansible execution
	AnsibleSkipTags string `json:"ansibleSkipTags,omitempty"`
}

// NetworkConfigSection is a specification of the Network configuration details
type NetworkConfigSection struct {

	// +kubebuilder:validation:Optional
	// Template - Contains a Ansible j2 nic config template to use when applying node
	// network configuration
	Template string `json:"template,omitempty" yaml:"template,omitempty"`
}

// UniqueSpecFields - the array of fields that must be unique between role and nodes
var UniqueSpecFields = []string{"NetworkAttachments"}

// AssertUniquenessBetween - compare specs for uniqueness
func AssertUniquenessBetween(spec interface{}, otherSpec interface{}, suffix string) []string {
	vSpec := reflect.ValueOf(spec)
	vOtherSpec := reflect.ValueOf(otherSpec)

	var errorMsgs []string
	for _, field := range UniqueSpecFields {
		value := vSpec.FieldByName(field)
		otherValue := vOtherSpec.FieldByName(field)
		if value.IsZero() || otherValue.IsZero() {
			continue
		}
		if !reflect.DeepEqual(value.Interface(), otherValue.Interface()) {
			errorMsgs = append(errorMsgs, field+" mismatch between "+suffix)
		}
	}
	return errorMsgs
}

// AnsibleEESpec is a specification of the ansible EE attributes
type AnsibleEESpec struct {
	// +kubebuilder:validation:Optional
	// NetworkAttachments is a list of NetworkAttachment resource names to pass to the ansibleee resource
	// which allows to connect the ansibleee runner to the given network
	NetworkAttachments []string `json:"networkAttachments,omitempty"`
	// +kubebuilder:validation:Optional
	// OpenStackAnsibleEERunnerImage image to use as the ansibleEE runner image
	OpenStackAnsibleEERunnerImage string `json:"openStackAnsibleEERunnerImage,omitempty"`
	// +kubebuilder:validation:Optional
	// AnsibleTags for ansible execution
	AnsibleTags string `json:"ansibleTags,omitempty"`
	// +kubebuilder:validation:Optional
	// AnsibleLimit for ansible execution
	AnsibleLimit string `json:"ansibleLimit,omitempty"`
	// +kubebuilder:validation:Optional
	// AnsibleSkipTags for ansible execution
	AnsibleSkipTags string `json:"ansibleSkipTags,omitempty"`
	// ExtraMounts containing files which can be mounted into an Ansible Execution Pod
	// +kubebuilder:validation:Optional
	ExtraMounts []storage.VolMounts `json:"extraMounts,omitempty"`
	// Env is a list containing the environment variables to pass to the pod
	Env []corev1.EnvVar `json:"env,omitempty"`
	// DNSConfig for setting dnsservers
	DNSConfig *corev1.PodDNSConfig `json:"dnsConfig,omitempty"`
}
