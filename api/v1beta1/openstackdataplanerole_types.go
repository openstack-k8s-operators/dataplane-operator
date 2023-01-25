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
	// DataPlaneNodes - List of nodes
	DataPlaneNodes []DataPlaneNodeSection `json:"dataPlaneNodes,omitempty"`

	// +kubebuilder:validation:Optional
	// NodeTemplate - node attributes specific to this roles
	NodeTemplate NodeSection `json:"nodeTemplate,omitempty"`
}

type DataPlaneNodeSection struct {
	// +kubebuilder:validation:Optional
	// Node - node attributes specific to this node
	Node NodeSection `json:"node,omitempty"`

	// +kubebuilder:validation:Optional
	// NodeFrom - Existing node name to reference. Can only be used if Node is
	// empty.
	NodeFrom string `json:"nodeFrom,omitempty"`
}

// OpenStackDataPlaneRoleStatus defines the observed state of OpenStackDataPlaneRole
type OpenStackDataPlaneRoleStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// OpenStackDataPlaneRole is the Schema for the openstackdataplaneroles API
type OpenStackDataPlaneRole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenStackDataPlaneRoleSpec   `json:"spec,omitempty"`
	Status OpenStackDataPlaneRoleStatus `json:"status,omitempty"`
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
