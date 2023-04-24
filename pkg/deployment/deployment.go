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
	"fmt"
	"strings"

	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"

	dataplanev1beta1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	dataplaneutil "github.com/openstack-k8s-operators/dataplane-operator/pkg/util"
	condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Deployer specifies the methods that must be defined to use the interface. This provides a standard
// and re-usable interface for all methods associated with the deployment of a Dataplane node.
type Deployer interface {
	Configuration
	Validation
	Installation
	Runner
}

// Configuration defines the Configure interface. This method is implemented by each module used to deploy
// the Dataplane nodes.
type Configuration interface {
	Configure(context.Context, *helper.Helper, client.Object, string, string, dataplanev1beta1.AnsibleEESpec) error
}

// Validation defines the Validate interface. This method is implemented by each module used to deploy
// the Dataplane nodes.
type Validation interface {
	Validate(context.Context, *helper.Helper, client.Object, string, string, dataplanev1beta1.AnsibleEESpec) error
}

// Installation defines the Install interface. This method is implemented by each module used to deploy
// the Dataplane nodes.
type Installation interface {
	Install(context.Context, *helper.Helper, client.Object, string, string, dataplanev1beta1.AnsibleEESpec) error
}

// Runner defines the Run interface. This method is implemented by each module used to deploy
// the Dataplane nodes.
type Runner interface {
	Run(context.Context, *helper.Helper, client.Object, string, string, dataplanev1beta1.AnsibleEESpec) error
}

// Networker provides the struct used for the network.go receiver functions.
type Networker struct{}

// OperatingSystem provides the struct used for the os.go receiver functions.
type OperatingSystem struct{}

// CephClient provides the struct used for the ceph_client.go receiver functions.
type CephClient struct{}

