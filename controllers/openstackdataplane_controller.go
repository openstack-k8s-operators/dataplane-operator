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
	"sort"
	"time"

	dataplanev1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/dataplane-operator/pkg/deployment"
	condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	baremetalv1 "github.com/openstack-k8s-operators/openstack-baremetal-operator/api/v1beta1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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
//+kubebuilder:rbac:groups=dataplane.openstack.org,resources=openstackdataplaneservices,verbs=get;list;watch

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
	instance := &dataplanev1.OpenStackDataPlane{}
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
			instance.Status.Conditions.MarkTrue(condition.ReadyCondition, dataplanev1.DataPlaneReadyMessage)
		} else {
			// something is not ready so reset the Ready condition
			instance.Status.Conditions.MarkUnknown(
				condition.ReadyCondition, condition.InitReason, condition.ReadyInitMessage)
			// and recalculate it based on the state of the rest of the conditions
			instance.Status.Conditions.Set(instance.Status.Conditions.Mirror(condition.ReadyCondition))
		}

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

	if instance.Status.Conditions.IsUnknown(dataplanev1.SetupReadyCondition) {
		instance.Status.Conditions.MarkFalse(dataplanev1.SetupReadyCondition, condition.RequestedReason, condition.SeverityInfo, condition.ReadyInitMessage)
	}

	// all setup tasks complete, mark SetupReadyCondition True
	instance.Status.Conditions.MarkTrue(dataplanev1.SetupReadyCondition, condition.ReadyMessage)

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
		r.Log.Info("Set DeploymentReadyCondition false", "instance", instance)
		instance.Status.Conditions.Set(condition.FalseCondition(condition.DeploymentReadyCondition, condition.RequestedReason, condition.SeverityInfo, condition.DeploymentReadyRunningMessage))
		roles := &dataplanev1.OpenStackDataPlaneRoleList{}

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
			condition.DeploymentReadyCondition,
			condition.ErrorReason,
			condition.SeverityError,
			dataplanev1.DataPlaneErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	}
	if shouldRequeue {
		r.Log.Info("one or more roles aren't ready, requeueing")
		return ctrl.Result{RequeueAfter: time.Second * 5}, nil
	}
	if instance.Spec.DeployStrategy.Deploy && len(deployErrors) == 0 {
		instance.Status.Deployed = true
		r.Log.Info("Set DeploymentReadyCondition true", "instance", instance)
		instance.Status.Conditions.Set(condition.TrueCondition(condition.DeploymentReadyCondition, condition.DeploymentReadyMessage))
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

	// Set DeploymentReadyCondition to False if it was unknown.
	// Handles the case where the Node is created with
	// DeployStrategy.Deploy=false.
	if instance.Status.Conditions.IsUnknown(condition.DeploymentReadyCondition) {
		r.Log.Info("Set DeploymentReadyCondition false")
		instance.Status.Conditions.Set(condition.FalseCondition(condition.DeploymentReadyCondition, condition.NotRequestedReason, condition.SeverityInfo, condition.DeploymentReadyInitMessage))
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpenStackDataPlaneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dataplanev1.OpenStackDataPlane{}).
		Owns(&dataplanev1.OpenStackDataPlaneNode{}).
		Owns(&dataplanev1.OpenStackDataPlaneRole{}).
		Complete(r)
}

// createOrPatchDataPlaneResources -
func createOrPatchDataPlaneResources(ctx context.Context, instance *dataplanev1.OpenStackDataPlane, helper *helper.Helper) (ctrl.Result, error) {
	// create DataPlaneRoles
	roleManagedHostMap := make(map[string]map[string]baremetalv1.InstanceSpec)
	err := createOrPatchDataPlaneRoles(ctx, instance, helper, roleManagedHostMap)
	if err != nil {
		instance.Status.Conditions.MarkFalse(
			condition.ReadyCondition,
			condition.ErrorReason,
			condition.SeverityError,
			dataplanev1.DataPlaneErrorMessage,
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
			dataplanev1.DataPlaneNodeErrorMessage,
			err.Error())
		return ctrl.Result{}, err
	}

	// Get All Nodes
	nodes := &dataplanev1.OpenStackDataPlaneNodeList{}
	listOpts := []client.ListOption{
		client.InNamespace(instance.GetNamespace()),
	}

	err = helper.GetClient().List(ctx, nodes, listOpts...)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(nodes.Items) < len(instance.Spec.Nodes) {
		util.LogForObject(helper, "All nodes not yet created, requeueing", instance)
		return ctrl.Result{RequeueAfter: time.Second * 10}, nil
	}

	// Order the nodes based on Name
	sort.SliceStable(nodes.Items, func(i, j int) bool {
		return nodes.Items[i].Name < nodes.Items[j].Name
	})

	err = deployment.BuildBMHHostMap(ctx, helper, instance, nodes, roleManagedHostMap)
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
			dataplanev1.DataPlaneErrorMessage,
			err.Error())
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// createOrPatchDataPlaneNodes Create or Patch DataPlaneNodes
func createOrPatchDataPlaneNodes(ctx context.Context, instance *dataplanev1.OpenStackDataPlane, helper *helper.Helper) error {
	logger := helper.GetLogger()
	client := helper.GetClient()

	for nodeName, nodeSpec := range instance.Spec.Nodes {
		logger.Info("CreateDataPlaneNode", "nodeName", nodeName)
		node := &dataplanev1.OpenStackDataPlaneNode{
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
	instance *dataplanev1.OpenStackDataPlane, helper *helper.Helper,
	roleManagedHostMap map[string]map[string]baremetalv1.InstanceSpec) error {
	client := helper.GetClient()
	logger := helper.GetLogger()
	for roleName, roleSpec := range instance.Spec.Roles {
		role := &dataplanev1.OpenStackDataPlaneRole{
			ObjectMeta: metav1.ObjectMeta{
				Name:      roleName,
				Namespace: instance.Namespace,
			},
		}
		err := client.Get(
			ctx, types.NamespacedName{Name: roleName, Namespace: instance.Namespace}, role)

		if err != nil && !k8s_errors.IsNotFound(err) {
			return err
		}

		logger.Info("Create Or Patch DataPlaneRole", "roleName", roleName)
		_, err = controllerutil.CreateOrPatch(ctx, client, role, func() error {
			// role.Spec.DeployStrategy is explicitly omitted. Otherwise, it
			// could get reset to False, and if the DataPlane deploy sets it to
			// True, the DataPlane and DataPlaneRole controllers will be stuck
			// looping trying to reconcile.
			role.Spec.DataPlane = instance.Name
			role.Spec.NodeTemplate = roleSpec.NodeTemplate
			role.Spec.NetworkAttachments = roleSpec.NetworkAttachments
			role.Spec.Env = roleSpec.Env
			role.Spec.Services = roleSpec.Services
			hostMap, ok := roleManagedHostMap[roleName]
			if ok {
				bmsTemplate := roleSpec.BaremetalSetTemplate.DeepCopy()
				bmsTemplate.BaremetalHosts = hostMap
				role.Spec.BaremetalSetTemplate = *bmsTemplate
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
