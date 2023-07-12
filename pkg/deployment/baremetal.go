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
	"net"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	dataplanev1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	infranetworkv1 "github.com/openstack-k8s-operators/infra-operator/apis/network/v1beta1"
	condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	utils "github.com/openstack-k8s-operators/lib-common/modules/common/util"
	baremetalv1 "github.com/openstack-k8s-operators/openstack-baremetal-operator/api/v1beta1"
)

// ManagedHostMap defines a struct to hold our HostMap. This HostMap is a map
// of NodeSet names and Baremetalv1 InstanceSpec.
type ManagedHostMap struct {
	HostMap map[string]map[string]baremetalv1.InstanceSpec
}

// DeployBaremetalSet Deploy OpenStackBaremetalSet
func DeployBaremetalSet(
	ctx context.Context, helper *helper.Helper, instance *dataplanev1.OpenStackDataPlaneNodeSet,
	ipSets map[string]infranetworkv1.IPSet,
	dnsAddresses []string,
) (bool, error) {
	baremetalSet := &baremetalv1.OpenStackBaremetalSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
		},
	}

	utils.LogForObject(helper, "Reconciling BaremetalSet", instance)
	_, err := controllerutil.CreateOrPatch(ctx, helper.GetClient(), baremetalSet, func() error {
		instance.Spec.BaremetalSetTemplate.DeepCopyInto(&baremetalSet.Spec)
		for nodeName, node := range instance.Spec.NodeTemplate.Nodes {
			hostName := node.HostName
			ipSet, ok := ipSets[nodeName]
			instanceSpec := baremetalSet.Spec.BaremetalHosts[hostName]
			if !ok {
				utils.LogForObject(helper, "IPAM Not configured for use, skipping", instance)
				instanceSpec.CtlPlaneIP = node.Ansible.AnsibleHost
			} else {
				for _, res := range ipSet.Status.Reservation {
					if res.Network == CtlPlaneNetwork {
						instanceSpec.CtlPlaneIP = res.Address
						baremetalSet.Spec.CtlplaneGateway = *res.Gateway
						baremetalSet.Spec.BootstrapDNS = dnsAddresses
						baremetalSet.Spec.DNSSearchDomains = []string{res.DNSDomain}
						_, ipNet, err := net.ParseCIDR(res.Cidr)
						if err == nil {
							baremetalSet.Spec.CtlplaneNetmask = net.IP(ipNet.Mask).String()
						}
					}
				}
			}
			baremetalSet.Spec.BaremetalHosts[hostName] = instanceSpec

		}
		err := controllerutil.SetControllerReference(
			helper.GetBeforeObject(), baremetalSet, helper.GetScheme())
		return err
	})

	if err != nil {
		instance.Status.Conditions.MarkFalse(
			dataplanev1.NodeSetBareMetalProvisionReadyCondition,
			condition.ErrorReason, condition.SeverityError,
			dataplanev1.NodeSetBaremetalProvisionErrorMessage)
		return false, err
	}

	// Check if baremetalSet is ready
	if !baremetalSet.IsReady() {
		utils.LogForObject(helper, "BaremetalSet not ready, waiting...", instance)
		instance.Status.Conditions.MarkFalse(
			dataplanev1.NodeSetBareMetalProvisionReadyCondition,
			condition.RequestedReason, condition.SeverityInfo,
			dataplanev1.NodeSetBaremetalProvisionReadyWaitingMessage)
		return false, nil
	}
	instance.Status.Conditions.MarkTrue(
		dataplanev1.NodeSetBareMetalProvisionReadyCondition,
		dataplanev1.NodeSetBaremetalProvisionReadyMessage)
	return true, nil
}

// BuildBMHHostMap  Build managed host map for all roles
func BuildBMHHostMap(ctx context.Context, helper *helper.Helper,
	instance *dataplanev1.OpenStackDataPlaneNodeSet,
	nodeSetManagedHostMap *ManagedHostMap) error {
	for _, node := range instance.Spec.NodeTemplate.Nodes {
		labels := instance.GetObjectMeta().GetLabels()
		nodeSetName, ok := labels["openstackdataplane"]
		if !ok {
			// Node does not have a label
			continue
		}
		if nodeSetManagedHostMap.HostMap == nil {
			nodeSetManagedHostMap.HostMap = make(map[string]map[string]baremetalv1.InstanceSpec)
		}
		if nodeSetManagedHostMap.HostMap[nodeSetName] == nil {
			nodeSetManagedHostMap.HostMap[nodeSetName] = make(map[string]baremetalv1.InstanceSpec)
		}

		if !instance.Spec.PreProvisioned {
			instanceSpec := baremetalv1.InstanceSpec{}
			instanceSpec.UserData = node.UserData
			instanceSpec.NetworkData = node.NetworkData
			nodeSetManagedHostMap.HostMap[nodeSetName][node.HostName] = instanceSpec
		}
	}
	return nil
}
