/*
Copyright 2022.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OpenStackDataPlaneNodeSpec defines the desired state of OpenStackDataPlaneNode
type OpenStackDataPlaneNodeSpec struct {

	// +kubebuilder:validation:Optional
	// Node - node attributes specific to this node
	Node NodeSection `json:"node,omitempty"`

	// +kubebuilder:validation:Optional
	// Role - role name for this node
	Role string `json:"templateRef,omitempty"`
}

type NodeSection struct {

	// +kubebuilder:validation:Optional
	// HostName - node name
	HostName string `json:"hostName,omitempty"`

	// +kubebuilder:validation:Optional
	// NetworkConfig - Network configuration details. Contains os-net-config
	// related properties.
	NetworkConfig NetworkConfigSection `json:"networkConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// Networks - Instance networks
	Networks []NetworksSection `json:"networks,omitempty"`

	// +kubebuilder:validation:Optional
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
	// AnsibleHost SSH host for Ansible connection
	AnsibleHost string `json:"ansibleHost,omitempty"`

	// +kubebuilder:validation:Optional
	// AnsiblePort SSH port for Ansible connection
	AnsiblePort int `json:"ansiblePort,omitempty"`
}

type NetworkConfigSection struct {

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=templates/net_config_bridge.j2
	// Template - ansible j2 nic config template to use when applying node
	// network configuration
	Template string `json:"template,omitempty"`
}

type NetworksSection struct {

	// +kubebuilder:validation:Optional
	// Network - Network name to configure
	Network string `json:"template,omitempty"`

	// +kubebuilder:validation:Optional
	// FixedIP - Specific IP address to use for this network
	FixedIP string `json:"fixedIP,omitempty"`
}

// OpenStackDataPlaneNodeStatus defines the observed state of OpenStackDataPlaneNode
type OpenStackDataPlaneNodeStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// OpenStackDataPlaneNode is the Schema for the openstackdataplanenodes API
type OpenStackDataPlaneNode struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenStackDataPlaneNodeSpec   `json:"spec,omitempty"`
	Status OpenStackDataPlaneNodeStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OpenStackDataPlaneNodeList contains a list of OpenStackDataPlaneNode
type OpenStackDataPlaneNodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenStackDataPlaneNode `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OpenStackDataPlaneNode{}, &OpenStackDataPlaneNodeList{})
}
