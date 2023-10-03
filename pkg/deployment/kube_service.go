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
	"net"
	"regexp"

	dataplanev1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	infranetworkv1 "github.com/openstack-k8s-operators/infra-operator/apis/network/v1beta1"
	helper "github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
	// We create only one KubeService per port. All the nodes will be IPs on the endpointslices
	// This will round-robin requests to the nodes, but it is also useful for Prometheus configuration
	for _, kubeService := range instance.Spec.Services {
		_, err := service(kubeService, instance, helper, labels)
		if err != nil {
			return err
		}

		var addressType discoveryv1.AddressType
		addresses := make([]string, len(nodeSet.Spec.Nodes))
		i := 0
		for name, item := range nodeSet.Spec.Nodes {
			namespacedName := &types.NamespacedName{
				Name:      name,
				Namespace: instance.GetNamespace(),
			}
			if len(item.Ansible.AnsibleHost) > 0 {
				addresses[i], addressType = getAddressFromAnsibleHost(&item, namespacedName, helper)
			} else {
				addresses[i], addressType = getAddressFromIPSet(&item, namespacedName, helper)
			}
			if addresses[i] == "" {
				// we were unable to find an IP or HostName for a node, so we do not go further
				return nil
			}
			i++
		}

		index := 0
		for i := 0; i < len(addresses); i += 100 {
			end := i + 100

			if end > len(addresses) {
				end = len(addresses)
			}

			_, err = endpointSlice(kubeService, instance, addresses[i:end], &addressType, index, helper, labels)
			if err != nil {
				return err
			}
			index++
		}

	}
	return nil
}

func getAddressFromIPSet(item *dataplanev1.NodeSection,
	namespacedName *types.NamespacedName,
	helper *helper.Helper,
) (string, discoveryv1.AddressType) {
	// we go search for an IPSet
	ipset := &infranetworkv1.IPSet{}
	err := helper.GetClient().Get(context.Background(), *namespacedName, ipset)
	if err != nil {
		// No IPsets found, lets try to get the HostName as last resource
		if isValidDomain(item.HostName) {
			return item.HostName, discoveryv1.AddressTypeFQDN
		}
		// No IP address or valid hostname found anywhere
		helper.GetLogger().Info("Did not found a valid hostname or IP address")
		return "", ""
	}
	// check that the reservations list is not empty
	if len(ipset.Status.Reservation) > 0 {
		// use the first IP in the reservations list. This is CtlPlane network.
		return ipset.Status.Reservation[0].Address, discoveryv1.AddressTypeIPv4
	}
	// if the reservations list is empty, we go find if HostName is a valid domain
	if isValidDomain(item.HostName) {
		return item.HostName, discoveryv1.AddressTypeFQDN
	}
	// No IP address or valid hostname found anywhere
	helper.GetLogger().Info("Did not found a valid hostname or IP address")
	return "", ""
}

func getAddressFromAnsibleHost(
	item *dataplanev1.NodeSection,
	namespacedName *types.NamespacedName,
	helper *helper.Helper,
) (string, discoveryv1.AddressType) {
	// check if ansiblehost is an IP
	addr := net.ParseIP(item.Ansible.AnsibleHost)
	if addr != nil {
		// it is an ip
		return item.Ansible.AnsibleHost, discoveryv1.AddressTypeIPv4
	}
	// it is not an ip, is it a valid hostname?
	if isValidDomain(item.Ansible.AnsibleHost) {
		// it is an valid domain name
		return item.Ansible.AnsibleHost, discoveryv1.AddressTypeFQDN
	}
	if isValidDomain(item.HostName) {
		// Maybe we find a valid domain name in HostName
		return item.HostName, discoveryv1.AddressTypeFQDN
	}
	// we did not find anything on the NodeSet, lets search in the IPSets
	return getAddressFromIPSet(item, namespacedName, helper)
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
	addressType *discoveryv1.AddressType,
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
		endpointSlice.AddressType = *addressType
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

// isValidDomain returns true if the domain is valid.
func isValidDomain(domain string) bool {
	var domainRegexp = regexp.MustCompile(`^(?i)[a-z0-9-]+(\.[a-z0-9-]+)+\.?$`)
	return domainRegexp.MatchString(domain)
}
