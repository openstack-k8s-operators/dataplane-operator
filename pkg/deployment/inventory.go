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
	var err error

	inventory := ansible.MakeInventory()
	roleNameGroup := inventory.AddGroup(instance.Name)
	err = resolveAnsibleVars(&instance.Spec.NodeTemplate, &ansible.Host{}, &roleNameGroup, defaultImages)
	if err != nil {
		return "", err
	}

	for nodeName, node := range instance.Spec.Nodes {
		host := roleNameGroup.AddHost(nodeName)
		var dnsSearchDomains []string

		// Use ansible_host if provided else use hostname. Fall back to
		// nodeName if all else fails.
		if node.Ansible.AnsibleHost != "" {
			host.Vars["ansible_host"] = node.Ansible.AnsibleHost
		} else if node.HostName != "" {
			host.Vars["ansible_host"] = node.HostName
		} else {
			host.Vars["ansible_host"] = nodeName
		}

		ipSet, ok := allIPSets[nodeName]
		if ok {
			populateInventoryFromIPAM(&ipSet, host, dnsAddresses)
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
					host.Vars["gateway_ip"] = res.Gateway
					host.Vars["ctlplane_dns_nameservers"] = dnsAddresses
					host.Vars["ctlplane_host_routes"] = res.Routes
					dnsSearchDomains = append(dnsSearchDomains, res.DNSDomain)
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
					//host.Vars[string.Join(entry, "_gateway_ip")] = res.Gateway
					host.Vars[entry+"_host_routes"] = res.Routes
					dnsSearchDomains = append(dnsSearchDomains, res.DNSDomain)
				}
				networkConfig := getAnsibleNetworkConfig(instance, nodeName)

				if networkConfig.Template != "" {
					host.Vars["edpm_network_config_template"] = NicConfigTemplateFile
				}

				host.Vars["ansible_user"] = getAnsibleUser(instance, nodeName)
				host.Vars["ansible_port"] = getAnsiblePort(instance, nodeName)
				host.Vars["management_network"] = getAnsibleManagementNetwork(instance, nodeName)
				host.Vars["networks"] = getAnsibleNetworks(instance, nodeName)

				ansibleVarsData, err := getAnsibleVars(helper, instance, nodeName)
				if err != nil {
					return "", err
				}
				for key, value := range ansibleVarsData {
					host.Vars[key] = value
				}
				host.Vars["dns_search_domains"] = dnsSearchDomains
			}
		}

		err = resolveNodeAnsibleVars(&node, &host, &ansible.Group{})
		if err != nil {
			return "", err
		}

		ipSet, ok = allIPSets[nodeName]
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
		"network":   string(instance.Spec.NodeTemplate.NetworkConfig.Template),
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

// getAnsibleUser returns the string value from the template unless it is set in the node
func getAnsibleUser(instance *dataplanev1.OpenStackDataPlaneNodeSet, nodeName string) string {
	if instance.Spec.Nodes[nodeName].Ansible.AnsibleUser != "" {
		return instance.Spec.Nodes[nodeName].Ansible.AnsibleUser
	}
	return instance.Spec.NodeTemplate.Ansible.AnsibleUser
}

// getAnsiblePort returns the string value from the template unless it is set in the node
func getAnsiblePort(instance *dataplanev1.OpenStackDataPlaneNodeSet, nodeName string) string {
	if instance.Spec.Nodes[nodeName].Ansible.AnsiblePort > 0 {
		return strconv.Itoa(instance.Spec.Nodes[nodeName].Ansible.AnsiblePort)
	}
	return strconv.Itoa(instance.Spec.NodeTemplate.Ansible.AnsiblePort)
}

// getAnsibleManagementNetwork returns the string value from the template unless it is set in the node
func getAnsibleManagementNetwork(
	instance *dataplanev1.OpenStackDataPlaneNodeSet,
	nodeName string) string {
	if instance.Spec.Nodes[nodeName].ManagementNetwork != "" {
		return instance.Spec.Nodes[nodeName].ManagementNetwork
	}
	return instance.Spec.NodeTemplate.ManagementNetwork
}

// getAnsibleNetworkConfig returns a JSON string value from the template unless it is set in the node
func getAnsibleNetworkConfig(instance *dataplanev1.OpenStackDataPlaneNodeSet, nodeName string) dataplanev1.NetworkConfigSection {
	if instance.Spec.Nodes[nodeName].NetworkConfig.Template != "" {
		return instance.Spec.Nodes[nodeName].NetworkConfig
	}
	return instance.Spec.NodeTemplate.NetworkConfig
}

// getAnsibleNetworks returns a JSON string mapping fixedIP and/or network name to their valules
func getAnsibleNetworks(instance *dataplanev1.OpenStackDataPlaneNodeSet, nodeName string) []infranetworkv1.IPSetNetwork {
	if len(instance.Spec.Nodes[nodeName].Networks) > 0 {
		return instance.Spec.Nodes[nodeName].Networks
	}
	return instance.Spec.NodeTemplate.Networks
}

