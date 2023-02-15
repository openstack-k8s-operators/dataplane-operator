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
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/storage/names"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	"github.com/openstack-k8s-operators/lib-common/modules/storage"
	ansibleeev1alpha1 "github.com/openstack-k8s-operators/openstack-ansibleee-operator/api/v1alpha1"
)

// AnsibleExecution creates a OpenStackAnsiblEE CR
func AnsibleExecution(instance client.Object, ctx context.Context, helper *helper.Helper, label string, sshKeySecret string, inventoryConfigMap string, play string, role ansibleeev1alpha1.Role) error {

	var err error
	ansibleEEs := &ansibleeev1alpha1.OpenStackAnsibleEEList{}

	listOpts := []client.ListOption{
		client.InNamespace(instance.GetNamespace()),
	}
	labelSelector := map[string]string{
		label: instance.GetName(),
	}
	if len(labelSelector) > 0 {
		labels := client.MatchingLabels(labelSelector)
		listOpts = append(listOpts, labels)
	}
	err = helper.GetClient().List(ctx, ansibleEEs, listOpts...)
	if err != nil {
		return err
	}

	var ansibleEE *ansibleeev1alpha1.OpenStackAnsibleEE
	if len(ansibleEEs.Items) == 0 {
		executionName := names.SimpleNameGenerator.GenerateName(label)
		ansibleEE = &ansibleeev1alpha1.OpenStackAnsibleEE{
			ObjectMeta: metav1.ObjectMeta{
				Name:      executionName,
				Namespace: instance.GetNamespace(),
				Labels: map[string]string{
					label: instance.GetName(),
				},
			},
		}
	} else if len(ansibleEEs.Items) == 1 {
		ansibleEE = &ansibleEEs.Items[0]
	} else {
		return errors.New(fmt.Sprintf("Multiple OpenStackAnsibleEE's found with label %s", label))
	}

	_, err = controllerutil.CreateOrPatch(ctx, helper.GetClient(), ansibleEE, func() error {
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

		ansibleEE.Spec.ExtraMounts = []storage.VolMounts{ansibleEEMounts}

		err := controllerutil.SetControllerReference(instance, ansibleEE, helper.GetScheme())
		if err != nil {
			return err
		}

		return nil

	})

	if err != nil {
		util.LogErrorForObject(helper, err, fmt.Sprintf("Unable to create AnsibleEE %s", ansibleEE.Name), ansibleEE)
		return err
	}

	return nil
}
