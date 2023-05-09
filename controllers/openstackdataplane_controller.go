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

package controllers

import (
	"context"
	"fmt"
	"time"

	dataplanev1beta1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	baremetalv1 "github.com/openstack-k8s-operators/openstack-baremetal-operator/api/v1beta1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
)

// OpenStackDataPlaneReconciler reconciles a OpenStackDataPlane object
type OpenStackDataPlaneReconciler struct {
	client.Client
	Kclient kubernetes.Interface
	Scheme  *runtime.Scheme
	Log     logr.Logger
}

//+kubebuilder:rbac:groups=dataplane.openstack.org,resources=openstackdataplanes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dataplane.openstack.org,resources=openstackdataplanenodes;openstackdataplaneroles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dataplane.openstack.org,resources=openstackdataplanes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dataplane.openstack.org,resources=openstackdataplanes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the OpenStackDataPlane object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *OpenStackDataPlaneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, _err error) {
	logger := log.FromContext(ctx)

	// Fetch the OpenStackDataPlane instance
	instance := &dataplanev1beta1.OpenStackDataPlane{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if k8s_errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected.
			// For additional cleanup logic use finalizers. Return and don't requeue.
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	helper, err := helper.NewHelper(
		instance,
		r.Client,
		r.Kclient,
		r.Scheme,
		logger,
	)
	if err != nil {
		// helper might be nil, so can't use util.LogErrorForObject since it requires helper as first arg
		r.Log.Error(err, fmt.Sprintf("unable to acquire helper for OpenStackDataPlane %s", instance.Name))
		return ctrl.Result{}, err
	}

	// Always patch the instance status when exiting this function so we can persist any changes.
	defer func() {
		// update the overall status condition if service is ready
		if instance.IsReady() {
			instance.Status.Conditions.MarkTrue(condition.ReadyCondition, dataplanev1beta1.DataPlaneNodeReadyMessage)
		} else {
			// something is not ready so reset the Ready condition
			instance.Status.Conditions.MarkUnknown(
				condition.ReadyCondition, condition.InitReason, condition.ReadyInitMessage)
			// and recalculate it based on the state of the rest of the conditions
			instance.Status.Conditions.Set(instance.Status.Conditions.Mirror(condition.ReadyCondition))
		}

		// Ensure conditions are always sorted by type
		instance.Status.Conditions.Sort()

		err := helper.PatchInstance(ctx, instance)
		if err != nil {
			r.Log.Error(_err, "PatchInstance error")
			_err = err
			return
		}
	}()

	if instance.Status.Conditions == nil {
		instance.InitConditions()
		// Register overall status immediately to have an early feedback e.g. in the cli
		return ctrl.Result{}, nil
	}

	if instance.Status.Conditions.IsUnknown(dataplanev1beta1.SetupReadyCondition) {
		instance.Status.Conditions.MarkFalse(dataplanev1beta1.SetupReadyCondition, condition.RequestedReason, condition.SeverityInfo, condition.ReadyInitMessage)
	}

	ctrlResult, err := createOrPatchDataPlaneResources(ctx, instance, helper)
	if err != nil {
		return ctrl.Result{}, err
	} else if (ctrlResult != ctrl.Result{}) {
		return ctrlResult, nil
	}

	var deployErrors []string
	shouldRequeue := false
	if instance.Spec.DeployStrategy.Deploy {
		r.Log.Info("Starting DataPlane deploy")
		r.Log.Info("Set ReadyCondition false")
		roles := &dataplanev1beta1.OpenStackDataPlaneRoleList{}

		listOpts := []client.ListOption{
			client.InNamespace(instance.GetNamespace()),
		}
		labelSelector := map[string]string{
			"openstackdataplane": instance.Name,
		}
		if len(labelSelector) > 0 {
			labels := client.MatchingLabels(labelSelector)
			listOpts = append(listOpts, labels)
		}
		err = r.Client.List(ctx, roles, listOpts...)
		if err != nil {
			return ctrl.Result{}, err
		}

		instance.Status.Conditions.Set(condition.FalseCondition(condition.ReadyCondition, condition.InitReason, condition.SeverityInfo, dataplanev1beta1.DataPlaneReadyWaitingMessage))
		for _, role := range roles.Items {
			logger.Info("DataPlane deploy", "role.Name", role.Name)
			if role.Spec.DataPlane != instance.Name {
				err = fmt.Errorf("role %s: role.DataPlane does not match with role.Label", role.Name)
				deployErrors = append(deployErrors, "role.Name: "+role.Name+" error: "+err.Error())
			}
			r.Log.Info("Role", "DeployStrategy.Deploy", role.Spec.DeployStrategy.Deploy, "Role.Namespace", instance.Namespace, "Role.Name", role.Name)
			if !role.Spec.DeployStrategy.Deploy {
				_, err := controllerutil.CreateOrPatch(ctx, helper.GetClient(), &role, func() error {
					r.Log.Info("Reconciling Role", "Role.Namespace", instance.Namespace, "Role.Name", role.Name)
					helper.GetLogger().Info("CreateOrPatch Role.DeployStrategy.Deploy", "Role.Namespace", instance.Namespace, "Role.Name", role.Name)
					role.Spec.DeployStrategy.Deploy = instance.Spec.DeployStrategy.Deploy
					if err != nil {
						deployErrors = append(deployErrors, "role.Name: "+role.Name+" error: "+err.Error())
					}
					return nil
				})
				if err != nil {
					deployErrors = append(deployErrors, "role.Name: "+role.Name+" error: "+err.Error())
				}
			}
			if !role.IsReady() {
				r.Log.Info("Role", "IsReady", role.IsReady(), "Role.Namespace", instance.Namespace, "Role.Name", role.Name)
				shouldRequeue = true
				mirroredCondition := role.Status.Conditions.Mirror(condition.ReadyCondition)
				if mirroredCondition != nil {
					r.Log.Info("Role", "Status", mirroredCondition.Message, "Role.Namespace", instance.Namespace, "Role.Name", role.Name)
					instance.Status.Conditions.Set(mirroredCondition)
					if condition.IsError(mirroredCondition) {
						deployErrors = append(deployErrors, "role.Name: "+role.Name+" error: "+mirroredCondition.Message)
					}
				}

			}
		}
	}

	if len(deployErrors) > 0 {
		util.LogErrorForObject(helper, err, fmt.Sprintf("Unable to deploy %s", instance.Name), instance)
		err = fmt.Errorf(fmt.Sprintf("DeployDataplane error(s): %s", deployErrors))
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.ReadyCondition,
			condition.ErrorReason,
			condition.SeverityError,
			dataplanev1beta1.DataPlaneErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	}
	if shouldRequeue {
		r.Log.Info("one or more roles aren't ready, requeueing")
		return ctrl.Result{RequeueAfter: time.Second * 5}, nil
	}
	if instance.Spec.DeployStrategy.Deploy && len(deployErrors) == 0 {
		r.Log.Info("Set ReadyCondition true")
		instance.Status.Conditions.Set(condition.TrueCondition(condition.ReadyCondition, dataplanev1beta1.DataPlaneReadyMessage))
	}

	// Set DeploymentReadyCondition to False if it was unknown.
	// Handles the case where the DataPlane is created with
	// DeployStrategy.Deploy=false.
	if instance.Status.Conditions.IsUnknown(condition.DeploymentReadyCondition) {
		r.Log.Info("Set DeploymentReadyCondition false")
		instance.Status.Conditions.Set(condition.FalseCondition(condition.DeploymentReadyCondition, condition.NotRequestedReason, condition.SeverityInfo, condition.DeploymentReadyInitMessage))
	}

	// Explicitly set instance.Spec.Deploy = false
	// We don't want another deploy triggered by any reconcile request, it should
	// only be triggered when the user (or another controller) specifically
	// sets it to true.
	instance.Spec.DeployStrategy.Deploy = false

	instance.Status.Conditions.MarkTrue(dataplanev1beta1.SetupReadyCondition, condition.ReadyMessage)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpenStackDataPlaneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dataplanev1beta1.OpenStackDataPlane{}).
		Owns(&dataplanev1beta1.OpenStackDataPlaneNode{}).
		Owns(&dataplanev1beta1.OpenStackDataPlaneRole{}).
		Complete(r)
}

