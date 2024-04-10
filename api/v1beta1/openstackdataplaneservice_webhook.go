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
	"context"
	"fmt"

	"github.com/openstack-k8s-operators/lib-common/modules/certmanager"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/secret"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var webhookHelper *helper.Helper
var openstackdataplaneservicelog = logf.Log.WithName("openstackdataplaneservice-resource")

// namespace of the service objects
var webhookNamespace string

// SetupWebhookWithManager sets up the webhook with the Manager
func (r *OpenStackDataPlaneService) SetupWebhookWithManager(mgr ctrl.Manager) error {
	if webhookClient == nil {
		webhookClient = mgr.GetClient()
	}

	if webhookHelper == nil {
		cfg, err := config.GetConfig()
		if err != nil {
			return err
		}

		kclient, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			return err
		}

		scheme := runtime.NewScheme()

		webhookHelper, _ = helper.NewHelper(
			r,
			webhookClient,
			kclient,
			scheme,
			openstackdataplaneservicelog,
		)
	}

	return ctrl.NewWebhookManagedBy(mgr).For(r).Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-dataplane-openstack-org-v1beta1-openstackdataplaneservice,mutating=true,failurePolicy=fail,sideEffects=None,groups=dataplane.openstack.org,resources=openstackdataplaneservices,verbs=create;update,versions=v1beta1,name=mopenstackdataplaneservice.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &OpenStackDataPlaneService{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *OpenStackDataPlaneService) Default() {

	openstackdataplaneservicelog.Info("default", "name", r.Name)
	r.Spec.Default()
}

// Default - set defaults for this OpenStackDataPlaneDeployment
func (spec *OpenStackDataPlaneServiceSpec) Default() {

}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:path=/validate-dataplane-openstack-org-v1beta1-openstackdataplaneservice,mutating=false,failurePolicy=fail,sideEffects=None,groups=dataplane.openstack.org,resources=openstackdataplaneservices,verbs=create;update,versions=v1beta1,name=vopenstackdataplaneservice.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &OpenStackDataPlaneService{}

func (r *OpenStackDataPlaneService) ValidateCreate() (admission.Warnings, error) {

	openstackdataplaneservicelog.Info("validate create", "name", r.Name)
	webhookNamespace = r.ObjectMeta.Namespace

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

func (r *OpenStackDataPlaneServiceSpec) ValidateCACerts() (string, error) {
	if len(r.CACerts) > 0 {
		_, _, err := secret.GetSecret(context.TODO(), webhookHelper, r.CACerts, webhookNamespace)
		return r.CACerts, err
	}
	return "", nil
}

func (r *OpenStackDataPlaneServiceSpec) ValidateIssuer() (string, error) {
	var label string
	if r.TLSCert != nil {
		if r.TLSCert.Issuer == "" {
			// by default, use the internal root Ca
			label = certmanager.RootCAIssuerInternalLabel
		} else {
			label = r.TLSCert.Issuer
		}

		issuerLabelSelector := map[string]string{label: ""}

		_, err := certmanager.GetIssuerByLabels(
			context.TODO(), webhookHelper, webhookNamespace, issuerLabelSelector)
		return label, err
	}
	return "", nil
}

func (r *OpenStackDataPlaneServiceSpec) ValidateCreate() field.ErrorList {
	var errors field.ErrorList

	// Validate issuers if present
	if label, err := r.ValidateIssuer(); err != nil {
		openstackdataplaneservicelog.Error(
			err, "Error validating OpenStackDataPlaneService issuer",
			"issuer", label)
		errors = append(errors, field.Invalid(
			field.NewPath("Spec").Child("TLSCert").Child("Issuer"),
			label,
			fmt.Sprintf("Error getting issuer with label %s: %s", label, err)))
	}

	// Validate cacerts if present
	if secretName, err := r.ValidateCACerts(); err != nil {
		openstackdataplaneservicelog.Error(
			err, "Error validating OpenStackDataPlaneService caCerts secret",
			"secretName", secretName)
		errors = append(errors, field.Invalid(
			field.NewPath("Spec").Child("CACerts"),
			secretName,
			fmt.Sprintf("Error getting cacerts secret %s: %s", secretName, err)))
	}

	return errors
}

func (r *OpenStackDataPlaneService) ValidateUpdate(original runtime.Object) (admission.Warnings, error) {
	openstackdataplaneservicelog.Info("validate update", "name", r.Name)
	webhookNamespace = r.ObjectMeta.Namespace

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
	var errors field.ErrorList

	// Validate issuers if present
	if label, err := r.ValidateIssuer(); err != nil {
		openstackdataplaneservicelog.Error(
			err, "Error validating OpenStackDataPlaneService issuer",
			"issuer", label)
		errors = append(errors, field.Invalid(
			field.NewPath("Spec").Child("TLSCert").Child("Issuer"),
			label,
			fmt.Sprintf("Error getting issuer with label %s: %s", label, err)))
	}

	// Validate cacerts if present
	if secretName, err := r.ValidateCACerts(); err != nil {
		openstackdataplaneservicelog.Error(
			err, "Error validating OpenStackDataPlaneService caCerts secret",
			"secretName", secretName)
		errors = append(errors, field.Invalid(
			field.NewPath("Spec").Child("CACerts"),
			secretName,
			fmt.Sprintf("Error getting cacerts secret %s: %s", secretName, err)))
	}

	return errors
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
