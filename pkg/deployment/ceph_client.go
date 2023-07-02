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

	"sigs.k8s.io/controller-runtime/pkg/client"

	dataplanev1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	dataplaneutil "github.com/openstack-k8s-operators/dataplane-operator/pkg/util"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	ansibleeev1 "github.com/openstack-k8s-operators/openstack-ansibleee-operator/api/v1alpha1"
)

// ConfigureCephClient ensures the Ceph client configuration files are on data plane nodes
func ConfigureCephClient(ctx context.Context, helper *helper.Helper, obj client.Object, sshKeySecret string, inventoryConfigMap string, aeeSpec dataplanev1.AnsibleEESpec, foundService dataplanev1.OpenStackDataPlaneService) error {

	tasks := []dataplaneutil.Task{
		{
			Name:          "Configure Ceph Client",
			RoleName:      "osp.edpm.edpm_ceph_client_files",
			RoleTasksFrom: "main.yml",
			Tags:          []string{"edpm_ceph_client_files"},
		},
	}

	role := ansibleeev1.Role{
		Name:     "Deploy EDPM Ceph client",
		Hosts:    "all",
		Strategy: "linear",
		Tasks:    dataplaneutil.PopulateTasks(tasks),
	}

	err := dataplaneutil.AnsibleExecution(ctx, helper, obj, ConfigureCephClientLabel, sshKeySecret, inventoryConfigMap, "", role, aeeSpec)
	if err != nil {
		helper.GetLogger().Error(err, "Unable to execute Ansible for ConfigureCephClient")
		return err
	}

	return nil

}
