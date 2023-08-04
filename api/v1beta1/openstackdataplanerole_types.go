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
	"reflect"

	condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	baremetalv1 "github.com/openstack-k8s-operators/openstack-baremetal-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OpenStackDataPlaneRoleSpec defines the desired state of OpenStackDataPlaneRole
type OpenStackDataPlaneRoleSpec struct {
	// +kubebuilder:validation:Optional
	// DataPlane name of OpenStackDataPlane for this role
	DataPlane string `json:"dataPlane,omitempty"`

	// +kubebuilder:validation:Optional
	// BaremetalSetTemplate Template for BaremetalSet for the Role
	BaremetalSetTemplate baremetalv1.OpenStackBaremetalSetSpec `json:"baremetalSetTemplate,omitempty"`

	// +kubebuilder:validation:Optional
	// NodeTemplate - node attributes specific to this roles
	NodeTemplate NodeSection `json:"nodeTemplate,omitempty"`

	// +kubebuilder:validation:Optional
	//
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	// PreProvisioned - Whether the nodes are actually pre-provisioned (True) or should be
	// preprovisioned (False)
	PreProvisioned bool `json:"preProvisioned,omitempty"`

	// Env is a list containing the environment variables to pass to the pod
	Env []corev1.EnvVar `json:"env,omitempty"`

	// +kubebuilder:validation:Optional
	// DeployStrategy section to control how the node is deployed
	DeployStrategy DeployStrategySection `json:"deployStrategy,omitempty"`

	// +kubebuilder:validation:Optional
	// NetworkAttachments is a list of NetworkAttachment resource names to pass to the ansibleee resource
	// which allows to connect the ansibleee runner to the given network
	NetworkAttachments []string `json:"networkAttachments,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default={configure-network,validate-network,install-os,configure-os,run-os,ovn}
	// Services list
	Services []string `json:"services"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+operator-sdk:csv:customresourcedefinitions:displayName="OpenStack Data Plane Role"
// +kubebuilder:resource:shortName=osdprole;osdproles
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[0].status",description="Status"
//+kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.conditions[0].message",description="Message"

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

// IsReady - returns true if the DataPlane is ready
func (instance OpenStackDataPlaneRole) IsReady() bool {
	return instance.Status.Conditions.IsTrue(condition.ReadyCondition)
}

// InitConditions - Initializes Status Conditons
func (instance *OpenStackDataPlaneRole) InitConditions() {
	instance.Status.Conditions = condition.Conditions{}

	cl := condition.CreateList(
		condition.UnknownCondition(condition.DeploymentReadyCondition, condition.InitReason, condition.InitReason),
		condition.UnknownCondition(condition.InputReadyCondition, condition.InitReason, condition.InitReason),
		condition.UnknownCondition(SetupReadyCondition, condition.InitReason, condition.InitReason),
	)

	// Only set Baremetal condition for baremetal provisioning
	if len(instance.Spec.BaremetalSetTemplate.BaremetalHosts) > 0 {
		cl = append(cl, *condition.UnknownCondition(RoleBareMetalProvisionReadyCondition, condition.InitReason, condition.InitReason))
	}

	if instance.Spec.Services != nil && instance.Spec.DeployStrategy.Deploy {
		for _, service := range instance.Spec.Services {
			readyCondition := condition.Type(fmt.Sprintf(ServiceReadyCondition, service))
			cl = append(cl, *condition.UnknownCondition(readyCondition, condition.InitReason, condition.InitReason))
		}
	}

	haveCephSecret := false
	for _, extraMount := range instance.Spec.NodeTemplate.ExtraMounts {
		if extraMount.ExtraVolType == "Ceph" {
			haveCephSecret = true
			break
		}
	}

	if haveCephSecret {
		cl.Set(condition.UnknownCondition(ConfigureCephClientReadyCondition, condition.InitReason, condition.InitReason))

	}
	instance.Status.Conditions.Init(&cl)
	instance.Status.Deployed = false
}

// Validate - validates the shared data between role and nodes
func (instance OpenStackDataPlaneRole) Validate(nodes []OpenStackDataPlaneNode) error {
	var errorMsgs []string
	containsEmptyField := false
	for _, field := range UniqueSpecFields {
		if reflect.ValueOf(instance.Spec).FieldByName(field).IsZero() {
			containsEmptyField = true
			break
		}
	}

	if !containsEmptyField {
		for _, node := range nodes {
			suffix := fmt.Sprintf("node: %s and role: %s", node.Name, instance.Name)
			msgs := AssertUniquenessBetween(node.Spec, instance.Spec, suffix)
			errorMsgs = append(errorMsgs, msgs...)
		}
	}

	// Compare nodes when role fields are empty
	if containsEmptyField {
		nodeMap := make(map[string]OpenStackDataPlaneNode)

		for _, node := range nodes {
			for _, field := range UniqueSpecFields {
				if len(nodeMap[field].Name) > 0 {
					suffix := fmt.Sprintf("node: %s and node: %s", node.Name, nodeMap[field].Name)
					msgs := AssertUniquenessBetween(node.Spec, nodeMap[field].Spec, suffix)
					errorMsgs = append(errorMsgs, msgs...)
				}
				if len(nodeMap[field].Name) == 0 && !reflect.ValueOf(node.Spec).FieldByName(field).IsZero() {
					nodeMap[field] = node
				}
			}
		}
	}

	if len(errorMsgs) > 0 {
		return fmt.Errorf("validation error(s): %s", errorMsgs)
	}
	return nil
}

// GetAnsibleEESpec - get the fields that will be passed to AEE
func (instance OpenStackDataPlaneRole) GetAnsibleEESpec() AnsibleEESpec {
	return AnsibleEESpec{
		NetworkAttachments: instance.Spec.NetworkAttachments,
		AnsibleTags:        instance.Spec.DeployStrategy.AnsibleTags,
		AnsibleLimit:       instance.Spec.DeployStrategy.AnsibleLimit,
		AnsibleSkipTags:    instance.Spec.DeployStrategy.AnsibleSkipTags,
		ExtraMounts:        instance.Spec.NodeTemplate.ExtraMounts,
		Env:                instance.Spec.Env,
	}
}
