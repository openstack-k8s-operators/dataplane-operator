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
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	yaml "gopkg.in/yaml.v3"

	dataplanev1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	infranetworkv1 "github.com/openstack-k8s-operators/infra-operator/apis/network/v1beta1"
	"github.com/openstack-k8s-operators/lib-common/modules/ansible"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/secret"
	utils "github.com/openstack-k8s-operators/lib-common/modules/common/util"
)

// GenerateNodeSetInventory yields a parsed Inventory for role
func GenerateNodeSetInventory(ctx context.Context, helper *helper.Helper,
	instance *dataplanev1.OpenStackDataPlaneNodeSet,
	allIPSets map[string]infranetworkv1.IPSet, dnsAddresses []string, defaultImages dataplanev1.DataplaneAnsibleImageDefaults) (string, error) {
	inventory := ansible.MakeInventory()
	nodeSetGroup := inventory.AddGroup(instance.Name)
	err := resolveGroupAnsibleVars(&instance.Spec.NodeTemplate, &nodeSetGroup, defaultImages)
	if err != nil {
		return "", err
	}
	for nodeName, node := range instance.Spec.Nodes {
		host := nodeSetGroup.AddHost(nodeName)
		// Use ansible_host if provided else use hostname. Fall back to
		// nodeName if all else fails.
		if node.Ansible.AnsibleHost != "" {
			host.Vars["ansible_host"] = node.Ansible.AnsibleHost
		} else if node.HostName != "" {
			host.Vars["ansible_host"] = node.HostName
		}

		err = resolveHostAnsibleVars(&node, &host)
		if err != nil {
			return "", err
		}

		ipSet, ok := allIPSets[nodeName]
		if ok {
			populateInventoryFromIPAM(&ipSet, host, dnsAddresses)
		}

	}

	invData, err := inventory.MarshalYAML()
	if err != nil {
		utils.LogErrorForObject(helper, err, "Could not parse NodeSet inventory", instance)
		return "", err
	}
	secretData := map[string]string{
		"inventory": string(invData),
	}
	secretName := fmt.Sprintf("dataplanenodeset-%s", instance.Name)
	template := []utils.Template{
		// Secret
		{
			Name:         secretName,
			Namespace:    instance.Namespace,
			Type:         utils.TemplateTypeNone,
			InstanceType: instance.Kind,
			CustomData:   secretData,
			Labels:       instance.ObjectMeta.Labels,
		},
	}
	err = secret.EnsureSecrets(ctx, helper, instance, template, nil)
	return secretName, err
}

// populateInventoryFromIPAM populates inventory from IPAM
func populateInventoryFromIPAM(
	ipSet *infranetworkv1.IPSet, host ansible.Host,
	dnsAddresses []string) {
	var dnsSearchDomains []string
	for _, res := range ipSet.Status.Reservation {
		// Build the vars for ips/routes etc
		switch n := res.Network; n {
		case CtlPlaneNetwork:
			host.Vars["ctlplane_ip"] = res.Address
			_, ipnet, err := net.ParseCIDR(res.Cidr)
			if err == nil {
				netCidr, _ := ipnet.Mask.Size()
				host.Vars["ctlplane_subnet_cidr"] = netCidr
			}
			host.Vars["ctlplane_mtu"] = res.MTU
			host.Vars["ctlplane_gateway_ip"] = res.Gateway
			host.Vars["ctlplane_dns_nameservers"] = dnsAddresses
			host.Vars["ctlplane_host_routes"] = res.Routes
		default:
			entry := toSnakeCase(string(n))
			host.Vars[entry+"_ip"] = res.Address
			_, ipnet, err := net.ParseCIDR(res.Cidr)
			if err == nil {
				netCidr, _ := ipnet.Mask.Size()
				host.Vars[entry+"_cidr"] = netCidr
			}
			host.Vars[entry+"_vlan_id"] = res.Vlan
			host.Vars[entry+"_mtu"] = res.MTU
			host.Vars[entry+"_gateway_ip"] = res.Gateway
			host.Vars[entry+"_host_routes"] = res.Routes
		}
		dnsSearchDomains = append(dnsSearchDomains, res.DNSDomain)
	}
	host.Vars["dns_search_domains"] = dnsSearchDomains
}