// getAnsibleVars returns ansible vars for a node
func getAnsibleVars(
	helper *helper.Helper, instance *dataplanev1.OpenStackDataPlaneNodeSet, nodeName string) (map[string]interface{}, error) {
	// Merge the ansibleVars from the role into the value set on the node.
	// Top level keys set on the node ansibleVars should override top level keys from the role AnsibleVars.
	// However, there is no "deep" merge of values. Only top level keys are comvar matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")

	// Unmarshal the YAML strings into two maps
	nodeSet := make(map[string]interface{})
	node := make(map[string]interface{})
	var nodeSetYamlError, nodeYamlError error
	for key, val := range instance.Spec.NodeTemplate.Ansible.AnsibleVars {
		var v interface{}
		nodeSetYamlError = yaml.Unmarshal(val, &v)
		if nodeSetYamlError != nil {
			utils.LogErrorForObject(
				helper,
				nodeSetYamlError,
				fmt.Sprintf("Failed to unmarshal YAML data from role AnsibleVar '%s'",
					key), instance)
			return nil, nodeSetYamlError
		}
		nodeSet[key] = v
	}

	for key, val := range instance.Spec.Nodes[nodeName].Ansible.AnsibleVars {
		var v interface{}
		nodeYamlError = yaml.Unmarshal(val, &v)
		if nodeYamlError != nil {
			utils.LogErrorForObject(
				helper,
				nodeYamlError,
				fmt.Sprintf("Failed to unmarshal YAML data from node AnsibleVar '%s'",
					key), instance)
			return nil, nodeYamlError
		}
		node[key] = v
	}

	if nodeSet == nil && node != nil {
		return node, nil
	}
	if nodeSet != nil && node == nil {
		return nodeSet, nil
	}

	// Merge the two maps
	for k, v := range node {
		nodeSet[k] = v
	}
	return nodeSet, nil
}

func resolveAnsibleVars(nodeTemplate *dataplanev1.NodeTemplate, host *ansible.Host, group *ansible.Group, defaultImages dataplanev1.DataplaneAnsibleImageDefaults) error {
	ansibleVarsData := make(map[string]interface{})

	if nodeTemplate.Ansible.AnsibleHost != "" {
		ansibleVarsData["ansible_user"] = nodeTemplate.Ansible.AnsibleUser
	}
	if nodeTemplate.Ansible.AnsiblePort > 0 {
		ansibleVarsData["ansible_port"] = nodeTemplate.Ansible.AnsiblePort
	}
	if nodeTemplate.ManagementNetwork != "" {
		ansibleVarsData["management_network"] = nodeTemplate.ManagementNetwork
	}
	if nodeTemplate.NetworkConfig.Template != "" {
		ansibleVarsData["edpm_network_config_template"] = NicConfigTemplateFile
	}
	if len(nodeTemplate.Networks) > 0 {
		ansibleVarsData["networks"] = nodeTemplate.Networks
	}

	// Set default Service Image Variables in they are not provided by the user.
	// This uses the default values provided by dataplanev1.DataplaneAnsibleImageDefaults
	if nodeTemplate.Ansible.AnsibleVars["edpm_frr_image"] == nil {
		ansibleVarsData["edpm_frr_image"] = defaultImages.Frr
	}
	if nodeTemplate.Ansible.AnsibleVars["edpm_iscsid_image"] == nil {
		ansibleVarsData["edpm_iscsid_image"] = defaultImages.IscsiD
	}
	if nodeTemplate.Ansible.AnsibleVars["edpm_logrotate_crond_image"] == nil {
		ansibleVarsData["edpm_logrotate_crond_image"] = defaultImages.Logrotate
	}
	if nodeTemplate.Ansible.AnsibleVars["edpm_nova_compute_image"] == nil {
		ansibleVarsData["edpm_nova_compute_image"] = defaultImages.NovaCompute
	}
	if nodeTemplate.Ansible.AnsibleVars["edpm_nova_libvirt_container_image"] == nil {
		ansibleVarsData["edpm_nova_libvirt_image"] = defaultImages.NovaLibvirt
	}
	if nodeTemplate.Ansible.AnsibleVars["edpm_ovn_controller_agent_image"] == nil {
		ansibleVarsData["edpm_ovn_controller_agent_image"] = defaultImages.OvnControllerAgent
	}
	if nodeTemplate.Ansible.AnsibleVars["edpm_ovn_metadata_agent_image"] == nil {
		ansibleVarsData["edpm_ovn_metadata_agent_image"] = defaultImages.OvnMetadataAgent
	}
	if nodeTemplate.Ansible.AnsibleVars["edpm_ovn_bgp_agent_image"] == nil {
		ansibleVarsData["edpm_ovn_bgp_agent_image"] = defaultImages.OvnBgpAgent
	}

	var err error
	for key, val := range nodeTemplate.Ansible.AnsibleVars {
		var v interface{}
		err = yaml.Unmarshal(val, &v)
		if err != nil {
			return err
		}
		ansibleVarsData[key] = v
	}

	if host.Vars != nil {
		for key, value := range ansibleVarsData {
			host.Vars[key] = value
		}
	}

	if group.Vars != nil {
		for key, value := range ansibleVarsData {
			group.Vars[key] = value
		}
	}

	return nil
}

func resolveNodeAnsibleVars(node *dataplanev1.NodeSection, host *ansible.Host, group *ansible.Group) error {
	ansibleVarsData := make(map[string]interface{})

	if node.Ansible.AnsibleUser != "" {
		ansibleVarsData["ansible_user"] = node.Ansible.AnsibleUser
	}
	if node.Ansible.AnsiblePort > 0 {
		ansibleVarsData["ansible_port"] = node.Ansible.AnsiblePort
	}
	if node.ManagementNetwork != "" {
		ansibleVarsData["management_network"] = node.ManagementNetwork
	}
	if node.NetworkConfig.Template != "" {
		ansibleVarsData["edpm_network_config_template"] = NicConfigTemplateFile
	}
	if len(node.Networks) > 0 {
		ansibleVarsData["networks"] = node.Networks
	}
	var err error
	for key, val := range node.Ansible.AnsibleVars {
		var v interface{}
		err = yaml.Unmarshal(val, &v)
		if err != nil {
			return err
		}
		ansibleVarsData[key] = v
	}

	if host.Vars != nil {
		for key, value := range ansibleVarsData {
			host.Vars[key] = value
		}
	}

	if group.Vars != nil {
		for key, value := range ansibleVarsData {
			group.Vars[key] = value
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