// createOrPatchDataPlaneResources -
func createOrPatchDataPlaneResources(ctx context.Context, instance *dataplanev1beta1.OpenStackDataPlane, helper *helper.Helper) (ctrl.Result, error) {
	// create DataPlaneRoles
	roleManagedHostMap := make(map[string]map[string]baremetalv1.InstanceSpec)
	err := createOrPatchDataPlaneRoles(ctx, instance, helper, roleManagedHostMap)
	if err != nil {
		instance.Status.Conditions.MarkFalse(
			condition.ReadyCondition,
			condition.ErrorReason,
			condition.SeverityError,
			dataplanev1beta1.DataPlaneErrorMessage,
			err.Error())
		return ctrl.Result{}, err
	}

	// Create DataPlaneNodes
	err = createOrPatchDataPlaneNodes(ctx, instance, helper)
	if err != nil {
		instance.Status.Conditions.MarkFalse(
			condition.ReadyCondition,
			condition.ErrorReason,
			condition.SeverityError,
			dataplanev1beta1.DataPlaneNodeErrorMessage,
			err.Error())
		return ctrl.Result{}, err
	}

	// Get All Nodes
	nodes := &dataplanev1beta1.OpenStackDataPlaneNodeList{}
	listOpts := []client.ListOption{
		client.InNamespace(instance.GetNamespace()),
	}

	err = helper.GetClient().List(ctx, nodes, listOpts...)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(nodes.Items) < len(instance.Spec.Nodes) {
		// All dataplane nodes are not created yet, requeue the request
		err = fmt.Errorf("All nodes not yet created, requeueing")
		return ctrl.Result{}, err
	}

	err = buildBMHHostMap(instance, nodes, roleManagedHostMap)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Patch the role again to provision the nodes
	err = createOrPatchDataPlaneRoles(ctx, instance, helper, roleManagedHostMap)
	if err != nil {
		instance.Status.Conditions.MarkFalse(
			condition.ReadyCondition,
			condition.ErrorReason,
			condition.SeverityError,
			dataplanev1beta1.DataPlaneErrorMessage,
			err.Error())
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// createOrPatchDataPlaneNodes Create or Patch DataPlaneNodes
func createOrPatchDataPlaneNodes(ctx context.Context, instance *dataplanev1beta1.OpenStackDataPlane, helper *helper.Helper) error {
	logger := helper.GetLogger()
	client := helper.GetClient()

	for nodeName, nodeSpec := range instance.Spec.Nodes {
		logger.Info("CreateDataPlaneNode", "nodeName", nodeName)
		node := &dataplanev1beta1.OpenStackDataPlaneNode{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nodeName,
				Namespace: instance.Namespace,
			},
		}
		_, err := controllerutil.CreateOrPatch(ctx, client, node, func() error {
			nodeSpec.DeepCopyInto(&node.Spec)
			err := controllerutil.SetControllerReference(instance, node, helper.GetScheme())
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// createOrPatchDataPlaneRoles Create or Patch DataPlaneRole
func createOrPatchDataPlaneRoles(ctx context.Context,
	instance *dataplanev1beta1.OpenStackDataPlane, helper *helper.Helper,
	roleManagedHostMap map[string]map[string]baremetalv1.InstanceSpec) error {
	client := helper.GetClient()
	logger := helper.GetLogger()
	for roleName, roleSpec := range instance.Spec.Roles {
		logger.Info("CreateDataPlaneRole", "roleName", roleName)
		role := &dataplanev1beta1.OpenStackDataPlaneRole{
			ObjectMeta: metav1.ObjectMeta{
				Name:      roleName,
				Namespace: instance.Namespace,
			},
		}
		_, err := controllerutil.CreateOrPatch(ctx, client, role, func() error {
			// role.Spec.DeployStrategy is explicitly omitted. Otherwise, it
			// could get reset to False, and if the DataPlane deploy sets it to
			// True, the DataPlane and DataPlaneRole controllers will be stuck
			// looping trying to reconcile.
			role.Spec.DataPlane = instance.Name
			role.Spec.NodeTemplate = roleSpec.NodeTemplate
			role.Spec.NetworkAttachments = roleSpec.NetworkAttachments
			role.Spec.OpenStackAnsibleEERunnerImage = roleSpec.OpenStackAnsibleEERunnerImage
			role.Spec.Env = roleSpec.Env
			role.Spec.Services = roleSpec.Services
			role.Spec.BaremetalSetTemplate = roleSpec.BaremetalSetTemplate
			hostMap, ok := roleManagedHostMap[roleName]
			if ok {
				role.Spec.BaremetalSetTemplate.BaremetalHosts = hostMap
			}
			err := controllerutil.SetControllerReference(instance, role, helper.GetScheme())
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// buildBMHHostMap  Build managed host map for all roles
func buildBMHHostMap(instance *dataplanev1beta1.OpenStackDataPlane,
	nodes *dataplanev1beta1.OpenStackDataPlaneNodeList,
	roleManagedHostMap map[string]map[string]baremetalv1.InstanceSpec) error {
	for _, node := range nodes.Items {
		labels := node.GetObjectMeta().GetLabels()
		roleName, ok := labels["openstackdataplanerole"]
		if !ok {
			// Node does not have a label
			continue
		}
		if roleManagedHostMap[roleName] == nil {
			roleManagedHostMap[roleName] = make(map[string]baremetalv1.InstanceSpec)
		}
		// Using AnsibleHost (assuming it to be the ctlplane ip atm)
		// Once IPAM has been implemented use that
		if !instance.Spec.Roles[roleName].PreProvisioned {
			instanceSpec := baremetalv1.InstanceSpec{}
			instanceSpec.CtlPlaneIP = node.Spec.AnsibleHost
			instanceSpec.UserData = node.Spec.Node.UserData
			instanceSpec.NetworkData = node.Spec.Node.NetworkData
			roleManagedHostMap[roleName][node.Spec.HostName] = instanceSpec

		}
	}
	return nil
}
