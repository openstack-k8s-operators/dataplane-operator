package functional

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	dataplanev1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	. "github.com/openstack-k8s-operators/lib-common/modules/common/test/helpers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Dataplane Deployment Test", func() {
	var dataplaneDeploymentName types.NamespacedName
	var dataplaneNodeSetName types.NamespacedName
	var dataplaneSSHSecretName types.NamespacedName

	BeforeEach(func() {
		dataplaneDeploymentName = types.NamespacedName{
			Name:      "edpm-deployment",
			Namespace: namespace,
		}
		dataplaneNodeSetName = types.NamespacedName{
			Name:      "edpm-compute-nodeset",
			Namespace: namespace,
		}
		dataplaneSSHSecretName = types.NamespacedName{
			Namespace: namespace,
			Name:      "dataplane-ansible-ssh-private-key-secret",
		}
		err := os.Setenv("OPERATOR_SERVICES", "../../config/services")
		Expect(err).NotTo(HaveOccurred())
	})

	When("A dataplaneDeployment is created with matching NodeSet", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateDataplaneNodeSet(dataplaneNodeSetName, DefaultDataPlaneNoNodeSetSpec()))
			DeferCleanup(th.DeleteInstance, CreateDataplaneDeployment(dataplaneDeploymentName, DefaultDataPlaneDeploymentSpec()))
			CreateSSHSecret(dataplaneSSHSecretName)
		})

		It("Should have Spec fields initialized", func() {
			dataplaneDeploymentInstance := GetDataplaneDeployment(dataplaneDeploymentName)
			expectedSpec := dataplanev1.OpenStackDataPlaneDeploymentSpec{
				NodeSets:        []string{"edpm-compute-nodeset"},
				AnsibleTags:     "",
				AnsibleLimit:    "",
				AnsibleSkipTags: "",
			}
			Expect(dataplaneDeploymentInstance.Spec).Should(Equal(expectedSpec))
		})

		It("should have conditions set", func() {
			th.ExpectCondition(
				dataplaneDeploymentName,
				ConditionGetterFunc(DataplaneDeploymentConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionFalse,
			)
			th.ExpectCondition(
				dataplaneDeploymentName,
				ConditionGetterFunc(DataplaneDeploymentConditionGetter),
				condition.InputReadyCondition,
				corev1.ConditionTrue,
			)
			th.ExpectCondition(
				dataplaneDeploymentName,
				ConditionGetterFunc(DataplaneDeploymentConditionGetter),
				dataplanev1.SetupReadyCondition,
				corev1.ConditionTrue,
			)
		})
	})
})
