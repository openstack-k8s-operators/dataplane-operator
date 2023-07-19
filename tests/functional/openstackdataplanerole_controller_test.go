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

var _ = Describe("Dataplane Role Test", func() {
	var dataplaneRoleName types.NamespacedName

	BeforeEach(func() {
		dataplaneRoleName = types.NamespacedName{
			Name:      "edpm-compute-no-nodes",
			Namespace: namespace,
		}
		err := os.Setenv("OPERATOR_TEMPLATES", "../../templates")
		Expect(err).NotTo(HaveOccurred())
	})

	When("A Dataplane resorce is created", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateDataplaneRoleNoNodes(dataplaneRoleName))
		})
		It("should have the Spec fields initialized", func() {
			dataplaneRoleInstance := GetDataplaneRole(dataplaneRoleName)
			Expect(dataplaneRoleInstance.Spec.DeployStrategy.AnsibleTags).Should(BeNil())
		})

		It("should have the Status fields initialized", func() {
			dataplaneRoleInstance := GetDataplaneRole(dataplaneRoleName)
			Expect(dataplaneRoleInstance.Status.Deployed).Should(BeFalse())
		})
	})
})
