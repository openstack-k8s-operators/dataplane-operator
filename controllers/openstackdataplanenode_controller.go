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
	corev1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	dataplanev1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/dataplane-operator/pkg/deployment"
	condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	nad "github.com/openstack-k8s-operators/lib-common/modules/common/networkattachment"
	"github.com/openstack-k8s-operators/lib-common/modules/common/secret"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	novav1beta1 "github.com/openstack-k8s-operators/nova-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/openstack-ansibleee-operator/api/v1alpha1"
)

// OpenStackDataPlaneNodeReconciler reconciles a OpenStackDataPlaneNode object
type OpenStackDataPlaneNodeReconciler struct {
	client.Client
	Kclient kubernetes.Interface
	Scheme  *runtime.Scheme
	Log     logr.Logger
}

//+kubebuilder:rbac:groups=dataplane.openstack.org,resources=openstackdataplanenodes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dataplane.openstack.org,resources=openstackdataplanenodes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dataplane.openstack.org,resources=openstackdataplanenodes/finalizers,verbs=update
//+kubebuilder:rbac:groups=ansibleee.openstack.org,resources=openstackansibleees,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nova.openstack.org,resources=novaexternalcomputes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete;
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete;
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete;
// +kubebuilder:rbac:groups=k8s.cni.cncf.io,resources=network-attachment-definitions,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the OpenStackDataPlaneNode object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *OpenStackDataPlaneNodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, _err error) {

	r.Log = log.FromContext(ctx)
	r.Log.Info("Reconciling Node")

	// Fetch the OpenStackDataPlaneNode instance
	instance := &dataplanev1.OpenStackDataPlaneNode{}
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
		r.Log,
	)
	if err != nil {
		return ctrl.Result{}, err
	}

	instanceRole, err := r.GetInstanceRole(ctx, instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(instance.Spec.Role) > 0 {
		if instance.ObjectMeta.Labels == nil {
			instance.ObjectMeta.Labels = make(map[string]string)
		}
		r.Log.Info(fmt.Sprintf("Adding label %s=%s", "openstackdataplanerole", instance.Spec.Role))
		instance.ObjectMeta.Labels["openstackdataplanerole"] = instance.Spec.Role
	} else if instance.ObjectMeta.Labels != nil {
		r.Log.Info(fmt.Sprintf("Removing label %s", "openstackdataplanerole"))
		delete(instance.ObjectMeta.Labels, "openstackdataplanerole")
	}

	// Always patch the instance status when exiting this function so we can persist any changes.
	defer func() {
		// update the Ready condition based on the sub conditions
		if instance.Status.Conditions.AllSubConditionIsTrue() {
			instance.Status.Deployed = true
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
			r.Log.Error(err, "Error updating instance status conditions")
			_err = err
			return
		}
	}()

	// Initialize Status
	if instance.Status.Conditions == nil {
		instance.InitConditions(instanceRole)
		// Register overall status immediately to have an early feedback e.g.
		// in the cli
		return ctrl.Result{}, nil
	}

	if instance.Status.Conditions.IsUnknown(dataplanev1.SetupReadyCondition) {
		instance.Status.Conditions.MarkFalse(dataplanev1.SetupReadyCondition, condition.RequestedReason, condition.SeverityInfo, condition.ReadyInitMessage)
	}
	ansibleSSHPrivateKeySecret := instance.Spec.Node.AnsibleSSHPrivateKeySecret

	_, result, err = secret.VerifySecret(
		ctx,
		types.NamespacedName{Namespace: instance.Namespace, Name: ansibleSSHPrivateKeySecret},
		[]string{
			AnsibleSSHPrivateKey,
		},
		helper.GetClient(),
		time.Duration(5)*time.Second,
	)

	if err != nil {
		if (result != ctrl.Result{}) {
			instance.Status.Conditions.MarkFalse(
				condition.InputReadyCondition,
				condition.RequestedReason,
				condition.SeverityInfo,
				dataplanev1.InputReadyWaitingMessage,
				"secret/"+ansibleSSHPrivateKeySecret)
		} else {
			instance.Status.Conditions.MarkFalse(
				condition.InputReadyCondition,
				condition.RequestedReason,
				condition.SeverityWarning,
				err.Error())
		}
		return result, err
	}

	// check if provided network attachments exist
	for _, netAtt := range instance.Spec.NetworkAttachments {
		_, err := nad.GetNADWithName(ctx, helper, netAtt, instance.Namespace)
		if err != nil {
			if k8s_errors.IsNotFound(err) {
				instance.Status.Conditions.MarkFalse(
					condition.InputReadyCondition,
					condition.RequestedReason,
					condition.SeverityInfo,
					dataplanev1.InputReadyWaitingMessage,
					"network-attachment-definition/"+netAtt)
				return ctrl.Result{RequeueAfter: time.Second * 10}, fmt.Errorf("network-attachment-definition %s not found", netAtt)
			}
			instance.Status.Conditions.MarkFalse(
				condition.InputReadyCondition,
				condition.ErrorReason,
				condition.SeverityWarning,
				condition.InputReadyErrorMessage,
				err.Error())
			return ctrl.Result{}, err
		}
	}

	// all our input checks out so report InputReady
	instance.Status.Conditions.MarkTrue(condition.InputReadyCondition, condition.InputReadyMessage)

	nodeConfigMap, err := deployment.GenerateNodeInventory(ctx, helper, instance, instanceRole)
	if err != nil {
		util.LogErrorForObject(helper, err, fmt.Sprintf("Unable to generate inventory for %s", instance.Name), instance)
		return ctrl.Result{}, err
	}

	// all setup tasks complete, mark SetupReadyCondition True
	instance.Status.Conditions.MarkTrue(dataplanev1.SetupReadyCondition, condition.ReadyMessage)

	r.Log.Info("Node", "DeployStrategy", instance.Spec.DeployStrategy.Deploy, "Node.Namespace", instance.Namespace, "Node.Name", instance.Name)
	if instance.Spec.DeployStrategy.Deploy {
		r.Log.Info("Starting DataPlaneNode deploy")
		r.Log.Info("Set DeploymentReadyCondition false", "instance", instance)
		instance.Status.Conditions.Set(condition.FalseCondition(condition.DeploymentReadyCondition, condition.RequestedReason, condition.SeverityInfo, condition.DeploymentReadyRunningMessage))
		nodes := &dataplanev1.OpenStackDataPlaneNodeList{
			Items: []dataplanev1.OpenStackDataPlaneNode{*instance},
		}
		deployResult, err := deployment.Deploy(
			ctx, helper, instance, nodes,
			ansibleSSHPrivateKeySecret, nodeConfigMap,
			&instance.Status, instance.GetAnsibleEESpec(*instanceRole),
			deployment.GetServices(instance, instanceRole), instanceRole)
		if err != nil {
			util.LogErrorForObject(helper, err, fmt.Sprintf("Unable to deploy %s", instance.Name), instance)
			instance.Status.Conditions.Set(condition.FalseCondition(
				condition.ReadyCondition,
				condition.ErrorReason,
				condition.SeverityError,
				dataplanev1.DataPlaneNodeErrorMessage,
				err.Error()))
			return ctrl.Result{}, err
		}
		if deployResult != nil {
			result = *deployResult
			return result, nil
		}

		instance.Status.Deployed = true
		r.Log.Info("Set DeploymentReadyCondition true", "instance", instance)
		instance.Status.Conditions.Set(condition.TrueCondition(condition.DeploymentReadyCondition, condition.DeploymentReadyMessage))

		// Explicitly set instance.Spec.Deploy = false
		// We don't want another deploy triggered by any reconcile request, it
		// should only be triggered when the user (or another controller)
		// specifically sets it to true.
		instance.Spec.DeployStrategy.Deploy = false

	}

	// Set DeploymentReadyCondition to False if it was unknown.
	// Handles the case where the Node is created with
	// DeployStrategy.Deploy=false.
	if instance.Status.Conditions.IsUnknown(condition.DeploymentReadyCondition) {
		r.Log.Info("Set DeploymentReadyCondition false")
		instance.Status.Conditions.Set(condition.FalseCondition(condition.DeploymentReadyCondition, condition.NotRequestedReason, condition.SeverityInfo, condition.DeploymentReadyInitMessage))
	}

	return result, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpenStackDataPlaneNodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &dataplanev1.OpenStackDataPlaneNode{}, "spec.role",
		func(rawObj client.Object) []string {
			node := rawObj.(*dataplanev1.OpenStackDataPlaneNode)
			return []string{node.Spec.Role}
		}); err != nil {
		return err
	}

	roleWatcher := handler.EnqueueRequestsFromMapFunc(func(obj client.Object) []reconcile.Request {
		var namespace string = obj.GetNamespace()
		var roleName string = obj.GetName()
		result := []reconcile.Request{}

		// Get all nodes for the role
		nodes := &dataplanev1.OpenStackDataPlaneNodeList{}

		listOpts := []client.ListOption{
			client.InNamespace(namespace),
		}
		fields := client.MatchingFields{"spec.role": roleName}
		listOpts = append(listOpts, fields)
		err := r.Client.List(context.Background(), nodes, listOpts...)
		if err != nil {
			r.Log.Error(err, "Unable to retrieve Node CRs %v")
			return nil
		}
		for _, node := range nodes.Items {
			name := client.ObjectKey{
				Namespace: namespace,
				Name:      node.Name,
			}
			result = append(result, reconcile.Request{NamespacedName: name})
		}
		return result
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&dataplanev1.OpenStackDataPlaneNode{}).
		Watches(&source.Kind{Type: &dataplanev1.OpenStackDataPlaneRole{}}, roleWatcher).
		Owns(&v1alpha1.OpenStackAnsibleEE{}).
		Owns(&novav1beta1.NovaExternalCompute{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}

// GetInstanceRole returns the role of a node based on the node's role name
func (r *OpenStackDataPlaneNodeReconciler) GetInstanceRole(ctx context.Context, instance *dataplanev1.OpenStackDataPlaneNode) (*dataplanev1.OpenStackDataPlaneRole, error) {
	// Use the instances's role name to get its role object
	var err error
	instanceRole := &dataplanev1.OpenStackDataPlaneRole{}
	if len(instance.Spec.Role) > 0 {
		err = r.Client.Get(ctx, client.ObjectKey{
			Namespace: instance.Namespace,
			Name:      instance.Spec.Role,
		}, instanceRole)
		if err == nil {
			err = instance.Validate(*instanceRole)
		}
	}
	return instanceRole, err
}
