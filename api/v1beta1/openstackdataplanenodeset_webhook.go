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
	baremetalv1 "github.com/openstack-k8s-operators/openstack-baremetal-operator/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var openstackdataplanenodesetlog = logf.Log.WithName("openstackdataplanenodeset-resource")

// SetupWebhookWithManager sets up the webhook with the Manager
func (r *OpenStackDataPlaneNodeSet) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-dataplane-openstack-org-v1beta1-openstackdataplanenodeset,mutating=true,failurePolicy=fail,sideEffects=None,groups=dataplane.openstack.org,resources=openstackdataplanenodesets,verbs=create;update,versions=v1beta1,name=mopenstackdataplanenodeset.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &OpenStackDataPlaneNodeSet{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *OpenStackDataPlaneNodeSet) Default() {
	openstackdataplanenodesetlog.Info("default", "name", r.Name)
	r.Spec.Default()
}

// Default - set defaults for this OpenStackDataPlaneNodeSet Spec
func (spec *OpenStackDataPlaneNodeSetSpec) Default() {
	for nodeName, node := range spec.NodeTemplate.Nodes {
		if node.HostName == "" {
			node.HostName = nodeName
		}
		spec.NodeTemplate.Nodes[nodeName] = *node.DeepCopy()
	}

	if spec.BaremetalSetTemplate.DeploymentSSHSecret == "" {
		spec.BaremetalSetTemplate.DeploymentSSHSecret = spec.NodeTemplate.AnsibleSSHPrivateKeySecret
	}

	if !spec.PreProvisioned && spec.BaremetalSetTemplate.BaremetalHosts == nil {
		nodeSetHostMap := make(map[string]baremetalv1.InstanceSpec)
		for _, node := range spec.NodeTemplate.Nodes {
			instanceSpec := baremetalv1.InstanceSpec{}
			instanceSpec.UserData = node.UserData
			instanceSpec.NetworkData = node.NetworkData
			nodeSetHostMap[node.HostName] = instanceSpec
		}
		spec.BaremetalSetTemplate.BaremetalHosts = nodeSetHostMap
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-dataplane-openstack-org-v1beta1-openstackdataplanenodeset,mutating=false,failurePolicy=fail,sideEffects=None,groups=dataplane.openstack.org,resources=openstackdataplanenodesets,verbs=create;update,versions=v1beta1,name=vopenstackdataplanenodeset.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &OpenStackDataPlaneNodeSet{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *OpenStackDataPlaneNodeSet) ValidateCreate() error {
	openstackdataplanenodesetlog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *OpenStackDataPlaneNodeSet) ValidateUpdate(old runtime.Object) error {
	openstackdataplanenodesetlog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *OpenStackDataPlaneNodeSet) ValidateDelete() error {
	openstackdataplanenodesetlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
