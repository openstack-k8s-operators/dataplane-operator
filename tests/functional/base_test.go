package functional

import (
	"fmt"

	. "github.com/onsi/gomega" //revive:disable:dot-imports
	dataplanev1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	infrav1 "github.com/openstack-k8s-operators/infra-operator/apis/network/v1beta1"
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/openstack-ansibleee-operator/api/v1beta1"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var DefaultEdpmServiceAnsibleVarList = []string{
	"edpm_frr_image",
	"edpm_iscsid_image",
	"edpm_logrotate_crond_image",
	"edpm_neutron_metadata_agent_image",
	"edpm_nova_compute_image",
	"edpm_ovn_controller_agent_image",
	"edpm_ovn_bgp_agent_image",
}

var CustomEdpmServiceDomainTag = "test-image:latest"

// Create OpenstackDataPlaneNodeSet in k8s and test that no errors occur
func CreateDataplaneNodeSet(name types.NamespacedName, spec map[string]interface{}) *unstructured.Unstructured {
	instance := DefaultDataplaneNodeSetTemplate(name, spec)
	return th.CreateUnstructured(instance)
}

// Create OpenStackDataPlaneDeployment in k8s and test that no errors occur
func CreateDataplaneDeployment(name types.NamespacedName, spec map[string]interface{}) *unstructured.Unstructured {
	instance := DefaultDataplaneDeploymentTemplate(name, spec)
	return th.CreateUnstructured(instance)
}

// Create an OpenStackDataPlaneService with a given NamespacedName, assert on success
func CreateDataplaneService(name types.NamespacedName, globalService bool) *unstructured.Unstructured {
	var raw map[string]interface{}
	if globalService {
		raw = DefaultDataplaneGlobalService(name)
	} else {
		raw = DefaultDataplaneService(name)
	}
	return th.CreateUnstructured(raw)
}

// Build CustomServiceImageSpec struct with empty `Nodes` list
func CustomServiceImageSpec() map[string]interface{} {

	var ansibleServiceVars = make(map[string]interface{})
	for _, svcName := range DefaultEdpmServiceAnsibleVarList {
		imageAddress := fmt.Sprintf(`"%s.%s"`, svcName, CustomEdpmServiceDomainTag)
		ansibleServiceVars[svcName] = imageAddress
	}

	return map[string]interface{}{
		"preProvisioned": true,
		"nodeTemplate": map[string]interface{}{
			"ansibleSSHPrivateKeySecret": "dataplane-ansible-ssh-private-key-secret",
			"ansible": map[string]interface{}{
				"ansibleVars": ansibleServiceVars,
			},
		},
		"nodes": map[string]interface{}{},
	}
}

func CreateNetConfig(name types.NamespacedName, spec map[string]interface{}) *unstructured.Unstructured {
	raw := DefaultNetConfig(name, spec)
	return th.CreateUnstructured(raw)
}

// Create SSHSecret
func CreateSSHSecret(name types.NamespacedName) *corev1.Secret {
	return th.CreateSecret(
		types.NamespacedName{Namespace: name.Namespace, Name: name.Name},
		map[string][]byte{
			"ssh-privatekey":  []byte("blah"),
			"authorized_keys": []byte("blih"),
		},
	)
}

// Struct initialization

