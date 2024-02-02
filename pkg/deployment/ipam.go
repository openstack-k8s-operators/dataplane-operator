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

	dataplanev1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	infranetworkv1 "github.com/openstack-k8s-operators/infra-operator/apis/network/v1beta1"
	condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
)

// EnsureIPSets Creates the IPSets
func EnsureIPSets(ctx context.Context, helper *helper.Helper,
	instance *dataplanev1.OpenStackDataPlaneNodeSet) (map[string]infranetworkv1.IPSet, bool, error) {
	allIPSets, err := reserveIPs(ctx, helper, instance)
	if err != nil {
		instance.Status.Conditions.MarkFalse(
			dataplanev1.NodeSetIPReservationReadyCondition,
			condition.ErrorReason, condition.SeverityError,
			dataplanev1.NodeSetIPReservationReadyErrorMessage)
		return nil, false, err
	}

	if len(allIPSets) == 0 {
		return nil, true, nil
	}

	for _, s := range allIPSets {
		if s.Status.Conditions.IsFalse(condition.ReadyCondition) {
			instance.Status.Conditions.MarkFalse(
				dataplanev1.NodeSetIPReservationReadyCondition,
				condition.RequestedReason, condition.SeverityInfo,
				dataplanev1.NodeSetIPReservationReadyWaitingMessage)
			return nil, false, nil
		}
	}
	instance.Status.Conditions.MarkTrue(
		dataplanev1.NodeSetIPReservationReadyCondition,
		dataplanev1.NodeSetIPReservationReadyMessage)
	return allIPSets, true, nil

}

