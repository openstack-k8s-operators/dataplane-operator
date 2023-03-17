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
	"strconv"
	"time"

	"github.com/go-logr/logr"
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

	dataplanev1beta1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/dataplane-operator/pkg/deployment"
	"github.com/openstack-k8s-operators/lib-common/modules/ansible"
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
	r.Log.Info("Reconciling")

	// Fetch the OpenStackDataPlaneNode instance
	instance := &dataplanev1beta1.OpenStackDataPlaneNode{}
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

	instanceRole, err := r.GetInstanceRole(ctx, instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	ansibleSSHPrivateKeySecret := r.GetAnsibleSSHPrivateKeySecret(instance, instanceRole)
	_, result, err = secret.VerifySecret(
		ctx,
		types.NamespacedName{Namespace: instance.Namespace, Name: ansibleSSHPrivateKeySecret},
		[]string{
			"ssh-privatekey",
		},
		helper.GetClient(),
		time.Duration(5)*time.Second,
	)
	if err != nil {
		return result, err
	}

	// Always patch the instance status when exiting this function so we can
	// persist any changes.
	defer func() {
		// update the overall status condition if service is ready
		if instance.IsReady() {
			instance.Status.Conditions.MarkTrue(condition.ReadyCondition, dataplanev1beta1.DataPlaneNodeReadyMessage)
		}
		c := instance.Status.Conditions.Mirror(condition.ReadyCondition)
		if c.Reason == condition.ErrorReason {
			instance.Status.Conditions.MarkFalse(
				condition.ReadyCondition,
				condition.ErrorReason,
				condition.SeverityError,
				c.Message)
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
		instance.Status.Conditions = condition.Conditions{}

		cl := condition.CreateList(
			condition.UnknownCondition(dataplanev1beta1.ConfigureNetworkReadyCondition, condition.InitReason, condition.InitReason),
			condition.UnknownCondition(dataplanev1beta1.ValidateNetworkReadyCondition, condition.InitReason, condition.InitReason),
			condition.UnknownCondition(dataplanev1beta1.InstallOSReadyCondition, condition.InitReason, condition.InitReason),
			condition.UnknownCondition(dataplanev1beta1.ConfigureOSReadyCondition, condition.InitReason, condition.InitReason),
			condition.UnknownCondition(dataplanev1beta1.RunOSReadyCondition, condition.InitReason, condition.InitReason),
			condition.UnknownCondition(dataplanev1beta1.ConfigureCephClientReadyCondition, condition.InitReason, condition.InitReason),
			condition.UnknownCondition(dataplanev1beta1.InstallOpenStackReadyCondition, condition.InitReason, condition.InitReason),
			condition.UnknownCondition(dataplanev1beta1.ConfigureOpenStackReadyCondition, condition.InitReason, condition.InitReason),
			condition.UnknownCondition(dataplanev1beta1.RunOpenStackReadyCondition, condition.InitReason, condition.InitReason),
		)

		instance.Status.Conditions.Init(&cl)

		instance.Status.Deployed = false

		// Register overall status immediately to have an early feedback e.g.
		// in the cli
		return ctrl.Result{}, nil
	}

	// check if provided network attachments exist
	for _, netAtt := range instance.Spec.NetworkAttachments {
		_, err := nad.GetNADWithName(ctx, helper, netAtt, instance.Namespace)
		if err != nil {
			if k8s_errors.IsNotFound(err) {
				instance.Status.Conditions.Set(condition.FalseCondition(
					condition.NetworkAttachmentsReadyCondition,
					condition.RequestedReason,
					condition.SeverityInfo,
					condition.NetworkAttachmentsReadyWaitingMessage,
					netAtt))
				return ctrl.Result{RequeueAfter: time.Second * 10}, fmt.Errorf("network-attachment-definition %s not found", netAtt)
			}
			instance.Status.Conditions.Set(condition.FalseCondition(
				condition.NetworkAttachmentsReadyCondition,
				condition.ErrorReason,
				condition.SeverityWarning,
				condition.NetworkAttachmentsReadyErrorMessage,
				err.Error()))
			return ctrl.Result{}, err
		}
	}

	if instance.Spec.Node.Managed {
		err = r.Provision(ctx, instance)
		if err != nil {
			util.LogErrorForObject(helper, err, fmt.Sprintf("Unable to OpenStackDataPlaneNode %s", instance.Name), instance)
			return ctrl.Result{}, err
		}
	}

	inventoryConfigMap, err := r.GenerateInventory(ctx, instance, instanceRole)
	if err != nil {
		util.LogErrorForObject(helper, err, fmt.Sprintf("Unable to generate inventory for %s", instance.Name), instance)
		return ctrl.Result{}, err
	}

	if instance.Spec.DeployStrategy.Deploy {
		result, err = deployment.Deploy(ctx, helper, instance, ansibleSSHPrivateKeySecret, inventoryConfigMap, &instance.Status, instance.Spec.NetworkAttachments, instance.Spec.OpenStackAnsibleEERunnerImage, instance.Spec.DeployStrategy.AnsibleTags, instance.Spec.Node.ExtraMounts)
		if err != nil {
			util.LogErrorForObject(helper, err, fmt.Sprintf("Unable to deploy %s", instance.Name), instance)
			instance.Status.Conditions.Set(condition.FalseCondition(
				condition.ReadyCondition,
				condition.ErrorReason,
				condition.SeverityError,
				dataplanev1beta1.DataPlaneNodeErrorMessage,
				err.Error()))
			return ctrl.Result{}, err
		}
		if result.RequeueAfter > 0 {
			return result, nil
		}

		var novaExternalCompute *novav1beta1.NovaExternalCompute
		result, novaExternalCompute, err = deployment.DeployNovaExternalCompute(
			ctx,
			helper,
			instance,
			ansibleSSHPrivateKeySecret,
			inventoryConfigMap,
			&instance.Status,
			instance.Spec.NetworkAttachments,
			instance.Spec.OpenStackAnsibleEERunnerImage)
		if err != nil {
			return result, err
		}

		novaReadyCondition := novaExternalCompute.Status.Conditions.Get(condition.ReadyCondition)
		if novaReadyCondition != nil {
			r.Log.Info(fmt.Sprintf("NovaExternalCompute ReadyCondition status: %s", novaReadyCondition.Status))
		} else {
			r.Log.Info("NovaExternalCompute ReadyCondition not yet set")
		}

		mirroredCondition := novaExternalCompute.Status.Conditions.Mirror(dataplanev1beta1.NovaComputeReadyCondition)
		if mirroredCondition != nil {
			instance.Status.Conditions.Set(mirroredCondition)
		}

		if condition.IsError(instance.Status.Conditions.Get(dataplanev1beta1.NovaComputeReadyCondition)) {
			r.Log.Info(fmt.Sprintf("%s error", dataplanev1beta1.NovaComputeReadyCondition))
			err = fmt.Errorf("failed: NovaExternalCompute name %s NovaExternalCompute namespace %s", novaExternalCompute.Name, novaExternalCompute.Namespace)
			return ctrl.Result{}, err
		}

		if instance.Status.Conditions.IsTrue(dataplanev1beta1.NovaComputeReadyCondition) {
			instance.Status.Deployed = true
			r.Log.Info("Set ReadyCondition true")
			instance.Status.Conditions.Set(condition.TrueCondition(condition.ReadyCondition, dataplanev1beta1.DataPlaneNodeReadyMessage))

			// Explicitly set instance.Spec.Deploy = false
			// We don't want another deploy triggered by any reconcile request, it
			// should only be triggered when the user (or another controller)
			// specifically sets it to true.
			instance.Spec.DeployStrategy.Deploy = false
		}

	}

	// Set ReadyCondition to False if it was unknown.
	// Handles the case where the Node is created with
	// DeployStrategy.Deploy=false.
	if instance.Status.Conditions.IsUnknown(condition.ReadyCondition) {
		r.Log.Info("Set ReadyCondition false")
		instance.Status.Conditions.Set(condition.FalseCondition(condition.ReadyCondition, condition.InitReason, condition.SeverityInfo, dataplanev1beta1.DataPlaneNodeReadyWaitingMessage))
	}
	return result, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpenStackDataPlaneNodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dataplanev1beta1.OpenStackDataPlaneNode{}).
		Owns(&v1alpha1.OpenStackAnsibleEE{}).
		Owns(&novav1beta1.NovaExternalCompute{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}

// Provision the data plane node
func (r *OpenStackDataPlaneNodeReconciler) Provision(ctx context.Context, instance *dataplanev1beta1.OpenStackDataPlaneNode) error {
	return nil
}

// GenerateInventory yields a parsed Inventory
func (r *OpenStackDataPlaneNodeReconciler) GenerateInventory(ctx context.Context, instance *dataplanev1beta1.OpenStackDataPlaneNode, instanceRole *dataplanev1beta1.OpenStackDataPlaneRole) (string, error) {
	var (
		err      error
		hostName string
	)

	inventory := ansible.MakeInventory()
	all := inventory.AddGroup("all")
	host := all.AddHost(instance.Name)

	host.Vars["ansible_user"] = r.GetAnsibleUser(instance, instanceRole)
	host.Vars["ansible_port"] = r.GetAnsiblePort(instance, instanceRole)
	host.Vars["managed"] = r.GetAnsibleManaged(instance, instanceRole)
	host.Vars["management_network"] = r.GetAnsibleManagementNetwork(instance, instanceRole)
	host.Vars["network_config"] = r.GetAnsibleNetworkConfig(instance, instanceRole)
	host.Vars["networks"] = r.GetAnsibleNetworks(instance, instanceRole)

	if instance.Spec.AnsibleHost == "" {
		hostName = instance.Spec.HostName
	} else {
		hostName = instance.Spec.AnsibleHost
	}
	host.Vars["ansible_host"] = hostName

	ansibleVarsData, err := r.GetAnsibleVars(instance, instanceRole)
	if err != nil {
		return "", err
	}
	for key, value := range ansibleVarsData {
		host.Vars[key] = value
	}

	configMapName := fmt.Sprintf("dataplanenode-%s-inventory", instance.Name)
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
		}
		return nil
	})
	if err != nil {
		return configMapName, err
	}

	return configMapName, nil
}

