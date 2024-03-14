package functional

import (
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2" //revive:disable:dot-imports
	. "github.com/onsi/gomega"    //revive:disable:dot-imports
	dataplanev1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"

	//revive:disable-next-line:dot-imports
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
	var dataplaneMultiNodesetDeploymentName types.NamespacedName
	var dataplaneServiceName types.NamespacedName
	var dataplaneGlobalServiceName types.NamespacedName

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
		dataplaneMultiNodesetDeploymentName = types.NamespacedName{
			Namespace: namespace,
			Name:      "edpm-compute-nodeset-global",
		}
		dataplaneServiceName = types.NamespacedName{
			Namespace: namespace,
			Name:      "foo-service",
		}
		dataplaneGlobalServiceName = types.NamespacedName{
			Name:      "global-service",
			Namespace: namespace,
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
			// DefaultDataPlanenodeSetSpec comes with two mock services, one marked for deployment on all nodesets
			CreateDataplaneService(dataplaneServiceName, false)
			CreateDataplaneService(dataplaneGlobalServiceName, true)

			DeferCleanup(th.DeleteService, dataplaneServiceName)
			DeferCleanup(th.DeleteService, dataplaneGlobalServiceName)
			DeferCleanup(th.DeleteInstance, CreateNetConfig(dataplaneNetConfigName, DefaultNetConfigSpec()))
			DeferCleanup(th.DeleteInstance, CreateDataplaneNodeSet(dataplaneNodeSetName, DefaultDataPlaneNodeSetSpec(dataplaneNodeSetName.Name)))
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
					ansibleEE := GetAnsibleeeByLabel(dataplaneNodeSetName.Name, dataplaneDeploymentName, dataplaneServiceName.Name)
					ansibleEE.Status.JobStatus = ansibleeev1.JobStatusSucceeded

					g.Expect(th.K8sClient.Status().Update(th.Ctx, ansibleEE)).To(Succeed())
					g.Expect(ansibleEE.Spec.ExtraVars).To(HaveKey("edpm_override_hosts"))
					if service.Spec.DeployOnAllNodeSets {
						g.Expect(string(ansibleEE.Spec.ExtraVars["edpm_override_hosts"])).To(Equal("\"all\""))
					} else {
						g.Expect(string(ansibleEE.Spec.ExtraVars["edpm_override_hosts"])).To(Equal(fmt.Sprintf("\"%s\"", dataplaneNodeSetName.Name)))
					}
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

	When("A dataplaneDeployment is created with two NodeSets", func() {
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

			alphaNodeSetName := types.NamespacedName{
				Name:      "alpha-nodeset",
				Namespace: namespace,
			}
			betaNodeSetName := types.NamespacedName{
				Name:      "beta-nodeset",
				Namespace: namespace,
			}

			// Two services on both nodesets
			CreateDataplaneService(dataplaneServiceName, false)
			CreateDataplaneService(dataplaneGlobalServiceName, true)

			DeferCleanup(th.DeleteService, dataplaneServiceName)
			DeferCleanup(th.DeleteService, dataplaneGlobalServiceName)

			DeferCleanup(th.DeleteInstance, CreateNetConfig(dataplaneNetConfigName, DefaultNetConfigSpec()))

			// Create both nodesets

			betaNodeSetSpec := map[string]interface{}{
				"preProvisioned": false,
				"services": []string{
					"foo-service",
				},
				"nodeTemplate": map[string]interface{}{
					"ansibleSSHPrivateKeySecret": "dataplane-ansible-ssh-private-key-secret",
					"ansible": map[string]interface{}{
						"ansibleUser": "cloud-user",
					},
				},
				"nodes": map[string]interface{}{
					fmt.Sprintf("%s-node-1", betaNodeSetName.Name): map[string]interface{}{
						"hostname": "edpm-bm-compute-2",
						"networks": []map[string]interface{}{{
							"name":       "CtlPlane",
							"fixedIP":    "172.20.12.77",
							"subnetName": "ctlplane_subnet",
						},
						},
					},
				},
				"baremetalSetTemplate": map[string]interface{}{
					"baremetalHosts": map[string]interface{}{
						"ctlPlaneIP": map[string]interface{}{},
					},
					"deploymentSSHSecret": "dataplane-ansible-ssh-private-key-secret",
					"ctlplaneInterface":   "172.20.12.1",
				},
				"tlsEnabled": true,
			}
			DeferCleanup(th.DeleteInstance, CreateDataplaneNodeSet(alphaNodeSetName, DefaultDataPlaneNodeSetSpec(alphaNodeSetName.Name)))
			DeferCleanup(th.DeleteInstance, CreateDataplaneNodeSet(betaNodeSetName, betaNodeSetSpec))

			deploymentSpec := map[string]interface{}{
				"nodeSets": []string{
					"alpha-nodeset",
					"beta-nodeset",
				},
			}
			DeferCleanup(th.DeleteInstance, CreateDataplaneDeployment(dataplaneMultiNodesetDeploymentName, deploymentSpec))
		})

		It("Should have Spec fields initialized", func() {
			dataplaneDeploymentInstance := GetDataplaneDeployment(dataplaneMultiNodesetDeploymentName)
			nodeSetsNames := []string{
				"alpha-nodeset",
				"beta-nodeset",
			}

			expectedSpec := dataplanev1.OpenStackDataPlaneDeploymentSpec{
				NodeSets:              nodeSetsNames,
				AnsibleTags:           "",
				AnsibleLimit:          "",
				AnsibleSkipTags:       "",
				DeploymentRequeueTime: 15,
			}
			Expect(dataplaneDeploymentInstance.Spec).Should(Equal(expectedSpec))
		})

		It("should have conditions set", func() {
			alphaNodeSetName := types.NamespacedName{
				Name:      "alpha-nodeset",
				Namespace: namespace,
			}
			betaNodeSetName := types.NamespacedName{
				Name:      "beta-nodeset",
				Namespace: namespace,
			}

			baremetalAlpha := baremetalv1.OpenStackBaremetalSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      alphaNodeSetName.Name,
					Namespace: alphaNodeSetName.Namespace,
				},
			}

			baremetalBeta := baremetalv1.OpenStackBaremetalSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      betaNodeSetName.Name,
					Namespace: betaNodeSetName.Namespace,
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

			nodeSetAlpha := *GetDataplaneNodeSet(alphaNodeSetName)
			nodeSetBeta := *GetDataplaneNodeSet(betaNodeSetName)

			// Set baremetal provisioning conditions to True
			Eventually(func(g Gomega) {
				// OpenStackBaremetalSet has the same name as OpenStackDataPlaneNodeSet
				g.Expect(th.K8sClient.Get(th.Ctx, alphaNodeSetName, &baremetalAlpha)).To(Succeed())
				baremetalAlpha.Status.Conditions.MarkTrue(
					baremetalv1.OpenStackBaremetalSetBmhProvisioningReadyCondition,
					baremetalv1.OpenStackBaremetalSetBmhProvisioningReadyMessage)
				g.Expect(th.K8sClient.Status().Update(th.Ctx, &baremetalAlpha)).To(Succeed())
				// OpenStackBaremetalSet has the same name as OpenStackDataPlaneNodeSet
				g.Expect(th.K8sClient.Get(th.Ctx, betaNodeSetName, &baremetalBeta)).To(Succeed())
				baremetalBeta.Status.Conditions.MarkTrue(
					baremetalv1.OpenStackBaremetalSetBmhProvisioningReadyCondition,
					baremetalv1.OpenStackBaremetalSetBmhProvisioningReadyMessage)
				g.Expect(th.K8sClient.Status().Update(th.Ctx, &baremetalBeta)).To(Succeed())

			}, th.Timeout, th.Interval).Should(Succeed())

			// Create all services necessary for deployment
			for _, serviceName := range nodeSetAlpha.Spec.Services {
				dataplaneServiceName := types.NamespacedName{
					Name:      serviceName,
					Namespace: namespace,
				}
				service := GetService(dataplaneServiceName)
				//Retrieve service AnsibleEE and set JobStatus to Successful
				Eventually(func(g Gomega) {
					ansibleEE := GetAnsibleeeByLabel(
						alphaNodeSetName.Name, dataplaneMultiNodesetDeploymentName, dataplaneServiceName.Name)
					if service.Spec.DeployOnAllNodeSets {
						g.Expect(ansibleEE.Spec.ExtraMounts[0].Volumes).Should(HaveLen(4))
					} else {
						g.Expect(ansibleEE.Spec.ExtraMounts[0].Volumes).Should(HaveLen(2))
					}
					ansibleEE.Status.JobStatus = ansibleeev1.JobStatusSucceeded
					g.Expect(th.K8sClient.Status().Update(th.Ctx, ansibleEE)).To(Succeed())
					if service.Spec.DeployOnAllNodeSets {
						g.Expect(string(ansibleEE.Spec.ExtraVars["edpm_override_hosts"])).To(Equal("\"all\""))
					}
				}, th.Timeout, th.Interval).Should(Succeed())
			}

			// Create all services necessary for deployment
			for _, serviceName := range nodeSetBeta.Spec.Services {
				dataplaneServiceName := types.NamespacedName{
					Name:      serviceName,
					Namespace: namespace,
				}
				service := GetService(dataplaneServiceName)
				//Retrieve service AnsibleEE and set JobStatus to Successful
				Eventually(func(g Gomega) {
					ansibleEE := GetAnsibleeeByLabel(
						betaNodeSetName.Name, dataplaneMultiNodesetDeploymentName, dataplaneServiceName.Name)
					if service.Spec.DeployOnAllNodeSets {
						g.Expect(ansibleEE.Spec.ExtraMounts[0].Volumes).Should(HaveLen(4))
					} else {
						g.Expect(ansibleEE.Spec.ExtraMounts[0].Volumes).Should(HaveLen(2))
					}
					ansibleEE.Status.JobStatus = ansibleeev1.JobStatusSucceeded
					g.Expect(th.K8sClient.Status().Update(th.Ctx, ansibleEE)).To(Succeed())
				}, th.Timeout, th.Interval).Should(Succeed())
			}

			th.ExpectCondition(
				dataplaneMultiNodesetDeploymentName,
				ConditionGetterFunc(DataplaneDeploymentConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionTrue,
			)
			th.ExpectCondition(
				dataplaneMultiNodesetDeploymentName,
				ConditionGetterFunc(DataplaneDeploymentConditionGetter),
				condition.InputReadyCondition,
				corev1.ConditionTrue,
			)
		})
	})
})
