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

	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	ansibleeev1 "github.com/openstack-k8s-operators/openstack-ansibleee-operator/api/v1beta1"
	baremetalv1 "github.com/openstack-k8s-operators/openstack-baremetal-operator/api/v1beta1"
)

var dataplaneAnsibleImageDefaults dataplanev1.DataplaneAnsibleImageDefaults

const (
	// FrrDefaultImage -
	FrrDefaultImage = "quay.io/podified-antelope-centos9/openstack-frr:current-podified"
	// IscsiDDefaultImage -
	IscsiDDefaultImage = "quay.io/podified-antelope-centos9/openstack-iscsid:current-podified"
	// LogrotateDefaultImage -
	LogrotateDefaultImage = "quay.io/podified-antelope-centos9/openstack-cron:current-podified"
	// NeutronMetadataAgentDefaultImage -
	NeutronMetadataAgentDefaultImage = "quay.io/podified-antelope-centos9/openstack-neutron-metadata-agent-ovn:current-podified"
	// NovaComputeDefaultImage -
	NovaComputeDefaultImage = "quay.io/podified-antelope-centos9/openstack-nova-compute:current-podified"
	// NovaLibvirtDefaultImage -
	NovaLibvirtDefaultImage = "quay.io/podified-antelope-centos9/openstack-nova-libvirt:current-podified"
	// OvnControllerAgentDefaultImage -
	OvnControllerAgentDefaultImage = "quay.io/podified-antelope-centos9/openstack-ovn-controller:current-podified"
	// OvnBgpAgentDefaultImage -
	OvnBgpAgentDefaultImage = "quay.io/podified-antelope-centos9/openstack-ovn-bgp-agent:current-podified"
)

// NodeConfigElements is a struct containing just the elements used for Ansible executions
type NodeConfigElements struct {
	BaremetalSetTemplate baremetalv1.OpenStackBaremetalSetSpec
	NodeTemplate         dataplanev1.NodeTemplate
	Nodes                map[string]dataplanev1.NodeSection
}

// SetupAnsibleImageDefaults -
func SetupAnsibleImageDefaults() {
	dataplaneAnsibleImageDefaults = dataplanev1.DataplaneAnsibleImageDefaults{
		Frr:                  util.GetEnvVar("RELATED_IMAGE_OPENSTACK_EDPM_FRR_DEFAULT_IMG", FrrDefaultImage),
		IscsiD:               util.GetEnvVar("RELATED_IMAGE_OPENSTACK_EDPM_ISCSID_DEFAULT_IMG", IscsiDDefaultImage),
		Logrotate:            util.GetEnvVar("RELATED_IMAGE_OPENSTACK_EDPM_LOGROTATE_CROND_DEFAULT_IMG", LogrotateDefaultImage),
		NeutronMetadataAgent: util.GetEnvVar("RELATED_IMAGE_OPENSTACK_EDPM_NEUTRON_METADATA_AGENT_DEFAULT_IMG", NeutronMetadataAgentDefaultImage),
		NovaCompute:          util.GetEnvVar("RELATED_IMAGE_OPENSTACK_EDPM_NOVA_COMPUTE_DEFAULT_IMG", NovaComputeDefaultImage),
		NovaLibvirt:          util.GetEnvVar("RELATED_IMAGE_OPENSTACK_EDPM_LIBVIRT_DEFAULT_IMG", NovaLibvirtDefaultImage),
		OvnControllerAgent:   util.GetEnvVar("RELATED_IMAGE_OPENSTACK_EDPM_OVN_CONTROLLER_AGENT_DEFAULT_IMG", OvnControllerAgentDefaultImage),
		OvnBgpAgent:          util.GetEnvVar("RELATED_IMAGE_OPENSTACK_EDPM_OVN_BGP_AGENT_IMAGE", OvnBgpAgentDefaultImage),
	}
}

const (
	// AnsibleSSHPrivateKey ssh private key
	AnsibleSSHPrivateKey = "ssh-privatekey"
	// AnsibleSSHAuthorizedKeys authorized keys
	AnsibleSSHAuthorizedKeys = "authorized_keys"
)

// OpenStackDataPlaneNodeSetReconciler reconciles a OpenStackDataPlaneNodeSet object
type OpenStackDataPlaneNodeSetReconciler struct {
	client.Client
	Kclient kubernetes.Interface
	Scheme  *runtime.Scheme
	Log     logr.Logger
}

