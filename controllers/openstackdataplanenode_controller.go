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

	"github.com/go-logr/logr"
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

	dataplanev1beta1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/dataplane-operator/pkg/deployment"
	"github.com/openstack-k8s-operators/lib-common/modules/ansible"
	condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
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
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete;

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
			condition.UnknownCondition(dataplanev1beta1.DataPlaneNodeReadyCondition, condition.InitReason, condition.InitReason),
			condition.UnknownCondition(dataplanev1beta1.ConfigureNetworkReadyCondition, condition.InitReason, condition.InitReason),
			condition.UnknownCondition(dataplanev1beta1.ValidateNetworkReadyCondition, condition.InitReason, condition.InitReason),
			condition.UnknownCondition(dataplanev1beta1.ConfigureOSReadyCondition, condition.InitReason, condition.InitReason),
		)

		instance.Status.Conditions.Init(&cl)

		instance.Status.Deployed = false

		// Register overall status immediately to have an early feedback e.g.
		// in the cli
		return ctrl.Result{}, nil

	}

	if instance.Spec.Node.Managed {
		err = r.Provision(ctx, instance)
		if err != nil {
			util.LogErrorForObject(helper, err, fmt.Sprintf("Unable to OpenStackDataPlaneNode %s", instance.Name), instance)
			return ctrl.Result{}, err
		}
	}

	inventoryConfigMap, err := r.GenerateInventory(ctx, instance)
	if err != nil {
		util.LogErrorForObject(helper, err, fmt.Sprintf("Unable to generate inventory for %s", instance.Name), instance)
		return ctrl.Result{}, err
	}

	if instance.Spec.Deploy {
		result, err = deployment.Deploy(ctx, helper, instance, instance.Spec.Node.AnsibleSSHPrivateKeySecret, inventoryConfigMap, &instance.Status)
		if err != nil {
			util.LogErrorForObject(helper, err, fmt.Sprintf("Unable to deploy %s", instance.Name), instance)
			return ctrl.Result{}, err
		}
		if result.RequeueAfter > 0 {
			return result, nil
		}
	}

	r.Log.Info("Set DataPlaneNodeReadyCondition true")
	instance.Status.Conditions.Set(condition.TrueCondition(dataplanev1beta1.DataPlaneNodeReadyCondition, dataplanev1beta1.DataPlaneNodeReadyMessage))

	return result, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpenStackDataPlaneNodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dataplanev1beta1.OpenStackDataPlaneNode{}).
		Owns(&v1alpha1.OpenStackAnsibleEE{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}

// Provision the data plane node
func (r *OpenStackDataPlaneNodeReconciler) Provision(ctx context.Context, instance *dataplanev1beta1.OpenStackDataPlaneNode) error {
	return nil
}

// GenerateInventory yields a parsed Inventory
func (r *OpenStackDataPlaneNodeReconciler) GenerateInventory(ctx context.Context, instance *dataplanev1beta1.OpenStackDataPlaneNode) (string, error) {
	var (
		err      error
		hostName string
	)

	inventory := ansible.MakeInventory()
	all := inventory.AddGroup("all")
	host := all.AddHost(instance.Name)
	host.Vars["ansible_user"] = instance.Spec.Node.AnsibleUser

	if instance.Spec.Node.AnsiblePort != 0 {
		host.Vars["ansible_port"] = instance.Spec.Node.AnsiblePort
	}

	if instance.Spec.AnsibleHost == "" {
		hostName = instance.Spec.HostName
	} else {
		hostName = instance.Spec.AnsibleHost
	}
	host.Vars["ansible_host"] = hostName

	ansibleVarsData := make(map[string]interface{})
	err = yaml.Unmarshal([]byte(instance.Spec.Node.AnsibleVars), ansibleVarsData)
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