// Build OpenStackDataPlaneNodeSetSpec struct and fill it with preset values
func DefaultDataPlaneNodeSetSpec(nodeSetName string) map[string]interface{} {

	return map[string]interface{}{
		"preProvisioned": false,
		"services": []string{
			"foo-service",
			"global-service",
		},
		"nodeTemplate": map[string]interface{}{
			"ansibleSSHPrivateKeySecret": "dataplane-ansible-ssh-private-key-secret",
			"ansible": map[string]interface{}{
				"ansibleUser": "cloud-user",
			},
		},
		"nodes": map[string]interface{}{
			fmt.Sprintf("%s-node-1", nodeSetName): map[string]interface{}{
				"hostname": "edpm-bm-compute-1",
				"networks": []map[string]interface{}{{
					"name":       "CtlPlane",
					"fixedIP":    "172.20.12.76",
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
}

// Build OpenStackDataPlaneNodeSetSpec struct with empty `Nodes` list
func DefaultDataPlaneNoNodeSetSpec(tlsEnabled bool) map[string]interface{} {
	spec := map[string]interface{}{
		"preProvisioned": true,
		"nodeTemplate": map[string]interface{}{
			"ansibleSSHPrivateKeySecret": "dataplane-ansible-ssh-private-key-secret",
		},
		"nodes": map[string]interface{}{},
	}
	if tlsEnabled {
		spec["tlsEnabled"] = true
	}
	return spec
}

// Build OpenStackDataPlnaeDeploymentSpec and fill it with preset values
func DefaultDataPlaneDeploymentSpec() map[string]interface{} {

	return map[string]interface{}{
		"nodeSets": []string{
			"edpm-compute-nodeset",
		},
	}
}

func DefaultNetConfigSpec() map[string]interface{} {
	return map[string]interface{}{
		"networks": []map[string]interface{}{{
			"dnsDomain": "test-domain.test",
			"mtu":       1500,
			"name":      "CtlPLane",
			"subnets": []map[string]interface{}{{
				"allocationRanges": []map[string]interface{}{{
					"end":   "172.20.12.120",
					"start": "172.20.12.0",
				},
				},
				"name":    "ctlplane_subnet",
				"cidr":    "172.20.12.0/16",
				"gateway": "172.20.12.1",
			},
			},
		},
		},
	}
}

// SimulateIPSetComplete - Simulates the result of the IPSet status
func SimulateIPSetComplete(name types.NamespacedName) {
	Eventually(func(g Gomega) {
		IPSet := &infrav1.IPSet{}
		g.Expect(th.K8sClient.Get(th.Ctx, name, IPSet)).Should(Succeed())
		gateway := "172.20.12.1"
		IPSet.Status.Reservation = []infrav1.IPSetReservation{
			{
				Address: "172.20.12.76",
				Cidr:    "172.20.12.0/16",
				MTU:     1500,
				Network: "CtlPlane",
				Subnet:  "subnet1",
				Gateway: &gateway,
			},
		}
		// This can return conflict so we have the gomega.Eventually block to retry
		g.Expect(th.K8sClient.Status().Update(th.Ctx, IPSet)).To(Succeed())

	}, th.Timeout, th.Interval).Should(Succeed())

	th.Logger.Info("Simulated DB completed", "on", name)
}

// Build OpenStackDataPlaneNodeSet struct and fill it with preset values
func DefaultDataplaneNodeSetTemplate(name types.NamespacedName, spec map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{

		"apiVersion": "dataplane.openstack.org/v1beta1",
		"kind":       "OpenStackDataPlaneNodeSet",
		"metadata": map[string]interface{}{
			"name":      name.Name,
			"namespace": name.Namespace,
		},
		"spec": spec,
	}
}

// Build OpenStackDataPlaneDeployment struct and fill it with preset values
func DefaultDataplaneDeploymentTemplate(name types.NamespacedName, spec map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{

		"apiVersion": "dataplane.openstack.org/v1beta1",
		"kind":       "OpenStackDataPlaneDeployment",

		"metadata": map[string]interface{}{
			"name":      name.Name,
			"namespace": name.Namespace,
		},
		"spec": spec,
	}
}

func DefaultNetConfig(name types.NamespacedName, spec map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"apiVersion": "network.openstack.org/v1beta1",
		"kind":       "NetConfig",
		"metadata": map[string]interface{}{
			"name":      name.Name,
			"namespace": name.Namespace,
		},
		"spec": spec,
	}
}

// Create an empty OpenStackDataPlaneService struct
// containing only given NamespacedName as metadata
func DefaultDataplaneService(name types.NamespacedName) map[string]interface{} {

	return map[string]interface{}{

		"apiVersion": "dataplane.openstack.org/v1beta1",
		"kind":       "OpenStackDataPlaneService",
		"metadata": map[string]interface{}{
			"name":      name.Name,
			"namespace": name.Namespace,
		}}
}

// Create an empty OpenStackDataPlaneService struct
// containing only given NamespacedName as metadata
func DefaultDataplaneGlobalService(name types.NamespacedName) map[string]interface{} {

	return map[string]interface{}{

		"apiVersion": "dataplane.openstack.org/v1beta1",
		"kind":       "OpenStackDataPlaneService",
		"metadata": map[string]interface{}{
			"name":      name.Name,
			"namespace": name.Namespace,
		},
		"spec": map[string]interface{}{
			"deployOnAllNodeSets": true,
		},
	}
}

// Get resources

// Retrieve OpenStackDataPlaneDeployment and check for errors
func GetDataplaneDeployment(name types.NamespacedName) *dataplanev1.OpenStackDataPlaneDeployment {
	instance := &dataplanev1.OpenStackDataPlaneDeployment{}
	Eventually(func(g Gomega) error {
		g.Expect(k8sClient.Get(ctx, name, instance)).Should(Succeed())
		return nil
	}, timeout, interval).Should(Succeed())
	return instance
}

// Retrieve OpenStackDataPlaneDeployment and check for errors
func GetDataplaneNodeSet(name types.NamespacedName) *dataplanev1.OpenStackDataPlaneNodeSet {
	instance := &dataplanev1.OpenStackDataPlaneNodeSet{}
	Eventually(func(g Gomega) error {
		g.Expect(k8sClient.Get(ctx, name, instance)).Should(Succeed())
		return nil
	}, timeout, interval).Should(Succeed())
	return instance
}

// Get service with given NamespacedName, assert on successful retrieval
func GetService(name types.NamespacedName) *dataplanev1.OpenStackDataPlaneService {
	foundService := &dataplanev1.OpenStackDataPlaneService{}
	Eventually(func(g Gomega) error {
		g.Expect(k8sClient.Get(ctx, name, foundService)).Should(Succeed())
		return nil
	}, timeout, interval).Should(Succeed())
	return foundService
}

// Get OpenStackDataPlaneNodeSet conditions
func DataplaneConditionGetter(name types.NamespacedName) condition.Conditions {
	instance := GetDataplaneNodeSet(name)
	return instance.Status.Conditions
}

// Get OpenStackDataPlaneDeployment conditions
func DataplaneDeploymentConditionGetter(name types.NamespacedName) condition.Conditions {
	instance := GetDataplaneDeployment(name)
	return instance.Status.Conditions
}

// Get OpenStackAnsibleEE resource by name
func GetAnsibleee(name types.NamespacedName) *v1beta1.OpenStackAnsibleEE {
	instance := &v1beta1.OpenStackAnsibleEE{}
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, name, instance)).Should(Succeed())
	}, timeout, interval).Should(Succeed())
	return instance
}

