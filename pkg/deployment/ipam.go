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
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	dataplanev1beta1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	infranetworkv1 "github.com/openstack-k8s-operators/infra-operator/apis/network/v1beta1"
	condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
)

// EnsureIPSets Creates the IPSets
func EnsureIPSets(ctx context.Context, helper *helper.Helper,
	instance *dataplanev1beta1.OpenStackDataPlaneRole,
	nodes *dataplanev1beta1.OpenStackDataPlaneNodeList) (map[string]infranetworkv1.IPSet, bool, error) {
	allIPSets, err := reserveIPs(ctx, helper, instance, nodes)
	if err != nil {
		instance.Status.Conditions.MarkFalse(
			dataplanev1beta1.RoleIPReservationReadyCondition,
			condition.ErrorReason, condition.SeverityError,
			dataplanev1beta1.RoleIPReservationReadyErrorMessage)
		return nil, false, err
	}

	if len(allIPSets) == 0 {
		return nil, true, nil
	}

	for _, s := range allIPSets {
		if s.Status.Conditions.IsFalse(condition.ReadyCondition) {
			return nil, false, nil
		}
	}
	instance.Status.Conditions.MarkTrue(
		dataplanev1beta1.RoleIPReservationReadyCondition,
		dataplanev1beta1.RoleIPReservationReadyMessage)
	return allIPSets, true, nil

}

// createOrPatchDNSData builds the DNSData
func createOrPatchDNSData(ctx context.Context, helper *helper.Helper,
	instance *dataplanev1beta1.OpenStackDataPlaneRole,
	nodes *dataplanev1beta1.OpenStackDataPlaneNodeList,
	allIPSets map[string]infranetworkv1.IPSet) error {

	var allDNSRecords []infranetworkv1.DNSHost
	// Build DNSData CR
	for _, node := range nodes.Items {
		nets := node.Spec.Node.Networks
		if len(nets) == 0 {
			nets = instance.Spec.NodeTemplate.Networks
		}

		if len(nets) > 0 {
			hostName := node.Spec.HostName
			if hostName == "" {
				hostName = node.Name

			}
			// Get IPSet
			ipSet, ok := allIPSets[node.Name]
			if ok {
				for _, res := range ipSet.Status.Reservation {
					dnsRecord := infranetworkv1.DNSHost{}
					dnsRecord.IP = res.Address
					var fqdnNames []string
					fqdnName := strings.Join([]string{hostName,
						strings.ToLower(string(res.Network)), res.DNSDomain}, ".")
					fqdnNames = append(fqdnNames, fqdnName)
					dnsRecord.Hostnames = fqdnNames
					allDNSRecords = append(allDNSRecords, dnsRecord)
				}
			}
		}
	}
	util.LogForObject(helper, "Reconciling DNSData", instance)
	dnsData := &infranetworkv1.DNSData{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: instance.Namespace,
			Name:      instance.Name,
		},
	}
	_, err := controllerutil.CreateOrPatch(ctx, helper.GetClient(), dnsData, func() error {
		dnsData.Spec.Hosts = allDNSRecords
		// Set controller reference to the DataPlaneNode object
		err := controllerutil.SetControllerReference(
			helper.GetBeforeObject(), dnsData, helper.GetScheme())
		return err
	})
	if err != nil {
		return err
	}
	return nil

}

// EnsureDNSData Ensures DNSData is created
func EnsureDNSData(ctx context.Context, helper *helper.Helper,
	instance *dataplanev1beta1.OpenStackDataPlaneRole,
	nodes *dataplanev1beta1.OpenStackDataPlaneNodeList,
	allIPSets map[string]infranetworkv1.IPSet) ([]string, bool, error) {

	// Verify dnsmasq CR exists
	dnsmasqList := &infranetworkv1.DNSMasqList{}
	listOpts := []client.ListOption{
		client.InNamespace(instance.GetNamespace()),
	}
	err := helper.GetClient().List(ctx, dnsmasqList, listOpts...)
	if err != nil {
		return nil, false, err
	}
	if len(dnsmasqList.Items) == 0 {
		util.LogForObject(helper, "No NetConfig CR exists yet, DNS Service won't be used", instance)
		return nil, true, nil
	}

	// Create or Patch DNSData
	err = createOrPatchDNSData(ctx, helper, instance, nodes, allIPSets)
	if err != nil {
		instance.Status.Conditions.MarkFalse(
			dataplanev1beta1.RoleDNSDataReadyCondition,
			condition.ErrorReason, condition.SeverityError,
			dataplanev1beta1.RoleDNSDataReadyErrorMessage)
		return nil, false, err
	}

	dnsData := &infranetworkv1.DNSData{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
		},
	}
	key := client.ObjectKeyFromObject(dnsData)
	err = helper.GetClient().Get(ctx, key, dnsData)
	if err != nil {
		return nil, false, err
	}

	if !dnsData.IsReady() {
		util.LogForObject(helper, "DNSData not ready yet waiting", instance)
		return nil, false, nil
	}
	instance.Status.Conditions.MarkTrue(
		dataplanev1beta1.RoleDNSDataReadyCondition,
		dataplanev1beta1.RoleDNSDataReadyMessage)
	dnsAddresses := dnsmasqList.Items[0].Status.DNSAddresses
	return dnsAddresses, true, nil
}

// reserveIPs Reserves IPs by creating IPSets
func reserveIPs(ctx context.Context, helper *helper.Helper,
	instance *dataplanev1beta1.OpenStackDataPlaneRole,
	nodes *dataplanev1beta1.OpenStackDataPlaneNodeList) (map[string]infranetworkv1.IPSet, error) {

	// Verify NetConfig CRs exist
	netConfigList := &infranetworkv1.NetConfigList{}
	listOpts := []client.ListOption{
		client.InNamespace(instance.GetNamespace()),
	}
	err := helper.GetClient().List(ctx, netConfigList, listOpts...)
	if err != nil {
		return nil, err
	}
	if len(netConfigList.Items) == 0 {
		util.LogForObject(helper, "No NetConfig CR exists yet, IPAM won't be used", instance)
		return nil, nil
	}

	allIPSets := make(map[string]infranetworkv1.IPSet)
	// CreateOrPatch IPSets
	for _, node := range nodes.Items {
		nets := node.Spec.Node.Networks

		if len(nets) == 0 {
			nets = instance.Spec.NodeTemplate.Networks
		}
		if len(nets) > 0 {
			util.LogForObject(helper, "Reconciling IPSet", instance)
			ipSet := &infranetworkv1.IPSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: instance.Namespace,
					Name:      node.Name,
				},
			}
			_, err := controllerutil.CreateOrPatch(ctx, helper.GetClient(), ipSet, func() error {
				ipSet.Spec.Networks = nets
				// Set controller reference to the DataPlaneNode object
				err := controllerutil.SetControllerReference(
					helper.GetBeforeObject(), ipSet, helper.GetScheme())
				return err
			})
			if err != nil {
				return nil, err
			}
			allIPSets[node.Name] = *ipSet
		}
	}

	return allIPSets, nil
}
