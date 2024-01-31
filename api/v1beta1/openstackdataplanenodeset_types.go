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
	infranetworkv1 "github.com/openstack-k8s-operators/infra-operator/apis/network/v1beta1"
	condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
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

	// Nodes - Map of Node Names and node specific data. Values here override defaults in the
	// upper level section.
	// +kubebuilder:validation:Required
	Nodes map[string]NodeSection `json:"nodes"`

	// +kubebuilder:validation:Optional
	//
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	// PreProvisioned - Set to true if the nodes have been Pre Provisioned.
	PreProvisioned bool `json:"preProvisioned,omitempty"`

	// Env is a list containing the environment variables to pass to the pod
	// +kubebuilder:validation:Optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// +kubebuilder:validation:Optional
	// NetworkAttachments is a list of NetworkAttachment resource names to pass to the ansibleee resource
	// which allows to connect the ansibleee runner to the given network
	NetworkAttachments []string `json:"networkAttachments,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default={download-cache,bootstrap,configure-network,validate-network,install-os,configure-os,run-os,reboot-os,install-certs,ovn,neutron-metadata,libvirt,nova,telemetry}
	// Services list
	Services []string `json:"services"`

	// TLSEnabled - Whether the node set has TLS enabled.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	TLSEnabled bool `json:"tlsEnabled" yaml:"tlsEnabled"`

	// ContainerImages sets values for corresponding ansible variables
	// +kubebuilder:validation:Optional
	ContainerImages DataplaneAnsibleContainerImages `json:"containerImages,omitempty"`
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

	// DeploymentStatuses
	DeploymentStatuses map[string]condition.Conditions `json:"deploymentStatuses,omitempty" optional:"true"`

	// DNSClusterAddresses
	DNSClusterAddresses []string `json:"dnsClusterAddresses,omitempty" optional:"true"`

	// CtlplaneSearchDomain
	CtlplaneSearchDomain string `json:"ctlplaneSearchDomain,omitempty" optional:"true"`

	// AllHostnames
	AllHostnames map[string]map[infranetworkv1.NetNameStr]string `json:"allHostnames,omitempty" optional:"true"`

	// AllIPs
	AllIPs map[string]map[infranetworkv1.NetNameStr]string `json:"allIPs,omitempty" optional:"true"`

	// ConfigMapHashes
	ConfigMapHashes map[string]string `json:"configMapHashes,omitempty" optional:"true"`

	// SecretHashes
	SecretHashes map[string]string `json:"secretHashes,omitempty" optional:"true"`

	// ConfigHash - holds the curret hash of the NodeTemplate and Node sections of the struct.
	// This hash is used to determine when new Ansible executions are required to roll
	// out config changes.
	ConfigHash string `json:"configHash,omitempty"`

	// DeployedConfigHash - holds the hash of the NodeTemplate and Node sections of the struct
	// that was last deployed.
	// This hash is used to determine when new Ansible executions are required to roll
	// out config changes.
	DeployedConfigHash string `json:"deployedConfigHash,omitempty"`

	// ContainerImages
	ContainerImages DataplaneAnsibleContainerImages `json:"containerImages,omitempty" optional:"true"`
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
	instance.Status.DeploymentStatuses = make(map[string]condition.Conditions)

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

	instance.Status.Conditions.Init(&cl)
	instance.Status.Deployed = false
}

// GetAnsibleEESpec - get the fields that will be passed to AEE
func (instance OpenStackDataPlaneNodeSet) GetAnsibleEESpec() AnsibleEESpec {
	return AnsibleEESpec{
		NetworkAttachments: instance.Spec.NetworkAttachments,
		ExtraMounts:        instance.Spec.NodeTemplate.ExtraMounts,
		Env:                instance.Spec.Env,
	}
}

// DataplaneAnsibleContainerImages default images for dataplane services
type DataplaneAnsibleContainerImages struct {
	Frr                  string `json:"frr,omitempty"`
	IscsiD               string `json:"iscsiD,omitempty"`
	Logrotate            string `json:"logrotate,omitempty"`
	NeutronMetadataAgent string `json:"neutronMetadataAgent,omitempty"`
	NovaCompute          string `json:"novaCompute,omitempty"`
	NovaLibvirt          string `json:"novaLibvirt,omitempty"`
	OvnControllerAgent   string `json:"ovnControllerAgent,omitempty"`
	OvnBgpAgent          string `json:"ovnBgpAgent,omitempty"`
}

const (
	// FrrDefaultImage -
	FrrDefaultImage = "quay.io/podified-antelope-centos9/openstack-frr:current-podified"
	// IscsiDDefaultImage -
	IscsiDDefaultImage = "quay.io/podified-antelope-centos9/openstack-iscsid:current-podified"
	// LogrotateDefaultImage -
	LogrotateDefaultImage = "quay.io/podified-antelope-centos9/openstack-cron:current-podified"
	// NeutronMetadataAgentDefaultImage -
	NeutronMetadataAgentDefaultImage = "quay.io/podified-antelope-centos9/openstack-neutron-metadata-agent-ovn:current-podified"
	// NovaComputeDefaultImage -
	NovaComputeDefaultImage = "quay.io/podified-antelope-centos9/openstack-nova-compute:current-podified"
	// NovaLibvirtDefaultImage -
	NovaLibvirtDefaultImage = "quay.io/podified-antelope-centos9/openstack-nova-libvirt:current-podified"
	// OvnControllerAgentDefaultImage -
	OvnControllerAgentDefaultImage = "quay.io/podified-antelope-centos9/openstack-ovn-controller:current-podified"
	// OvnBgpAgentDefaultImage -
	OvnBgpAgentDefaultImage = "quay.io/podified-antelope-centos9/openstack-ovn-bgp-agent:current-podified"
)

// SetupDefaults - initializes any CRD field defaults based on environment variables (the defaulting mechanism itself is implemented via webhooks)
func SetupDefaults() {
	// Acquire environmental defaults and initialize dataplane defaults with them
	dataplaneAnsibleContainerImagesDefaults = DataplaneAnsibleContainerImages{
		Frr:                  util.GetEnvVar("RELATED_IMAGE_OPENSTACK_EDPM_FRR_DEFAULT_IMG", FrrDefaultImage),
		IscsiD:               util.GetEnvVar("RELATED_IMAGE_OPENSTACK_EDPM_ISCSID_DEFAULT_IMG", IscsiDDefaultImage),
		Logrotate:            util.GetEnvVar("RELATED_IMAGE_OPENSTACK_EDPM_LOGROTATE_CROND_DEFAULT_IMG", LogrotateDefaultImage),
		NeutronMetadataAgent: util.GetEnvVar("RELATED_IMAGE_OPENSTACK_EDPM_NEUTRON_METADATA_AGENT_DEFAULT_IMG", NeutronMetadataAgentDefaultImage),
		NovaCompute:          util.GetEnvVar("RELATED_IMAGE_OPENSTACK_EDPM_NOVA_COMPUTE_DEFAULT_IMG", NovaComputeDefaultImage),
		NovaLibvirt:          util.GetEnvVar("RELATED_IMAGE_OPENSTACK_EDPM_LIBVIRT_DEFAULT_IMG", NovaLibvirtDefaultImage),
		OvnControllerAgent:   util.GetEnvVar("RELATED_IMAGE_OPENSTACK_EDPM_OVN_CONTROLLER_AGENT_DEFAULT_IMG", OvnControllerAgentDefaultImage),
		OvnBgpAgent:          util.GetEnvVar("RELATED_IMAGE_OPENSTACK_EDPM_OVN_BGP_AGENT_IMAGE", OvnBgpAgentDefaultImage),
	}
	SetupDataplaneAnsibleContainerImagesDefaults(dataplaneAnsibleContainerImagesDefaults)
}
