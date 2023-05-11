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
	"fmt"

	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	corev1 "k8s.io/api/core/v1"
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

	// Env is a list containing the environment variables to pass to the pod
	Env []corev1.EnvVar `json:"env,omitempty"`

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
	return instance.Status.Conditions.IsTrue(condition.DeploymentReadyCondition)
}

// InitConditions - Initializes Status Conditons
func (instance *OpenStackDataPlaneNode) InitConditions() {
	instance.Status.Conditions = condition.Conditions{}

	cl := condition.CreateList(
		condition.UnknownCondition(condition.DeploymentReadyCondition, condition.InitReason, condition.InitReason),
		condition.UnknownCondition(SetupReadyCondition, condition.InitReason, condition.InitReason),
		condition.UnknownCondition(ConfigureNetworkReadyCondition, condition.InitReason, condition.InitReason),
		condition.UnknownCondition(ValidateNetworkReadyCondition, condition.InitReason, condition.InitReason),
		condition.UnknownCondition(InstallOSReadyCondition, condition.InitReason, condition.InitReason),
		condition.UnknownCondition(ConfigureOSReadyCondition, condition.InitReason, condition.InitReason),
		condition.UnknownCondition(RunOSReadyCondition, condition.InitReason, condition.InitReason),
		condition.UnknownCondition(ConfigureCephClientReadyCondition, condition.InitReason, condition.InitReason),
		condition.UnknownCondition(InstallOpenStackReadyCondition, condition.InitReason, condition.InitReason),
		condition.UnknownCondition(ConfigureOpenStackReadyCondition, condition.InitReason, condition.InitReason),
		condition.UnknownCondition(RunOpenStackReadyCondition, condition.InitReason, condition.InitReason),
	)

	instance.Status.Conditions.Init(&cl)
	instance.Status.Deployed = false
}

// Validate - validates the shared data between node and role
func (instance OpenStackDataPlaneNode) Validate(role OpenStackDataPlaneRole) error {
	suffix := fmt.Sprintf("node: %s and role: %s", instance.Name, role.Name)
	errorMsgs := AssertUniquenessBetween(instance.Spec, role.Spec, suffix)

	if len(errorMsgs) > 0 {
		return fmt.Errorf("validation error(s): %s", errorMsgs)
	}
	return nil
}

// GetAnsibleEESpec - get the fields that will be passed to AEE
func (instance OpenStackDataPlaneNode) GetAnsibleEESpec(role OpenStackDataPlaneRole) AnsibleEESpec {
	aee := AnsibleEESpec{}
	if len(instance.Spec.Node.ExtraMounts) > 0 {
		aee.ExtraMounts = instance.Spec.Node.ExtraMounts
	} else {
		aee.ExtraMounts = role.Spec.NodeTemplate.ExtraMounts
	}
	if len(instance.Spec.NetworkAttachments) > 0 {
		aee.NetworkAttachments = instance.Spec.NetworkAttachments
	} else {
		aee.NetworkAttachments = role.Spec.NetworkAttachments
	}
	if len(instance.Spec.DeployStrategy.AnsibleTags) > 0 {
		aee.AnsibleTags = instance.Spec.DeployStrategy.AnsibleTags
	} else {
		aee.AnsibleTags = role.Spec.DeployStrategy.AnsibleTags
	}
	if len(instance.Spec.DeployStrategy.AnsibleLimit) > 0 {
		aee.AnsibleLimit = instance.Spec.DeployStrategy.AnsibleLimit
	} else {
		aee.AnsibleLimit = role.Spec.DeployStrategy.AnsibleLimit
	}
	if len(instance.Spec.DeployStrategy.AnsibleSkipTags) > 0 {
		aee.AnsibleSkipTags = instance.Spec.DeployStrategy.AnsibleSkipTags
	} else {
		aee.AnsibleSkipTags = role.Spec.DeployStrategy.AnsibleSkipTags
	}
	if len(instance.Spec.Env) > 0 {
		aee.Env = instance.Spec.Env
	} else {
		aee.Env = role.Spec.Env
	}
	if len(instance.Spec.OpenStackAnsibleEERunnerImage) > 0 {
		aee.OpenStackAnsibleEERunnerImage = instance.Spec.OpenStackAnsibleEERunnerImage
	} else {
		aee.OpenStackAnsibleEERunnerImage = role.Spec.OpenStackAnsibleEERunnerImage
	}
	if len(instance.Spec.DeployStrategy.DeployIdentifier) > 0 {
		aee.DeployIdentifier = instance.Spec.DeployStrategy.DeployIdentifier
	} else {
		aee.DeployIdentifier = role.Spec.DeployStrategy.DeployIdentifier
	}
	return aee
}
