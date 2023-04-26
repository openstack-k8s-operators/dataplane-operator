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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	dataplanev1beta1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	novav1beta1 "github.com/openstack-k8s-operators/nova-operator/api/v1beta1"
)

// DeployNovaExternalCompute deploys the nova compute configuration and services
func DeployNovaExternalCompute(ctx context.Context, helper *helper.Helper, obj client.Object, owner client.Object, sshKeySecret string, inventoryConfigMap string, status *dataplanev1beta1.OpenStackDataPlaneStatus, aeeSpec dataplanev1beta1.AnsibleEESpec) (*novav1beta1.NovaExternalCompute, error) {

	log := helper.GetLogger()
	log.Info(fmt.Sprintf("NovaExternalCompute deploy for %s", obj.GetName()))

	novaExternalCompute := &novav1beta1.NovaExternalCompute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetName(),
			Namespace: obj.GetNamespace(),
		},
	}

	_, err := controllerutil.CreateOrPatch(ctx, helper.GetClient(), novaExternalCompute, func() error {
		if novaExternalCompute.ObjectMeta.Labels == nil {
			novaExternalCompute.ObjectMeta.Labels = make(map[string]string)
		}
		log.Info(fmt.Sprintf("NovaExternalCompute: Adding label %s=%s", "openstackdataplanenode", obj.GetName()))
		novaExternalCompute.ObjectMeta.Labels["openstackdataplanenode"] = obj.GetName()

		// We need to call the default ctor to get the unspecified fields defaulted according to the CRD defaults
		// as otherwise golang would defaul those field to the golang empty value instead.
		novaExternalCompute.Spec = novav1beta1.NewNovaExternalComputeSpec(inventoryConfigMap, sshKeySecret)
		novaExternalCompute.Spec.Deploy = true
		novaExternalCompute.Spec.NetworkAttachments = aeeSpec.NetworkAttachments
		novaExternalCompute.Spec.AnsibleEEContainerImage = aeeSpec.OpenStackAnsibleEERunnerImage

		err := controllerutil.SetControllerReference(owner, novaExternalCompute, helper.GetScheme())
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		util.LogErrorForObject(helper, err, fmt.Sprintf("Unable to CreateOrPatch NovaExternalCompute %s", novaExternalCompute.Name), novaExternalCompute)
		return nil, err
	}

	return novaExternalCompute, nil

}
