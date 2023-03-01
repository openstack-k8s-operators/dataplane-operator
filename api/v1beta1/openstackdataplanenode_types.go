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
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OpenStackDataPlaneNodeSpec defines the desired state of OpenStackDataPlaneNode
type OpenStackDataPlaneNodeSpec struct {

	// +kubebuilder:validation:Optional
	// HostName - node name
	HostName string `json:"hostName,omitempty"`

	// +kubebuilder:validation:Optional
	// Node - node attributes specific to this node
	Node NodeSection `json:"node,omitempty"`

	// +kubebuilder:validation:Optional
	// Role - role name for this node
	Role string `json:"role,omitempty"`

	// +kubebuilder:validation:Optional
	// AnsibleHost SSH host for Ansible connection
	AnsibleHost string `json:"ansibleHost,omitempty"`

	// +kubebuilder:default=true
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	// Deploy boolean to trigger ansible execution
	Deploy bool `json:"deploy"`

	// +kubebuilder:validation:Optional
	// NodeFrom - Existing node name to reference. Can only be used if Node is
	// empty.
	NodeFrom string `json:"nodeFrom,omitempty"`

	// +kubebuilder:validation:Optional
	// NetworkAttachments is a list of NetworkAttachment resource names to pass to the ansibleee resource
	// which allows to connect the ansibleee runner to the given network
	NetworkAttachments []string `json:"networkAttachments"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="quay.io/openstack-k8s-operators/openstack-ansibleee-runner:latest"
	// OpenStackAnsibleEERunnerImage image to use as the ansibleEE runner image
	OpenStackAnsibleEERunnerImage string `json:"openStackAnsibleEERunnerImage"`
}

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
}

// NetworkConfigSection is a specification of the Network configuration details
type NetworkConfigSection struct {

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=templates/net_config_bridge.j2
	// Template - ansible j2 nic config template to use when applying node
	// network configuration
	Template string `json:"template,omitempty"`
}

// NetworksSection is a specification of the network attributes
type NetworksSection struct {

	// +kubebuilder:validation:Optional
	// Network - Network name to configure
	Network string `json:"network,omitempty"`

	// +kubebuilder:validation:Optional
	// FixedIP - Specific IP address to use for this network
	FixedIP string `json:"fixedIP,omitempty"`
}

// OpenStackDataPlaneNodeStatus defines the observed state of OpenStackDataPlaneNode
type OpenStackDataPlaneNodeStatus struct {

	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors={"urn:alm:descriptor:io.kubernetes.conditions"}
	// Conditions
	Conditions condition.Conditions `json:"conditions,omitempty" optional:"true"`

	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	// Deployed
	Deployed bool `json:"deployed,omitempty" optional:"true"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+operator-sdk:csv:customresourcedefinitions:displayName="OpenStack Data Plane Node"
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[0].status",description="Status"
//+kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.conditions[0].message",description="Message"

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

// IsReady - returns true if the DataPlane is ready
func (instance OpenStackDataPlaneNode) IsReady() bool {
	return instance.Status.Conditions.IsTrue(DataPlaneNodeReadyCondition)
}
