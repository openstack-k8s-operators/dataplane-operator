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

	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"

	dataplanev1beta1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	dataplaneutil "github.com/openstack-k8s-operators/dataplane-operator/pkg/util"
	condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	novav1beta1 "github.com/openstack-k8s-operators/nova-operator/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// deployFuncDef so we can pass a function to ConditionalDeploy
type deployFuncDef func(context.Context, *helper.Helper, client.Object, string, string, dataplanev1beta1.AnsibleEESpec, dataplanev1beta1.OpenStackDataPlaneService) error

// Deploy function encapsulating primary deloyment handling
func Deploy(
	ctx context.Context,
	helper *helper.Helper,
	obj client.Object,
	nodes *dataplanev1beta1.OpenStackDataPlaneNodeList,
	sshKeySecret string,
	inventoryConfigMap string,
	status *dataplanev1beta1.OpenStackDataPlaneStatus,
	aeeSpec dataplanev1beta1.AnsibleEESpec,
	services []string,
	role *dataplanev1beta1.OpenStackDataPlaneRole,
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
	var foundService dataplanev1beta1.OpenStackDataPlaneService

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
		readyCondition = condition.Type(fmt.Sprintf(dataplanev1beta1.ServiceReadyCondition, service))
		readyWaitingMessage = fmt.Sprintf(dataplanev1beta1.ServiceReadyWaitingMessage, deployName)
		readyMessage = fmt.Sprintf(dataplanev1beta1.ServiceReadyMessage, deployName)
		readyErrorMessage = dataplanev1beta1.ServiceErrorMessage
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

	// InstallOS
	readyCondition = dataplanev1beta1.InstallOSReadyCondition
	readyWaitingMessage = dataplanev1beta1.InstallOSReadyWaitingMessage
	readyMessage = dataplanev1beta1.InstallOSReadyMessage
	readyErrorMessage = dataplanev1beta1.InstallOSErrorMessage
	deployFunc = InstallOS
	deployName = "InstallOS"
	deployLabel = InstallOSLabel
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
		foundService)

	if err != nil || !status.Conditions.IsTrue(readyCondition) {
		return &ctrl.Result{}, err
	}
	log.Info(fmt.Sprintf("Condition %s ready", readyCondition))

	// ConfigureOS
	readyCondition = dataplanev1beta1.ConfigureOSReadyCondition
	readyWaitingMessage = dataplanev1beta1.ConfigureOSReadyWaitingMessage
	readyMessage = dataplanev1beta1.ConfigureOSReadyMessage
	readyErrorMessage = dataplanev1beta1.ConfigureOSErrorMessage
	deployFunc = ConfigureOS
	deployName = "ConfigureOS"
	deployLabel = ConfigureOSLabel
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
		foundService)

	if err != nil || !status.Conditions.IsTrue(readyCondition) {
		return &ctrl.Result{}, err
	}
	log.Info(fmt.Sprintf("Condition %s ready", readyCondition))

	// RunOS
	readyCondition = dataplanev1beta1.RunOSReadyCondition
	readyWaitingMessage = dataplanev1beta1.RunOSReadyWaitingMessage
	readyMessage = dataplanev1beta1.RunOSReadyMessage
	readyErrorMessage = dataplanev1beta1.RunOSErrorMessage
	deployFunc = RunOS
	deployName = "RunOS"
	deployLabel = RunOSLabel
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
		foundService)

	if err != nil || !status.Conditions.IsTrue(readyCondition) {
		return &ctrl.Result{}, err
	}
	log.Info(fmt.Sprintf("Condition %s ready", readyCondition))

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
		readyCondition = dataplanev1beta1.ConfigureCephClientReadyCondition
		readyWaitingMessage = dataplanev1beta1.ConfigureCephClientReadyWaitingMessage
		readyMessage = dataplanev1beta1.ConfigureCephClientReadyMessage
		readyErrorMessage = dataplanev1beta1.ConfigureCephClientErrorMessage
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

	// InstallOpenStack
	readyCondition = dataplanev1beta1.InstallOpenStackReadyCondition
	readyWaitingMessage = dataplanev1beta1.InstallOpenStackReadyWaitingMessage
	readyMessage = dataplanev1beta1.InstallOpenStackReadyMessage
	readyErrorMessage = dataplanev1beta1.InstallOpenStackErrorMessage
	deployFunc = InstallOpenStack
	deployName = "InstallOpenStack"
	deployLabel = InstallOpenStackLabel
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
		foundService)

	if err != nil || !status.Conditions.IsTrue(readyCondition) {
		return &ctrl.Result{}, err
	}
	log.Info(fmt.Sprintf("Condition %s ready", readyCondition))

	// ConfigureOpenStack
	readyCondition = dataplanev1beta1.ConfigureOpenStackReadyCondition
	readyWaitingMessage = dataplanev1beta1.ConfigureOpenStackReadyWaitingMessage
	readyMessage = dataplanev1beta1.ConfigureOpenStackReadyMessage
	readyErrorMessage = dataplanev1beta1.ConfigureOpenStackErrorMessage
	deployFunc = ConfigureOpenStack
	deployName = "ConfigureOpenStack"
	deployLabel = ConfigureOpenStackLabel
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
		foundService)

	if err != nil || !status.Conditions.IsTrue(readyCondition) {
		return &ctrl.Result{}, err
	}
	log.Info(fmt.Sprintf("Condition %s ready", readyCondition))

	// RunOpenStack
	readyCondition = dataplanev1beta1.RunOpenStackReadyCondition
	readyWaitingMessage = dataplanev1beta1.RunOpenStackReadyWaitingMessage
	readyMessage = dataplanev1beta1.RunOpenStackReadyMessage
	readyErrorMessage = dataplanev1beta1.RunOpenStackErrorMessage
	deployFunc = RunOpenStack
	deployName = "RunOpenStack"
	deployLabel = RunOpenStackLabel
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
		foundService)

	if err != nil || !status.Conditions.IsTrue(readyCondition) {
		return &ctrl.Result{}, err
	}
	log.Info(fmt.Sprintf("Condition %s ready", readyCondition))

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
	status.Conditions.Set(condition.TrueCondition(dataplanev1beta1.NovaComputeReadyCondition, dataplanev1beta1.NovaComputeReadyMessage))

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
	status *dataplanev1beta1.OpenStackDataPlaneStatus,
	readyCondition condition.Type,
	readyMessage string,
	readyWaitingMessage string,
	readyErrorMessage string,
	deployFunc deployFuncDef,
	deployName string,
	deployLabel string,
	aeeSpec dataplanev1beta1.AnsibleEESpec,
	foundService dataplanev1beta1.OpenStackDataPlaneService,
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

	} else if status.Conditions.IsFalse(readyCondition) {
		ansibleEEJob, err := dataplaneutil.GetAnsibleExecutionJob(ctx, helper, obj, deployLabel)
		if err != nil && k8s_errors.IsNotFound(err) {
			log.Info(fmt.Sprintf("%s OpenStackAnsibleEE Job not yet found", readyCondition))
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
		} else if ansibleEEJob.Status.Succeeded > 0 {
			log.Info(fmt.Sprintf("Condition %s ready", readyCondition))
			status.Conditions.Set(condition.TrueCondition(
				readyCondition,
				readyMessage))
		} else if ansibleEEJob.Status.Failed > 0 {
			log.Info(fmt.Sprintf("Condition %s error", readyCondition))
			err = fmt.Errorf("failed: job.name %s job.namespace %s", ansibleEEJob.Name, ansibleEEJob.Namespace)
			status.Conditions.Set(condition.FalseCondition(
				readyCondition,
				condition.ErrorReason,
				condition.SeverityError,
				readyErrorMessage,
				err.Error()))
			return err
		} else {
			log.Info(fmt.Sprintf("Condition %s not yet ready", readyCondition))
			return nil
		}

	}

	return err

}
