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

	yaml "gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
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
	dataplanev1beta1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/dataplane-operator/pkg/deployment"
	"github.com/openstack-k8s-operators/lib-common/modules/ansible"
	condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/secret"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	novav1beta1 "github.com/openstack-k8s-operators/nova-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/openstack-ansibleee-operator/api/v1alpha1"
	baremetalv1 "github.com/openstack-k8s-operators/openstack-baremetal-operator/api/v1beta1"
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
//+kubebuilder:rbac:groups=ansibleee.openstack.org,resources=openstackansibleees,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=baremetal.openstack.org,resources=openstackbaremetalsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nova.openstack.org,resources=novaexternalcomputes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete;
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete;
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete;
// +kubebuilder:rbac:groups=k8s.cni.cncf.io,resources=network-attachment-definitions,verbs=get;list;watch

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
	instance := &dataplanev1beta1.OpenStackDataPlaneRole{}
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
			instance.Status.Conditions.MarkTrue(condition.ReadyCondition, dataplanev1beta1.DataPlaneRoleReadyMessage)
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

	if instance.Status.Conditions.IsUnknown(dataplanev1beta1.SetupReadyCondition) {
		instance.Status.Conditions.MarkFalse(dataplanev1beta1.SetupReadyCondition, condition.RequestedReason, condition.SeverityInfo, condition.ReadyInitMessage)
	}

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

	// Reconcile BaremetalSet if required
	if len(instance.Spec.BaremetalSetTemplate.BaremetalHosts) > 0 {
		ctrlResult, err := r.ReconcileBaremetalSet(ctx, instance, helper)
		if err != nil || ctrlResult != nil {
			return *ctrlResult, err
		}
	}

	nodes := &dataplanev1beta1.OpenStackDataPlaneNodeList{}

	listOpts := []client.ListOption{
		client.InNamespace(instance.GetNamespace()),
	}
	labelSelector := map[string]string{
		"openstackdataplanerole": instance.Name,
	}
	if len(labelSelector) > 0 {
		labels := client.MatchingLabels(labelSelector)
		listOpts = append(listOpts, labels)
	}
	err = r.Client.List(ctx, nodes, listOpts...)
	if err != nil {
		return ctrl.Result{}, err
	}
	nodeNames := ""
	for _, node := range nodes.Items {
		if node.Spec.Role != instance.Name {
			err = fmt.Errorf("node %s: node.Role does not match with node.Label", node.Name)
			return ctrl.Result{}, err
		}
		nodeNames = nodeNames + node.Name + ","
	}
	r.Log.Info("Role", "Nodes", nodeNames, "Role.Namespace", instance.Namespace, "Role.Name", instance.Name)
	err = instance.Validate(nodes.Items)
	if err != nil {
		return ctrl.Result{}, err
	}

	roleConfigMap, err := r.GenerateInventory(ctx, instance, nodes.Items)
	if err != nil {
		util.LogErrorForObject(helper, err, fmt.Sprintf("Unable to generate inventory for %s", instance.Name), instance)
		return ctrl.Result{}, err
	}

	ansibleSSHPrivateKeySecret := instance.Spec.NodeTemplate.AnsibleSSHPrivateKeySecret
	_, result, err = secret.VerifySecret(
		ctx,
		types.NamespacedName{Namespace: instance.Namespace, Name: ansibleSSHPrivateKeySecret},
		[]string{
			"ssh-privatekey",
		},
		r.Client,
		time.Duration(5)*time.Second,
	)
	if err != nil {
		return result, err
	}

	// all setup tasks complete, mark SetupReadyCondition True
	instance.Status.Conditions.MarkTrue(dataplanev1beta1.SetupReadyCondition, condition.ReadyMessage)

	r.Log.Info("Role", "DeployStrategy", instance.Spec.DeployStrategy.Deploy, "Role.Namespace", instance.Namespace, "Role.Name", instance.Name)
	if instance.Spec.DeployStrategy.Deploy {
		r.Log.Info("Starting DataPlaneRole deploy")
		r.Log.Info("Set DeploymentReadyCondition false", "instance", instance)
		instance.Status.Conditions.Set(condition.FalseCondition(condition.DeploymentReadyCondition, condition.RequestedReason, condition.SeverityInfo, condition.DeploymentReadyRunningMessage))

		deployResult, err := deployment.Deploy(ctx, helper, instance, nodes, ansibleSSHPrivateKeySecret, roleConfigMap, &instance.Status, instance.GetAnsibleEESpec(), *instance.Spec.NodeTemplate.Services, instance)
		if err != nil {
			util.LogErrorForObject(helper, err, fmt.Sprintf("Unable to deploy %s", instance.Name), instance)
			instance.Status.Conditions.Set(condition.FalseCondition(
				condition.ReadyCondition,
				condition.ErrorReason,
				condition.SeverityWarning,
				dataplanev1beta1.DataPlaneRoleErrorMessage,
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

// ReconcileBaremetalSet Reconcile OpenStackBaremetalSet
func (r *OpenStackDataPlaneRoleReconciler) ReconcileBaremetalSet(ctx context.Context, instance *dataplanev1beta1.OpenStackDataPlaneRole, helper *helper.Helper,
) (*ctrl.Result, error) {
	baremetalSet := &baremetalv1.OpenStackBaremetalSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
		},
	}

	helper.GetLogger().Info("Reconciling BaremetalSet")
	_, err := controllerutil.CreateOrPatch(ctx, helper.GetClient(), baremetalSet, func() error {
		instance.Spec.BaremetalSetTemplate.DeepCopyInto(&baremetalSet.Spec)
		err := controllerutil.SetControllerReference(helper.GetBeforeObject(), baremetalSet, helper.GetScheme())
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		instance.Status.Conditions.MarkFalse(
			dataplanev1beta1.RoleBareMetalProvisionReadyCondition,
			condition.ErrorReason, condition.SeverityError,
			dataplanev1beta1.RoleBaremetalProvisionErrorMessage)
		return &ctrl.Result{}, err
	}

	// Check if baremetalSet is ready
	if !baremetalSet.IsReady() {
		util.LogForObject(helper, "BaremetalSet not ready, Requeueing", instance)
		return &ctrl.Result{}, nil
	}
	instance.Status.Conditions.MarkTrue(
		dataplanev1beta1.RoleBareMetalProvisionReadyCondition,
		dataplanev1beta1.RoleBaremetalProvisionReadyMessage)
	return nil, nil
}

// GenerateInventory yields a parsed Inventory
func (r *OpenStackDataPlaneRoleReconciler) GenerateInventory(ctx context.Context, instance *dataplanev1beta1.OpenStackDataPlaneRole, nodes []dataplanev1beta1.OpenStackDataPlaneNode) (string, error) {
	var (
		err      error
		hostName string
	)

	inventory := ansible.MakeInventory()
	roleNameGroup := inventory.AddGroup(instance.Name)
	err = resolveAnsibleVars(&instance.Spec.NodeTemplate, &ansible.Host{}, &roleNameGroup)
	if err != nil {
		return "", err
	}

	for _, node := range nodes {
		host := roleNameGroup.AddHost(node.Name)
		if node.Spec.AnsibleHost == "" {
			hostName = node.Spec.HostName
		} else {
			hostName = node.Spec.AnsibleHost
		}
		host.Vars["ansible_host"] = hostName
		err = resolveAnsibleVars(&node.Spec.Node, &host, &ansible.Group{})
		if err != nil {
			return "", err
		}
	}

	configMapName := fmt.Sprintf("dataplanerole-%s", instance.Name)
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: instance.Namespace,
			Labels:    instance.ObjectMeta.Labels,
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
		cm.TypeMeta = metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		}
		cm.ObjectMeta = metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: instance.Namespace,
			Labels:    instance.ObjectMeta.Labels,
		}
		invData, err := inventory.MarshalYAML()
		if err != nil {
			return err
		}
		cm.Data = map[string]string{
			"inventory": string(invData),
			"network":   string(instance.Spec.NodeTemplate.NetworkConfig.Template),
		}
		return nil
	})
	if err != nil {
		return configMapName, err
	}

	return configMapName, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpenStackDataPlaneRoleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dataplanev1beta1.OpenStackDataPlaneRole{}).
		Owns(&v1alpha1.OpenStackAnsibleEE{}).
		Owns(&novav1beta1.NovaExternalCompute{}).
		Owns(&baremetalv1.OpenStackBaremetalSet{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}

func resolveAnsibleVars(node *dataplanev1beta1.NodeSection, host *ansible.Host, group *ansible.Group) error {
	ansibleVarsData := make(map[string]interface{})

	if node.AnsibleUser != "" {
		ansibleVarsData["ansible_user"] = node.AnsibleUser
	}
	if node.AnsiblePort > 0 {
		ansibleVarsData["ansible_port"] = node.AnsiblePort
	}
	if node.ManagementNetwork != "" {
		ansibleVarsData["management_network"] = node.ManagementNetwork
	}
	if node.NetworkConfig.Template != "" {
		ansibleVarsData["edpm_network_config_template"] = deployment.NicConfigTemplateFile
	}
	if len(node.Networks) > 0 {
		ansibleVarsData["networks"] = node.Networks
	}

	err := yaml.Unmarshal([]byte(node.AnsibleVars), ansibleVarsData)
	if err != nil {
		return err
	}

	if host.Vars != nil {
		for key, value := range ansibleVarsData {
			host.Vars[key] = value
		}
	}

	if group.Vars != nil {
		for key, value := range ansibleVarsData {
			group.Vars[key] = value
		}
	}

	return nil
}
