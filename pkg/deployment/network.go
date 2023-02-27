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

	dataplaneutil "github.com/openstack-k8s-operators/dataplane-operator/pkg/util"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	ansibleeev1alpha1 "github.com/openstack-k8s-operators/openstack-ansibleee-operator/api/v1alpha1"
)

// ConfigureNetwork ensures the network config
func ConfigureNetwork(ctx context.Context, helper *helper.Helper, obj client.Object, sshKeySecret string, inventoryConfigMap string, networkAttachments []string, openStackAnsibleEERunnerImage string) error {

	role := ansibleeev1alpha1.Role{
		Name:     "edpm_network_config",
		Hosts:    "all",
		Strategy: "linear",
		Tasks: []ansibleeev1alpha1.Task{
			{
				Name: "import edpm_network_config",
				ImportRole: ansibleeev1alpha1.ImportRole{
					Name:      "edpm_network_config",
					TasksFrom: "main.yml",
				},
			},
		},
	}

	err := dataplaneutil.AnsibleExecution(ctx, helper, obj, ConfigureNetworkLabel, sshKeySecret, inventoryConfigMap, "", role, networkAttachments, openStackAnsibleEERunnerImage)
	if err != nil {
		helper.GetLogger().Error(err, "Unable to execute Ansible for ConfigureNetwork")
		return err
	}

	return nil

}

// ValidateNetwork ensures the node network config
func ValidateNetwork(ctx context.Context, helper *helper.Helper, obj client.Object, sshKeySecret string, inventoryConfigMap string, networkAttachments []string, openStackAnsibleEERunnerImage string) error {

	role := ansibleeev1alpha1.Role{
		Name:     "edpm_nodes_validation",
		Hosts:    "all",
		Strategy: "linear",
		Tasks: []ansibleeev1alpha1.Task{
			{
				Name: "import edpm_nodes_validation",
				ImportRole: ansibleeev1alpha1.ImportRole{
					Name:      "edpm_nodes_validation",
					TasksFrom: "main.yml",
				},
			},
		},
	}

	err := dataplaneutil.AnsibleExecution(ctx, helper, obj, ValidateNetworkLabel, sshKeySecret, inventoryConfigMap, "", role, networkAttachments, openStackAnsibleEERunnerImage)
	if err != nil {
		helper.GetLogger().Error(err, "Unable to execute Ansible for ValidateNetwork")
		return err
	}

	return nil

}
