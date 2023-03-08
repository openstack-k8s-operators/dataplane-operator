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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// OpenStackDataPlaneRoleSpec defines the desired state of OpenStackDataPlaneRole
type OpenStackDataPlaneRoleSpec struct {
	// +kubebuilder:validation:Optional
	// DataPlane name of OpenStackDataPlane for this role
	DataPlane string `json:"dataPlane,omitempty"`

	// +kubebuilder:validation:Optional
	// NodeTemplate - node attributes specific to this roles
	NodeTemplate NodeSection `json:"nodeTemplate,omitempty"`

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
//+operator-sdk:csv:customresourcedefinitions:displayName="OpenStack Data Plane Role"

// OpenStackDataPlaneRole is the Schema for the openstackdataplaneroles API
type OpenStackDataPlaneRole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenStackDataPlaneRoleSpec `json:"spec,omitempty"`
	Status OpenStackDataPlaneStatus   `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OpenStackDataPlaneRoleList contains a list of OpenStackDataPlaneRole
type OpenStackDataPlaneRoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenStackDataPlaneRole `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OpenStackDataPlaneRole{}, &OpenStackDataPlaneRoleList{})
}
