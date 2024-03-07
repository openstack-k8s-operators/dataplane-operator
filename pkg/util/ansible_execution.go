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

package util

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	dataplanev1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	"github.com/openstack-k8s-operators/lib-common/modules/storage"
	ansibleeev1 "github.com/openstack-k8s-operators/openstack-ansibleee-operator/api/v1beta1"
)

// AnsibleExecution creates a OpenStackAnsiblEE CR
func AnsibleExecution(
	ctx context.Context,
	helper *helper.Helper,
	openStackDeployment client.Object,
	service *dataplanev1.OpenStackDataPlaneService,
	sshKeySecrets map[string]string,
	inventorySecrets map[string]string,
	aeeSpec *dataplanev1.AnsibleEESpec,
	targetNodeSet string,
) error {
	var err error
	var cmdLineArguments strings.Builder
	var inventoryVolume corev1.Volume
	var inventoryName string
	var inventoryMountPath string
	var sshKeyName string
	var sshKeyMountPath string
	var sshKeyMountSubPath string

	ansibleEEMounts := storage.VolMounts{}

	ansibleEE, err := GetAnsibleExecution(ctx, helper, openStackDeployment, service.Name, targetNodeSet)
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}
	if ansibleEE == nil {

		executionName := fmt.Sprintf("%s-%s-%s-", GetAnsibleExecutionNamePrefix(service), openStackDeployment.GetName(), targetNodeSet)
		ansibleEE = &ansibleeev1.OpenStackAnsibleEE{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: executionName,
				Namespace:    openStackDeployment.GetNamespace(),
				Labels: map[string]string{
					"openstackdataplaneservice":    service.Name,
					"openstackdataplanedeployment": openStackDeployment.GetName(),
					"openstackdataplanenodeset":    targetNodeSet,
				},
			},
		}
	}

	_, err = controllerutil.CreateOrPatch(ctx, helper.GetClient(), ansibleEE, func() error {
		ansibleEE.Spec.NetworkAttachments = aeeSpec.NetworkAttachments
		if aeeSpec.DNSConfig != nil {
			ansibleEE.Spec.DNSConfig = aeeSpec.DNSConfig
		}
		if len(aeeSpec.OpenStackAnsibleEERunnerImage) > 0 {
			ansibleEE.Spec.Image = aeeSpec.OpenStackAnsibleEERunnerImage
		}
		if len(aeeSpec.ExtraVars) > 0 {
			ansibleEE.Spec.ExtraVars = aeeSpec.ExtraVars
		}
		if len(aeeSpec.AnsibleTags) > 0 {
			fmt.Fprintf(&cmdLineArguments, "--tags %s ", aeeSpec.AnsibleTags)
		}
		if len(aeeSpec.AnsibleLimit) > 0 {
			fmt.Fprintf(&cmdLineArguments, "--limit %s ", aeeSpec.AnsibleLimit)
		}
		if len(aeeSpec.AnsibleSkipTags) > 0 {
			fmt.Fprintf(&cmdLineArguments, "--skip-tags %s ", aeeSpec.AnsibleSkipTags)
		}
		if cmdLineArguments.Len() > 0 {
			ansibleEE.Spec.CmdLine = strings.TrimSpace(cmdLineArguments.String())
		}

		if len(service.Spec.Play) > 0 {
			ansibleEE.Spec.Play = service.Spec.Play
		}
		if len(service.Spec.Playbook) > 0 {
			ansibleEE.Spec.Playbook = service.Spec.Playbook
		}

		// If we have a service that ought to be deployed everywhere
		// substitute the existing play target with 'all'
		// Check if we have ExtraVars before accessing it
		if ansibleEE.Spec.ExtraVars == nil {
			ansibleEE.Spec.ExtraVars = make(map[string]json.RawMessage)
		}
		if service.Spec.DeployOnAllNodeSets {
			ansibleEE.Spec.ExtraVars["edpm_override_hosts"] = json.RawMessage([]byte("\"all\""))
			util.LogForObject(helper, fmt.Sprintf("for service %s, substituting existing ansible play host with 'all'.", service.Name), ansibleEE)
		} else {
			ansibleEE.Spec.ExtraVars["edpm_override_hosts"] = json.RawMessage([]byte(fmt.Sprintf("\"%s\"", targetNodeSet)))
			util.LogForObject(helper, fmt.Sprintf("for service %s, substituting existing ansible play host with '%s'.", service.Name, targetNodeSet), ansibleEE)
		}

		for sshKeyNodeName, sshKeySecret := range sshKeySecrets {
			if service.Spec.DeployOnAllNodeSets {
				sshKeyName = fmt.Sprintf("ssh-key-%s", sshKeyNodeName)
				sshKeyMountSubPath = fmt.Sprintf("ssh_key_%s", sshKeyNodeName)
				sshKeyMountPath = fmt.Sprintf("/runner/env/ssh_key/%s", sshKeyMountSubPath)
			} else {
				if sshKeyNodeName != targetNodeSet {
					continue
				}
				sshKeyName = "ssh-key"
				sshKeyMountSubPath = "ssh_key"
				sshKeyMountPath = "/runner/env/ssh_key"
			}
			sshKeyVolume := corev1.Volume{
				Name: sshKeyName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: sshKeySecret,
						Items: []corev1.KeyToPath{
							{
								Key:  "ssh-privatekey",
								Path: sshKeyMountSubPath,
							},
						},
					},
				},
			}
			sshKeyMount := corev1.VolumeMount{
				Name:      sshKeyName,
				MountPath: sshKeyMountPath,
				SubPath:   sshKeyMountSubPath,
			}
			// Mount ssh secrets
			ansibleEEMounts.Mounts = append(ansibleEEMounts.Mounts, sshKeyMount)
			ansibleEEMounts.Volumes = append(ansibleEEMounts.Volumes, sshKeyVolume)
		}

		// order the inventory keys otherwise it could lead to changing order and mount order changing
		invKeys := make([]string, 0)
		for k := range inventorySecrets {
			invKeys = append(invKeys, k)
		}
		sort.Strings(invKeys)

		// Mounting inventory and secrets
		for inventoryIndex, nodeName := range invKeys {
			if service.Spec.DeployOnAllNodeSets {
				inventoryName = fmt.Sprintf("inventory-%d", inventoryIndex)
				inventoryMountPath = fmt.Sprintf("/runner/inventory/%s", inventoryName)
			} else {
				if nodeName != targetNodeSet {
					continue
				}
				inventoryName = "inventory"
				inventoryMountPath = "/runner/inventory/hosts"
			}

			inventoryVolume = corev1.Volume{
				Name: inventoryName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: inventorySecrets[nodeName],
						Items: []corev1.KeyToPath{
							{
								Key:  "inventory",
								Path: inventoryName,
							},
						},
					},
				},
			}
			inventoryMount := corev1.VolumeMount{
				Name:      inventoryName,
				MountPath: inventoryMountPath,
				SubPath:   inventoryName,
			}
			// Inventory mount
			ansibleEEMounts.Mounts = append(ansibleEEMounts.Mounts, inventoryMount)
			ansibleEEMounts.Volumes = append(ansibleEEMounts.Volumes, inventoryVolume)
		}

		ansibleEE.Spec.ExtraMounts = append(aeeSpec.ExtraMounts, []storage.VolMounts{ansibleEEMounts}...)
		ansibleEE.Spec.Env = aeeSpec.Env

		err := controllerutil.SetControllerReference(openStackDeployment, ansibleEE, helper.GetScheme())
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		util.LogErrorForObject(helper, err, fmt.Sprintf("Unable to create AnsibleEE %s", ansibleEE.Name), ansibleEE)
		return err
	}
	util.LogForObject(helper, fmt.Sprintf("AnsibleEE %s created", ansibleEE.Name), ansibleEE)
	return nil
}

