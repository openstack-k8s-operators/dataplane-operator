/*
Copyright 2024.

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

	"golang.org/x/exp/slices"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var openstackdataplaneservicelog = logf.Log.WithName("openstackdataplaneservice-resource")

// SetupWebhookWithManager sets up the webhook with the Manager
func (r *OpenStackDataPlaneService) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(r).Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-dataplane-openstack-org-v1beta1-openstackdataplaneservice,mutating=true,failurePolicy=fail,sideEffects=None,groups=dataplane.openstack.org,resources=openstackdataplaneservices,verbs=create;update,versions=v1beta1,name=mopenstackdataplaneservice.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &OpenStackDataPlaneService{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *OpenStackDataPlaneService) Default() {

	openstackdataplaneservicelog.Info("default", "name", r.Name)
	r.Spec.Default(r.Name)
}

// Default - set defaults for this OpenStackDataPlaneService
func (spec *OpenStackDataPlaneServiceSpec) Default(name string) {
	if spec.EDPMServiceType == "" {
		spec.EDPMServiceType = name
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:path=/validate-dataplane-openstack-org-v1beta1-openstackdataplaneservice,mutating=false,failurePolicy=fail,sideEffects=None,groups=dataplane.openstack.org,resources=openstackdataplaneservices,verbs=create;update,versions=v1beta1,name=vopenstackdataplaneservice.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &OpenStackDataPlaneService{}

func (r *OpenStackDataPlaneService) ValidateCreate() (admission.Warnings, error) {

	openstackdataplaneservicelog.Info("validate create", "name", r.Name)

	errors := r.Spec.ValidateCreate()

	if len(errors) != 0 {
		openstackdataplaneservicelog.Info("validation failed", "name", r.Name)
		return nil, apierrors.NewInvalid(
			schema.GroupKind{Group: "dataplane.openstack.org", Kind: "OpenStackDataPlaneService"},
			r.Name,
			errors,
		)
	}

	return nil, nil
}

func (r *OpenStackDataPlaneServiceSpec) ValidateCreate() field.ErrorList {
	var errs field.ErrorList

	if r.TLSCerts != nil {
		for _, v := range r.TLSCerts {
			errs = append(errs, v.ValidateContents()...)
		}
	}

	return errs
}

func (r *OpenStackDataPlaneService) ValidateUpdate(original runtime.Object) (admission.Warnings, error) {
	openstackdataplaneservicelog.Info("validate update", "name", r.Name)
	errors := r.Spec.ValidateUpdate()

	if len(errors) != 0 {
		openstackdataplaneservicelog.Info("validation failed", "name", r.Name)
		return nil, apierrors.NewInvalid(
			schema.GroupKind{Group: "dataplane.openstack.org", Kind: "OpenStackDataPlaneService"},
			r.Name,
			errors,
		)
	}
	return nil, nil
}

func (r *OpenStackDataPlaneServiceSpec) ValidateUpdate() field.ErrorList {
	var errs field.ErrorList

	if r.TLSCerts != nil {
		for _, v := range r.TLSCerts {
			errs = append(errs, v.ValidateContents()...)
		}
	}

	return errs
}

func (r *OpenStackDataPlaneService) ValidateDelete() (admission.Warnings, error) {
	openstackdataplaneservicelog.Info("validate delete", "name", r.Name)

	errors := r.Spec.ValidateDelete()

	if len(errors) != 0 {
		openstackdataplaneservicelog.Info("validation failed", "name", r.Name)
		return nil, apierrors.NewInvalid(
			schema.GroupKind{Group: "dataplane.openstack.org", Kind: "OpenStackDataPlaneService"},
			r.Name,
			errors,
		)
	}
	return nil, nil
}

func (r *OpenStackDataPlaneServiceSpec) ValidateDelete() field.ErrorList {
	// TODO(user): fill in your validation logic upon object creation.

	return field.ErrorList{}
}

func (r *OpenstackDataPlaneServiceCert) ValidateContents() field.ErrorList {

	var errs field.ErrorList
	// "dnsnames" and "ips" are only allowed usages
	allowedContents := []string{
		"dnsnames",
		"ips",
	}
	for _, val := range r.Contents {

		if !slices.Contains(allowedContents, val) {
			errs = append(errs, field.Invalid(field.NewPath("spec.tlsCert.Contents"),
				r.KeyUsages,
				fmt.Sprintf("error validating contents of TLSCert, %s, only valid contents are %v ", val, allowedContents)))
		}
	}
	return errs
}
