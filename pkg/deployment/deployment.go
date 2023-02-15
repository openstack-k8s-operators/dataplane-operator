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

// Deploy function encapsulating primary deloyment handling
func Deploy(ctx context.Context, helper *helper.Helper, obj client.Object, sshKeySecret string, inventoryConfigMap string, status *dataplanev1beta1.OpenStackDataPlaneNodeStatus) (ctrl.Result, error) {

	var err error
	log := helper.GetLogger()

	// ConfigureNetwork
	if status.Conditions.IsUnknown(dataplanev1beta1.ConfigureNetworkReadyCondition) {
		log.Info("ConfigureNetworkReadyCondition Unknown, starting ConfigureNetwork")
		err = ConfigureNetwork(ctx, helper, obj, sshKeySecret, inventoryConfigMap)
		if err != nil {
			util.LogErrorForObject(helper, err, fmt.Sprintf("Unable to configure network for %s", obj.GetName()), obj)
			return ctrl.Result{}, err
		}

		status.Conditions.Set(condition.FalseCondition(
			dataplanev1beta1.ConfigureNetworkReadyCondition,
			condition.RequestedReason,
			condition.SeverityInfo,
			dataplanev1beta1.ConfigureNetworkReadyWaitingMessage))

		log.Info("ConfigureNetworkReadyCondition not yet ready, requeueing")
		return ctrl.Result{RequeueAfter: time.Second * 2}, nil

	} else if status.Conditions.IsFalse(dataplanev1beta1.ConfigureNetworkReadyCondition) {
		ansibleEEJob, err := dataplaneutil.GetAnsibleExecutionJob(ctx, helper, obj, ConfigureNetworkLabel)
		if err != nil && k8s_errors.IsNotFound(err) {
			log.Info("ConfigureNetworkReadyCondition not yet ready, requeueing")
			return ctrl.Result{RequeueAfter: time.Second * 2}, nil
		} else if err != nil {
			log.Error(err, "Error getting ansibleEE job for ConfigureNetwork")
			return ctrl.Result{}, err
		} else if ansibleEEJob.Status.Succeeded > 0 {
			log.Info("ConfigureNetworkReadyCondition ready")
			status.Conditions.Set(condition.TrueCondition(
				dataplanev1beta1.ConfigureNetworkReadyCondition,
				dataplanev1beta1.ConfigureNetworkReadyMessage))
		} else {
			log.Info("ConfigureNetworkReadyCondition not yet ready, requeueing")
			return ctrl.Result{RequeueAfter: time.Second * 2}, nil
		}

	} else if status.Conditions.IsTrue(dataplanev1beta1.ConfigureNetworkReadyCondition) {
		log.Info("ConfigureNetworkReadyCondition already ready")
	}

	Status.Deployed = true
	return ctrl.Result{}, nil

}