// GetAnsibleExecution gets and returns an OpenStackAnsibleEE with given
// labels:
// openstackdataplaneservice:    service-name,
// openstackdataplanedeployment: deployment-name,
// openstackdataplanenodeset:    nodeset-name,
// If multiple or no resources are found, return nil and error
func GetAnsibleExecution(ctx context.Context, helper *helper.Helper, openStackDeployment client.Object, openStackService string, targetNodeSet string) (*ansibleeev1.OpenStackAnsibleEE, error) {
	var err error
	ansibleEEs := &ansibleeev1.OpenStackAnsibleEEList{}

	listOpts := []client.ListOption{
		client.InNamespace(openStackDeployment.GetNamespace()),
	}
	labelSelector := map[string]string{
		"openstackdataplaneservice":    openStackService,
		"openstackdataplanedeployment": openStackDeployment.GetName(),
		"openstackdataplanenodeset":    targetNodeSet,
	}
	if len(labelSelector) > 0 {
		labels := client.MatchingLabels(labelSelector)
		listOpts = append(listOpts, labels)
	}
	err = helper.GetClient().List(ctx, ansibleEEs, listOpts...)
	if err != nil {
		return nil, err
	}

	var ansibleEE *ansibleeev1.OpenStackAnsibleEE
	if len(ansibleEEs.Items) == 0 {
		return nil, k8serrors.NewNotFound(appsv1.Resource("OpenStackAnsibleEE"), fmt.Sprintf("with labels %v", labelSelector))
	} else if len(ansibleEEs.Items) == 1 {
		ansibleEE = &ansibleEEs.Items[0]
	} else {
		return nil, fmt.Errorf("multiple OpenStackAnsibleEE's found with labels %v", labelSelector)
	}

	return ansibleEE, nil
}

// GetAnsibleExecutionNamePrefix compute the name of the AnsibleEE
func GetAnsibleExecutionNamePrefix(service *dataplanev1.OpenStackDataPlaneService) string {
	var executionNamePrefix string
	if len(service.Name) > AnsibleExecutionServiceNameLen {
		executionNamePrefix = service.Name[:AnsibleExecutionServiceNameLen]
	} else {
		executionNamePrefix = service.Name
	}
	return executionNamePrefix
}
