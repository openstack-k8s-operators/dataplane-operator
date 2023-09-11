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

	condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	baremetalv1 "github.com/openstack-k8s-operators/openstack-baremetal-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OpenStackDataPlaneNodeSetSpec defines the desired state of OpenStackDataPlaneNodeSet
type OpenStackDataPlaneNodeSetSpec struct {
	// +kubebuilder:validation:Optional
	// BaremetalSetTemplate Template for BaremetalSet for the NodeSet
	BaremetalSetTemplate baremetalv1.OpenStackBaremetalSetSpec `json:"baremetalSetTemplate,omitempty"`

	// +kubebuilder:validation:Required
	// NodeTemplate - node attributes specific to nodes defined by this resource. These
	// attributes can be overriden at the individual node level, else take their defaults
	// from valus in this section.
	NodeTemplate NodeTemplate `json:"nodeTemplate"`

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
	// +kubebuilder:default={download-cache,configure-network,validate-network,install-os,configure-os,run-os,ovn,libvirt,nova,telemetry}
	// +kubebuilder:default={download-cache,configure-network,validate-network,install-os,configure-os,run-os,ovn,libvirt,nova}
	// Services list
	Services []string `json:"services"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+operator-sdk:csv:customresourcedefinitions:displayName="OpenStack Data Plane NodeSet"
//+kubebuilder:resource:shortName=osdpns;osdpnodeset;osdpnodesets
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[0].status",description="Status"
//+kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.conditions[0].message",description="Message"

// OpenStackDataPlaneNodeSet is the Schema for the openstackdataplanenodesets API
type OpenStackDataPlaneNodeSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenStackDataPlaneNodeSetSpec   `json:"spec,omitempty"`
	Status OpenStackDataPlaneNodeSetStatus `json:"status,omitempty"`
}

// OpenStackDataPlaneNodeSetStatus defines the observed state of OpenStackDataPlaneNodeSet
type OpenStackDataPlaneNodeSetStatus struct {
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors={"urn:alm:descriptor:io.kubernetes.conditions"}
	// Conditions
	Conditions condition.Conditions `json:"conditions,omitempty" optional:"true"`

	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	// Deployed
	Deployed bool `json:"deployed,omitempty" optional:"true"`
}

//+kubebuilder:object:root=true

// OpenStackDataPlaneNodeSetList contains a list of OpenStackDataPlaneNodeSets
type OpenStackDataPlaneNodeSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenStackDataPlaneNodeSet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OpenStackDataPlaneNodeSet{}, &OpenStackDataPlaneNodeSetList{})
}

// IsReady - returns true if the DataPlane is ready
func (instance OpenStackDataPlaneNodeSet) IsReady() bool {
	return instance.Status.Conditions.IsTrue(condition.ReadyCondition)
}

// InitConditions - Initializes Status Conditons
func (instance *OpenStackDataPlaneNodeSet) InitConditions() {
	instance.Status.Conditions = condition.Conditions{}

	cl := condition.CreateList(
		condition.UnknownCondition(condition.DeploymentReadyCondition, condition.InitReason, condition.InitReason),
		condition.UnknownCondition(condition.InputReadyCondition, condition.InitReason, condition.InitReason),
		condition.UnknownCondition(SetupReadyCondition, condition.InitReason, condition.InitReason),
		condition.UnknownCondition(NodeSetIPReservationReadyCondition, condition.InitReason, condition.InitReason),
		condition.UnknownCondition(NodeSetDNSDataReadyCondition, condition.InitReason, condition.InitReason),
	)

	// Only set Baremetal related conditions if we have baremetal hosts included in the
	// baremetalSetTemplate.
	if len(instance.Spec.BaremetalSetTemplate.BaremetalHosts) > 0 {
		cl = append(cl, *condition.UnknownCondition(NodeSetBareMetalProvisionReadyCondition, condition.InitReason, condition.InitReason))
	}

	if instance.Spec.Services != nil && instance.Spec.DeployStrategy.Deploy {
		for _, service := range instance.Spec.Services {
			readyCondition := condition.Type(fmt.Sprintf(ServiceReadyCondition, service))
			cl = append(cl, *condition.UnknownCondition(readyCondition, condition.InitReason, condition.InitReason))
		}
	}

	instance.Status.Conditions.Init(&cl)
	instance.Status.Deployed = false
}

// GetAnsibleEESpec - get the fields that will be passed to AEE
func (instance OpenStackDataPlaneNodeSet) GetAnsibleEESpec() AnsibleEESpec {
	return AnsibleEESpec{
		NetworkAttachments: instance.Spec.NetworkAttachments,
		AnsibleTags:        instance.Spec.DeployStrategy.AnsibleTags,
		AnsibleLimit:       instance.Spec.DeployStrategy.AnsibleLimit,
		AnsibleSkipTags:    instance.Spec.DeployStrategy.AnsibleSkipTags,
		ExtraMounts:        instance.Spec.NodeTemplate.ExtraMounts,
		Env:                instance.Spec.Env,
	}
}