// createOrPatchDNSData builds the DNSData
func createOrPatchDNSData(ctx context.Context, helper *helper.Helper,
	instance *dataplanev1.OpenStackDataPlaneNodeSet,
	allIPSets map[string]infranetworkv1.IPSet) (
	string, map[string]map[infranetworkv1.NetNameStr]string,
	map[string]map[infranetworkv1.NetNameStr]string, error) {

	var allDNSRecords []infranetworkv1.DNSHost
	var ctlplaneSearchDomain string
	allHostnames := map[string]map[infranetworkv1.NetNameStr]string{}
	allIPs := map[string]map[infranetworkv1.NetNameStr]string{}

	// Build DNSData CR
	for _, node := range instance.Spec.Nodes {
		var shortName string
		nets := node.Networks
		hostName := node.HostName

		allHostnames[hostName] = map[infranetworkv1.NetNameStr]string{}
		allIPs[hostName] = map[infranetworkv1.NetNameStr]string{}

		shortName = strings.Split(hostName, ".")[0]
		if len(nets) == 0 {
			nets = instance.Spec.NodeTemplate.Networks
		}
		if len(nets) > 0 {
			// Get IPSet
			ipSet, ok := allIPSets[hostName]
			if ok {
				for _, res := range ipSet.Status.Reservation {
					var fqdnNames []string
					dnsRecord := infranetworkv1.DNSHost{}
					dnsRecord.IP = res.Address
					netLower := strings.ToLower(string(res.Network))
					fqdnName := strings.Join([]string{shortName, res.DNSDomain}, ".")
					if fqdnName != hostName {
						fqdnNames = append(fqdnNames, fqdnName)
						allHostnames[hostName][res.Network] = fqdnName
					}
					if isFQDN(hostName) && netLower == CtlPlaneNetwork {
						fqdnNames = append(fqdnNames, hostName)
						allHostnames[hostName][res.Network] = hostName
					}
					allIPs[hostName][res.Network] = res.Address
					dnsRecord.Hostnames = fqdnNames
					allDNSRecords = append(allDNSRecords, dnsRecord)
					// Adding only ctlplane domain for ansibleee.
					// TODO (rabi) This is not very efficient.
					if netLower == CtlPlaneNetwork && ctlplaneSearchDomain == "" {
						ctlplaneSearchDomain = res.DNSDomain
					}
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
		// TODO (rabi) DNSDataLabelSelectorValue can probably be
		// used from dnsmasq(?)
		dnsData.Spec.DNSDataLabelSelectorValue = "dnsdata"
		// Set controller reference to the DataPlaneNode object
		err := controllerutil.SetControllerReference(
			helper.GetBeforeObject(), dnsData, helper.GetScheme())
		return err
	})
	if err != nil {
		return "", allHostnames, allIPs, err
	}
	return ctlplaneSearchDomain, allHostnames, allIPs, nil

}

// EnsureDNSData Ensures DNSData is created
func EnsureDNSData(ctx context.Context, helper *helper.Helper,
	instance *dataplanev1.OpenStackDataPlaneNodeSet,
	allIPSets map[string]infranetworkv1.IPSet) (
	[]string, []string, string, bool, map[string]map[infranetworkv1.NetNameStr]string,
	map[string]map[infranetworkv1.NetNameStr]string, error) {

	// Verify dnsmasq CR exists
	dnsAddresses, dnsClusterAddresses, isReady, err := CheckDNSService(
		ctx, helper, instance)

	if err != nil || !isReady || dnsClusterAddresses == nil {
		if err != nil {
			instance.Status.Conditions.MarkFalse(
				dataplanev1.NodeSetDNSDataReadyCondition,
				condition.ErrorReason, condition.SeverityError,
				dataplanev1.NodeSetDNSDataReadyErrorMessage)
		}
		if !isReady {
			instance.Status.Conditions.MarkFalse(
				dataplanev1.NodeSetDNSDataReadyCondition,
				condition.RequestedReason, condition.SeverityInfo,
				dataplanev1.NodeSetDNSDataReadyWaitingMessage)
		}
		if dnsClusterAddresses == nil {
			instance.Status.Conditions.Remove(dataplanev1.NodeSetDNSDataReadyCondition)
		}
		return nil, nil, "", isReady, nil, nil, err
	}
	// Create or Patch DNSData
	ctlplaneSearchDomain, allHostnames, allIPs, err := createOrPatchDNSData(
		ctx, helper, instance, allIPSets)
	if err != nil {
		instance.Status.Conditions.MarkFalse(
			dataplanev1.NodeSetDNSDataReadyCondition,
			condition.ErrorReason, condition.SeverityError,
			dataplanev1.NodeSetDNSDataReadyErrorMessage)
		return nil, nil, "", false, nil, nil, err
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
		instance.Status.Conditions.MarkFalse(
			dataplanev1.NodeSetDNSDataReadyCondition,
			condition.ErrorReason, condition.SeverityError,
			dataplanev1.NodeSetDNSDataReadyErrorMessage)
		return nil, nil, "", false, nil, nil, err
	}

	if !dnsData.IsReady() {
		util.LogForObject(helper, "DNSData not ready yet waiting", instance)
		instance.Status.Conditions.MarkFalse(
			dataplanev1.NodeSetDNSDataReadyCondition,
			condition.RequestedReason, condition.SeverityInfo,
			dataplanev1.NodeSetDNSDataReadyWaitingMessage)
		return nil, nil, "", false, nil, nil, nil
	}
	instance.Status.Conditions.MarkTrue(
		dataplanev1.NodeSetDNSDataReadyCondition,
		dataplanev1.NodeSetDNSDataReadyMessage)
	return dnsAddresses, dnsClusterAddresses, ctlplaneSearchDomain, true, allHostnames, allIPs, nil
}

// reserveIPs Reserves IPs by creating IPSets
func reserveIPs(ctx context.Context, helper *helper.Helper,
	instance *dataplanev1.OpenStackDataPlaneNodeSet) (map[string]infranetworkv1.IPSet, error) {

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
		instance.Status.Conditions.Remove(dataplanev1.NodeSetIPReservationReadyCondition)
		return nil, nil
	}

	ipamUsed := false
	allIPSets := make(map[string]infranetworkv1.IPSet)
	// CreateOrPatch IPSets
	for _, node := range instance.Spec.Nodes {
		nets := node.Networks
		hostName := node.HostName
		if len(nets) == 0 {
			nets = instance.Spec.NodeTemplate.Networks
		}

		if len(nets) > 0 {
			ipamUsed = true
			util.LogForObject(helper, "Reconciling IPSet", instance)
			ipSet := &infranetworkv1.IPSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: instance.Namespace,
					Name:      hostName,
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
			allIPSets[hostName] = *ipSet
		}
	}
	if !ipamUsed {
		util.LogForObject(helper, "No Networks defined for nodes, IPAM won't be used", instance)
		instance.Status.Conditions.Remove(dataplanev1.NodeSetIPReservationReadyCondition)
	}

	return allIPSets, nil
}

// CheckDNSService checks if DNS is configured and ready
func CheckDNSService(ctx context.Context, helper *helper.Helper,
	instance client.Object) ([]string, []string, bool, error) {
	dnsmasqList := &infranetworkv1.DNSMasqList{}
	listOpts := []client.ListOption{
		client.InNamespace(instance.GetNamespace()),
	}
	err := helper.GetClient().List(ctx, dnsmasqList, listOpts...)
	if err != nil {
		return nil, nil, false, err
	}
	if len(dnsmasqList.Items) == 0 {
		util.LogForObject(helper, "No DNSMasq CR exists yet, DNS Service won't be used", instance)
		return nil, nil, true, nil
	} else if !dnsmasqList.Items[0].IsReady() {
		util.LogForObject(helper, "DNSMasq service exists, but not ready yet ", instance)
		return nil, nil, false, nil
	}
	dnsClusterAddresses := dnsmasqList.Items[0].Status.DNSClusterAddresses
	dnsAddresses := dnsmasqList.Items[0].Status.DNSAddresses
	return dnsAddresses, dnsClusterAddresses, true, nil
}
