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
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DataPlaneNetwork Type
type DataPlaneNetwork struct {
	// +kubebuilder:validation:Required
	// Network Name
	Name string `json:"name"`

	// +kubebuilder:validation:Required
	// Subnet Name
	SubnetName string `json:"subnetName"`

	// +kubebuilder:validation:Optional
	// Fixed Ip
	FixedIP string `json:"fixedIP,omitempty"`
}

// DataPlaneIPSetSpec defines the desired state of DataPlaneIPSet
type DataPlaneIPSetSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// VIP flag to indicate ipset is a request for a VIP
	VIP bool `json:"vip"`

	// +kubebuilder:validation:Optional
	// Host Networks used to generate IPs
	DataPlaneNetworks []DataPlaneNetwork `json:"networks,omitempty"`
}

// DataPlaneIPSetStatus defines the observed state of DataPlaneIPSet
type DataPlaneIPSetStatus struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:deepcopy-gen=false
	IPAddresses map[string]IPReservation `json:"ipaddresses,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	Allocated bool `json:"allocated,omitempty"`

	// Conditions - conditions to display in the OpenShift GUI, which reflect CurrentState
	// +kubebuilder:validation:Optional
	Conditions condition.Conditions `json:"conditions,omitempty"`
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// DataPlaneIPSet is the Schema for the dataplaneipsets API
type DataPlaneIPSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DataPlaneIPSetSpec   `json:"spec,omitempty"`
	Status DataPlaneIPSetStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DataPlaneIPSetList contains a list of DataPlaneIPSet
type DataPlaneIPSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DataPlaneIPSet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DataPlaneIPSet{}, &DataPlaneIPSetList{})
}

// InitCondition  Initializes conditions
func (instance DataPlaneIPSet) InitCondition() {
	if instance.Status.Conditions == nil {
		instance.Status.Conditions = condition.Conditions{}
	}
	cl := condition.CreateList(
		condition.UnknownCondition(DataPlaneIPSetReadyCondition, condition.InitReason, condition.ReadyInitMessage))
	// initialize conditions used later as Status=Unknown
	instance.Status.Conditions.Init(&cl)
}
