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
	obj client.Object,
	service *dataplanev1.OpenStackDataPlaneService,
	sshKeySecret string,
	inventorySecret string,
	aeeSpec *dataplanev1.AnsibleEESpec,
) error {
	var err error
	var cmdLineArguments strings.Builder

	ansibleEE, err := GetAnsibleExecution(ctx, helper, obj, service.Spec.Label)
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}
	if ansibleEE == nil {
		var executionName string
		if len(service.Spec.Label) > 0 {
			executionName = fmt.Sprintf("%s-%s", service.Spec.Label, obj.GetName())
		} else {
			executionName = obj.GetName()
		}
		ansibleEE = &ansibleeev1.OpenStackAnsibleEE{
			ObjectMeta: metav1.ObjectMeta{
				Name:      executionName,
				Namespace: obj.GetNamespace(),
				Labels: map[string]string{
					service.Spec.Label: string(obj.GetUID()),
					"osdpd":            obj.GetName(),
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
				Secret: &corev1.SecretVolumeSource{
					SecretName: inventorySecret,
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
func GetAnsibleExecution(ctx context.Context, helper *helper.Helper, obj client.Object, label string) (*ansibleeev1.OpenStackAnsibleEE, error) {
	var err error
	ansibleEEs := &ansibleeev1.OpenStackAnsibleEEList{}

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

	var ansibleEE *ansibleeev1.OpenStackAnsibleEE
	if len(ansibleEEs.Items) == 0 {
		return nil, k8serrors.NewNotFound(appsv1.Resource("OpenStackAnsibleEE"), fmt.Sprintf("with label %s=%s", label, obj.GetUID()))
	} else if len(ansibleEEs.Items) == 1 {
		ansibleEE = &ansibleEEs.Items[0]
	} else {
		return nil, fmt.Errorf("multiple OpenStackAnsibleEE's found with label %s=%s", label, obj.GetUID())
	}

	return ansibleEE, nil
}