// Deploy function encapsulating primary deloyment handling
func Deploy(
	ctx context.Context,
	helper *helper.Helper,
	obj client.Object,
	sshKeySecret string,
	inventoryConfigMap string,
	status *dataplanev1beta1.OpenStackDataPlaneStatus,
	aeeSpec dataplanev1beta1.AnsibleEESpec,
) (ctrl.Result, error) {

	log := helper.GetLogger()

	var err error
	var readyCondition condition.Type
	var readyMessage string
	var readyWaitingMessage string
	var readyErrorMessage string
	var deployName string
	var deployLabel string

	// Set ReadyCondition to requested
	status.Conditions.Set(condition.FalseCondition(
		condition.ReadyCondition,
		condition.RequestedReason,
		condition.SeverityInfo,
		dataplanev1beta1.DataPlaneNodeReadyWaitingMessage))

	// ConfigureNetwork
	readyCondition = dataplanev1beta1.ConfigureNetworkReadyCondition
	readyWaitingMessage = dataplanev1beta1.ConfigureNetworkReadyWaitingMessage
	readyMessage = dataplanev1beta1.ConfigureNetworkReadyMessage
	readyErrorMessage = dataplanev1beta1.ConfigureNetworkErrorMessage
	deployName = "ConfigureNetwork"
	deployLabel = ConfigureNetworkLabel
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
		Networker{},
		deployName,
		deployLabel,
		aeeSpec,
	)

	if err != nil || !status.Conditions.IsTrue(readyCondition) {
		return ctrl.Result{}, err
	}
	log.Info(fmt.Sprintf("Condition %s ready", readyCondition))

	// ValidateNetwork
	readyCondition = dataplanev1beta1.ValidateNetworkReadyCondition
	readyWaitingMessage = dataplanev1beta1.ValidateNetworkReadyWaitingMessage
	readyMessage = dataplanev1beta1.ValidateNetworkReadyMessage
	readyErrorMessage = dataplanev1beta1.ValidateNetworkErrorMessage
	deployName = "ValidateNetwork"
	deployLabel = ValidateNetworkLabel
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
		Networker{},
		deployName,
		deployLabel,
		aeeSpec,
	)

	if err != nil || !status.Conditions.IsTrue(readyCondition) {
		return ctrl.Result{}, err
	}
	log.Info(fmt.Sprintf("Condition %s ready", readyCondition))

	// InstallOS
	readyCondition = dataplanev1beta1.InstallOSReadyCondition
	readyWaitingMessage = dataplanev1beta1.InstallOSReadyWaitingMessage
	readyMessage = dataplanev1beta1.InstallOSReadyMessage
	readyErrorMessage = dataplanev1beta1.InstallOSErrorMessage
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
		OperatingSystem{},
		deployName,
		deployLabel,
		aeeSpec)

	if err != nil || !status.Conditions.IsTrue(readyCondition) {
		return ctrl.Result{}, err
	}
	log.Info(fmt.Sprintf("Condition %s ready", readyCondition))

	// ConfigureOS
	readyCondition = dataplanev1beta1.ConfigureOSReadyCondition
	readyWaitingMessage = dataplanev1beta1.ConfigureOSReadyWaitingMessage
	readyMessage = dataplanev1beta1.ConfigureOSReadyMessage
	readyErrorMessage = dataplanev1beta1.ConfigureOSErrorMessage
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
		OperatingSystem{},
		deployName,
		deployLabel,
		aeeSpec)

	if err != nil || !status.Conditions.IsTrue(readyCondition) {
		return ctrl.Result{}, err
	}
	log.Info(fmt.Sprintf("Condition %s ready", readyCondition))

	// RunOS
	readyCondition = dataplanev1beta1.RunOSReadyCondition
	readyWaitingMessage = dataplanev1beta1.RunOSReadyWaitingMessage
	readyMessage = dataplanev1beta1.RunOSReadyMessage
	readyErrorMessage = dataplanev1beta1.RunOSErrorMessage
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
		OperatingSystem{},
		deployName,
		deployLabel,
		aeeSpec)

	if err != nil || !status.Conditions.IsTrue(readyCondition) {
		return ctrl.Result{}, err
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
		helper.GetLogger().Info("Skipping execution of Ansible for ConfigureCephClient because extraMounts does not have an extraVolType of Ceph.")
	} else {
		readyCondition = dataplanev1beta1.ConfigureCephClientReadyCondition
		readyWaitingMessage = dataplanev1beta1.ConfigureCephClientReadyWaitingMessage
		readyMessage = dataplanev1beta1.ConfigureCephClientReadyMessage
		readyErrorMessage = dataplanev1beta1.ConfigureCephClientErrorMessage
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
			CephClient{},
			deployName,
			deployLabel,
			aeeSpec,
		)

		if err != nil || !status.Conditions.IsTrue(readyCondition) {
			return ctrl.Result{}, err
		}
		log.Info(fmt.Sprintf("Condition %s ready", readyCondition))
	}

	// InstallOpenStack
	readyCondition = dataplanev1beta1.InstallOpenStackReadyCondition
	readyWaitingMessage = dataplanev1beta1.InstallOpenStackReadyWaitingMessage
	readyMessage = dataplanev1beta1.InstallOpenStackReadyMessage
	readyErrorMessage = dataplanev1beta1.InstallOpenStackErrorMessage
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
		OperatingSystem{},
		deployName,
		deployLabel,
		aeeSpec)

	if err != nil || !status.Conditions.IsTrue(readyCondition) {
		return ctrl.Result{}, err
	}
	log.Info(fmt.Sprintf("Condition %s ready", readyCondition))

	// ConfigureOpenStack
	readyCondition = dataplanev1beta1.ConfigureOpenStackReadyCondition
	readyWaitingMessage = dataplanev1beta1.ConfigureOpenStackReadyWaitingMessage
	readyMessage = dataplanev1beta1.ConfigureOpenStackReadyMessage
	readyErrorMessage = dataplanev1beta1.ConfigureOpenStackErrorMessage
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
		OperatingSystem{},
		deployName,
		deployLabel,
		aeeSpec)

	if err != nil || !status.Conditions.IsTrue(readyCondition) {
		return ctrl.Result{}, err
	}
	log.Info(fmt.Sprintf("Condition %s ready", readyCondition))

	// RunOpenStack
	readyCondition = dataplanev1beta1.RunOpenStackReadyCondition
	readyWaitingMessage = dataplanev1beta1.RunOpenStackReadyWaitingMessage
	readyMessage = dataplanev1beta1.RunOpenStackReadyMessage
	readyErrorMessage = dataplanev1beta1.RunOpenStackErrorMessage
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
		OperatingSystem{},
		deployName,
		deployLabel,
		aeeSpec)

	if err != nil || !status.Conditions.IsTrue(readyCondition) {
		return ctrl.Result{}, err
	}
	log.Info(fmt.Sprintf("Condition %s ready", readyCondition))

	return ctrl.Result{}, nil

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
	serviceInterface Deployer,
	deployName string,
	deployLabel string,
	aeeSpec dataplanev1beta1.AnsibleEESpec,
) error {

	var err error
	log := helper.GetLogger()

	if status.Conditions.IsUnknown(readyCondition) {
		log.Info(fmt.Sprintf("%s Unknown, starting %s", readyCondition, deployName))
		switch true {
		case strings.Contains(deployName, "Configure"):
			err = serviceInterface.Configure(ctx, helper, obj, sshKeySecret, inventoryConfigMap, aeeSpec)
		case strings.Contains(deployName, "Validate"):
			err = serviceInterface.Validate(ctx, helper, obj, sshKeySecret, inventoryConfigMap, aeeSpec)
		case strings.Contains(deployName, "Install"):
			err = serviceInterface.Install(ctx, helper, obj, sshKeySecret, inventoryConfigMap, aeeSpec)
		case strings.Contains(deployName, "Run"):
			err = serviceInterface.Run(ctx, helper, obj, sshKeySecret, inventoryConfigMap, aeeSpec)
		}

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
