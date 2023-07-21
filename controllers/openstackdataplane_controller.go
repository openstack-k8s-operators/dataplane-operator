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

	"github.com/go-logr/logr"
	dataplanev1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// OpenStackDataPlaneReconciler reconciles a OpenStackDataPlane object
type OpenStackDataPlaneReconciler struct {
	client.Client
	Kclient kubernetes.Interface
	Scheme  *runtime.Scheme
	Log     logr.Logger
}

//+kubebuilder:rbac:groups=dataplane.openstack.org,resources=openstackdataplanes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dataplane.openstack.org,resources=openstackdataplanenodesets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dataplane.openstack.org,resources=openstackdataplanes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dataplane.openstack.org,resources=openstackdataplanes/finalizers,verbs=update
//+kubebuilder:rbac:groups=dataplane.openstack.org,resources=openstackdataplaneservices,verbs=get;list;watch
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete;

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
		logger.Error(err, fmt.Sprintf("unable to acquire helper for OpenStackDataPlane %s", instance.Name))
		return ctrl.Result{}, err
	}

	// Always patch the instance status when exiting this function so we can persist any changes.
	defer func() {
		// update the Ready condition based on the sub conditions
		if instance.Status.Conditions.AllSubConditionIsTrue() {
			instance.Status.Conditions.MarkTrue(
				condition.ReadyCondition, dataplanev1.DataPlaneReadyMessage)
		} else {
			// something is not ready so reset the Ready condition
			instance.Status.Conditions.MarkUnknown(
				condition.ReadyCondition, condition.InitReason, condition.ReadyInitMessage)
			// and recalculate it based on the state of the rest of the conditions
			instance.Status.Conditions.Set(
				instance.Status.Conditions.Mirror(condition.ReadyCondition))
		}
		err := helper.PatchInstance(ctx, instance)
		if err != nil {
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

	var deployErrors []string
	shouldRequeue := false
	if instance.Spec.DeployStrategy.Deploy {
		logger.Info("Starting DataPlane deploy")
		logger.Info("Set DeploymentReadyCondition false", "instance", instance)
		instance.Status.Conditions.Set(condition.FalseCondition(condition.DeploymentReadyCondition, condition.RequestedReason, condition.SeverityInfo, condition.DeploymentReadyRunningMessage))
		nodeSets := &dataplanev1.OpenStackDataPlaneNodeSetList{}

		labelSelector := labels.NewSelector()
		labelReq, err := labels.NewRequirement("openstackdataplane", selection.In, []string{instance.Name})
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to format labelSelector: %w", err)
		}
		labelSelector.Add(*labelReq)

		listOpts := client.ListOptions{
			Namespace:     instance.GetNamespace(),
			LabelSelector: labelSelector,
		}

		err = r.Client.List(ctx, nodeSets, &listOpts)
		if err != nil {
			return ctrl.Result{}, err
		}

		// If we didn't find any nodeSets with the labelSelector, we wont be able to deploy anything. Let's Log
		// this and requeue.
		if len(nodeSets.Items) == 0 {
			logger.Info(fmt.Sprintf("No nodeSets were found with matching label: %v", labelSelector))
			return ctrl.Result{RequeueAfter: time.Second * 5}, nil
		}

		for _, nodeSet := range nodeSets.Items {
			logger.Info("DataPlane deploy", "nodeSet.Name", nodeSet.Name)
			// We don't expect that our nodeSet list will be overly large. As such, linear search is likely
			// to be computationally more efficient compared to importing and using slices.Contains() here.
			nodeSetFound := false
			for _, nodeSetName := range instance.Spec.NodeSets {
				if nodeSetName == nodeSet.Name {
					nodeSetFound = true
				}
			}

			if !nodeSetFound {
				err = fmt.Errorf("nodeSet %s: nodeSet.DataPlane does not match with nodeSet.Label", nodeSet.Name)
				deployErrors = append(deployErrors, "nodeSet.Name: "+nodeSet.Name+" error: "+err.Error())
			}

			if !nodeSet.IsReady() {
				logger.Info("NodeSet", "IsReady", nodeSet.IsReady(), "NodeSet.Namespace", instance.Namespace, "NodeSet.Name", nodeSet.Name)
				shouldRequeue = true
				mirroredCondition := nodeSet.Status.Conditions.Mirror(condition.ReadyCondition)
				if mirroredCondition != nil {
					logger.Info("NodeSet", "Status", mirroredCondition.Message, "NodeSet.Namespace", instance.Namespace, "NodeSet.Name", nodeSet.Name)
					if condition.IsError(mirroredCondition) {
						deployErrors = append(deployErrors, "nodeSet.Name: "+nodeSet.Name+" error "+mirroredCondition.Message)
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
		logger.Info("one or more nodeSets aren't ready, requeueing")
		return ctrl.Result{RequeueAfter: time.Second * 5}, nil
	}
	if instance.Spec.DeployStrategy.Deploy && len(deployErrors) == 0 {
		instance.Status.Deployed = true
		logger.Info("Set DeploymentReadyCondition true", "instance", instance)
		instance.Status.Conditions.Set(condition.TrueCondition(condition.DeploymentReadyCondition, condition.DeploymentReadyMessage))
	}

	// Set DeploymentReadyCondition to False if it was unknown.
	// Handles the case where the DataPlane is created with
	// DeployStrategy.Deploy=false.
	if instance.Status.Conditions.IsUnknown(condition.DeploymentReadyCondition) {
		logger.Info("Set DeploymentReadyCondition false")
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
		logger.Info("Set DeploymentReadyCondition false")
		instance.Status.Conditions.Set(condition.FalseCondition(condition.DeploymentReadyCondition, condition.NotRequestedReason, condition.SeverityInfo, condition.DeploymentReadyInitMessage))
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpenStackDataPlaneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dataplanev1.OpenStackDataPlane{}).
		Complete(r)
}
