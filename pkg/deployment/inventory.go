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
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	dataplanev1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	infranetworkv1 "github.com/openstack-k8s-operators/infra-operator/apis/network/v1beta1"
	"github.com/openstack-k8s-operators/lib-common/modules/ansible"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/secret"
	utils "github.com/openstack-k8s-operators/lib-common/modules/common/util"
)

// GenerateRoleInventory yields a parsed Inventory for role
func GenerateRoleInventory(ctx context.Context, helper *helper.Helper,
	instance *dataplanev1.OpenStackDataPlaneRole,
	nodes []dataplanev1.OpenStackDataPlaneNode,
	allIPSets map[string]infranetworkv1.IPSet, dnsAddresses []string) (string, error) {
	var err error

	inventory := ansible.MakeInventory()
	roleNameGroup := inventory.AddGroup(instance.Name)
	err = resolveAnsibleVars(&instance.Spec.NodeTemplate, &ansible.Host{}, &roleNameGroup)
	if err != nil {
		return "", err
	}

	for _, node := range nodes {
		host := roleNameGroup.AddHost(node.Name)
		// Use if provided else use hostname
		if node.Spec.AnsibleHost != "" {
			host.Vars["ansible_host"] = node.Spec.AnsibleHost
		} else {
			host.Vars["ansible_host"] = node.Spec.HostName
		}

		err = resolveAnsibleVars(&node.Spec.Node, &host, &ansible.Group{})
		if err != nil {
			return "", err
		}

		ipSet, ok := allIPSets[node.Name]
		if ok {
			populateInventoryFromIPAM(&ipSet, host, dnsAddresses)
		}

	}

	invData, err := inventory.MarshalYAML()
	if err != nil {
		utils.LogErrorForObject(helper, err, "Could not parse Role inventory", instance)
		return "", err
	}
	secretData := map[string]string{
		"inventory": string(invData),
		"network":   string(instance.Spec.NodeTemplate.NetworkConfig.Template),
	}
	secretName := fmt.Sprintf("dataplanerole-%s", instance.Name)
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

// GenerateNodeInventory yields a parsed Inventory for node
func GenerateNodeInventory(ctx context.Context, helper *helper.Helper,
	instance *dataplanev1.OpenStackDataPlaneNode,
	instanceRole *dataplanev1.OpenStackDataPlaneRole) (string, error) {
	var (
		err      error
		hostName string
	)

	inventory := ansible.MakeInventory()
	all := inventory.AddGroup("all")
	host := all.AddHost(instance.Name)

	ipSet := &infranetworkv1.IPSet{}
	err = helper.GetClient().Get(ctx,
		types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, ipSet)
	if err != nil {
		if !k8s_errors.IsNotFound(err) {
			return "", err
		}
		// Don't try to popluare inventory
		utils.LogForObject(helper, "Networks not configured for Role", instance)
	} else {
		dnsAddresses, _, _, err := checkDNSService(ctx, helper, instance)
		if err != nil {
			return "", err
		}
		populateInventoryFromIPAM(ipSet, host, dnsAddresses)
	}
	networkConfig := getAnsibleNetworkConfig(instance, instanceRole)

	if networkConfig.Template != "" {
		host.Vars["edpm_network_config_template"] = NicConfigTemplateFile
	}

	host.Vars["ansible_user"] = getAnsibleUser(instance, instanceRole)
	host.Vars["ansible_port"] = getAnsiblePort(instance, instanceRole)
	host.Vars["management_network"] = getAnsibleManagementNetwork(instance, instanceRole)
	host.Vars["networks"] = getAnsibleNetworks(instance, instanceRole)

	if instance.Spec.AnsibleHost == "" {
		hostName = instance.Spec.HostName
	} else {
		hostName = instance.Spec.AnsibleHost
	}
	host.Vars["ansible_host"] = hostName

	ansibleVarsData, err := getAnsibleVars(helper, instance, instanceRole)
	if err != nil {
		return "", err
	}
	for key, value := range ansibleVarsData {
		host.Vars[key] = value
	}

	invData, err := inventory.MarshalYAML()
	if err != nil {
		utils.LogErrorForObject(helper, err, "Could not Parse node inventory", instance)
		return "", err
	}
	secretData := map[string]string{
		"inventory": string(invData),
		"network":   string(networkConfig.Template),
	}
	secretName := fmt.Sprintf("dataplanenode-%s", instance.Name)
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
func getAnsibleUser(instance *dataplanev1.OpenStackDataPlaneNode, instanceRole *dataplanev1.OpenStackDataPlaneRole) string {
	if instance.Spec.Node.AnsibleUser != "" {
		return instance.Spec.Node.AnsibleUser
	}
	return instanceRole.Spec.NodeTemplate.AnsibleUser
}

// getAnsiblePort returns the string value from the template unless it is set in the node
func getAnsiblePort(instance *dataplanev1.OpenStackDataPlaneNode,
	instanceRole *dataplanev1.OpenStackDataPlaneRole) string {
	if instance.Spec.Node.AnsiblePort > 0 {
		return strconv.Itoa(instance.Spec.Node.AnsiblePort)
	}
	return strconv.Itoa(instanceRole.Spec.NodeTemplate.AnsiblePort)
}

// getAnsibleManagementNetwork returns the string value from the template unless it is set in the node
func getAnsibleManagementNetwork(
	instance *dataplanev1.OpenStackDataPlaneNode,
	instanceRole *dataplanev1.OpenStackDataPlaneRole) string {
	if instance.Spec.Node.ManagementNetwork != "" {
		return instance.Spec.Node.ManagementNetwork
	}
	return instanceRole.Spec.NodeTemplate.ManagementNetwork
}

// getAnsibleNetworkConfig returns a JSON string value from the template unless it is set in the node
func getAnsibleNetworkConfig(instance *dataplanev1.OpenStackDataPlaneNode,
	instanceRole *dataplanev1.OpenStackDataPlaneRole) dataplanev1.NetworkConfigSection {
	if instance.Spec.Node.NetworkConfig.Template != "" {
		return instance.Spec.Node.NetworkConfig
	}
	return instanceRole.Spec.NodeTemplate.NetworkConfig
}

// getAnsibleNetworks returns a JSON string mapping fixedIP and/or network name to their valules
func getAnsibleNetworks(instance *dataplanev1.OpenStackDataPlaneNode,
	instanceRole *dataplanev1.OpenStackDataPlaneRole) []infranetworkv1.IPSetNetwork {
	if len(instance.Spec.Node.Networks) > 0 {
		return instance.Spec.Node.Networks
	}
	return instanceRole.Spec.NodeTemplate.Networks
}

// getAnsibleVars returns ansible vars for a node
func getAnsibleVars(helper *helper.Helper, instance *dataplanev1.OpenStackDataPlaneNode,
	instanceRole *dataplanev1.OpenStackDataPlaneRole) (map[string]interface{}, error) {
	// Merge the ansibleVars from the role into the value set on the node.
	// Top level keys set on the node ansibleVars should override top level keys from the role AnsibleVars.
	// However, there is no "deep" merge of values. Only top level keys are comvar matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")

	// Unmarshal the YAML strings into two maps
	role := make(map[string]interface{})
	node := make(map[string]interface{})
	var roleYamlError, nodeYamlError error
	for key, val := range instanceRole.Spec.NodeTemplate.AnsibleVars {
		var v interface{}
		roleYamlError = yaml.Unmarshal(val, &v)
		if roleYamlError != nil {
			utils.LogErrorForObject(
				helper,
				roleYamlError,
				fmt.Sprintf("Failed to unmarshal YAML data from role AnsibleVar '%s'",
					key), instance)
			return nil, roleYamlError
		}
		role[key] = v
	}

	for key, val := range instance.Spec.Node.AnsibleVars {
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
	if role == nil && node != nil {
		return node, nil
	}
	if role != nil && node == nil {
		return role, nil
	}

	// Merge the two maps
	for k, v := range node {
		role[k] = v
	}
	return role, nil
}

func resolveAnsibleVars(node *dataplanev1.NodeSection, host *ansible.Host, group *ansible.Group) error {
	ansibleVarsData := make(map[string]interface{})

	if node.AnsibleUser != "" {
		ansibleVarsData["ansible_user"] = node.AnsibleUser
	}
	if node.AnsiblePort > 0 {
		ansibleVarsData["ansible_port"] = node.AnsiblePort
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
	for key, val := range node.AnsibleVars {
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
