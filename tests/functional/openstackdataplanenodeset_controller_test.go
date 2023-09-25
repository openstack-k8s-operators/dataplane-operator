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
package functional

import (
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	dataplanev1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	. "github.com/openstack-k8s-operators/lib-common/modules/common/test/helpers"
	"github.com/openstack-k8s-operators/openstack-baremetal-operator/api/v1beta1"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Ansible Inventory Structs for testing specific values
type AnsibleInventory struct {
	EdpmComputeNodeset struct {
		Vars struct {
			AnsibleUser string `yaml:"ansible_user"`
		} `yaml:"vars"`
		Hosts struct {
			Node struct {
				AnsibleHost       string        `yaml:"ansible_host"`
				AnsiblePort       string        `yaml:"ansible_port"`
				AnsibleUser       string        `yaml:"ansible_user"`
				CtlPlaneIP        string        `yaml:"ctlplane_ip"`
				DNSSearchDomains  []interface{} `yaml:"dns_search_domains"`
				ManagementNetwork string        `yaml:"management_network"`
				Networks          []interface{} `yaml:"networks"`
			} `yaml:"edpm-compute-node-1"`
		} `yaml:"hosts"`
	} `yaml:"edpm-compute-nodeset"`
}

var _ = Describe("Dataplane NodeSet Test", func() {
	var dataplaneNodeSetName types.NamespacedName
	var dataplaneSecretName types.NamespacedName
	var dataplaneSSHSecretName types.NamespacedName
	var dataplaneNetConfigName types.NamespacedName
	var dataplaneIPSetName types.NamespacedName

	defaultEdpmServiceList := []string{
		"edpm_frr_image",
		"edpm_iscsid_image",
		"edpm_logrotate_crond_image",
		"edpm_nova_compute_image",
		"edpm_nova_libvirt_image",
		"edpm_ovn_controller_agent_image",
		"edpm_ovn_metadata_agent_image",
		"edpm_ovn_bgp_agent_image",
	}

	BeforeEach(func() {
		dataplaneNodeSetName = types.NamespacedName{
			Name:      "edpm-compute-nodeset",
			Namespace: namespace,
		}
		dataplaneSecretName = types.NamespacedName{
			Namespace: namespace,
			Name:      "dataplanenodeset-edpm-compute-nodeset",
		}
		dataplaneSSHSecretName = types.NamespacedName{
			Namespace: namespace,
			Name:      "dataplane-ansible-ssh-private-key-secret",
		}
		dataplaneNetConfigName = types.NamespacedName{
			Namespace: namespace,
			Name:      "dataplane-netconfig",
		}
		dataplaneIPSetName = types.NamespacedName{
			Namespace: namespace,
			Name:      "edpm-compute-node-1",
		}
		err := os.Setenv("OPERATOR_SERVICES", "../../config/services")
		Expect(err).NotTo(HaveOccurred())
	})

	When("A Dataplane resorce is created with PreProvisioned nodes, no deployment", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateDataplaneNodeSet(dataplaneNodeSetName, DefaultDataPlaneNoNodeSetSpec()))
		})
		It("should have the Spec fields initialized", func() {
			dataplaneNodeSetInstance := GetDataplaneNodeSet(dataplaneNodeSetName)
			emptyNodeSpec := dataplanev1.OpenStackDataPlaneNodeSetSpec{
				BaremetalSetTemplate: v1beta1.OpenStackBaremetalSetSpec{
					BaremetalHosts:        nil,
					OSImage:               "",
					UserData:              nil,
					NetworkData:           nil,
					AutomatedCleaningMode: "",
					ProvisionServerName:   "",
					ProvisioningInterface: "",
					DeploymentSSHSecret:   "",
					CtlplaneInterface:     "",
					CtlplaneGateway:       "",
					CtlplaneNetmask:       "",
					BmhNamespace:          "",
					BmhLabelSelector:      nil,
					HardwareReqs: v1beta1.HardwareReqs{
						CPUReqs: v1beta1.CPUReqs{
							Arch:     "",
							CountReq: v1beta1.CPUCountReq{Count: 0, ExactMatch: false},
							MhzReq:   v1beta1.CPUMhzReq{Mhz: 0, ExactMatch: false},
						},
						MemReqs: v1beta1.MemReqs{
							GbReq: v1beta1.MemGbReq{Gb: 0, ExactMatch: false},
						},
						DiskReqs: v1beta1.DiskReqs{
							GbReq:  v1beta1.DiskGbReq{Gb: 0, ExactMatch: false},
							SSDReq: v1beta1.DiskSSDReq{SSD: false, ExactMatch: false},
						},
					},
					PasswordSecret:   nil,
					CloudUserName:    "",
					DomainName:       "",
					BootstrapDNS:     nil,
					DNSSearchDomains: nil,
				},
				NodeTemplate: dataplanev1.NodeTemplate{
					AnsibleSSHPrivateKeySecret: "dataplane-ansible-ssh-private-key-secret",
					Networks:                   nil,
					ManagementNetwork:          "",
					Ansible: dataplanev1.AnsibleOpts{
						AnsibleUser: "",
						AnsibleHost: "",
						AnsiblePort: 0,
						AnsibleVars: nil,
					},
					ExtraMounts: nil,
					UserData:    nil,
					NetworkData: nil,
				},
				Env:                nil,
				PreProvisioned:     true,
				NetworkAttachments: nil,
				Nodes:              map[string]dataplanev1.NodeSection{},
				Services: []string{
					"configure-network",
					"validate-network",
					"download-cache",
					"install-os",
					"configure-os",
					"run-os",
					"ovn",
					"libvirt",
					"nova"},
			}
			Expect(dataplaneNodeSetInstance.Spec).Should(Equal(emptyNodeSpec))
		})

		It("should have the Status fields initialized", func() {
			dataplaneNodeSetInstance := GetDataplaneNodeSet(dataplaneNodeSetName)
			Expect(dataplaneNodeSetInstance.Status.Deployed).Should(BeFalse())
		})

		It("should have input not ready and unknown Conditions initialized", func() {
			th.ExpectCondition(
				dataplaneNodeSetName,
				ConditionGetterFunc(DataplaneConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionFalse,
			)
			th.ExpectCondition(
				dataplaneNodeSetName,
				ConditionGetterFunc(DataplaneConditionGetter),
				condition.InputReadyCondition,
				corev1.ConditionFalse,
			)
			th.ExpectCondition(
				dataplaneNodeSetName,
				ConditionGetterFunc(DataplaneConditionGetter),
				dataplanev1.SetupReadyCondition,
				corev1.ConditionFalse,
			)
		})

		It("Should not have created a Secret", func() {
			th.AssertSecretDoesNotExist(dataplaneSecretName)
		})
	})

	When("A Dataplane resorce is created without PreProvisioned nodes and ordered deployment", func() {
		BeforeEach(func() {
			spec := DefaultDataPlaneNoNodeSetSpec()
			spec["metadata"] = map[string]interface{}{"ansiblesshprivatekeysecret": ""}
			spec["preProvisioned"] = false
			DeferCleanup(th.DeleteInstance, CreateDataplaneNodeSet(dataplaneNodeSetName, spec))
		})
		It("should have the Spec fields initialized", func() {
			dataplaneNodeSetInstance := GetDataplaneNodeSet(dataplaneNodeSetName)
			Expect(dataplaneNodeSetInstance.Spec.PreProvisioned).Should(BeFalse())
		})

		It("should have the Status fields initialized", func() {
			dataplaneNodeSetInstance := GetDataplaneNodeSet(dataplaneNodeSetName)
			Expect(dataplaneNodeSetInstance.Status.Deployed).Should(BeFalse())
		})

		It("should have ReadyCondition, InputReadyCondition and SetupReadyCondition set to false, and DeploymentReadyCondition set to Unknown", func() {
			th.ExpectCondition(
				dataplaneNodeSetName,
				ConditionGetterFunc(DataplaneConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionFalse,
			)
			th.ExpectCondition(
				dataplaneNodeSetName,
				ConditionGetterFunc(DataplaneConditionGetter),
				condition.InputReadyCondition,
				corev1.ConditionFalse,
			)
			th.ExpectCondition(
				dataplaneNodeSetName,
				ConditionGetterFunc(DataplaneConditionGetter),
				dataplanev1.SetupReadyCondition,
				corev1.ConditionFalse,
			)
			th.ExpectCondition(
				dataplaneNodeSetName,
				ConditionGetterFunc(DataplaneConditionGetter),
				condition.DeploymentReadyCondition,
				corev1.ConditionUnknown,
			)
		})

		It("Should not have created a Secret", func() {
			th.AssertSecretDoesNotExist(dataplaneSecretName)
		})
	})

	When("A Dataplane resorce is created without PreProvisioned nodes but is marked as PreProvisioned, with ordered deployment", func() {
		BeforeEach(func() {
			spec := DefaultDataPlaneNoNodeSetSpec()
			spec["metadata"] = map[string]interface{}{"ansiblesshprivatekeysecret": ""}
			DeferCleanup(th.DeleteInstance, CreateDataplaneNodeSet(dataplaneNodeSetName, spec))
		})
		It("should have the Spec fields initialized", func() {
			dataplaneNodeSetInstance := GetDataplaneNodeSet(dataplaneNodeSetName)
			Expect(dataplaneNodeSetInstance.Spec.PreProvisioned).Should(BeTrue())
		})

		It("should have the Status fields initialized", func() {
			dataplaneNodeSetInstance := GetDataplaneNodeSet(dataplaneNodeSetName)
			Expect(dataplaneNodeSetInstance.Status.Deployed).Should(BeFalse())
		})

		It("should have ReadyCondition, InputReadCondition and SetupReadyCondition set to false, and DeploymentReadyCondition set to unknown", func() {
			th.ExpectCondition(
				dataplaneNodeSetName,
				ConditionGetterFunc(DataplaneConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionFalse,
			)
			th.ExpectCondition(
				dataplaneNodeSetName,
				ConditionGetterFunc(DataplaneConditionGetter),
				condition.InputReadyCondition,
				corev1.ConditionFalse,
			)
			th.ExpectCondition(
				dataplaneNodeSetName,
				ConditionGetterFunc(DataplaneConditionGetter),
				dataplanev1.SetupReadyCondition,
				corev1.ConditionFalse,
			)
			th.ExpectCondition(
				dataplaneNodeSetName,
				ConditionGetterFunc(DataplaneConditionGetter),
				condition.DeploymentReadyCondition,
				corev1.ConditionUnknown,
			)
		})

		It("Should not have created a Secret", func() {
			th.AssertSecretDoesNotExist(dataplaneSecretName)
		})
	})

	When("A ssh secret is created", func() {

		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateDataplaneNodeSet(dataplaneNodeSetName, DefaultDataPlaneNoNodeSetSpec()))
			CreateSSHSecret(dataplaneSSHSecretName)
		})
		It("Should have created a Secret", func() {
			secret := th.GetSecret(dataplaneSecretName)
			Expect(secret.Data["inventory"]).Should(
				ContainSubstring("edpm-compute-nodeset"))
		})
		It("Should set Input ready", func() {
			th.ExpectCondition(
				dataplaneNodeSetName,
				ConditionGetterFunc(DataplaneConditionGetter),
				condition.InputReadyCondition,
				corev1.ConditionTrue,
			)
		})
	})

	When("No default service image is provided", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateDataplaneNodeSet(dataplaneNodeSetName, DefaultDataPlaneNoNodeSetSpec()))
			CreateSSHSecret(dataplaneSSHSecretName)
		})
		It("Should have default service values provided", func() {
			secret := th.GetSecret(dataplaneSecretName)
			for _, svcImage := range defaultEdpmServiceList {
				Expect(secret.Data["inventory"]).Should(
					ContainSubstring(svcImage))
			}
		})
	})

	When("A user provides a custom service image", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateDataplaneNodeSet(dataplaneNodeSetName, CustomServiceImageSpec()))
			CreateSSHSecret(dataplaneSSHSecretName)
		})
		It("Should have the user defined image in the inventory", func() {
			secret := th.GetSecret(dataplaneSecretName)
			for _, svcAnsibleVar := range DefaultEdpmServiceAnsibleVarList {
				Expect(secret.Data["inventory"]).Should(
					ContainSubstring(fmt.Sprintf("%s.%s", svcAnsibleVar, CustomEdpmServiceDomainTag)))
			}
		})
	})

	When("No default service image is provided", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateDataplaneNodeSet(dataplaneNodeSetName, DefaultDataPlaneNoNodeSetSpec()))
			CreateSSHSecret(dataplaneSSHSecretName)
		})
		It("Should have default service values provided", func() {
			secret := th.GetSecret(dataplaneSecretName)
			for _, svcAnsibleVar := range DefaultEdpmServiceAnsibleVarList {
				Expect(secret.Data["inventory"]).Should(
					ContainSubstring(svcAnsibleVar))
			}
		})
	})

	When("A user provides a custom service image", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateDataplaneNodeSet(dataplaneNodeSetName, CustomServiceImageSpec()))
			CreateSSHSecret(dataplaneSSHSecretName)
		})
		It("Should have the user defined image in the inventory", func() {
			secret := th.GetSecret(dataplaneSecretName)
			for _, svcAnsibleVar := range DefaultEdpmServiceAnsibleVarList {
				Expect(secret.Data["inventory"]).Should(
					ContainSubstring(fmt.Sprintf("%s.%s", svcAnsibleVar, CustomEdpmServiceDomainTag)))
			}
		})
	})

	When("The nodeTemplate contains a ansibleUser but the individual node does not", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateNetConfig(dataplaneNetConfigName, DefaultNetConfigSpec()))
			DeferCleanup(th.DeleteInstance, CreateDataplaneNodeSet(dataplaneNodeSetName, DefaultDataPlaneNodeSetSpec()))
			CreateSSHSecret(dataplaneSSHSecretName)
			SimulateIPSetComplete(dataplaneIPSetName)
		})
		It("Should not have set the node specific ansible_user variable", func() {
			secret := th.GetSecret(dataplaneSecretName)
			secretData := secret.Data["inventory"]

			var inv AnsibleInventory
			err := yaml.Unmarshal(secretData, &inv)
			if err != nil {
				fmt.Printf("Error: %v", err)
			}
			Expect(inv.EdpmComputeNodeset.Vars.AnsibleUser).Should(Equal("cloud-user"))
			Expect(inv.EdpmComputeNodeset.Hosts.Node.AnsibleUser).Should(BeEmpty())
		})
	})

	When("The individual node has a AnsibleUser override", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateNetConfig(dataplaneNetConfigName, DefaultNetConfigSpec()))
			nodeOverrideSpec := map[string]interface{}{
				"hostname": "edpm-bm-compute-1",
				"networks": []map[string]interface{}{{
					"name":       "CtlPlane",
					"fixedIP":    "172.20.12.76",
					"subnetName": "ctlplane_subnet",
				},
				},
				"ansible": map[string]interface{}{
					"ansibleUser": "test-user",
				},
			}

			nodeTemplateOverrideSpec := map[string]interface{}{
				"ansibleSSHPrivateKeySecret": "dataplane-ansible-ssh-private-key-secret",
				"ansible": map[string]interface{}{
					"ansibleUser": "cloud-user",
				},
			}

			nodeSetSpec := DefaultDataPlaneNoNodeSetSpec()
			nodeSetSpec["nodes"].(map[string]interface{})["edpm-compute-node-1"] = nodeOverrideSpec
			nodeSetSpec["nodeTemplate"] = nodeTemplateOverrideSpec

			DeferCleanup(th.DeleteInstance, CreateDataplaneNodeSet(dataplaneNodeSetName, nodeSetSpec))
			CreateSSHSecret(dataplaneSSHSecretName)
			SimulateIPSetComplete(dataplaneIPSetName)
		})
		It("Should have a node specific override that is different to the group", func() {
			secret := th.GetSecret(dataplaneSecretName)
			secretData := secret.Data["inventory"]

			var inv AnsibleInventory
			err := yaml.Unmarshal(secretData, &inv)
			if err != nil {
				fmt.Printf("Error: %v", err)
			}
			Expect(inv.EdpmComputeNodeset.Hosts.Node.AnsibleUser).Should(Equal("test-user"))
			Expect(inv.EdpmComputeNodeset.Vars.AnsibleUser).Should(Equal("cloud-user"))
		})
	})

	When("A nodeSet is created with IPAM", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateNetConfig(dataplaneNetConfigName, DefaultNetConfigSpec()))
			DeferCleanup(th.DeleteInstance, CreateDataplaneNodeSet(dataplaneNodeSetName, DefaultDataPlaneNodeSetSpec()))
			CreateSSHSecret(dataplaneSSHSecretName)
			SimulateIPSetComplete(dataplaneIPSetName)
		})
		It("Should set the ctlplane_ip variable in the Ansible inventory secret", func() {
			Eventually(func() string {
				secret := th.GetSecret(dataplaneSecretName)
				return getCtlPlaneIP(&secret)
			}).Should(Equal("172.20.12.76"))
		})
	})
})
