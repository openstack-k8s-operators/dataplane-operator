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
	ansibleeev1 "github.com/openstack-k8s-operators/openstack-ansibleee-operator/api/v1alpha1"
)

// KubeService represents a Kubernetes Service. It is called like this to avoid the extreme overloading of the
// Service term in this context
type KubeService struct {
	// Name of the Service will have in kubernetes
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Port is the port of the service
	// +kubebuilder:validation:Required
	Port int `json:"port"`

	// Protocol is the protocol used to connect to the endpoint
	// +kubebuilder:default=http
	Protocol string `json:"protocol,omitempty"`
}

// OpenStackDataPlaneServiceSpec defines the desired state of OpenStackDataPlaneService
type OpenStackDataPlaneServiceSpec struct {
	// Label to use for service
	// +kubebuilder:validation:Optional
	Label string `json:"label,omitempty"`

	// Services to create to expose possible external services in computes
	// +kubebuilder:validation:Optional
	Services []KubeService `json:"services,omitempty"`

	// Play is an inline playbook contents that ansible will run on execution.
	// If both Play and Roles are specified, Play takes precedence
	Play string `json:"play,omitempty"`

	// Playbook is a path to the playbook that ansible will run on this execution
	Playbook string `json:"playbook,omitempty"`

	// Role is the description of an Ansible Role
	// +kubebuilder:validation:Optional
	Role *ansibleeev1.Role `json:"role,omitempty"`

	// +kubebuilder:validation:Optional
	// OpenStackAnsibleEERunnerImage image to use as the ansibleEE runner image
	OpenStackAnsibleEERunnerImage string `json:"openStackAnsibleEERunnerImage,omitempty"`
}

// OpenStackDataPlaneServiceStatus defines the observed state of OpenStackDataPlaneService
type OpenStackDataPlaneServiceStatus struct {
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors={"urn:alm:descriptor:io.kubernetes.conditions"}
	// Conditions
	Conditions condition.Conditions `json:"conditions,omitempty" optional:"true"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:resource:shortName=osdpservice;osdpservices
//+operator-sdk:csv:customresourcedefinitions:displayName="OpenStack Data Plane Service"

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
