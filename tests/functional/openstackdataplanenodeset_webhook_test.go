package functional

import (
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

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
			nodeSetSpec["nodes"] = map[string]interface{}{
				"compute-0": map[string]interface{}{
					"hostName": "compute-0"},
			}
			nodeSetSpec["baremetalSetTemplate"] = baremetalv1.OpenStackBaremetalSetSpec{
				CloudUserName: "test-user",
				BmhLabelSelector: map[string]string{
					"app": "test-openstack",
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

	When("A user tries to redeclare an existing node in a new NodeSet", func() {
		BeforeEach(func() {
			nodeSetSpec := DefaultDataPlaneNoNodeSetSpec()
			nodeSetSpec["preProvisioned"] = true
			nodeSetSpec["nodes"] = map[string]interface{}{
				"compute-0": map[string]interface{}{
					"hostName": "compute-0"},
			}
			DeferCleanup(th.DeleteInstance, CreateDataplaneNodeSet(dataplaneNodeSetName, nodeSetSpec))
		})

		It("Should block duplicate node declaration", func() {
			Eventually(func(g Gomega) string {
				newNodeSetSpec := DefaultDataPlaneNoNodeSetSpec()
				newNodeSetSpec["preProvisioned"] = true
				newNodeSetSpec["nodes"] = map[string]interface{}{
					"compute-0": map[string]interface{}{
						"hostName": "compute-0"},
				}
				newInstance := DefaultDataplaneNodeSetTemplate(types.NamespacedName{Name: "test-duplicate-node", Namespace: namespace}, newNodeSetSpec)
				unstructuredObj := &unstructured.Unstructured{Object: newInstance}
				_, err := controllerutil.CreateOrPatch(
					th.Ctx, th.K8sClient, unstructuredObj, func() error { return nil })
				return fmt.Sprintf("%s", err)
			}).Should(ContainSubstring("node already exists"))
		})

		It("Should block NodeSets if they contain a duplicate ansibleHost", func() {
			Eventually(func(g Gomega) string {
				newNodeSetSpec := DefaultDataPlaneNoNodeSetSpec()
				newNodeSetSpec["preProvisioned"] = true
				newNodeSetSpec["nodes"] = map[string]interface{}{
					"compute-3": map[string]interface{}{
						"hostName": "compute-3",
						"ansible": map[string]interface{}{
							"ansibleHost": "compute-3",
						},
					},
					"compute-2": map[string]interface{}{
						"hostName": "compute-2"},
					"compute-8": map[string]interface{}{
						"hostName": "compute-8"},
					"compute-0": map[string]interface{}{
						"ansible": map[string]interface{}{
							"ansibleHost": "compute-0",
						},
					},
				}
				newInstance := DefaultDataplaneNodeSetTemplate(types.NamespacedName{Name: "test-nodeset-with-duplicate-node", Namespace: namespace}, newNodeSetSpec)
				unstructuredObj := &unstructured.Unstructured{Object: newInstance}
				_, err := controllerutil.CreateOrPatch(
					th.Ctx, th.K8sClient, unstructuredObj, func() error { return nil })
				return fmt.Sprintf("%s", err)
			}).Should(ContainSubstring("node already exists"))
		})
	})
})
