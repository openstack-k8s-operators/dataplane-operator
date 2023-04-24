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

	"sigs.k8s.io/controller-runtime/pkg/client"

	dataplanev1beta1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	dataplaneutil "github.com/openstack-k8s-operators/dataplane-operator/pkg/util"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	ansibleeev1alpha1 "github.com/openstack-k8s-operators/openstack-ansibleee-operator/api/v1alpha1"
)

// Configure ensures the network config
func (n Networker) Configure(ctx context.Context, helper *helper.Helper, obj client.Object, sshKeySecret string, inventoryConfigMap string, aeeSpec dataplanev1beta1.AnsibleEESpec) error {

	tasks := []dataplaneutil.Task{
		{
			Name:          "Configure Hosts Entries",
			RoleName:      "osp.edpm.edpm_hosts_entries",
			RoleTasksFrom: "main.yml",
			Tags:          []string{"edpm_hosts_entries"},
		},
		{
			Name:          "import edpm_network_config",
			RoleName:      "osp.edpm.edpm_network_config",
			RoleTasksFrom: "main.yml",
			Tags:          []string{"edpm_network_config"},
		},
	}
	role := ansibleeev1alpha1.Role{
		Name:     "Deploy EDPM Network",
		Hosts:    "all",
		Strategy: "linear",
		Tasks:    dataplaneutil.PopulateTasks(tasks),
	}

	err := dataplaneutil.AnsibleExecution(ctx, helper, obj, ConfigureNetworkLabel, sshKeySecret, inventoryConfigMap, "", role, aeeSpec)
	if err != nil {
		helper.GetLogger().Error(err, "Unable to execute Ansible for ConfigureNetwork")
		return err
	}

	return nil

}

// Validate ensures the node network config
func (n Networker) Validate(ctx context.Context, helper *helper.Helper, obj client.Object, sshKeySecret string, inventoryConfigMap string, aeeSpec dataplanev1beta1.AnsibleEESpec) error {

	role := ansibleeev1alpha1.Role{
		Name:     "osp.edpm.edpm_nodes_validation",
		Hosts:    "all",
		Strategy: "linear",
		Tasks: []ansibleeev1alpha1.Task{
			{
				Name: "import edpm_nodes_validation",
				ImportRole: ansibleeev1alpha1.ImportRole{
					Name:      "osp.edpm.edpm_nodes_validation",
					TasksFrom: "main.yml",
				},
			},
		},
	}

	err := dataplaneutil.AnsibleExecution(ctx, helper, obj, ValidateNetworkLabel, sshKeySecret, inventoryConfigMap, "", role, aeeSpec)
	if err != nil {
		helper.GetLogger().Error(err, "Unable to execute Ansible for ValidateNetwork")
		return err
	}

	return nil

}

// Install - Networker does not implement this method but is required to satisfy the interface contract
func (n Networker) Install(ctx context.Context, helper *helper.Helper, obj client.Object, sshKeySecret string, inventoryConfigMap string, aeeSpec dataplanev1beta1.AnsibleEESpec) error {
	return fmt.Errorf("Network does not implement the Install method")
}

// Run - Networker does not implement this method but is required to satisfy the interface contract
func (n Networker) Run(ctx context.Context, helper *helper.Helper, obj client.Object, sshKeySecret string, inventoryConfigMap string, aeeSpec dataplanev1beta1.AnsibleEESpec) error {
	return fmt.Errorf("Network does not implement the Run method")
}
