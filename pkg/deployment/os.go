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

	dataplanev1beta1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	dataplaneutil "github.com/openstack-k8s-operators/dataplane-operator/pkg/util"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	ansibleeev1alpha1 "github.com/openstack-k8s-operators/openstack-ansibleee-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// InstallOS ensures the node Operating System is installed
func InstallOS(ctx context.Context, helper *helper.Helper, obj client.Object, sshKeySecret string, inventoryConfigMap string, aeeSpec dataplanev1beta1.AnsibleEESpec) error {

	tasks := []dataplaneutil.Task{
		{
			Name:          "Install edpm_bootstrap",
			RoleName:      "osp.edpm.edpm_bootstrap",
			RoleTasksFrom: "bootstrap.yml",
			Tags:          []string{"edpm_bootstrap"},
		},
		{
			Name:          "Install edpm_kernel",
			RoleName:      "osp.edpm.edpm_kernel",
			RoleTasksFrom: "main.yml",
			Tags:          []string{"edpm_kernel"},
		},
		{
			Name:          "Install edpm_podman",
			RoleName:      "osp.edpm.edpm_podman",
			RoleTasksFrom: "install.yml",
			Tags:          []string{"edpm_podman"},
		},
		{
			Name:          "Install edpm_sshd",
			RoleName:      "osp.edpm.edpm_sshd",
			RoleTasksFrom: "install.yml",
			Tags:          []string{"edpm_sshd"},
		},
		{
			Name:          "Install edpm_chrony",
			RoleName:      "osp.edpm.edpm_chrony",
			RoleTasksFrom: "install.yml",
			Tags:          []string{"edpm_chrony"},
		},
		{
			Name:          "Install edpm_ovn",
			RoleName:      "osp.edpm.edpm_ovn",
			RoleTasksFrom: "install.yml",
			Tags:          []string{"edpm_ovn"},
		},
	}
	role := ansibleeev1alpha1.Role{
		Name:           "Deploy EDPM Operating System Install",
		Hosts:          "all",
		Strategy:       "linear",
		GatherFacts:    false,
		Become:         true,
		AnyErrorsFatal: true,
		Tasks:          dataplaneutil.PopulateTasks(tasks),
	}

	err := dataplaneutil.AnsibleExecution(ctx, helper, obj, InstallOSLabel, sshKeySecret, inventoryConfigMap, "", role, aeeSpec)
	if err != nil {
		helper.GetLogger().Error(err, "Unable to execute Ansible for InstallOS")
		return err
	}

	return nil

}

// ConfigureOS ensures the node Operating System config
func ConfigureOS(ctx context.Context, helper *helper.Helper, obj client.Object, sshKeySecret string, inventoryConfigMap string, aeeSpec dataplanev1beta1.AnsibleEESpec) error {

	tasks := []dataplaneutil.Task{
		{
			Name:          "Configure edpm_podman",
			RoleName:      "osp.edpm.edpm_podman",
			RoleTasksFrom: "configure.yml",
			Tags:          []string{"edpm_podman"},
		},
		{
			Name:          "Manage edpm container services",
			RoleName:      "osp.edpm.edpm_container_manage",
			RoleTasksFrom: "shutdown.yml",
			Tags:          []string{"edpm_container_manage"},
		},
		{
			Name:          "Prepare nftables",
			RoleName:      "osp.edpm.edpm_nftables",
			RoleTasksFrom: "service-bootstrap.yml",
			Tags:          []string{"edpm_firewall"},
		},
		{
			Name:          "Configure edpm_sshd",
			RoleName:      "osp.edpm.edpm_sshd",
			RoleTasksFrom: "configure.yml",
			Tags:          []string{"edpm_sshd"},
		},
		{
			Name:          "Configure edpm_chrony",
			RoleName:      "osp.edpm.edpm_chrony",
			RoleTasksFrom: "config.yml",
			Tags:          []string{"edpm_chrony"},
		},
		{
			Name:          "Configure edpm_timezone",
			RoleName:      "osp.edpm.edpm_timezone",
			RoleTasksFrom: "configure.yml",
			Tags:          []string{"edpm_timezone"},
		},
		{
			Name:          "Configure edpm_ovn",
			RoleName:      "osp.edpm.edpm_ovn",
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

	err := dataplaneutil.AnsibleExecution(ctx, helper, obj, ConfigureOSLabel, sshKeySecret, inventoryConfigMap, "", role, aeeSpec)
	if err != nil {
		helper.GetLogger().Error(err, "Unable to execute Ansible for ConfigureOS")
		return err
	}

	return nil

}

// RunOS ensures the node Operating System is running
func RunOS(ctx context.Context, helper *helper.Helper, obj client.Object, sshKeySecret string, inventoryConfigMap string, aeeSpec dataplanev1beta1.AnsibleEESpec) error {

	tasks := []dataplaneutil.Task{
		{
			Name:          "Run edpm_sshd",
			RoleName:      "osp.edpm.edpm_sshd",
			RoleTasksFrom: "run.yml",
			Tags:          []string{"edpm_sshd"},
		},
		{
			Name:          "Run edpm_chrony",
			RoleName:      "osp.edpm.edpm_chrony",
			RoleTasksFrom: "run.yml",
			Tags:          []string{"edpm_chrony"},
		},
		{
			Name:          "Run edpm_chrony (online)",
			RoleName:      "osp.edpm.edpm_chrony",
			RoleTasksFrom: "online.yml",
			Tags:          []string{"edpm_chrony"},
		},
		{
			Name:          "Run edpm_chrony (sync)",
			RoleName:      "osp.edpm.edpm_chrony",
			RoleTasksFrom: "sync.yml",
			Tags:          []string{"edpm_chrony"},
		},
		{
			Name:          "Run edpm_timezone",
			RoleName:      "osp.edpm.edpm_timezone",
			RoleTasksFrom: "run.yml",
			Tags:          []string{"edpm_timezone"},
		},
		{
			Name:          "Run edpm_ovn",
			RoleName:      "osp.edpm.edpm_ovn",
			RoleTasksFrom: "run.yml",
			Tags:          []string{"edpm_ovn"},
		},
	}
	role := ansibleeev1alpha1.Role{
		Name:           "Deploy EDPM Operating System Run",
		Hosts:          "all",
		Strategy:       "linear",
		GatherFacts:    false,
		Become:         true,
		AnyErrorsFatal: true,
		Tasks:          dataplaneutil.PopulateTasks(tasks),
	}

	err := dataplaneutil.AnsibleExecution(ctx, helper, obj, RunOSLabel, sshKeySecret, inventoryConfigMap, "", role, aeeSpec)
	if err != nil {
		helper.GetLogger().Error(err, "Unable to execute Ansible for RunOS")
		return err
	}

	return nil

}
