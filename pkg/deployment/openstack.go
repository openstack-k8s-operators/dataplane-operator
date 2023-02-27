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

// InstallOpenStack ensures the node OpenStack is installed
func InstallOpenStack(ctx context.Context, helper *helper.Helper, obj client.Object, sshKeySecret string, inventoryConfigMap string, networkAttachments []string) error {

	tasks := []dataplaneutil.Task{
		{
			Name:          "Install edpm_logrotate_crond",
			RoleName:      "edpm_logrotate_crond",
			RoleTasksFrom: "install.yml",
			Tags:          []string{"edpm_logrotate_crond"},
		},
		{
			Name:          "Install edpm_iscsid",
			RoleName:      "edpm_iscsid",
			RoleTasksFrom: "install.yml",
			Tags:          []string{"edpm_iscsid"},
		},
	}
	role := ansibleeev1alpha1.Role{
		Name:           "Deploy EDPM OpenStack Install",
		Hosts:          "all",
		Strategy:       "linear",
		GatherFacts:    false,
		AnyErrorsFatal: true,
		Tasks:          dataplaneutil.PopulateTasks(tasks),
	}

	err := dataplaneutil.AnsibleExecution(ctx, helper, obj, InstallOpenStackLabel, sshKeySecret, inventoryConfigMap, "", role, networkAttachments)
	if err != nil {
		helper.GetLogger().Error(err, "Unable to execute Ansible for InstallOpenStack")
		return err
	}

	return nil

}

// ConfigureOpenStack ensures the node OpenStack config
func ConfigureOpenStack(ctx context.Context, helper *helper.Helper, obj client.Object, sshKeySecret string, inventoryConfigMap string, networkAttachments []string) error {

	tasks := []dataplaneutil.Task{
		{
			Name:          "Configure edpm_ssh_known_hosts",
			RoleName:      "edpm_ssh_known_hosts",
			RoleTasksFrom: "main.yml",
			Tags:          []string{"edpm_ssh_known_hosts"},
		},
		{
			Name:          "Configure edpm_logrotate_crond",
			RoleName:      "edpm_logrotate_crond",
			RoleTasksFrom: "configure.yml",
			Tags:          []string{"edpm_logrotate_crond"},
		},
		{
			Name:          "Configure edpm_iscsid",
			RoleName:      "edpm_iscsid",
			RoleTasksFrom: "configure.yml",
			Tags:          []string{"edpm_iscsid"},
		},
		{
			Name:          "Configure nftables",
			RoleName:      "edpm_nftables",
			RoleTasksFrom: "configure.yml",
			Tags:          []string{"edpm_firewall"},
		},
	}
	role := ansibleeev1alpha1.Role{
		Name:           "Deploy EDPM OpenStack Configure",
		Hosts:          "all",
		Strategy:       "linear",
		GatherFacts:    false,
		AnyErrorsFatal: true,
		Tasks:          dataplaneutil.PopulateTasks(tasks),
	}

	err := dataplaneutil.AnsibleExecution(ctx, helper, obj, ConfigureOpenStackLabel, sshKeySecret, inventoryConfigMap, "", role, networkAttachments)
	if err != nil {
		helper.GetLogger().Error(err, "Unable to execute Ansible for ConfigureOpenStack")
		return err
	}

	return nil

}

// RunOpenStack ensures the node OpenStack is running
func RunOpenStack(ctx context.Context, helper *helper.Helper, obj client.Object, sshKeySecret string, inventoryConfigMap string, networkAttachments []string) error {

	tasks := []dataplaneutil.Task{
		{
			Name:          "Apply nftables configuration",
			RoleName:      "edpm_nftables",
			RoleTasksFrom: "run.yml",
			When:          "deploy_edpm_openstack_run_firewall|default(true)|bool",
			Tags:          []string{"edpm_firewall"},
		},
		{
			Name:          "Run edpm_logrotate_crond",
			RoleName:      "edpm_logrotate_crond",
			RoleTasksFrom: "run.yml",
			Tags:          []string{"edpm_logrotate_crond"},
		},
		{
			Name:          "Run edpm_iscsid",
			RoleName:      "edpm_iscsid",
			RoleTasksFrom: "run.yml",
			Tags:          []string{"edpm_iscsid"},
		},
	}
	role := ansibleeev1alpha1.Role{
		Name:           "Deploy EDPM OpenStack Run",
		Hosts:          "all",
		Strategy:       "linear",
		GatherFacts:    false,
		AnyErrorsFatal: true,
		Tasks:          dataplaneutil.PopulateTasks(tasks),
	}

	err := dataplaneutil.AnsibleExecution(ctx, helper, obj, RunOpenStackLabel, sshKeySecret, inventoryConfigMap, "", role, networkAttachments)
	if err != nil {
		helper.GetLogger().Error(err, "Unable to execute Ansible for RunOpenStack")
		return err
	}

	return nil

}
