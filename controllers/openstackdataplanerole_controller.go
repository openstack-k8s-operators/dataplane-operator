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
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	dataplanev1beta1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/lib-common/modules/ansible"
	condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
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

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the OpenStackDataPlaneRole object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *OpenStackDataPlaneRoleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

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
	err = helper.GetClient().List(ctx, nodes, listOpts...)
	if err != nil {
		return ctrl.Result{}, err
	}

	for _, node := range nodes.Items {
		if node.Spec.Role != instance.Name {
			err = fmt.Errorf("node %s: node.Role does not match with node.Label", node.Name)
			return ctrl.Result{}, err
		}
	}

	_, err = r.GenerateInventory(ctx, instance, nodes.Items)
	if err != nil {
		util.LogErrorForObject(helper, err, fmt.Sprintf("Unable to generate inventory for %s", instance.Name), instance)
		return ctrl.Result{}, err
	}

	// Always patch the instance status when exiting this function so we can
	// persist any changes.
	defer func() {
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
			condition.UnknownCondition(dataplanev1beta1.DataPlaneRoleReadyCondition, condition.InitReason, condition.InitReason),
			condition.UnknownCondition(dataplanev1beta1.ConfigureNetworkReadyCondition, condition.InitReason, condition.InitReason),
			condition.UnknownCondition(dataplanev1beta1.ValidateNetworkReadyCondition, condition.InitReason, condition.InitReason),
			condition.UnknownCondition(dataplanev1beta1.InstallOSReadyCondition, condition.InitReason, condition.InitReason),
			condition.UnknownCondition(dataplanev1beta1.ConfigureOSReadyCondition, condition.InitReason, condition.InitReason),
			condition.UnknownCondition(dataplanev1beta1.RunOSReadyCondition, condition.InitReason, condition.InitReason),
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

	// Set DataPlaneRoleReadyCondition to False if it was unknown
	if instance.Status.Conditions.IsUnknown(dataplanev1beta1.DataPlaneRoleReadyCondition) {
		r.Log.Info("Set DataPlaneRoleReadyCondition false")
		instance.Status.Conditions.Set(condition.FalseCondition(dataplanev1beta1.DataPlaneRoleReadyCondition, condition.InitReason, condition.SeverityInfo, dataplanev1beta1.DataPlaneRoleReadyWaitingMessage))
	}

	return ctrl.Result{}, nil
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

	configMapName := fmt.Sprintf("dataplanerole-%s-inventory", instance.Name)
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

// SetupWithManager sets up the controller with the Manager.
func (r *OpenStackDataPlaneRoleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dataplanev1beta1.OpenStackDataPlaneRole{}).
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
	if node.Managed {
		ansibleVarsData["managed"] = node.Managed
	}
	if node.ManagementNetwork != "" {
		ansibleVarsData["management_network"] = node.ManagementNetwork
	}
	if node.NetworkConfig.Template != "" {
		ansibleVarsData["network_config"] = node.NetworkConfig
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
