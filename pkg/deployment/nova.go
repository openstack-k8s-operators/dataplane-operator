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

	"dario.cat/mergo"
	dataplanev1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	novav1beta1 "github.com/openstack-k8s-operators/nova-operator/api/v1beta1"
)

// DeployNovaExternalCompute deploys the nova compute configuration and services
func DeployNovaExternalCompute(
	ctx context.Context,
	helper *helper.Helper,
	node *dataplanev1.OpenStackDataPlaneNode,
	owner client.Object,
	sshKeySecret string,
	inventoryConfigMap string,
	status *dataplanev1.OpenStackDataPlaneStatus,
	aeeSpec dataplanev1.AnsibleEESpec,
	template dataplanev1.NovaTemplate,
) (*novav1beta1.NovaExternalCompute, error) {
	log := helper.GetLogger()

	log.Info("NovaExternalCompute deploy", "OpenStackControlPlaneNode", node.Name, "novaTemplate", template)

	novaExternalCompute := &novav1beta1.NovaExternalCompute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      node.GetName(),
			Namespace: node.GetNamespace(),
		},
	}

	_, err := controllerutil.CreateOrPatch(ctx, helper.GetClient(), novaExternalCompute, func() error {
		if novaExternalCompute.ObjectMeta.Labels == nil {
			novaExternalCompute.ObjectMeta.Labels = make(map[string]string)
		}
		log.Info(fmt.Sprintf("NovaExternalCompute: Adding label %s=%s", "openstackdataplanenode", node.GetName()))
		novaExternalCompute.ObjectMeta.Labels["openstackdataplanenode"] = node.GetName()

		// We need to call the default ctor to get the unspecified fields defaulted according to the CRD defaults
		// as otherwise golang would default those field to the golang empty value instead.
		novaExternalCompute.Spec = novav1beta1.NewNovaExternalComputeSpec(inventoryConfigMap, sshKeySecret)
		novaExternalCompute.Spec.CellName = template.CellName
		novaExternalCompute.Spec.NovaInstance = template.NovaInstance
		novaExternalCompute.Spec.CustomServiceConfig = template.CustomServiceConfig
		// NOTE(gibi): if DeployStrategy.Deploy is false but Nova.Deploy is true
		// then we never reach this point, so the Deploy true will not be passed
		// to NovaExternalCompute
		novaExternalCompute.Spec.Deploy = template.Deploy
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

// getNovaTemplate returns the NovaTemplate instance to be used. The NovaTemplate
// in the OpenStackDataPlaneNode if defined takes precedence over the NovaTemplate
// in the OpenStackDataPlaneRole.
func getNovaTemplate(
	node *dataplanev1.OpenStackDataPlaneNode,
	role *dataplanev1.OpenStackDataPlaneRole,
) (*dataplanev1.NovaTemplate, error) {
	if node.Spec.Node.Nova == nil {
		return role.Spec.NodeTemplate.Nova, nil
	}

	if node.Spec.Node.Nova != nil && role.Spec.NodeTemplate.Nova != nil {
		return mergeNovaTemplates(
			*node.Spec.Node.Nova, *role.Spec.NodeTemplate.Nova)
	}

	return node.Spec.Node.Nova, nil
}

func mergeNovaTemplates(
	node dataplanev1.NovaTemplate,
	role dataplanev1.NovaTemplate,
) (*dataplanev1.NovaTemplate, error) {
	merged := node.DeepCopy()
	err := mergo.Merge(merged, role)
	return merged, err
}
