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
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Dataplane Test", func() {
	var dataplaneName types.NamespacedName
	var dataplaneRoleName types.NamespacedName

	BeforeEach(func() {
		dataplaneName = types.NamespacedName{
			Name:      "dataplane-test",
			Namespace: namespace,
		}
		dataplaneRoleName = types.NamespacedName{
			Name:      "edpm-compute-no-nodes",
			Namespace: namespace,
		}
		err := os.Setenv("OPERATOR_TEMPLATES", "../../templates")
		Expect(err).NotTo(HaveOccurred())
	})

	When("A Dataplane resorce is created", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateDataPlane(dataplaneName, DefaultDataPlaneSpec()))
		})
		It("should have the Spec fields initialized", func() {
			dataplaneInstance := GetDataplane(dataplaneName)
			Expect(dataplaneInstance.Spec.DeployStrategy.Deploy).Should(BeFalse())
		})

		It("should have the Status fields initialized", func() {
			dataplaneInstance := GetDataplane(dataplaneName)
			Expect(dataplaneInstance.Status.Deployed).Should(BeFalse())
		})

		It("Should have created a OpenStackDataplaneRole", func() {
			dataplaneRoleInstance := GetDataplaneRole(dataplaneRoleName)
			Expect(dataplaneRoleInstance).NotTo(BeNil())
			Expect(dataplaneRoleInstance.Spec.DeployStrategy.Deploy).Should(BeFalse())
			Expect(dataplaneRoleInstance.Spec.NodeTemplate.AnsibleSSHPrivateKeySecret).Should(Equal("dataplane-ansible-ssh-private-key-secret"))
		})
	})
	When("A dataplane resource is created with nic-config overrides", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateDataPlane(dataplaneName, DataPlaneWithNodeSpec()))
			CreateSSHSecret(dataplaneName.Namespace, dataplaneName.Name)
		})
		It("Should have a configMap with ansibleVars for 'edpm_network_config_template'", func() {
			dataplaneInstanceCM := th.GetConfigMap(types.NamespacedName{Namespace: namespace, Name: "dataplanenode-edpm-compute-0"})
			Expect(dataplaneInstanceCM.Data["inventory"]).Should(ContainSubstring("network_config"))
		})
	})
})
