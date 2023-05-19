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
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	dataplanev1beta1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	"github.com/openstack-k8s-operators/lib-common/modules/storage"
	ansibleeev1alpha1 "github.com/openstack-k8s-operators/openstack-ansibleee-operator/api/v1alpha1"
)

// AnsibleExecution creates a OpenStackAnsiblEE CR
func AnsibleExecution(
	ctx context.Context,
	helper *helper.Helper,
	obj client.Object,
	label string,
	sshKeySecret string,
	inventoryConfigMap string,
	play string,
	role ansibleeev1alpha1.Role,
	aeeSpec dataplanev1beta1.AnsibleEESpec,
) error {

	var err error
	var cmdLineArguments strings.Builder

	ansibleEE, err := GetAnsibleExecution(ctx, helper, obj, label)
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}
	if ansibleEE == nil {
		executionName := fmt.Sprintf("%s-%s", label, obj.GetName())
		ansibleEE = &ansibleeev1alpha1.OpenStackAnsibleEE{
			ObjectMeta: metav1.ObjectMeta{
				Name:      executionName,
				Namespace: obj.GetNamespace(),
				Labels: map[string]string{
					label:              string(obj.GetUID()),
					"deployIdentifier": aeeSpec.DeployIdentifier,
				},
			},
		}
	}

	_, err = controllerutil.CreateOrPatch(ctx, helper.GetClient(), ansibleEE, func() error {
		ansibleEE.Spec.Image = aeeSpec.OpenStackAnsibleEERunnerImage
		ansibleEE.Spec.NetworkAttachments = aeeSpec.NetworkAttachments
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

		if len(play) > 0 {
			ansibleEE.Spec.Play = play
		}
		ansibleEE.Spec.Role = &role

		ansibleEEMounts := storage.VolMounts{}
		sshKeyVolume := corev1.Volume{
			Name: "ssh-key",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: sshKeySecret,
					Items: []corev1.KeyToPath{
						{
							Key:  "ssh-privatekey",
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
						{
							Key:  "network",
							Path: "network",
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
		networkConfigMount := corev1.VolumeMount{
			Name:      "inventory",
			MountPath: "/runner/network/nic-config-template",
			SubPath:   "network",
		}

		ansibleEEMounts.Volumes = append(ansibleEEMounts.Volumes, sshKeyVolume)
		ansibleEEMounts.Volumes = append(ansibleEEMounts.Volumes, inventoryVolume)
		ansibleEEMounts.Mounts = append(ansibleEEMounts.Mounts, sshKeyMount)
		ansibleEEMounts.Mounts = append(ansibleEEMounts.Mounts, inventoryMount)
		ansibleEEMounts.Mounts = append(ansibleEEMounts.Mounts, networkConfigMount)

		ansibleEE.Spec.ExtraMounts = append(aeeSpec.ExtraMounts, []storage.VolMounts{ansibleEEMounts}...)
		ansibleEE.Spec.Env = aeeSpec.Env

		err := controllerutil.SetControllerReference(obj, ansibleEE, helper.GetScheme())
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

// GetAnsibleExecution gets and returns an OpenStackAnsibleEE with the given
// label where <label>=<node UID>
// If none is found, return nil
func GetAnsibleExecution(ctx context.Context, helper *helper.Helper, obj client.Object, label string) (*ansibleeev1alpha1.OpenStackAnsibleEE, error) {

	var err error
	ansibleEEs := &ansibleeev1alpha1.OpenStackAnsibleEEList{}

	listOpts := []client.ListOption{
		client.InNamespace(obj.GetNamespace()),
	}
	labelSelector := map[string]string{
		label: string(obj.GetUID()),
	}
	if len(labelSelector) > 0 {
		labels := client.MatchingLabels(labelSelector)
		listOpts = append(listOpts, labels)
	}
	err = helper.GetClient().List(ctx, ansibleEEs, listOpts...)
	if err != nil {
		return nil, err
	}

	var ansibleEE *ansibleeev1alpha1.OpenStackAnsibleEE
	if len(ansibleEEs.Items) == 0 {
		return nil, k8serrors.NewNotFound(appsv1.Resource("OpenStackAnsibleEE"), fmt.Sprintf("with label %s=%s", label, obj.GetUID()))
	} else if len(ansibleEEs.Items) == 1 {
		ansibleEE = &ansibleEEs.Items[0]
	} else {
		return nil, fmt.Errorf("Multiple OpenStackAnsibleEE's found with label %s=%s", label, obj.GetUID())
	}

	return ansibleEE, nil

}

// GetAnsibleExecutionJob returns OpenStackAnsibleEE Job for given label
func GetAnsibleExecutionJob(ctx context.Context, helper *helper.Helper, obj client.Object, label string) (*batchv1.Job, error) {

	foundJob := &batchv1.Job{}
	ansibleEE, err := GetAnsibleExecution(ctx, helper, obj, label)
	if err != nil {
		return foundJob, err
	} else if ansibleEE == nil {
		return foundJob, fmt.Errorf("OpenStackAnsibleEE not found with label %s=%s", label, obj.GetUID())
	}

	err = helper.GetClient().Get(ctx, types.NamespacedName{Name: ansibleEE.Name, Namespace: ansibleEE.Namespace}, foundJob)
	if err != nil {
		return foundJob, err
	}

	return foundJob, nil
}
