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

package deployment

import (
	"context"
	"errors"
	"fmt"
	"path"
	"sort"
	"strconv"

	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	dataplanev1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	dataplaneutil "github.com/openstack-k8s-operators/dataplane-operator/pkg/util"
	condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	"github.com/openstack-k8s-operators/lib-common/modules/storage"
	novav1beta1 "github.com/openstack-k8s-operators/nova-operator/api/v1beta1"
	ansibleeev1alpha1 "github.com/openstack-k8s-operators/openstack-ansibleee-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// deployFuncDef so we can pass a function to ConditionalDeploy
type deployFuncDef func(context.Context, *helper.Helper, client.Object, string, string, dataplanev1.AnsibleEESpec, dataplanev1.OpenStackDataPlaneService) error

// Deploy function encapsulating primary deloyment handling
func Deploy(
	ctx context.Context,
	helper *helper.Helper,
	obj client.Object,
	nodes *dataplanev1.OpenStackDataPlaneNodeList,
	sshKeySecret string,
	inventoryConfigMap string,
	status *dataplanev1.OpenStackDataPlaneStatus,
	aeeSpec dataplanev1.AnsibleEESpec,
	services []string,
	role *dataplanev1.OpenStackDataPlaneRole,
) (*ctrl.Result, error) {

	log := helper.GetLogger()

	var err error
	var readyCondition condition.Type
	var readyMessage string
	var readyWaitingMessage string
	var readyErrorMessage string
	var deployFunc deployFuncDef
	var deployName string
	var deployLabel string
	var foundService dataplanev1.OpenStackDataPlaneService

	// (slagle) For the prototype, we deploy all the composable services first
	for _, service := range services {
		log.Info("Deploying service", "service", service)
		foundService, err := GetService(ctx, helper, service)
		if err != nil {
			return &ctrl.Result{}, err
		}
		deployFunc = DeployService
		deployName = foundService.Name
		deployLabel = foundService.Spec.Label
		readyCondition = condition.Type(fmt.Sprintf(dataplanev1.ServiceReadyCondition, service))
		readyWaitingMessage = fmt.Sprintf(dataplanev1.ServiceReadyWaitingMessage, deployName)
		readyMessage = fmt.Sprintf(dataplanev1.ServiceReadyMessage, deployName)
		readyErrorMessage = dataplanev1.ServiceErrorMessage
		aeeSpec.OpenStackAnsibleEERunnerImage = foundService.Spec.OpenStackAnsibleEERunnerImage
		aeeSpec, err = addServiceExtraMounts(ctx, helper, aeeSpec, foundService)
		if err != nil {
			return &ctrl.Result{}, err
		}
		err = ConditionalDeploy(
			ctx,
			helper,
			obj,
			sshKeySecret,
			inventoryConfigMap,
			status,
			readyCondition,
			readyMessage,
			readyWaitingMessage,
			readyErrorMessage,
			deployFunc,
			deployName,
			deployLabel,
			aeeSpec,
			foundService,
		)
		// Some OpenStackDataPlaneService might need Kubernetes Services to be created
		if len(foundService.Spec.Services) > 0 {
			errKube := CreateKubeServices(&foundService, nodes, helper, make(map[string]string))
			if errKube != nil {
				return &ctrl.Result{}, errKube
			}
		}

		if err != nil || !status.Conditions.IsTrue(readyCondition) {
			return &ctrl.Result{}, err
		}
		log.Info(fmt.Sprintf("Condition %s ready", readyCondition))
	}

	// ConfigureCephClient
	haveCephSecret := false
	for _, extraMount := range aeeSpec.ExtraMounts {
		if extraMount.ExtraVolType == "Ceph" {
			haveCephSecret = true
			break
		}
	}
	if !haveCephSecret {
		log.Info("Skipping execution of Ansible for ConfigureCephClient because extraMounts does not have an extraVolType of Ceph.")
	} else {
		readyCondition = dataplanev1.ConfigureCephClientReadyCondition
		readyWaitingMessage = dataplanev1.ConfigureCephClientReadyWaitingMessage
		readyMessage = dataplanev1.ConfigureCephClientReadyMessage
		readyErrorMessage = dataplanev1.ConfigureCephClientErrorMessage
		deployFunc = ConfigureCephClient
		deployName = "ConfigureCephClient"
		deployLabel = ConfigureCephClientLabel
		err = ConditionalDeploy(
			ctx,
			helper,
			obj,
			sshKeySecret,
			inventoryConfigMap,
			status,
			readyCondition,
			readyMessage,
			readyWaitingMessage,
			readyErrorMessage,
			deployFunc,
			deployName,
			deployLabel,
			aeeSpec,
			foundService,
		)

		if err != nil || !status.Conditions.IsTrue(readyCondition) {
			return &ctrl.Result{}, err
		}
		log.Info(fmt.Sprintf("Condition %s ready", readyCondition))
	}

	// Call DeployNovaExternalCompute individually for each node
	var novaExternalComputes []*novav1beta1.NovaExternalCompute
	var novaReadyConditionsTrue []*condition.Condition
	var novaErrors []error
	for _, node := range nodes.Items {
		template, err := getNovaTemplate(&node, role)
		if err != nil {
			log.Error(err, "Failed to get merged NovaTemplate")
			novaErrors = append(novaErrors, err)
			continue
		}
		if template == nil {
			// If the Nova template is not defined neither in the Node nor in
			// the Role then it means the Node is not a compute node. So skip
			// NovaExternalCompute deployment.
			log.Info("Skip creating NovaExternalCompute as the Node is not a compute", "node", node.Name)
			continue
		}

		nodeConfigMapName := fmt.Sprintf("dataplanenode-%s", node.Name)
		novaExternalCompute, err := DeployNovaExternalCompute(
			ctx,
			helper,
			&node,
			obj,
			sshKeySecret,
			nodeConfigMapName,
			status,
			aeeSpec,
			*template,
		)
		if err != nil {
			novaErrors = append(novaErrors, err)
			continue
		}
		novaExternalComputes = append(novaExternalComputes, novaExternalCompute)
		novaReadyCondition := novaExternalCompute.Status.Conditions.Get(condition.ReadyCondition)
		log.Info("Nova Status", "NovaExternalCompute", node.Name, "IsReady", novaExternalCompute.IsReady())
		if novaExternalCompute.IsReady() {
			novaReadyConditionsTrue = append(novaReadyConditionsTrue, novaReadyCondition)

		}
	}

	// When any errors are found, wrap all into a single error, and return
	// it
	errStr := "DeployNovaExternalCompute error:"
	if len(novaErrors) > 0 {
		for _, err := range novaErrors {
			errStr = fmt.Sprintf("%s: %s", errStr, err.Error())
		}
		err = errors.New(errStr)
		return &ctrl.Result{}, err
	}

	// Return when any condition is not ready, otherwise set the role as
	// deployed.
	if len(novaReadyConditionsTrue) < len(novaExternalComputes) {
		log.Info("Not all NovaExternalCompute ReadyConditions are true.")
		return &ctrl.Result{}, nil
	}

	log.Info("All NovaExternalCompute ReadyConditions are true")
	status.Conditions.Set(condition.TrueCondition(dataplanev1.NovaComputeReadyCondition, dataplanev1.NovaComputeReadyMessage))

	return nil, nil

}

// ConditionalDeploy function encapsulating primary deloyment handling with
// conditions.
func ConditionalDeploy(
	ctx context.Context,
	helper *helper.Helper,
	obj client.Object,
	sshKeySecret string,
	inventoryConfigMap string,
	status *dataplanev1.OpenStackDataPlaneStatus,
	readyCondition condition.Type,
	readyMessage string,
	readyWaitingMessage string,
	readyErrorMessage string,
	deployFunc deployFuncDef,
	deployName string,
	deployLabel string,
	aeeSpec dataplanev1.AnsibleEESpec,
	foundService dataplanev1.OpenStackDataPlaneService,
) error {

	var err error
	log := helper.GetLogger()

	if status.Conditions.IsUnknown(readyCondition) {
		log.Info(fmt.Sprintf("%s Unknown, starting %s", readyCondition, deployName))
		err = deployFunc(ctx, helper, obj, sshKeySecret, inventoryConfigMap, aeeSpec, foundService)
		if err != nil {
			util.LogErrorForObject(helper, err, fmt.Sprintf("Unable to %s for %s", deployName, obj.GetName()), obj)
			return err
		}

		status.Conditions.Set(condition.FalseCondition(
			readyCondition,
			condition.RequestedReason,
			condition.SeverityInfo,
			readyWaitingMessage))

		log.Info(fmt.Sprintf("Condition %s unknown", readyCondition))
		return nil

	}

	if status.Conditions.IsFalse(readyCondition) {
		ansibleEE, err := dataplaneutil.GetAnsibleExecution(ctx, helper, obj, deployLabel)
		if err != nil && k8s_errors.IsNotFound(err) {
			log.Info(fmt.Sprintf("%s OpenStackAnsibleEE not yet found", readyCondition))
			return nil
		} else if err != nil {
			log.Error(err, fmt.Sprintf("Error getting ansibleEE job for %s", deployName))
			status.Conditions.Set(condition.FalseCondition(
				readyCondition,
				condition.ErrorReason,
				condition.SeverityWarning,
				readyErrorMessage,
				err.Error()))
			return err
		}

		if ansibleEE.Status.JobStatus == ansibleeev1alpha1.JobStatusSucceeded {
			log.Info(fmt.Sprintf("Condition %s ready", readyCondition))
			status.Conditions.Set(condition.TrueCondition(
				readyCondition,
				readyMessage))
			return nil
		}

		if ansibleEE.Status.JobStatus == ansibleeev1alpha1.JobStatusRunning || ansibleEE.Status.JobStatus == ansibleeev1alpha1.JobStatusPending {
			log.Info(fmt.Sprintf("AnsibleEE job is not yet completed: Execution: %s, Status: %s", ansibleEE.Name, ansibleEE.Status.JobStatus))
			status.Conditions.Set(condition.FalseCondition(
				readyCondition,
				condition.RequestedReason,
				condition.SeverityInfo,
				readyWaitingMessage))
			return nil
		}

		if ansibleEE.Status.JobStatus == ansibleeev1alpha1.JobStatusFailed {
			log.Info(fmt.Sprintf("Condition %s error", readyCondition))
			err = fmt.Errorf("Execution.name %s Execution.namespace %s Execution.status.jobstatus: %s", ansibleEE.Name, ansibleEE.Namespace, ansibleEE.Status.JobStatus)
			status.Conditions.Set(condition.FalseCondition(
				readyCondition,
				condition.ErrorReason,
				condition.SeverityError,
				readyErrorMessage,
				err.Error()))
			return err
		}

	}

	return err

}

// addServiceExtraMounts adds the service configs as ExtraMounts to aeeSpec
func addServiceExtraMounts(
	ctx context.Context,
	helper *helper.Helper,
	aeeSpec dataplanev1.AnsibleEESpec,
	service dataplanev1.OpenStackDataPlaneService,
) (dataplanev1.AnsibleEESpec, error) {

	client := helper.GetClient()
	baseMountPath := path.Join(ConfigPaths, service.Name)

	for _, cmName := range service.Spec.ConfigMaps {

		volMounts := storage.VolMounts{}
		cm := &corev1.ConfigMap{}
		err := client.Get(ctx, types.NamespacedName{Name: cmName, Namespace: service.Namespace}, cm)
		if err != nil {
			return aeeSpec, err
		}

		keys := []string{}
		for key := range cm.Data {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for idx, key := range keys {
			name := fmt.Sprintf("%s-%s", cmName, strconv.Itoa(idx))
			volume := corev1.Volume{
				Name: name,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: cmName,
						},
						Items: []corev1.KeyToPath{
							{
								Key:  key,
								Path: key,
							},
						},
					},
				},
			}

			volumeMount := corev1.VolumeMount{
				Name:      name,
				MountPath: path.Join(baseMountPath, key),
				SubPath:   key,
			}

			volMounts.Volumes = append(volMounts.Volumes, volume)
			volMounts.Mounts = append(volMounts.Mounts, volumeMount)

		}

		aeeSpec.ExtraMounts = append(aeeSpec.ExtraMounts, volMounts)
	}

	return aeeSpec, nil

}