// GetInstanceRole returns the role of a node based on the node's role name
func (r *OpenStackDataPlaneNodeReconciler) GetInstanceRole(ctx context.Context, instance *dataplanev1beta1.OpenStackDataPlaneNode) (*dataplanev1beta1.OpenStackDataPlaneRole, error) {
	// Use the instances's role name to get its role object
	var err error
	instanceRole := &dataplanev1beta1.OpenStackDataPlaneRole{}
	if len(instance.Spec.Role) > 0 {
		err = r.Client.Get(ctx, client.ObjectKey{
			Namespace: instance.Namespace,
			Name:      instance.Spec.Role,
		}, instanceRole)
	}
	return instanceRole, err
}

// GetAnsibleUser returns the string value from the template unless it is set in the node
func (r *OpenStackDataPlaneNodeReconciler) GetAnsibleUser(instance *dataplanev1beta1.OpenStackDataPlaneNode, instanceRole *dataplanev1beta1.OpenStackDataPlaneRole) string {
	if instance.Spec.Node.AnsibleUser != "" {
		return instance.Spec.Node.AnsibleUser
	}
	return instanceRole.Spec.NodeTemplate.AnsibleUser
}

// GetAnsiblePort returns the string value from the template unless it is set in the node
func (r *OpenStackDataPlaneNodeReconciler) GetAnsiblePort(instance *dataplanev1beta1.OpenStackDataPlaneNode, instanceRole *dataplanev1beta1.OpenStackDataPlaneRole) string {
	if instance.Spec.Node.AnsiblePort > 0 {
		return strconv.Itoa(instance.Spec.Node.AnsiblePort)
	}
	return strconv.Itoa(instanceRole.Spec.NodeTemplate.AnsiblePort)
}

