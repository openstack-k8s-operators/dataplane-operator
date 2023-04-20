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
			instance.Status.Conditions.MarkTrue(condition.ReadyCondition, dataplanev1beta1.DataPlaneReadyMessage)
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

	// Reset all ReadyConditons to 'Unknown'
	instance.InitConditions()

	ctrlResult, err := r.CreateDataPlaneResources(ctx, instance, helper)
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

	// Set ReadyCondition to False if it was unknown
	if instance.Status.Conditions.IsUnknown(condition.ReadyCondition) {
		r.Log.Info("Set ReadyCondition false")
		instance.Status.Conditions.Set(condition.FalseCondition(condition.ReadyCondition, condition.RequestedReason, condition.SeverityInfo, dataplanev1beta1.DataPlaneReadyWaitingMessage))
	}

	// Explicitly set instance.Spec.Deploy = false
	// We don't want another deploy triggered by any reconcile request, it should
	// only be triggered when the user (or another controller) specifically
	// sets it to true.
	instance.Spec.DeployStrategy.Deploy = false

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

// CreateDataPlaneResources -
func (r *OpenStackDataPlaneReconciler) CreateDataPlaneResources(ctx context.Context, instance *dataplanev1beta1.OpenStackDataPlane, helper *helper.Helper) (ctrl.Result, error) {
	err := r.CreateDataPlaneRole(ctx, instance, helper)
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.ReadyCondition,
			condition.ErrorReason,
			condition.SeverityError,
			dataplanev1beta1.DataPlaneErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	}
	err = r.CreateDataPlaneNode(ctx, instance, helper)
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.ReadyCondition,
			condition.ErrorReason,
			condition.SeverityError,
			dataplanev1beta1.DataPlaneNodeErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil

}

// CreateDataPlaneNode -
func (r *OpenStackDataPlaneReconciler) CreateDataPlaneNode(ctx context.Context, instance *dataplanev1beta1.OpenStackDataPlane, helper *helper.Helper) error {

	for nodeName, nodeSpec := range instance.Spec.Nodes {
		r.Log.Info("CreateDataPlaneNode", "nodeName", nodeName)
		node := &dataplanev1beta1.OpenStackDataPlaneNode{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nodeName,
				Namespace: instance.Namespace,
			},
		}
		_, err := controllerutil.CreateOrPatch(ctx, r.Client, node, func() error {
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

// CreateDataPlaneRole -
func (r *OpenStackDataPlaneReconciler) CreateDataPlaneRole(ctx context.Context, instance *dataplanev1beta1.OpenStackDataPlane, helper *helper.Helper) error {

	for roleName, roleSpec := range instance.Spec.Roles {
		r.Log.Info("CreateDataPlaneRole", "roleName", roleName)
		role := &dataplanev1beta1.OpenStackDataPlaneRole{
			ObjectMeta: metav1.ObjectMeta{
				Name:      roleName,
				Namespace: instance.Namespace,
			},
		}
		_, err := controllerutil.CreateOrPatch(ctx, r.Client, role, func() error {
			// role.Spec.DeployStrategy is explicitly omitted. Otherwise, it
			// could get reset to False, and if the DataPlane deploy sets it to
			// True, the DataPlane and DataPlaneRole controllers will be stuck
			// looping trying to reconcile.
			role.Spec.DataPlane = instance.Name
			role.Spec.NodeTemplate = roleSpec.NodeTemplate
			role.Spec.NetworkAttachments = roleSpec.NetworkAttachments
			role.Spec.OpenStackAnsibleEERunnerImage = roleSpec.OpenStackAnsibleEERunnerImage
			role.Spec.Env = roleSpec.Env
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
