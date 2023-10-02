/*
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

	dataplanev1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	helper "github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"k8s.io/apimachinery/pkg/util/intstr"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
)

// CreateKubeServices creates a service in Kubernetes for each each port
func CreateKubeServices(
	instance *dataplanev1.OpenStackDataPlaneService,
	nodeSet *dataplanev1.OpenStackDataPlaneNodeSet,
	helper *helper.Helper,
	labels map[string]string,
) error {
	log := helper.GetLogger()
	// We create one Service per port and per node, as the Service configuration states that if we
	// just add all the compute nodes IPs to one service, the Service will round-robin between them.
	// Our wanted behaviour is to expose all the compute nodes services at the same time.
	for _, kubeService := range instance.Spec.Services {
		_, err := service(kubeService, instance, helper, labels)
		if err != nil {
			return err
		}

		addresses := make([]string, len(nodeSet.Spec.Nodes))
		i := 0
		for _, item := range nodeSet.Spec.Nodes {
			addresses[i] = item.Ansible.AnsibleHost
			i++
		}

		if len(addresses) == 0 {
			log.Info("There are no addresses in the NodeSet, cannot create Services")
			return nil
		}

		index := 0
		for i := 0; i < len(addresses); i += 100 {
			end := i + 100

			if end > len(addresses) {
				end = len(addresses)
			}

			_, err = endpointSlice(kubeService, instance, addresses[i:end], index, helper, labels)
			if err != nil {
				return err
			}
			index++
		}

	}
	return nil
}

// service creates a service in Kubernetes for the appropiate port
func service(
	kubeService dataplanev1.KubeService,
	instance *dataplanev1.OpenStackDataPlaneService,
	helper *helper.Helper,
	labels map[string]string,
) (*corev1.Service, error) {

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kubeService.Name,
			Namespace: instance.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(context.TODO(), helper.GetClient(), service, func() error {
		labels["kubernetes.io/service-name"] = kubeService.Name
		service.Labels = labels
		service.Spec.Ports = []corev1.ServicePort{{
			Protocol:   "TCP",
			Port:       int32(kubeService.Port),
			TargetPort: intstr.FromInt(kubeService.Port),
		}}

		err := controllerutil.SetControllerReference(instance, service, helper.GetScheme())
		if err != nil {
			return err
		}

		return nil
	})

	return service, err
}

// endpointSlice creates endpointslice in Kubernetes for the appropiate port
func endpointSlice(
	kubeService dataplanev1.KubeService,
	instance *dataplanev1.OpenStackDataPlaneService,
	addresses []string,
	index int,
	helper *helper.Helper,
	labels map[string]string,
) (*discoveryv1.EndpointSlice, error) {

	if len(addresses) > 100 {
		err := fmt.Errorf("an EndpointSlice cannot contain more than 100 endpoint addresses")
		return nil, err
	}

	endpointSlice := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%v", kubeService.Name, index),
			Namespace: instance.Namespace,
		},
	}
	_, err := controllerutil.CreateOrUpdate(context.TODO(), helper.GetClient(), endpointSlice, func() error {
		labels["kubernetes.io/service-name"] = kubeService.Name
		endpointSlice.Labels = labels
		endpointSlice.AddressType = "IPv4"
		appProtocol := kubeService.Protocol
		protocol := corev1.ProtocolTCP
		port := int32(kubeService.Port)
		endpointSlice.Ports = []discoveryv1.EndpointPort{{
			AppProtocol: &appProtocol,
			Protocol:    &protocol,
			Port:        &port,
		}}

		endpointSlice.Endpoints = []discoveryv1.Endpoint{{
			Addresses: addresses,
		}}

		err := controllerutil.SetControllerReference(instance, endpointSlice, helper.GetScheme())
		if err != nil {
			return err
		}

		return nil
	})

	return endpointSlice, err
}
