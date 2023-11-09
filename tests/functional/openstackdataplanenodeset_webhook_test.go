package functional

import (
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"

	baremetalv1 "github.com/openstack-k8s-operators/openstack-baremetal-operator/api/v1beta1"
)

var _ = Describe("DataplaneNodeSet Webhook", func() {

	var dataplaneNodeSetName types.NamespacedName

	BeforeEach(func() {
		dataplaneNodeSetName = types.NamespacedName{
			Name:      "edpm-compute-nodeset",
			Namespace: namespace,
		}
		err := os.Setenv("OPERATOR_SERVICES", "../../config/services")
		Expect(err).NotTo(HaveOccurred())
	})

	When("User tries to change forbidden items in the baremetalSetTemplate", func() {
		BeforeEach(func() {
			nodeSetSpec := DefaultDataPlaneNoNodeSetSpec()
			nodeSetSpec["preProvisioned"] = false
			nodeSetSpec["baremetalSetTemplate"] = baremetalv1.OpenStackBaremetalSetSpec{
				CloudUserName: "test-user",
				BmhLabelSelector: map[string]string{
					"app": "test-openstack",
				},
				BaremetalHosts: map[string]baremetalv1.InstanceSpec{
					"compute-0": {
						CtlPlaneIP: "192.168.1.12",
					},
				},
			}
			DeferCleanup(th.DeleteInstance, CreateDataplaneNodeSet(dataplaneNodeSetName, nodeSetSpec))
		})

		It("Should block changes to the BmhLabelSelector object in baremetalSetTemplate spec", func() {
			Eventually(func(g Gomega) string {
				instance := GetDataplaneNodeSet(dataplaneNodeSetName)
				instance.Spec.BaremetalSetTemplate = baremetalv1.OpenStackBaremetalSetSpec{
					CloudUserName: "new-user",
					BmhLabelSelector: map[string]string{
						"app": "openstack1",
					},
				}
				err := th.K8sClient.Update(th.Ctx, instance)
				return fmt.Sprintf("%s", err)
			}).Should(ContainSubstring("Forbidden: cannot change"))
		})
	})

	When("A user changes an allowed field in the baremetalSetTemplate", func() {
		BeforeEach(func() {
			nodeSetSpec := DefaultDataPlaneNoNodeSetSpec()
			nodeSetSpec["preProvisioned"] = false
			nodeSetSpec["baremetalSetTemplate"] = baremetalv1.OpenStackBaremetalSetSpec{
				CloudUserName: "test-user",
				BmhLabelSelector: map[string]string{
					"app": "test-openstack",
				},
				BaremetalHosts: map[string]baremetalv1.InstanceSpec{
					"compute-0": {
						CtlPlaneIP: "192.168.1.12",
					},
				},
			}
			DeferCleanup(th.DeleteInstance, CreateDataplaneNodeSet(dataplaneNodeSetName, nodeSetSpec))
		})

		It("Should allow changes to the CloudUserName", func() {
			Eventually(func(g Gomega) error {
				instance := GetDataplaneNodeSet(dataplaneNodeSetName)
				instance.Spec.BaremetalSetTemplate = baremetalv1.OpenStackBaremetalSetSpec{
					CloudUserName: "new-user",
					BmhLabelSelector: map[string]string{
						"app": "test-openstack",
					},
					BaremetalHosts: map[string]baremetalv1.InstanceSpec{
						"compute-0": {
							CtlPlaneIP: "192.168.1.12",
						},
					},
				}
				return th.K8sClient.Update(th.Ctx, instance)
			}).Should(Succeed())
		})
	})
})
