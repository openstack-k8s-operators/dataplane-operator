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
	"time"

	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"

	dataplanev1beta1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	dataplaneutil "github.com/openstack-k8s-operators/dataplane-operator/pkg/util"
	condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// deployFuncDef so we can pass a function to ConditionalDeploy
type deployFuncDef func(context.Context, *helper.Helper, client.Object, string, string) error

// Deploy function encapsulating primary deloyment handling
func Deploy(ctx context.Context, helper *helper.Helper, obj client.Object, sshKeySecret string, inventoryConfigMap string, status *dataplanev1beta1.OpenStackDataPlaneNodeStatus) (ctrl.Result, error) {

	var result ctrl.Result
	var err error
	var readyCondition condition.Type
	var readyMessage string
	var readyWaitingMessage string
	var deployFunc deployFuncDef
	var deployName string
	var deployLabel string

	// Set DataPlaneNodeReadyCondition to requested
	status.Conditions.Set(condition.FalseCondition(
		dataplanev1beta1.DataPlaneNodeReadyCondition,
		condition.RequestedReason,
		condition.SeverityInfo,
		dataplanev1beta1.DataPlaneNodeReadyWaitingMessage))

	// ConfigureNetwork
	readyCondition = dataplanev1beta1.ConfigureNetworkReadyCondition
	readyWaitingMessage = dataplanev1beta1.ConfigureNetworkReadyWaitingMessage
	readyMessage = dataplanev1beta1.ConfigureNetworkReadyMessage
	deployFunc = ConfigureNetwork
	deployName = "ConfigureNetwork"
	deployLabel = ConfigureNetworkLabel
	result, err = ConditionalDeploy(ctx, helper, obj, sshKeySecret, inventoryConfigMap, status, readyCondition, readyMessage, readyWaitingMessage, deployFunc, deployName, deployLabel)
	if err != nil || result.RequeueAfter > 0 {
		return result, err
	}

	// ValidateNetwork
	readyCondition = dataplanev1beta1.ValidateNetworkReadyCondition
	readyWaitingMessage = dataplanev1beta1.ValidateNetworkReadyWaitingMessage
	readyMessage = dataplanev1beta1.ValidateNetworkReadyMessage
	deployFunc = ValidateNetwork
	deployName = "ValidateNetwork"
	deployLabel = ValidateNetworkLabel
	result, err = ConditionalDeploy(ctx, helper, obj, sshKeySecret, inventoryConfigMap, status, readyCondition, readyMessage, readyWaitingMessage, deployFunc, deployName, deployLabel)
	if err != nil || result.RequeueAfter > 0 {
		return result, err
	}

	// ConfigureOS
	readyCondition = dataplanev1beta1.ConfigureOSReadyCondition
	readyWaitingMessage = dataplanev1beta1.ConfigureOSReadyWaitingMessage
	readyMessage = dataplanev1beta1.ConfigureOSReadyMessage
	deployFunc = ConfigureOS
	deployName = "ConfigureOS"
	deployLabel = ConfigureOSLabel
	result, err = ConditionalDeploy(ctx, helper, obj, sshKeySecret, inventoryConfigMap, status, readyCondition, readyMessage, readyWaitingMessage, deployFunc, deployName, deployLabel)
	if err != nil || result.RequeueAfter > 0 {
		return result, err
	}

	status.Deployed = true
	return ctrl.Result{}, nil

}

// ConditionalDeploy function encapsulating primary deloyment handling with
// conditions.
func ConditionalDeploy(ctx context.Context, helper *helper.Helper, obj client.Object, sshKeySecret string, inventoryConfigMap string, status *dataplanev1beta1.OpenStackDataPlaneNodeStatus, readyCondition condition.Type, readyMessage string, readyWaitingMessage string, deployFunc deployFuncDef, deployName string, deployLabel string) (ctrl.Result, error) {

	var err error

	log := helper.GetLogger()

	if status.Conditions.IsUnknown(readyCondition) {
		log.Info(fmt.Sprintf("%s Unknown, starting %s", readyCondition, deployName))
		err = deployFunc(ctx, helper, obj, sshKeySecret, inventoryConfigMap)
		if err != nil {
			util.LogErrorForObject(helper, err, fmt.Sprintf("Unable to %s for %s", deployName, obj.GetName()), obj)
			return ctrl.Result{}, err
		}

		status.Conditions.Set(condition.FalseCondition(
			readyCondition,
			condition.RequestedReason,
			condition.SeverityInfo,
			readyWaitingMessage))

		log.Info(fmt.Sprintf("%s not yet ready, requeueing", readyCondition))
		return ctrl.Result{RequeueAfter: time.Second * 2}, nil

	} else if status.Conditions.IsFalse(readyCondition) {
		ansibleEEJob, err := dataplaneutil.GetAnsibleExecutionJob(ctx, helper, obj, deployLabel)
		if err != nil && k8s_errors.IsNotFound(err) {
			log.Info(fmt.Sprintf("%s not yet ready, requeueing", readyCondition))
			return ctrl.Result{RequeueAfter: time.Second * 2}, nil
		} else if err != nil {
			log.Error(err, fmt.Sprintf("Error getting ansibleEE job for %s", deployName))
			return ctrl.Result{}, err
		} else if ansibleEEJob.Status.Succeeded > 0 {
			log.Info(fmt.Sprintf("%s ready", readyCondition))
			status.Conditions.Set(condition.TrueCondition(
				readyCondition,
				readyMessage))
		} else {
			log.Info(fmt.Sprintf("%s not yet ready, requeueing", readyCondition))
			return ctrl.Result{RequeueAfter: time.Second * 2}, nil
		}

	} else if status.Conditions.IsTrue(readyCondition) {
		log.Info(fmt.Sprintf("%s already ready", readyCondition))
	}

	return ctrl.Result{}, err

}
