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
	"context"
	"fmt"
	"reflect"

	"golang.org/x/exp/slices"

	baremetalv1 "github.com/openstack-k8s-operators/openstack-baremetal-operator/api/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// Client needed for API calls (manager's client, set by first SetupWebhookWithManager() call
// to any particular webhook)
var webhookClient client.Client

// log is for logging in this package.
var openstackdataplanenodesetlog = logf.Log.WithName("openstackdataplanenodeset-resource")

// SetupWebhookWithManager sets up the webhook with the Manager
func (r *OpenStackDataPlaneNodeSet) SetupWebhookWithManager(mgr ctrl.Manager) error {
	if webhookClient == nil {
		webhookClient = mgr.GetClient()
	}

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
	for nodeName, node := range spec.Nodes {
		if node.HostName == "" {
			node.HostName = nodeName
		}
		spec.Nodes[nodeName] = *node.DeepCopy()
	}

	if !spec.PreProvisioned {
		spec.NodeTemplate.Ansible.AnsibleUser = spec.BaremetalSetTemplate.CloudUserName
	} else if spec.NodeTemplate.Ansible.AnsibleUser == "" {
		spec.NodeTemplate.Ansible.AnsibleUser = "cloud-admin"
	}

	if spec.BaremetalSetTemplate.DeploymentSSHSecret == "" {
		spec.BaremetalSetTemplate.DeploymentSSHSecret = spec.NodeTemplate.AnsibleSSHPrivateKeySecret
	}

	if !spec.PreProvisioned {
		nodeSetHostMap := make(map[string]baremetalv1.InstanceSpec)
		for _, node := range spec.Nodes {
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

	// Currently, this check is only valid for PreProvisioned nodes. Since we can't possibly
	// have duplicates in Baremetal Deployments, we can exit early here for Baremetal NodeSets.
	if !r.Spec.PreProvisioned {
		return nil
	}

	var errors field.ErrorList

	nodeSetList := &OpenStackDataPlaneNodeSetList{}
	opts := &client.ListOptions{
		Namespace: r.ObjectMeta.Namespace,
	}

	err := webhookClient.List(context.TODO(), nodeSetList, opts)
	if err != nil {
		return err
	}

	// If this is the first NodeSet being created, then there can be no duplicates
	// we can exit early here.
	if len(nodeSetList.Items) == 0 {
		return nil
	}

	errors = r.duplicateNodeCheck(nodeSetList)

	if errors != nil {
		return apierrors.NewInvalid(
			schema.GroupKind{Group: "dataplane.openstack.org", Kind: "OpenStackDataPlaneNodeSet"},
			r.Name,
			errors)
	}

	return nil
}

// duplicateNodeCheck checks the NodeSetList for pre-existing nodes. If the user is trying to redefine an
// existing node, we will return an error and block resource creation.
func (r *OpenStackDataPlaneNodeSet) duplicateNodeCheck(nodeSetList *OpenStackDataPlaneNodeSetList) (errors field.ErrorList) {
	existingNodeNames := make([]string, 0)
	for _, existingNode := range nodeSetList.Items {
		for _, node := range existingNode.Spec.Nodes {
			existingNodeNames = append(existingNodeNames, node.HostName)
			if node.Ansible.AnsibleHost != "" {
				existingNodeNames = append(existingNodeNames, node.Ansible.AnsibleHost)
			}
		}
	}

	for _, newNodeName := range r.Spec.Nodes {
		if slices.Contains(existingNodeNames, newNodeName.HostName) || slices.Contains(existingNodeNames, newNodeName.Ansible.AnsibleHost) {
			errors = append(errors, field.Invalid(
				field.NewPath("Spec").Child("nodes"),
				r.Name,
				fmt.Sprintf("node already exists in the cluster: %s", newNodeName.HostName)))
		}
	}

	return
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *OpenStackDataPlaneNodeSet) ValidateUpdate(old runtime.Object) error {
	openstackdataplanenodesetlog.Info("validate update", "name", r.Name)
	oldNodeSet, ok := old.(*OpenStackDataPlaneNodeSet)
	if !ok {
		return apierrors.NewInternalError(
			fmt.Errorf("expected a OpenStackDataPlaneNodeSet object, but got %T", oldNodeSet))
	}

	var errors field.ErrorList
	// Some changes to the baremetalSetTemplate after the initial deployment would necessitate
	// a redeploy of the node. Thus we should block these changes and require the user to
	// delete and redeploy should they wish to make such changes after the initial deploy.
	// If the BaremetalSetTemplate is changed, we will offload the parsing of these details
	// to the openstack-baremetal-operator webhook to avoid duplicating logic.
	if !reflect.DeepEqual(r.Spec.BaremetalSetTemplate, oldNodeSet.Spec.BaremetalSetTemplate) {

		// Call openstack-baremetal-operator webhook Validate() to parse changes
		err := r.Spec.BaremetalSetTemplate.Validate(oldNodeSet.Spec.BaremetalSetTemplate)
		if err != nil {
			errors = append(errors, field.Forbidden(
				field.NewPath("spec.baremetalSetTemplate"),
				fmt.Sprintf("%s", err)))
		}
	}

	if errors != nil {
		return apierrors.NewInvalid(
			schema.GroupKind{Group: "dataplane.openstack.org", Kind: "OpenStackDataPlaneNodeSet"},
			r.Name,
			errors,
		)
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *OpenStackDataPlaneNodeSet) ValidateDelete() error {
	openstackdataplanenodesetlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
