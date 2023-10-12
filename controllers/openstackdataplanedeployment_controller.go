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

package controllers

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	dataplanev1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/dataplane-operator/pkg/deployment"
	condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	"github.com/openstack-k8s-operators/openstack-ansibleee-operator/api/v1alpha1"
)

// OpenStackDataPlaneDeploymentReconciler reconciles a OpenStackDataPlaneDeployment object
type OpenStackDataPlaneDeploymentReconciler struct {
	client.Client
	Kclient kubernetes.Interface
	Scheme  *runtime.Scheme
	Log     logr.Logger
}

//+kubebuilder:rbac:groups=dataplane.openstack.org,resources=openstackdataplanedeployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dataplane.openstack.org,resources=openstackdataplanedeployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dataplane.openstack.org,resources=openstackdataplanedeployments/finalizers,verbs=update
//+kubebuilder:rbac:groups=dataplane.openstack.org,resources=openstackdataplanenodesets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dataplane.openstack.org,resources=openstackdataplanenodesets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dataplane.openstack.org,resources=openstackdataplanenodesets/finalizers,verbs=update
//+kubebuilder:rbac:groups=ansibleee.openstack.org,resources=openstackansibleees,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete;
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete;
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete;
//+kubebuilder:rbac:groups=k8s.cni.cncf.io,resources=network-attachment-definitions,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *OpenStackDataPlaneDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, _err error) {

	logger := log.FromContext(ctx)
	logger.Info("Reconciling Deployment")

	// Fetch the OpenStackDataPlaneDeployment instance
	instance := &dataplanev1.OpenStackDataPlaneDeployment{}
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

	helper, _ := helper.NewHelper(
		instance,
		r.Client,
		r.Kclient,
		r.Scheme,
		logger,
	)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Always patch the instance status when exiting this function so we can persist any changes.
	defer func() { // update the Ready condition based on the sub conditions
		if instance.Status.Conditions.AllSubConditionIsTrue() {
			instance.Status.Conditions.MarkTrue(
				condition.ReadyCondition, condition.ReadyMessage)
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
			logger.Error(err, "Error updating instance status conditions")
			_err = err
			return
		}
	}()

	// If the deploy is already done, return immediately.
	if instance.Status.Deployed {
		logger.Info("Already deployed", "instance.Status.Deployed", instance.Status.Deployed)
		return ctrl.Result{}, nil
	}

	// Initialize Status
	if instance.Status.Conditions == nil {
		instance.InitConditions()
		// Register overall status immediately to have an early feedback e.g.
		// in the cli
		return ctrl.Result{}, nil
	}

	// All setup tasks done
	// Mark SetupReadyCondition=True
	instance.Status.Conditions.MarkTrue(dataplanev1.SetupReadyCondition, condition.ReadyMessage)

	// Ensure NodeSets
	nodeSets := dataplanev1.OpenStackDataPlaneNodeSetList{}
	for _, nodeSet := range instance.Spec.NodeSets {

		// Fetch the OpenStackDataPlaneNodeSet instance
		nodeSetInstance := &dataplanev1.OpenStackDataPlaneNodeSet{}
		err := r.Client.Get(
			ctx,
			types.NamespacedName{
				Namespace: instance.GetNamespace(),
				Name:      nodeSet},
			nodeSetInstance)
		if err != nil {
			// NodeSet not found, force a requeue
			if k8s_errors.IsNotFound(err) {
				logger.Info("NodeSet not found", "NodeSet", nodeSet)
				return ctrl.Result{RequeueAfter: time.Second * 15}, nil
			}
			// Error reading the object - requeue the request.
			return ctrl.Result{}, err
		}
		nodeSets.Items = append(nodeSets.Items, *nodeSetInstance)
	}

	// Check that all nodeSets are SetupReady
	for _, nodeSet := range nodeSets.Items {
		if !nodeSet.Status.Conditions.IsTrue(dataplanev1.SetupReadyCondition) {
			logger.Info("NodeSet SetupReadyCondition is not True", "NodeSet", nodeSet.Name)
			return ctrl.Result{RequeueAfter: time.Second * 15}, nil
		}
	}

	// All nodeSets successfully fetched.
	// Mark InputReadyCondition=True
	instance.Status.Conditions.MarkTrue(condition.InputReadyCondition, condition.ReadyMessage)

	// Deploy each nodeSet
	// The loop starts and checks NodeSet deployments sequentially. However, after they
	// are started, they are running in parallel, since the loop does not wait
	// for the first started NodeSet to finish before starting the next.
	shouldRequeue := false
	haveError := false
	ansibleEESpec := instance.GetAnsibleEESpec()

	logger.Info("Deploying NodeSets")
	logger.Info("Set Status.Deployed to false", "instance", instance)
	instance.Status.Deployed = false
	logger.Info("Set DeploymentReadyCondition false", "instance", instance)
	instance.Status.Conditions.MarkFalse(
		condition.DeploymentReadyCondition, condition.RequestedReason,
		condition.SeverityInfo, condition.DeploymentReadyRunningMessage)
	for _, nodeSet := range nodeSets.Items {
		nodeSetSecretInv := fmt.Sprintf("dataplanenodeset-%s", nodeSet.Name)

		// Move this later
		if nodeSet.Status.DNSClusterAddresses != nil && nodeSet.Status.CtlplaneSearchDomain != "" {
			ansibleEESpec.DNSConfig = &corev1.PodDNSConfig{
				Nameservers: nodeSet.Status.DNSClusterAddresses,
				Searches:    []string{nodeSet.Status.CtlplaneSearchDomain},
			}
		}
		var deployResult *ctrl.Result
		deployResult, err = deployment.Deploy(
			ctx, helper, &nodeSet, instance,
			nodeSetSecretInv, &instance.Status, ansibleEESpec,
			instance.Spec.Services)

		if err != nil {
			util.LogErrorForObject(helper, err, fmt.Sprintf(
				"OpenStackDeployment error for NodeSet %s", nodeSet.Name), instance)
			haveError = true
			instance.Status.Conditions.MarkFalse(
				condition.ReadyCondition,
				condition.ErrorReason,
				condition.SeverityWarning,
				dataplanev1.DataPlaneNodeSetErrorMessage,
				err.Error())
		}

		if deployResult != nil {
			shouldRequeue = true
		} else {
			logger.Info("OpenStackDeployment succeeded for NodeSet", "NodeSet", nodeSet.Name)
			logger.Info("Set NodeSetDeploymentReadyCondition true", "nodeSet", nodeSet.Name)
			instance.Status.Conditions.MarkTrue(
				condition.Type(fmt.Sprintf(dataplanev1.NodeSetDeploymentReadyCondition, nodeSet.Name)),
				condition.DeploymentReadyMessage)
		}
	}

	if haveError {
		return ctrl.Result{}, err
	}

	if shouldRequeue {
		logger.Info("Not all NodeSets done for OpenStackDeployment")
		return ctrl.Result{}, nil
	}

	logger.Info("Set DeploymentReadyCondition true", "instance", instance)
	instance.Status.Conditions.MarkTrue(
		condition.DeploymentReadyCondition, condition.DeploymentReadyMessage)
	instance.Status.Deployed = true
	logger.Info("Set status deploy true", "instance", instance)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpenStackDataPlaneDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dataplanev1.OpenStackDataPlaneDeployment{}).
		Owns(&v1alpha1.OpenStackAnsibleEE{}).
		Complete(r)
}
