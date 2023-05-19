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

	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	ansibleeev1alpha1 "github.com/openstack-k8s-operators/openstack-ansibleee-operator/api/v1alpha1"
)

// OpenStackDataPlaneServiceSpec defines the desired state of OpenStackDataPlaneService
type OpenStackDataPlaneServiceSpec struct {
	// Label to use for service
	// +kubebuilder:validation:Optional
	Label string `json:"label,omitempty"`

	// Play is the playbook contents that ansible will run on execution.
	// If both Play and Role are specified, Play takes precedence
	Play string `json:"play,omitempty"`

	// Role is the description of an Ansible Role
	// +kubebuilder:validation:Optional
	Role *ansibleeev1alpha1.Role `json:"role,omitempty"`
}

// OpenStackDataPlaneServiceStatus defines the observed state of OpenStackDataPlaneService
type OpenStackDataPlaneServiceStatus struct {
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors={"urn:alm:descriptor:io.kubernetes.conditions"}
	// Conditions
	Conditions condition.Conditions `json:"conditions,omitempty" optional:"true"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// OpenStackDataPlaneService is the Schema for the openstackdataplaneservices API
type OpenStackDataPlaneService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenStackDataPlaneServiceSpec   `json:"spec,omitempty"`
	Status OpenStackDataPlaneServiceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OpenStackDataPlaneServiceList contains a list of OpenStackDataPlaneService
type OpenStackDataPlaneServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenStackDataPlaneService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OpenStackDataPlaneService{}, &OpenStackDataPlaneServiceList{})
}

// IsReady - returns true if service is ready to serve requests
func (instance OpenStackDataPlaneService) IsReady() bool {
	return instance.Status.Conditions.IsTrue(condition.ReadyCondition)
}

// InitConditions - Initializes Status Conditons
func (instance OpenStackDataPlaneService) InitConditions() {
	if instance.Status.Conditions == nil {
		instance.Status.Conditions = condition.Conditions{}
	}
	cl := condition.CreateList(condition.UnknownCondition(condition.ReadyCondition, condition.InitReason, condition.InitReason))
	// initialize conditions used later as Status=Unknown
	instance.Status.Conditions.Init(&cl)
}