// GetAnsibleManaged returns the string (from boolean) value from the template unless it is set in the node
func (r *OpenStackDataPlaneNodeReconciler) GetAnsibleManaged(instance *dataplanev1beta1.OpenStackDataPlaneNode, instanceRole *dataplanev1beta1.OpenStackDataPlaneRole) string {
	if instance.Spec.Node.Managed {
		return strconv.FormatBool(instance.Spec.Node.Managed)
	}
	return strconv.FormatBool(instanceRole.Spec.NodeTemplate.Managed)
}

// GetAnsibleManagementNetwork returns the string value from the template unless it is set in the node
func (r *OpenStackDataPlaneNodeReconciler) GetAnsibleManagementNetwork(instance *dataplanev1beta1.OpenStackDataPlaneNode, instanceRole *dataplanev1beta1.OpenStackDataPlaneRole) string {
	if instance.Spec.Node.ManagementNetwork != "" {
		return instance.Spec.Node.ManagementNetwork
	}
	return instanceRole.Spec.NodeTemplate.ManagementNetwork
}

// GetAnsibleNetworkConfig returns a JSON string value from the template unless it is set in the node
func (r *OpenStackDataPlaneNodeReconciler) GetAnsibleNetworkConfig(instance *dataplanev1beta1.OpenStackDataPlaneNode, instanceRole *dataplanev1beta1.OpenStackDataPlaneRole) dataplanev1beta1.NetworkConfigSection {
	if instance.Spec.Node.NetworkConfig.Template != "" {
		return instance.Spec.Node.NetworkConfig
	}
	return instanceRole.Spec.NodeTemplate.NetworkConfig
}

