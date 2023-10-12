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
	"fmt"

	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/storage"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OpenStackDataPlaneDeploymentSpec defines the desired state of OpenStackDataPlaneDeployment
type OpenStackDataPlaneDeploymentSpec struct {

	// +kubebuilder:validation:Required
	// NodeSets is the list of NodeSets deployed
	NodeSets []string `json:"nodeSets"`

	// +kubebuilder:default={download-cache,configure-network,validate-network,install-os,configure-os,run-os,ovn,libvirt,nova,telemetry}
	// Services list
	Services []string `json:"services"`

	// +kubebuilder:validation:Optional
	// Env is a list containing the environment variables to pass to the pod
	Env []corev1.EnvVar `json:"env,omitempty"`

	// +kubebuilder:validation:Optional
	// NetworkAttachments is a list of NetworkAttachment resource names to pass to the ansibleee resource
	// which allows to connect the ansibleee runner to the given network
	NetworkAttachments []string `json:"networkAttachments,omitempty"`

	// ExtraMounts containing files which can be mounted into an Ansible Execution Pod
	// +kubebuilder:validation:Optional
	ExtraMounts []storage.VolMounts `json:"extraMounts,omitempty"`

	// AnsibleTags for ansible execution
	// +kubebuilder:validation:Optional
	AnsibleTags string `json:"ansibleTags,omitempty"`

	// AnsibleLimit for ansible execution
	// +kubebuilder:validation:Optional
	AnsibleLimit string `json:"ansibleLimit,omitempty"`

	// AnsibleSkipTags for ansible execution
	// +kubebuilder:validation:Optional
	AnsibleSkipTags string `json:"ansibleSkipTags,omitempty"`
}

// OpenStackDataPlaneDeploymentStatus defines the observed state of OpenStackDataPlaneDeployment
type OpenStackDataPlaneDeploymentStatus struct {
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors={"urn:alm:descriptor:io.kubernetes.conditions"}
	// Conditions
	Conditions condition.Conditions `json:"conditions,omitempty" optional:"true"`

	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	// Deployed
	Deployed bool `json:"deployed,omitempty" optional:"true"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+operator-sdk:csv:customresourcedefinitions:displayName="OpenStack Data Plane Deployments"
//+kubebuilder:resource:shortName=osdpd;osdpdeployment;osdpdeployments
//+kubebuilder:printcolumn:name="NodeSets",type="string",JSONPath=".spec.nodeSets",description="NodeSets"
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[0].status",description="Status"
//+kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.conditions[0].message",description="Message"

// OpenStackDataPlaneDeployment is the Schema for the openstackdataplanedeployments API
type OpenStackDataPlaneDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenStackDataPlaneDeploymentSpec   `json:"spec,omitempty"`
	Status OpenStackDataPlaneDeploymentStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OpenStackDataPlaneDeploymentList contains a list of OpenStackDataPlaneDeployment
type OpenStackDataPlaneDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenStackDataPlaneDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OpenStackDataPlaneDeployment{}, &OpenStackDataPlaneDeploymentList{})
}

// IsReady - returns true if the OpenStackDataPlaneDeployment is ready
func (instance OpenStackDataPlaneDeployment) IsReady() bool {
	return instance.Status.Conditions.IsTrue(condition.ReadyCondition)
}

// InitConditions - Initializes Status Conditons
func (instance *OpenStackDataPlaneDeployment) InitConditions() {
	instance.Status.Conditions = condition.Conditions{}

	cl := condition.CreateList(
		condition.UnknownCondition(condition.DeploymentReadyCondition, condition.InitReason, condition.InitReason),
		condition.UnknownCondition(condition.InputReadyCondition, condition.InitReason, condition.InitReason),
	)

	if instance.Spec.NodeSets != nil {
		for _, nodeSet := range instance.Spec.NodeSets {
			readyCondition := condition.Type(fmt.Sprintf(NodeSetDeploymentReadyCondition, nodeSet))
			cl = append(cl, *condition.UnknownCondition(readyCondition, condition.InitReason, condition.InitReason))
		}
	}

	instance.Status.Conditions.Init(&cl)
	instance.Status.Deployed = false
}

// GetAnsibleEESpec - get the fields that will be passed to AEE
func (instance OpenStackDataPlaneDeployment) GetAnsibleEESpec() AnsibleEESpec {
	return AnsibleEESpec{
		NetworkAttachments: instance.Spec.NetworkAttachments,
		ExtraMounts:        instance.Spec.ExtraMounts,
		Env:                instance.Spec.Env,
		AnsibleTags:        instance.Spec.AnsibleTags,
		AnsibleLimit:       instance.Spec.AnsibleLimit,
		AnsibleSkipTags:    instance.Spec.AnsibleSkipTags,
	}
}