func resolveGroupAnsibleVars(template *dataplanev1.NodeTemplate, group *ansible.Group,
	defaultImages dataplanev1.DataplaneAnsibleImageDefaults) error {
	groupVars := make(map[string]interface{})

	if template.Ansible.AnsibleUser != "" {
		groupVars["ansible_user"] = template.Ansible.AnsibleUser
	}
	if template.Ansible.AnsiblePort > 0 {
		groupVars["ansible_port"] = strconv.Itoa(template.Ansible.AnsiblePort)
	}
	if template.ManagementNetwork != "" {
		groupVars["management_network"] = template.ManagementNetwork
	}
	if len(template.Networks) > 0 {
		groupVars["networks"] = template.Networks
	}
	// Set default Service Image Variables in they are not provided by the user.
	// This uses the default values provided by dataplanev1.DataplaneAnsibleImageDefaults
	if template.Ansible.AnsibleVars["edpm_frr_image"] == nil {
		groupVars["edpm_frr_image"] = defaultImages.Frr
	}
	if template.Ansible.AnsibleVars["edpm_iscsid_image"] == nil {
		groupVars["edpm_iscsid_image"] = defaultImages.IscsiD
	}
	if template.Ansible.AnsibleVars["edpm_logrotate_crond_image"] == nil {
		groupVars["edpm_logrotate_crond_image"] = defaultImages.Logrotate
	}
	if template.Ansible.AnsibleVars["edpm_nova_compute_image"] == nil {
		groupVars["edpm_nova_compute_image"] = defaultImages.NovaCompute
	}
	if template.Ansible.AnsibleVars["edpm_nova_libvirt_container_image"] == nil {
		groupVars["edpm_nova_libvirt_image"] = defaultImages.NovaLibvirt
	}
	if template.Ansible.AnsibleVars["edpm_ovn_controller_agent_image"] == nil {
		groupVars["edpm_ovn_controller_agent_image"] = defaultImages.OvnControllerAgent
	}
	if template.Ansible.AnsibleVars["edpm_ovn_metadata_agent_image"] == nil {
		groupVars["edpm_ovn_metadata_agent_image"] = defaultImages.OvnMetadataAgent
	}
	if template.Ansible.AnsibleVars["edpm_ovn_bgp_agent_image"] == nil {
		groupVars["edpm_ovn_bgp_agent_image"] = defaultImages.OvnBgpAgent
	}

	for key, val := range template.Ansible.AnsibleVars {
		var v interface{}
		err := yaml.Unmarshal(val, &v)
		if err != nil {
			return err
		}
		groupVars[key] = v
	}
	if group.Vars != nil {
		for key, value := range groupVars {
			group.Vars[key] = value
		}
	}
	return nil
}

func resolveHostAnsibleVars(node *dataplanev1.NodeSection, host *ansible.Host) error {
	hostVars := make(map[string]interface{})

	if node.Ansible.AnsibleUser != "" {
		hostVars["ansible_user"] = node.Ansible.AnsibleUser
	}
	if node.Ansible.AnsiblePort > 0 {
		hostVars["ansible_port"] = strconv.Itoa(node.Ansible.AnsiblePort)
	}
	if node.ManagementNetwork != "" {
		hostVars["management_network"] = node.ManagementNetwork
	}
	if len(node.Networks) > 0 {
		hostVars["networks"] = node.Networks
	}

	for key, val := range node.Ansible.AnsibleVars {
		var v interface{}
		err := yaml.Unmarshal(val, &v)
		if err != nil {
			return err
		}
		hostVars[key] = v
	}

	if host.Vars != nil {
		for key, value := range hostVars {
			host.Vars[key] = value
		}
	}
	return nil
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}
