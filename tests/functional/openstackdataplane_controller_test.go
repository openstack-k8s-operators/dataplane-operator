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
	var dataplaneNodeSetName types.NamespacedName

	BeforeEach(func() {
		dataplaneName = types.NamespacedName{
			Name:      "dataplane-test",
			Namespace: namespace,
		}
		dataplaneNodeSetName = types.NamespacedName{
			Name:      "edpm-compute-node-set",
			Namespace: namespace,
		}
		err := os.Setenv("OPERATOR_TEMPLATES", "../../templates")
		Expect(err).NotTo(HaveOccurred())
	})

	When("A Dataplane resorce is created", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateDataplaneNodeSet(dataplaneNodeSetName, DefaultDataPlaneNodeSetSpec()))
			DeferCleanup(th.DeleteInstance, CreateDataPlane(dataplaneName, DefaultDataPlaneSpec()))
		})
		It("should have the Spec fields initialized", func() {
			dataplaneInstance := GetDataplane(dataplaneName)
			Expect(dataplaneInstance.Spec.DeployStrategy.Deploy).Should(BeTrue())
		})

		It("should have the Status fields initialized", func() {
			dataplaneInstance := GetDataplane(dataplaneName)
			Expect(dataplaneInstance.Status.Deployed).Should(BeFalse())
		})

		It("Should have put a label on the OpenStackDataPlaneNodeSet CR", func() {
			dataplaneNodeSet := GetDataplaneNodeSet(dataplaneNodeSetName)
			Expect(dataplaneNodeSet.ObjectMeta.Annotations["edpm.openstack.org/deploy"]).Should(Equal("true"))
		})
	})
})