// GetAnsibleNetworks returns a JSON string mapping fixedIP and/or network name to their valules
func (r *OpenStackDataPlaneNodeReconciler) GetAnsibleNetworks(instance *dataplanev1beta1.OpenStackDataPlaneNode, instanceRole *dataplanev1beta1.OpenStackDataPlaneRole) []dataplanev1beta1.NetworksSection {
	if len(instance.Spec.Node.Networks) > 0 {
		return instance.Spec.Node.Networks
	}
	return instanceRole.Spec.NodeTemplate.Networks
}

// GetAnsibleVars returns a map of strings representing ansible vars which were merged from the role template vars and node vars
func (r *OpenStackDataPlaneNodeReconciler) GetAnsibleVars(instance *dataplanev1beta1.OpenStackDataPlaneNode, instanceRole *dataplanev1beta1.OpenStackDataPlaneRole) (map[string]interface{}, error) {
	// Merge the ansibleVars from the role into the value set on the node.
	// Top level keys set on the node ansibleVars should override top level keys from the role AnsibleVars.
	// However, there is no "deep" merge of values. Only top level keys are compared for merging

	// Unmarshal the YAML strings into two maps
	var role, node map[string]interface{}
	roleYamlError := yaml.Unmarshal([]byte(instanceRole.Spec.NodeTemplate.AnsibleVars), &role)
	if roleYamlError != nil {
		r.Log.Error(roleYamlError, fmt.Sprintf("Failed to unmarshal YAML data from role AnsibleVars '%s'", instanceRole.Spec.NodeTemplate.AnsibleVars))
		return nil, roleYamlError
	}
	nodeYamlError := yaml.Unmarshal([]byte(instance.Spec.Node.AnsibleVars), &node)
	if nodeYamlError != nil {
		r.Log.Error(nodeYamlError, fmt.Sprintf("Failed to unmarshal YAML data from node AnsibleVars '%s'", instance.Spec.Node.AnsibleVars))
		return nil, nodeYamlError
	}

	if role == nil && node != nil {
		return node, nil
	}
	if role != nil && node == nil {
		return role, nil
	}

	// Merge the two maps
	for k, v := range node {
		role[k] = v
	}
	return role, nil
}

// GetAnsibleSSHPrivateKeySecret returns the secret name holding the private SSH key
func (r *OpenStackDataPlaneNodeReconciler) GetAnsibleSSHPrivateKeySecret(instance *dataplanev1beta1.OpenStackDataPlaneNode, instanceRole *dataplanev1beta1.OpenStackDataPlaneRole) string {
	if instance.Spec.Node.AnsibleSSHPrivateKeySecret != "" {
		return instance.Spec.Node.AnsibleSSHPrivateKeySecret
	}
	return instanceRole.Spec.NodeTemplate.AnsibleSSHPrivateKeySecret
}