// Get OpenStackAnsibleEE resource by label
func GetAnsibleeeByLabel(targetNodeSet string, deploymentName types.NamespacedName, service string) *v1beta1.OpenStackAnsibleEE {
	ansibleEEs := &v1beta1.OpenStackAnsibleEEList{}
	instance := &v1beta1.OpenStackAnsibleEE{}
	Eventually(func(g Gomega) {
		deployment := GetDataplaneDeployment(deploymentName)
		listOpts := []client.ListOption{
			client.InNamespace(deployment.GetNamespace()),
		}
		labelSelector := map[string]string{
			"openstackdataplaneservice":    service,
			"openstackdataplanedeployment": deployment.Name,
			"openstackdataplanenodeset":    targetNodeSet,
		}
		labels := client.MatchingLabels(labelSelector)
		listOpts = append(listOpts, labels)
		g.Expect(k8sClient.List(ctx, ansibleEEs, listOpts...)).Should(Succeed())

		g.Expect(ansibleEEs.Items).To(HaveLen(1))
		if len(ansibleEEs.Items) == 1 {
			*instance = ansibleEEs.Items[0]
		}

	}, timeout, interval).Should(Succeed())

	return instance
}

// Delete resources

// Delete namespace from k8s, check for errors
func DeleteNamespace(name string) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	Expect(k8sClient.Delete(ctx, ns)).Should(Succeed())
}

func getCtlPlaneIP(secret *corev1.Secret) string {
	secretData := secret.Data["inventory"]

	var inv AnsibleInventory
	err := yaml.Unmarshal(secretData, &inv)
	if err != nil {
		fmt.Printf("Error unmarshalling secretData: %v", err)
	}
	return inv.EdpmComputeNodeset.Hosts.Node.CtlPlaneIP
}
