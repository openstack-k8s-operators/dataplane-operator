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
	"sort"
	"time"

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

	"github.com/go-logr/logr"
	dataplanev1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/dataplane-operator/pkg/deployment"
	infranetworkv1 "github.com/openstack-k8s-operators/infra-operator/apis/network/v1beta1"
	condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/secret"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	novav1beta1 "github.com/openstack-k8s-operators/nova-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/openstack-ansibleee-operator/api/v1alpha1"
	baremetalv1 "github.com/openstack-k8s-operators/openstack-baremetal-operator/api/v1beta1"
)

const (
	// AnsibleSSHPrivateKey ssh private key
	AnsibleSSHPrivateKey = "ssh-privatekey"
	// AnsibleSSHAuthorizedKeys authorized keys
	AnsibleSSHAuthorizedKeys = "authorized_keys"
)

// OpenStackDataPlaneRoleReconciler reconciles a OpenStackDataPlaneRole object
type OpenStackDataPlaneRoleReconciler struct {
	client.Client
	Kclient kubernetes.Interface
	Scheme  *runtime.Scheme
	Log     logr.Logger
}

//+kubebuilder:rbac:groups=dataplane.openstack.org,resources=openstackdataplaneroles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dataplane.openstack.org,resources=openstackdataplaneroles/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dataplane.openstack.org,resources=openstackdataplaneroles/finalizers,verbs=update
//+kubebuilder:rbac:groups=dataplane.openstack.org,resources=openstackdataplaneservices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dataplane.openstack.org,resources=openstackdataplaneservices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dataplane.openstack.org,resources=openstackdataplaneservices/finalizers,verbs=update
//+kubebuilder:rbac:groups=ansibleee.openstack.org,resources=openstackansibleees,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=baremetal.openstack.org,resources=openstackbaremetalsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nova.openstack.org,resources=novaexternalcomputes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete;
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete;
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete;
//+kubebuilder:rbac:groups=k8s.cni.cncf.io,resources=network-attachment-definitions,verbs=get;list;watch
//+kubebuilder:rbac:groups=network.openstack.org,resources=ipsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=network.openstack.org,resources=ipsets/status,verbs=get
//+kubebuilder:rbac:groups=network.openstack.org,resources=ipsets/finalizers,verbs=update
//+kubebuilder:rbac:groups=network.openstack.org,resources=netconfigs,verbs=get;list;watch
//+kubebuilder:rbac:groups=network.openstack.org,resources=dnsmasqs,verbs=get;list;watch
//+kubebuilder:rbac:groups=network.openstack.org,resources=dnsmasqs/status,verbs=get
//+kubebuilder:rbac:groups=network.openstack.org,resources=dnsdata,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=network.openstack.org,resources=dnsdata/status,verbs=get
//+kubebuilder:rbac:groups=network.openstack.org,resources=dnsdata/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the OpenStackDataPlaneRole object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *OpenStackDataPlaneRoleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, _err error) {

	r.Log = log.FromContext(ctx)
	r.Log.Info("Reconciling Role")

	// Fetch the OpenStackDataPlaneRole instance
	instance := &dataplanev1.OpenStackDataPlaneRole{}
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

	// Always patch the instance status when exiting this function so we can
	// persist any changes.
	defer func() {
		// update the overall status condition if service is ready
		if instance.IsReady() {
			instance.Status.Conditions.MarkTrue(condition.ReadyCondition, dataplanev1.DataPlaneRoleReadyMessage)
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

	// Initialize Status
	if instance.Status.Conditions == nil {
		instance.InitConditions()
		// Register overall status immediately to have an early feedback e.g.
		// in the cli
		return ctrl.Result{}, nil
	}

	if instance.Status.Conditions.IsUnknown(dataplanev1.SetupReadyCondition) {
		instance.Status.Conditions.MarkFalse(dataplanev1.SetupReadyCondition, condition.RequestedReason, condition.SeverityInfo, condition.ReadyInitMessage)
	}

	// Ensure Services
	err = deployment.EnsureServices(ctx, helper, instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Add/Remove openstackdataplane Label
	if len(instance.Spec.DataPlane) > 0 {
		if instance.ObjectMeta.Labels == nil {
			instance.ObjectMeta.Labels = make(map[string]string)
		}
		r.Log.Info(fmt.Sprintf("Adding label %s=%s", "openstackdataplane", instance.Spec.DataPlane))
		instance.ObjectMeta.Labels["openstackdataplane"] = instance.Spec.DataPlane
	} else if instance.ObjectMeta.Labels != nil {
		r.Log.Info(fmt.Sprintf("Removing label %s", "openstackdataplane"))
		delete(instance.ObjectMeta.Labels, "openstackdataplane")
	}

	// Get List of Nodes with matching Role Label
	nodes := &dataplanev1.OpenStackDataPlaneNodeList{}

	listOpts := []client.ListOption{
		client.InNamespace(instance.GetNamespace()),
	}
	fields := client.MatchingFields{"spec.role": instance.Name}
	listOpts = append(listOpts, fields)

	err = r.Client.List(ctx, nodes, listOpts...)
	if err != nil {
		return ctrl.Result{}, err
	}
	r.Log.Info("found nodes", "total", len(nodes.Items))

	// Order the nodes based on Name
	sort.SliceStable(nodes.Items, func(i, j int) bool {
		return nodes.Items[i].Name < nodes.Items[j].Name
	})

	// Validate NodeSpecs
	err = instance.Validate(nodes.Items)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Ensure IPSets Required for Nodes
	allIPSets, isReady, err := deployment.EnsureIPSets(ctx, helper, instance, nodes)
	if err != nil || !isReady {
		return ctrl.Result{}, err
	}

	// Ensure DNSData Required for Nodes
	dnsAddresses, ctlplaneSearchDomain, isReady, err := deployment.EnsureDNSData(ctx, helper,
		instance, nodes, allIPSets)
	if err != nil || !isReady {
		return ctrl.Result{}, err
	}

	ansibleSSHPrivateKeySecret := instance.Spec.NodeTemplate.AnsibleSSHPrivateKeySecret

	if ansibleSSHPrivateKeySecret != "" {
		var secretKeys = []string{}
		secretKeys = append(secretKeys, AnsibleSSHPrivateKey)
		if len(instance.Spec.BaremetalSetTemplate.BaremetalHosts) > 0 {
			secretKeys = append(secretKeys, AnsibleSSHAuthorizedKeys)
		}
		_, result, err = secret.VerifySecret(
			ctx,
			types.NamespacedName{Namespace: instance.Namespace,
				Name: ansibleSSHPrivateKeySecret},
			secretKeys,
			helper.GetClient(),
			time.Duration(5)*time.Second,
		)

		if err != nil {
			if (result != ctrl.Result{}) {
				instance.Status.Conditions.MarkFalse(
					condition.InputReadyCondition,
					condition.RequestedReason,
					condition.SeverityInfo,
					fmt.Sprintf(dataplanev1.InputReadyWaitingMessage,
						"secret/"+ansibleSSHPrivateKeySecret))
			} else {
				instance.Status.Conditions.MarkFalse(
					condition.InputReadyCondition,
					condition.RequestedReason,
					condition.SeverityWarning,
					err.Error())
			}
			return result, err
		}
	}

	// all our input checks out so report InputReady
	instance.Status.Conditions.MarkTrue(condition.InputReadyCondition, condition.InputReadyMessage)

	// Reconcile BaremetalSet if required
	if len(instance.Spec.BaremetalSetTemplate.BaremetalHosts) > 0 {
		isReady, err := deployment.DeployBaremetalSet(ctx, helper, instance,
			nodes, allIPSets, dnsAddresses)
		if err != nil || !isReady {
			return ctrl.Result{}, err
		}
	}

	// Ensure all nodes are in SetupReady state
	setupReadyNodes := 0
	for _, node := range nodes.Items {
		if node.IsSetupReady() {
			setupReadyNodes++
		}
	}

	if setupReadyNodes < len(nodes.Items) {
		return ctrl.Result{}, err
	}

	// Generate Role Inventory
	roleConfigMap, err := deployment.GenerateRoleInventory(ctx, helper, instance,
		nodes.Items, allIPSets, dnsAddresses)
	if err != nil {
		util.LogErrorForObject(helper, err, fmt.Sprintf("Unable to generate inventory for %s", instance.Name), instance)
		return ctrl.Result{}, err
	}

	// all setup tasks complete, mark SetupReadyCondition True
	instance.Status.Conditions.MarkTrue(dataplanev1.SetupReadyCondition, condition.ReadyMessage)

	r.Log.Info("Role", "DeployStrategy", instance.Spec.DeployStrategy.Deploy,
		"Role.Namespace", instance.Namespace, "Role.Name", instance.Name)
	if instance.Spec.DeployStrategy.Deploy {
		r.Log.Info("Starting DataPlaneRole deploy")
		r.Log.Info("Set DeploymentReadyCondition false", "instance", instance)
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.DeploymentReadyCondition, condition.RequestedReason,
			condition.SeverityInfo, condition.DeploymentReadyRunningMessage))
		ansibleEESpec := instance.GetAnsibleEESpec()
		if dnsAddresses != nil && ctlplaneSearchDomain != "" {
			ansibleEESpec.DNSConfig = &corev1.PodDNSConfig{
				Nameservers: dnsAddresses,
				Searches:    []string{ctlplaneSearchDomain},
			}
		}
		deployResult, err := deployment.Deploy(
			ctx, helper, instance, nodes, ansibleSSHPrivateKeySecret,
			roleConfigMap, &instance.Status, ansibleEESpec,
			instance.Spec.Services, instance)
		if err != nil {
			util.LogErrorForObject(helper, err, fmt.Sprintf("Unable to deploy %s", instance.Name), instance)
			instance.Status.Conditions.Set(condition.FalseCondition(
				condition.ReadyCondition,
				condition.ErrorReason,
				condition.SeverityWarning,
				dataplanev1.DataPlaneRoleErrorMessage,
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
		// We don't want another deploy triggered by any reconcile request, it should
		// only be triggered when the user (or another controller) specifically
		// sets it to true.
		r.Log.Info("Set DeployStrategy.Deploy to false")
		instance.Spec.DeployStrategy.Deploy = false

	}

	// Set DeploymentReadyCondition to False if it was unknown.
	// Handles the case where the Role is created with
	// DeployStrategy.Deploy=false.
	if instance.Status.Conditions.IsUnknown(condition.DeploymentReadyCondition) {
		r.Log.Info("Set DeploymentReadyCondition false")
		instance.Status.Conditions.Set(condition.FalseCondition(condition.DeploymentReadyCondition, condition.NotRequestedReason, condition.SeverityInfo, condition.DeploymentReadyInitMessage))
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpenStackDataPlaneRoleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	reconcileFunction := handler.EnqueueRequestsFromMapFunc(func(o client.Object) []reconcile.Request {
		result := []reconcile.Request{}

		// For each DNSMasq change event get the list of all
		// OpenStackDataPlaneRole to trigger reconcile for the
		// ones in the same namespace.
		roles := &dataplanev1.OpenStackDataPlaneRoleList{}

		listOpts := []client.ListOption{
			client.InNamespace(o.GetNamespace()),
		}
		if err := r.Client.List(context.Background(), roles, listOpts...); err != nil {
			r.Log.Error(err, "Unable to retrieve OpenStackDataPlaneRoleList %w")
			return nil
		}

		// For each role instance create a reconcile request
		for _, i := range roles.Items {
			name := client.ObjectKey{
				Namespace: o.GetNamespace(),
				Name:      i.Name,
			}
			result = append(result, reconcile.Request{NamespacedName: name})
		}
		if len(result) > 0 {
			r.Log.Info(fmt.Sprintf("Reconcile request for: %+v", result))

			return result
		}
		return nil
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&dataplanev1.OpenStackDataPlaneRole{}).
		Owns(&v1alpha1.OpenStackAnsibleEE{}).
		Owns(&novav1beta1.NovaExternalCompute{}).
		Owns(&baremetalv1.OpenStackBaremetalSet{}).
		Owns(&infranetworkv1.IPSet{}).
		Owns(&infranetworkv1.DNSData{}).
		Owns(&corev1.ConfigMap{}).
		Watches(&source.Kind{Type: &infranetworkv1.DNSMasq{}},
			reconcileFunction).
		Complete(r)
}
