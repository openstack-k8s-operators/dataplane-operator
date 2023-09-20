package functional

import (
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	dataplanev1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
)

// Resource creation

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
func CreateDataplaneService(name types.NamespacedName) *unstructured.Unstructured {
	raw := DefaultDataplaneService(name)
	return th.CreateUnstructured(raw)
}

// Build CustomServiceImageSpec struct with empty `Nodes` list
func CustomServiceImageSpec() map[string]interface{} {

	return map[string]interface{}{
		"preProvisioned": true,
		"nodeTemplate": map[string]interface{}{
			"ansibleSSHPrivateKeySecret": "dataplane-ansible-ssh-private-key-secret",
			"ansible": map[string]interface{}{
				"ansibleVars": map[string]interface{}{
					"edpm_nova_compute_image": "blah.test-image:latest",
				},
			},
		},
		"nodes": map[string]interface{}{},
	}
}

// Create SSHSecret
func CreateSSHSecret(name types.NamespacedName) *corev1.Secret {
	return th.CreateSecret(
		types.NamespacedName{Namespace: name.Namespace, Name: name.Name},
		map[string][]byte{
			"ssh-privatekey": []byte("blah"),
		},
	)
}

// Struct initialization

// Build OpenStackDataPlaneNodeSetSpec struct and fill it with preset values
func DefaultDataPlaneNodeSetSpec() map[string]interface{} {

	return map[string]interface{}{
		"preProvisioned": false,
		"nodeTemplate": map[string]interface{}{
			"ansibleSSHPrivateKeySecret": "dataplane-ansible-ssh-private-key-secret",
		},
		"nodes": map[string]interface{}{
			"edpm-compute-node-set": map[string]interface{}{
				"hostname": "edpm-bm-compute-1",
			},
		},
	}
}

// Build OpenStackDataPlaneNodeSetSpec struct with empty `Nodes` list
func DefaultDataPlaneNoNodeSetSpec() map[string]interface{} {

	return map[string]interface{}{
		"preProvisioned": true,
		"nodeTemplate": map[string]interface{}{
			"ansibleSSHPrivateKeySecret": "dataplane-ansible-ssh-private-key-secret",
		},
		"nodes": map[string]interface{}{},
	}
}

// Build OpenStackDataPlnaeDeploymentSpec and fill it with preset values
func DefaultDataPlaneDeploymentSpec() map[string]interface{} {

	return map[string]interface{}{
		"nodeSets": []string{
			"edpm-compute-nodeset",
		},
	}
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
