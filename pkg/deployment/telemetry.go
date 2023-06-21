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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"

	dataplanev1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	telemetryv1 "github.com/openstack-k8s-operators/telemetry-operator/api/v1beta1"
)

func DeployTelemetry(
	ctx context.Context,
	instance *dataplanev1.OpenStackDataPlaneRole,
	helper *helper.Helper,
	roleConfigMap string,
	ansibleSSHPrivateKeySecret string) error {

	log := helper.GetLogger()

	// deploy ceilometercompute
	ceilometercompute, op, err := CeilometerComputeCreateOrUpdate(instance, helper, roleConfigMap, ansibleSSHPrivateKeySecret)
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			telemetryv1.CeilometerComputeReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			telemetryv1.CeilometerComputeReadyErrorMessage,
			err.Error()))
		return err
	}
	if op != controllerutil.OperationResultNone {
		log.Info(fmt.Sprintf("Deployment %s successfully reconciled - operation: %s", instance.Name, string(op)))
	}

	// Mirror ceilometercompute's condition status
	ccompute := ceilometercompute.Status.Conditions.Mirror(telemetryv1.CeilometerComputeReadyCondition)
	if ccompute != nil {
		instance.Status.Conditions.Set(ccompute)
	}
	// end deploy ceilometercompute

	// deploy infracompute
	infracompute, op, err := InfraComputeCreateOrUpdate(instance, helper, roleConfigMap, ansibleSSHPrivateKeySecret)
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			telemetryv1.InfraComputeReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			telemetryv1.InfraComputeReadyErrorMessage,
			err.Error()))
		return err
	}
	if op != controllerutil.OperationResultNone {
		log.Info(fmt.Sprintf("Deployment %s successfully reconciled - operation: %s", instance.Name, string(op)))
	}

	// Mirror ceilometercompute's condition status
	icompute := infracompute.Status.Conditions.Mirror(telemetryv1.InfraComputeReadyCondition)
	if icompute != nil {
		instance.Status.Conditions.Set(icompute)
	}
	// end deploy infracompute

	return nil
}

func CeilometerComputeCreateOrUpdate(
	instance *dataplanev1.OpenStackDataPlaneRole,
	helper *helper.Helper,
	roleConfigMap string,
	ansibleSSHPrivateKeySecret string) (*telemetryv1.CeilometerCompute, controllerutil.OperationResult, error) {
	ccompute := &telemetryv1.CeilometerCompute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-ceilometer-compute", instance.Name),
			Namespace: instance.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(context.TODO(), helper.GetClient(), ccompute, func() error {
		ccompute.Spec = instance.Spec.CeilometerCompute
		ccompute.Spec.DataplaneInventoryConfigMap = roleConfigMap
		ccompute.Spec.DataplaneSSHSecret = ansibleSSHPrivateKeySecret

		err := controllerutil.SetControllerReference(instance, ccompute, helper.GetScheme())
		if err != nil {
			return err
		}

		return nil
	})

	return ccompute, op, err
}

func InfraComputeCreateOrUpdate(
	instance *dataplanev1.OpenStackDataPlaneRole,
	helper *helper.Helper,
	roleConfigMap string,
	ansibleSSHPrivateKeySecret string) (*telemetryv1.InfraCompute, controllerutil.OperationResult, error) {
	icompute := &telemetryv1.InfraCompute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-infra-compute", instance.Name),
			Namespace: instance.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(context.TODO(), helper.GetClient(), icompute, func() error {
		icompute.Spec = instance.Spec.InfraCompute
		icompute.Spec.DataplaneInventoryConfigMap = roleConfigMap
		icompute.Spec.DataplaneSSHSecret = ansibleSSHPrivateKeySecret

		err := controllerutil.SetControllerReference(instance, icompute, helper.GetScheme())
		if err != nil {
			return err
		}

		return nil
	})

	return icompute, op, err
}
