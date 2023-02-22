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

	dataplaneutil "github.com/openstack-k8s-operators/dataplane-operator/pkg/util"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	ansibleeev1alpha1 "github.com/openstack-k8s-operators/openstack-ansibleee-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ConfigureOS ensures the node Operating System config
func ConfigureOS(ctx context.Context, helper *helper.Helper, obj client.Object, sshKeySecret string, inventoryConfigMap string) error {

	tasks := []dataplaneutil.Task{
		{
			Name:          "Configure edpm_podman",
			RoleName:      "edpm_podman",
			RoleTasksFrom: "configure.yml",
			Tags:          []string{"edpm_podman"},
		},
		{
			Name:          "Manage edpm container services",
			RoleName:      "edpm_container_manage",
			RoleTasksFrom: "shutdown.yml",
			Tags:          []string{"edpm_container_manage"},
		},
		{
			Name:          "Prepare nftables",
			RoleName:      "edpm_nftables",
			RoleTasksFrom: "service-bootstrap.yml",
			Tags:          []string{"edpm_firewall"},
		},
		{
			Name:          "Configure edpm_sshd",
			RoleName:      "edpm_sshd",
			RoleTasksFrom: "configure.yml",
			Tags:          []string{"edpm_sshd"},
		},
		{
			Name:          "Configure chrony",
			RoleName:      "chrony",
			RoleTasksFrom: "config.yml",
			Tags:          []string{"chrony"},
		},
		{
			Name:          "Configure edpm_timezone",
			RoleName:      "edpm_timezone",
			RoleTasksFrom: "configure.yml",
			Tags:          []string{"edpm_timezone"},
		},
		{
			Name:          "Configure edpm_ovn",
			RoleName:      "edpm_ovn",
			RoleTasksFrom: "configure.yml",
			Tags:          []string{"edpm_ovn"},
		},
	}
	role := ansibleeev1alpha1.Role{
		Name:           "Deploy EDPM Operating System Configure",
		Hosts:          "all",
		Strategy:       "linear",
		GatherFacts:    false,
		Become:         true,
		AnyErrorsFatal: true,
		Tasks:          dataplaneutil.PopulateTasks(tasks),
	}

	err := dataplaneutil.AnsibleExecution(ctx, helper, obj, ConfigureOSLabel, sshKeySecret, inventoryConfigMap, "", role)
	if err != nil {
		helper.GetLogger().Error(err, "Unable to execute Ansible for ConfigureOS")
		return err
	}

	return nil

}
