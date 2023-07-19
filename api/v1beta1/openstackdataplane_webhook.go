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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

const errNoSSHKey = "SSH key not provided for Node %s"

// log is for logging in this package.
var openstackdataplanelog = logf.Log.WithName("openstackdataplane-resource")

// SetupWebhookWithManager sets up the webhook with the Manager
func (r *OpenStackDataPlane) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-dataplane-openstack-org-v1beta1-openstackdataplane,mutating=true,failurePolicy=fail,sideEffects=None,groups=dataplane.openstack.org,resources=openstackdataplanes,verbs=create;update,versions=v1beta1,name=mopenstackdataplane.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &OpenStackDataPlane{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *OpenStackDataPlane) Default() {
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-dataplane-openstack-org-v1beta1-openstackdataplane,mutating=false,failurePolicy=fail,sideEffects=None,groups=dataplane.openstack.org,resources=openstackdataplanes,verbs=create;update,versions=v1beta1,name=vopenstackdataplane.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &OpenStackDataPlane{}

// Validate implements common validations
func (spec *OpenStackDataPlaneSpec) Validate() field.ErrorList {
	return nil
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *OpenStackDataPlane) ValidateCreate() error {
	openstackdataplanelog.Info("validate create", "name", r.Name)
	allErrs := r.Spec.Validate()
	if len(allErrs) != 0 {
		return apierrors.NewInvalid(
			schema.GroupKind{Group: "dataplane.openstack.org", Kind: "OpenStackDataPlane"},
			r.Name, allErrs)
	}
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *OpenStackDataPlane) ValidateUpdate(old runtime.Object) error {
	openstackdataplanelog.Info("validate update", "name", r.Name)
	allErrs := r.Spec.Validate()
	if len(allErrs) != 0 {
		return apierrors.NewInvalid(
			schema.GroupKind{Group: "dataplane.openstack.org", Kind: "OpenStackDataPlane"},
			r.Name, allErrs)
	}
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *OpenStackDataPlane) ValidateDelete() error {
	openstackdataplanelog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
