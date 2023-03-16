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
	condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"
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

	// +kubebuilder:validation:Optional
	// DeployStrategy section to control how the node is deployed
	DeployStrategy DeployStrategySection `json:"deployStrategy,omitempty"`

	// +kubebuilder:validation:Optional
	// NetworkAttachments is a list of NetworkAttachment resource names to pass to the ansibleee resource
	// which allows to connect the ansibleee runner to the given network
	NetworkAttachments []string `json:"networkAttachments"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="quay.io/openstack-k8s-operators/openstack-ansibleee-runner:latest"
	// OpenStackAnsibleEERunnerImage image to use as the ansibleEE runner image
	OpenStackAnsibleEERunnerImage string `json:"openStackAnsibleEERunnerImage"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+operator-sdk:csv:customresourcedefinitions:displayName="OpenStack Data Plane Node"
// +kubebuilder:resource:shortName=osdpnode;osdpnodes
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[0].status",description="Status"
//+kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.conditions[0].message",description="Message"

// OpenStackDataPlaneNode is the Schema for the openstackdataplanenodes API
type OpenStackDataPlaneNode struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenStackDataPlaneNodeSpec `json:"spec,omitempty"`
	Status OpenStackDataPlaneStatus   `json:"status,omitempty"`
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
	return instance.Status.Conditions.IsTrue(condition.ReadyCondition)
}
