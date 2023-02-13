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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	"github.com/openstack-k8s-operators/lib-common/modules/storage"
	ansibleeev1alpha1 "github.com/openstack-k8s-operators/openstack-ansibleee-operator/api/v1alpha1"
)

// AnsibleExecution creates a OpenStackAnsiblEE CR
func AnsibleExecution(ctx context.Context, helper *helper.Helper, executionName string, namespace string, sshKeySecret string, inventoryConfigMap string, play string, role ansibleeev1alpha1.Role) error {
	ansibleEE := &ansibleeev1alpha1.OpenStackAnsibleEE{
		ObjectMeta: metav1.ObjectMeta{
			Name:      executionName,
			Namespace: namespace,
		},
	}
	ansibleEE.Spec.Image = "quay.io/openstack-k8s-operators/openstack-ansibleee-runner:latest"
	// TODO(slagle): Handle either play or role being specified
	ansibleEE.Spec.Role = role
	// 	ansibleEE.Spec.Play = `
	// - name: Play
	//   hosts: localhost
	//   gather_facts: no
	//   tasks:
	//     - name: sleep
	//       shell: sleep infinity
	// `

	ansibleEEMounts := storage.VolMounts{}
	sshKeyVolume := corev1.Volume{
		Name: "ssh-key",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: sshKeySecret,
				Items: []corev1.KeyToPath{
					{
						Key:  "private_ssh_key",
						Path: "ssh_key",
					},
				},
			},
		},
	}
	sshKeyMount := corev1.VolumeMount{
		Name:      "ssh-key",
		MountPath: "/runner/env/ssh_key",
		SubPath:   "ssh_key",
	}

	inventoryVolume := corev1.Volume{
		Name: "inventory",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: inventoryConfigMap,
				},
				Items: []corev1.KeyToPath{
					{
						Key:  "inventory",
						Path: "inventory",
					},
				},
			},
		},
	}
	inventoryMount := corev1.VolumeMount{
		Name:      "inventory",
		MountPath: "/runner/inventory/hosts",
		SubPath:   "inventory",
	}

	ansibleEEMounts.Volumes = append(ansibleEEMounts.Volumes, sshKeyVolume)
	ansibleEEMounts.Volumes = append(ansibleEEMounts.Volumes, inventoryVolume)
	ansibleEEMounts.Mounts = append(ansibleEEMounts.Mounts, sshKeyMount)
	ansibleEEMounts.Mounts = append(ansibleEEMounts.Mounts, inventoryMount)

	ansibleEE.Spec.ExtraMounts = append(ansibleEE.Spec.ExtraMounts, ansibleEEMounts)

	err := helper.GetClient().Create(ctx, ansibleEE)
	if err != nil {
		util.LogErrorForObject(helper, err, fmt.Sprintf("Unable to create AnsibleEE %s", ansibleEE.Name), ansibleEE)
		return err
	}

	return nil
}
