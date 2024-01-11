package functional

import (
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	dataplanev1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	. "github.com/openstack-k8s-operators/lib-common/modules/common/test/helpers"
	ansibleeev1 "github.com/openstack-k8s-operators/openstack-ansibleee-operator/api/v1beta1"
	baremetalv1 "github.com/openstack-k8s-operators/openstack-baremetal-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Dataplane Deployment Test", func() {
	var dataplaneDeploymentName types.NamespacedName
	var dataplaneNodeSetName types.NamespacedName
	var dataplaneSSHSecretName types.NamespacedName
	var neutronOvnMetadataSecretName types.NamespacedName
	var novaNeutronMetadataSecretName types.NamespacedName
	var novaCellComputeConfigSecretName types.NamespacedName
	var novaMigrationSSHKey types.NamespacedName
	var ceilometerConfigSecretName types.NamespacedName
	var dataplaneNetConfigName types.NamespacedName

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
		neutronOvnMetadataSecretName = types.NamespacedName{
			Namespace: namespace,
			Name:      "neutron-ovn-metadata-agent-neutron-config",
		}
		novaNeutronMetadataSecretName = types.NamespacedName{
			Namespace: namespace,
			Name:      "nova-metadata-neutron-config",
		}
		novaCellComputeConfigSecretName = types.NamespacedName{
			Namespace: namespace,
			Name:      "nova-cell1-compute-config",
		}
		novaMigrationSSHKey = types.NamespacedName{
			Namespace: namespace,
			Name:      "nova-migration-ssh-key",
		}
		ceilometerConfigSecretName = types.NamespacedName{
			Namespace: namespace,
			Name:      "ceilometer-compute-config-data",
		}
		dataplaneNetConfigName = types.NamespacedName{
			Namespace: namespace,
			Name:      "dataplane-netconfig",
		}
		err := os.Setenv("OPERATOR_SERVICES", "../../config/services")
		Expect(err).NotTo(HaveOccurred())
	})

	When("A dataplaneDeployment is created with matching NodeSet", func() {
		BeforeEach(func() {
			CreateSSHSecret(dataplaneSSHSecretName)
			DeferCleanup(th.DeleteInstance, th.CreateSecret(neutronOvnMetadataSecretName, map[string][]byte{
				"fake_keys": []byte("blih"),
			}))
			DeferCleanup(th.DeleteInstance, th.CreateSecret(novaNeutronMetadataSecretName, map[string][]byte{
				"fake_keys": []byte("blih"),
			}))
			DeferCleanup(th.DeleteInstance, th.CreateSecret(novaCellComputeConfigSecretName, map[string][]byte{
				"fake_keys": []byte("blih"),
			}))
			DeferCleanup(th.DeleteInstance, th.CreateSecret(novaMigrationSSHKey, map[string][]byte{
				"ssh-privatekey": []byte("fake-ssh-private-key"),
				"ssh-publickey":  []byte("fake-ssh-public-key"),
			}))
			DeferCleanup(th.DeleteInstance, th.CreateSecret(ceilometerConfigSecretName, map[string][]byte{
				"fake_keys": []byte("blih"),
			}))
			DeferCleanup(th.DeleteInstance, CreateNetConfig(dataplaneNetConfigName, DefaultNetConfigSpec()))
			DeferCleanup(th.DeleteInstance, CreateDataplaneNodeSet(dataplaneNodeSetName, DefaultDataPlaneNodeSetSpec()))
			DeferCleanup(th.DeleteInstance, CreateDataplaneDeployment(dataplaneDeploymentName, DefaultDataPlaneDeploymentSpec()))
		})

		It("Should have Spec fields initialized", func() {
			dataplaneDeploymentInstance := GetDataplaneDeployment(dataplaneDeploymentName)
			expectedSpec := dataplanev1.OpenStackDataPlaneDeploymentSpec{
				NodeSets:              []string{"edpm-compute-nodeset"},
				AnsibleTags:           "",
				AnsibleLimit:          "",
				AnsibleSkipTags:       "",
				DeploymentRequeueTime: 15,
			}
			Expect(dataplaneDeploymentInstance.Spec).Should(Equal(expectedSpec))
		})

		It("should have conditions set", func() {

			nodeSet := dataplanev1.OpenStackDataPlaneNodeSet{}
			baremetal := baremetalv1.OpenStackBaremetalSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      nodeSet.Name,
					Namespace: nodeSet.Namespace,
				},
			}
			// Create config map for OVN service
			ovnConfigMapName := types.NamespacedName{
				Namespace: namespace,
				Name:      "ovncontroller-config",
			}
			mapData := map[string]interface{}{
				"ovsdb-config": "test-ovn-config",
			}
			th.CreateConfigMap(ovnConfigMapName, mapData)

			nodeSet = *GetDataplaneNodeSet(dataplaneNodeSetName)

			// Set baremetal provisioning conditions to True
			Eventually(func(g Gomega) {
				// OpenStackBaremetalSet has the same name as OpenStackDataPlaneNodeSet
				g.Expect(th.K8sClient.Get(th.Ctx, dataplaneNodeSetName, &baremetal)).To(Succeed())
				baremetal.Status.Conditions.MarkTrue(
					baremetalv1.OpenStackBaremetalSetBmhProvisioningReadyCondition,
					baremetalv1.OpenStackBaremetalSetBmhProvisioningReadyMessage)
				g.Expect(th.K8sClient.Status().Update(th.Ctx, &baremetal)).To(Succeed())

			}, th.Timeout, th.Interval).Should(Succeed())

			// Create all services necessary for deployment
			for _, serviceName := range nodeSet.Spec.Services {
				dataplaneServiceName := types.NamespacedName{
					Name:      serviceName,
					Namespace: namespace,
				}
				service := GetService(dataplaneServiceName)
				//Retrieve service AnsibleEE and set JobStatus to Successful
				Eventually(func(g Gomega) {
					// Make an AnsibleEE name for each service
					ansibleeeName := types.NamespacedName{
						Name:      fmt.Sprintf("%s-%s", service.Spec.Label, dataplaneDeploymentName.Name),
						Namespace: dataplaneDeploymentName.Namespace,
					}
					ansibleEE := &ansibleeev1.OpenStackAnsibleEE{
						ObjectMeta: metav1.ObjectMeta{
							Name:      ansibleeeName.Name,
							Namespace: ansibleeeName.Namespace,
						}}
					g.Expect(th.K8sClient.Get(th.Ctx, ansibleeeName, ansibleEE)).To(Succeed())
					ansibleEE.Status.JobStatus = ansibleeev1.JobStatusSucceeded

					g.Expect(th.K8sClient.Status().Update(th.Ctx, ansibleEE)).To(Succeed())
				}, th.Timeout, th.Interval).Should(Succeed())
			}

			th.ExpectCondition(
				dataplaneDeploymentName,
				ConditionGetterFunc(DataplaneDeploymentConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionTrue,
			)
			th.ExpectCondition(
				dataplaneDeploymentName,
				ConditionGetterFunc(DataplaneDeploymentConditionGetter),
				condition.InputReadyCondition,
				corev1.ConditionTrue,
			)
		})
	})
})