//+kubebuilder:rbac:groups=dataplane.openstack.org,resources=openstackdataplanenodesets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dataplane.openstack.org,resources=openstackdataplanenodesets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dataplane.openstack.org,resources=openstackdataplanenodesets/finalizers,verbs=update
//+kubebuilder:rbac:groups=dataplane.openstack.org,resources=openstackdataplaneservices,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups=dataplane.openstack.org,resources=openstackdataplaneservices/finalizers,verbs=update
//+kubebuilder:rbac:groups=baremetal.openstack.org,resources=openstackbaremetalsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=baremetal.openstack.org,resources=openstackbaremetalsets/status,verbs=get
//+kubebuilder:rbac:groups=baremetal.openstack.org,resources=openstackbaremetalsets/finalizers,verbs=update
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
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete;
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the OpenStackDataPlaneNodeSet object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *OpenStackDataPlaneNodeSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, _err error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling NodeSet")

	// Fetch the OpenStackDataPlaneNodeSet instance
	instance := &dataplanev1.OpenStackDataPlaneNodeSet{}
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

	// Always patch the instance status when exiting this function so we can persist any changes.
	defer func() { // update the Ready condition based on the sub conditions
		if instance.Status.Conditions.AllSubConditionIsTrue() {
			instance.Status.Conditions.MarkTrue(
				condition.ReadyCondition, dataplanev1.NodeSetReadyMessage)
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

	// Initialize Status
	if instance.Status.Conditions == nil {
		instance.InitConditions()
		// Register overall status immediately to have an early feedback e.g.
		// in the cli
		return ctrl.Result{}, nil
	}
	if instance.Status.ConfigMapHashes == nil {
		instance.Status.ConfigMapHashes = make(map[string]string)
	}
	if instance.Status.SecretHashes == nil {
		instance.Status.SecretHashes = make(map[string]string)
	}

	instance.Status.Conditions.MarkFalse(dataplanev1.SetupReadyCondition, condition.RequestedReason, condition.SeverityInfo, condition.ReadyInitMessage)

	// Detect config changes and set Status ConfigHash
	configHash, err := r.GetSpecConfigHash(instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	if configHash != instance.Status.DeployedConfigHash {
		instance.Status.ConfigHash = configHash
	}

	// Ensure Services
	err = deployment.EnsureServices(ctx, helper, instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Ensure IPSets Required for Nodes
	allIPSets, isReady, err := deployment.EnsureIPSets(ctx, helper, instance)
	if err != nil || !isReady {
		return ctrl.Result{}, err
	}

	// Ensure DNSData Required for Nodes
	dnsAddresses, dnsClusterAddresses, ctlplaneSearchDomain, isReady, allHostnames, allIPs, err := deployment.EnsureDNSData(
		ctx, helper,
		instance, allIPSets)
	if err != nil || !isReady {
		return ctrl.Result{}, err
	}
	instance.Status.DNSClusterAddresses = dnsClusterAddresses
	instance.Status.CtlplaneSearchDomain = ctlplaneSearchDomain
	instance.Status.AllHostnames = allHostnames
	instance.Status.AllIPs = allIPs

	ansibleSSHPrivateKeySecret := instance.Spec.NodeTemplate.AnsibleSSHPrivateKeySecret

	secretKeys := []string{}
	secretKeys = append(secretKeys, AnsibleSSHPrivateKey)
	if !instance.Spec.PreProvisioned {
		secretKeys = append(secretKeys, AnsibleSSHAuthorizedKeys)
	}
	_, result, err = secret.VerifySecret(
		ctx,
		types.NamespacedName{
			Namespace: instance.Namespace,
			Name:      ansibleSSHPrivateKeySecret,
		},
		secretKeys,
		helper.GetClient(),
		time.Second*5,
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

	// all our input checks out so report InputReady
	instance.Status.Conditions.MarkTrue(condition.InputReadyCondition, condition.InputReadyMessage)

	// Reconcile BaremetalSet if required
	if !instance.Spec.PreProvisioned {
		// Reset the NodeSetBareMetalProvisionReadyCondition to unknown
		instance.Status.Conditions.MarkUnknown(dataplanev1.NodeSetBareMetalProvisionReadyCondition,
			condition.InitReason, condition.InitReason)
		isReady, err := deployment.DeployBaremetalSet(ctx, helper, instance,
			allIPSets, dnsAddresses)
		if err != nil || !isReady {
			return ctrl.Result{}, err
		}
	}

	if instance.Status.Deployed && instance.DeletionTimestamp.IsZero() {
		// The role is already deployed and not being deleted, so reconciliation
		// is already complete.
		logger.Info("NodeSet already deployed", "instance", instance)
		return ctrl.Result{}, nil
	}

	// Generate NodeSet Inventory
	_, err = deployment.GenerateNodeSetInventory(ctx, helper, instance,
		allIPSets, dnsAddresses, dataplaneAnsibleImageDefaults)
	if err != nil {
		util.LogErrorForObject(helper, err, fmt.Sprintf("Unable to generate inventory for %s", instance.Name), instance)
		return ctrl.Result{}, err
	}

	// all setup tasks complete, mark SetupReadyCondition True
	instance.Status.Conditions.MarkTrue(dataplanev1.SetupReadyCondition, condition.ReadyMessage)

	// Set DeploymentReadyCondition to False if it was unknown.
	// Handles the case where the NodeSet is created, but not yet deployed.
	if instance.Status.Conditions.IsUnknown(condition.DeploymentReadyCondition) {
		logger.Info("Set DeploymentReadyCondition false")
		instance.Status.Conditions.MarkFalse(condition.DeploymentReadyCondition,
			condition.NotRequestedReason, condition.SeverityInfo,
			condition.DeploymentReadyInitMessage)
	}

	deploymentExists, isDeploymentReady, err := checkDeployment(helper, instance)
	if err != nil {
		logger.Error(err, "Unable to get deployed OpenStackDataPlaneDeployments.")
		return ctrl.Result{}, err
	}
	if isDeploymentReady {
		logger.Info("Set NodeSet DeploymentReadyCondition true")
		instance.Status.Conditions.MarkTrue(condition.DeploymentReadyCondition,
			condition.DeploymentReadyMessage)
		instance.Status.DeployedConfigHash = configHash
	} else if deploymentExists {
		logger.Info("Set NodeSet DeploymentReadyCondition false")
		instance.Status.Conditions.MarkFalse(condition.DeploymentReadyCondition,
			condition.RequestedReason, condition.SeverityInfo,
			condition.DeploymentReadyRunningMessage)
	} else {
		logger.Info("Set NodeSet DeploymentReadyCondition false")
		instance.Status.Conditions.MarkFalse(condition.DeploymentReadyCondition,
			condition.RequestedReason, condition.SeverityInfo,
			condition.DeploymentReadyInitMessage)
	}
	return ctrl.Result{}, nil
}

func checkDeployment(helper *helper.Helper,
	instance *dataplanev1.OpenStackDataPlaneNodeSet,
) (bool, bool, error) {
	// Get all completed deployments
	deployments := &dataplanev1.OpenStackDataPlaneDeploymentList{}
	opts := []client.ListOption{
		client.InNamespace(instance.Namespace),
	}
	err := helper.GetClient().List(context.Background(), deployments, opts...)
	if err != nil {
		helper.GetLogger().Error(err, "Unable to retrieve OpenStackDataPlaneDeployment CRs %v")
		return false, false, err
	}

	isDeploymentReady := false
	deploymentExists := false

	// Sort deployments from oldest to newest by the LastTransitionTime of
	// their DeploymentReadyCondition
	slices.SortFunc(deployments.Items, func(a, b dataplanev1.OpenStackDataPlaneDeployment) int {
		aReady := a.Status.Conditions.Get(condition.DeploymentReadyCondition)
		bReady := b.Status.Conditions.Get(condition.DeploymentReadyCondition)
		if aReady != nil && bReady != nil {
			if aReady.LastTransitionTime.Before(&bReady.LastTransitionTime) {
				return -1
			}
		}
		return 1
	})

	for _, deployment := range deployments.Items {
		if !deployment.DeletionTimestamp.IsZero() {
			continue
		}
		if slices.Contains(
			deployment.Spec.NodeSets, instance.Name) {
			deploymentExists = true
			isDeploymentReady = false
			if deployment.Status.Deployed {
				isDeploymentReady = true
				for k, v := range deployment.Status.ConfigMapHashes {
					instance.Status.ConfigMapHashes[k] = v
				}
				for k, v := range deployment.Status.SecretHashes {
					instance.Status.SecretHashes[k] = v
				}
			}
			deploymentConditions := deployment.Status.NodeSetConditions[instance.Name]
			if instance.Status.DeploymentStatuses == nil {
				instance.Status.DeploymentStatuses = make(map[string]condition.Conditions)
			}
			instance.Status.DeploymentStatuses[deployment.Name] = deploymentConditions
		}
	}

	return deploymentExists, isDeploymentReady, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpenStackDataPlaneNodeSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	reconcileFunction := handler.EnqueueRequestsFromMapFunc(func(o client.Object) []reconcile.Request {
		result := []reconcile.Request{}

		// For each DNSMasq change event get the list of all
		// OpenStackDataPlaneNodeSet to trigger reconcile for the
		// ones in the same namespace.
		nodeSets := &dataplanev1.OpenStackDataPlaneNodeSetList{}

		listOpts := []client.ListOption{
			client.InNamespace(o.GetNamespace()),
		}
		if err := r.Client.List(context.Background(), nodeSets, listOpts...); err != nil {
			r.Log.Error(err, "Unable to retrieve OpenStackDataPlaneNodeSetList %w")
			return nil
		}

		// For each role instance create a reconcile request
		for _, i := range nodeSets.Items {
			name := client.ObjectKey{
				Namespace: o.GetNamespace(),
				Name:      i.Name,
			}
			result = append(result, reconcile.Request{NamespacedName: name})
		}
		return result
	})

	deploymentWatcher := handler.EnqueueRequestsFromMapFunc(func(obj client.Object) []reconcile.Request {
		var namespace string = obj.GetNamespace()
		result := []reconcile.Request{}

		deployment := obj.(*dataplanev1.OpenStackDataPlaneDeployment)
		for _, nodeSet := range deployment.Spec.NodeSets {
			name := client.ObjectKey{
				Namespace: namespace,
				Name:      nodeSet,
			}
			result = append(result, reconcile.Request{NamespacedName: name})
		}
		podsInterface := r.Kclient.CoreV1().Pods(namespace)
		// List service pods in the given namespace
		podsList, err := podsInterface.List(context.TODO(), v1.ListOptions{
			LabelSelector: fmt.Sprintf("osdpd=%s", deployment.Name),
			FieldSelector: "status.phase=Failed",
		})

		if err != nil {
			r.Log.Error(err, "unable to retrieve list of pods for dataplane diagnostic")
		} else {
			for _, pod := range podsList.Items {
				r.Log.Info(
					fmt.Sprintf(
						"openstack dataplane pod %s failed due to %s message: %s",
						pod.Name, pod.Status.Reason, pod.Status.Message))
			}
		}
		return result
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&dataplanev1.OpenStackDataPlaneNodeSet{}).
		Owns(&ansibleeev1.OpenStackAnsibleEE{}).
		Owns(&baremetalv1.OpenStackBaremetalSet{}).
		Owns(&infranetworkv1.IPSet{}).
		Owns(&infranetworkv1.DNSData{}).
		Owns(&corev1.Secret{}).
		Watches(&source.Kind{Type: &infranetworkv1.DNSMasq{}},
			reconcileFunction).
		Watches(&source.Kind{Type: &dataplanev1.OpenStackDataPlaneDeployment{}},
			deploymentWatcher).
		Complete(r)
}

// GetSpecConfigHash initialises a new struct with only the field we want to check for variances in.
// We then hash the contents of the new struct using md5 and return the hashed string.
func (r *OpenStackDataPlaneNodeSetReconciler) GetSpecConfigHash(instance *dataplanev1.OpenStackDataPlaneNodeSet) (string, error) {
	nodeConfig := &NodeConfigElements{
		BaremetalSetTemplate: instance.Spec.BaremetalSetTemplate,
		NodeTemplate:         instance.Spec.NodeTemplate,
		Nodes:                instance.Spec.Nodes,
	}
	configHash, err := util.ObjectHash(&nodeConfig)
	if err != nil {
		return "", err
	}

	return configHash, nil
}
