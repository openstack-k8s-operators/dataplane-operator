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
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	dataplanev1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	. "github.com/openstack-k8s-operators/lib-common/modules/common/test/helpers"
	"github.com/openstack-k8s-operators/openstack-baremetal-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Dataplane NodeSet Test", func() {
	var dataplaneNodeSetName types.NamespacedName
	var dataplaneSecretName types.NamespacedName
	var dataplaneSSHSecretName types.NamespacedName

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
					NetworkConfig: dataplanev1.NetworkConfigSection{
						Template: "",
					},
					Networks:          nil,
					ManagementNetwork: "",
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
					"download-cache",
					"configure-network",
					"validate-network",
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
	})
})
