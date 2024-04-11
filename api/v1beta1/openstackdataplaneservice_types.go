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

	certmgrv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	infranetworkv1 "github.com/openstack-k8s-operators/infra-operator/apis/network/v1beta1"
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
)

// OpenstackDataPlaneServiceCert defines the property of a TLS cert issued for
// a dataplane service
type OpenstackDataPlaneServiceCert struct {
	// Contents of the certificate
	// This is a list of strings for properties that are needed in the cert
	// +kubebuilder:validation:Required
	Contents []string `json:"contents"`

	// Networks to include in SNI for the cert
	// +kubebuilder:validation:Optional
	Networks []infranetworkv1.NetNameStr `json:"networks,omitempty"`

	// Issuer is the label for the issuer to issue the cert
	// Only one issuer should have this label
	// +kubebuilder:validation:Optional
	Issuer string `json:"issuer,omitempty"`

	// KeyUsages to be added to the issued cert
	// +kubebuilder:validation:Optional
	KeyUsages []certmgrv1.KeyUsage `json:"keyUsages,omitempty" yaml:"keyUsages,omitempty"`
}

// OpenStackDataPlaneServiceSpec defines the desired state of OpenStackDataPlaneService
type OpenStackDataPlaneServiceSpec struct {
	// Play is an inline playbook contents that ansible will run on execution.
	Play string `json:"play,omitempty"`

	// Playbook is a path to the playbook that ansible will run on this execution
	Playbook string `json:"playbook,omitempty"`

	// ConfigMaps list of ConfigMap names to mount as ExtraMounts for the OpenStackAnsibleEE
	// +kubebuilder:validation:Optional
	ConfigMaps []string `json:"configMaps,omitempty" yaml:"configMaps,omitempty"`

	// Secrets list of Secret names to mount as ExtraMounts for the OpenStackAnsibleEE
	// +kubebuilder:validation:Optional
	Secrets []string `json:"secrets,omitempty"`

	// OpenStackAnsibleEERunnerImage image to use as the ansibleEE runner image
	// +kubebuilder:validation:Optional
	OpenStackAnsibleEERunnerImage string `json:"openStackAnsibleEERunnerImage,omitempty" yaml:"openStackAnsibleEERunnerImage,omitempty"`

	// TLSCert tls certs to be generated
	// +kubebuilder:validation:Optional
	TLSCert *OpenstackDataPlaneServiceCert `json:"tlsCert,omitempty" yaml:"tlsCert,omitempty"`

	// CACerts - Secret containing the CA certificate chain
	// +kubebuilder:validation:Optional
	CACerts string `json:"caCerts,omitempty" yaml:"caCerts,omitempty"`

	// AddCertMounts - Whether to add cert mounts
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	AddCertMounts bool `json:"addCertMounts" yaml:"addCertMounts"`

	// DeployOnAllNodeSets - should the service be deploy across all nodesets
	// This will override default target of a service play, setting it to 'all'.
	// +kubebuilder:validation:Optional
	DeployOnAllNodeSets bool `json:"deployOnAllNodeSets,omitempty" yaml:"deployOnAllNodeSets,omitempty"`

	//ServiceType - denotes what purpose does the service perform for the dataplane
	// +kubebuilder:validation:Optional
	// +kubekuilder:default=standard
	// +kubebuilder:validation:Enum=standard;nova
	ServiceType string `json:"serviceType,omitempty"`
}

// OpenStackDataPlaneServiceStatus defines the observed state of OpenStackDataPlaneService
type OpenStackDataPlaneServiceStatus struct {
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors={"urn:alm:descriptor:io.kubernetes.conditions"}
	// Conditions
	Conditions condition.Conditions `json:"conditions,omitempty" optional:"true"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:resource:shortName=osdps;osdpservice;osdpservices
//+operator-sdk:csv:customresourcedefinitions:displayName="OpenStack Data Plane Service"

// OpenStackDataPlaneService is the Schema for the openstackdataplaneservices API
// OpenStackDataPlaneService name must be a valid RFC1123 as it is used in labels
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
