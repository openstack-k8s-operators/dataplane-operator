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

func CreateDataplaneNodeSet(name types.NamespacedName, spec dataplanev1.OpenStackDataPlaneNodeSetSpec) *dataplanev1.OpenStackDataPlaneNodeSet {
	instance := DefaultDataplaneNodeSetTemplate(name, spec)
	err := k8sClient.Create(ctx, instance)
	Expect(err).NotTo(HaveOccurred())

	return instance
}

func DefaultDataPlaneNodeSetSpec() dataplanev1.OpenStackDataPlaneNodeSetSpec {
	return dataplanev1.OpenStackDataPlaneNodeSetSpec{
		PreProvisioned: false,
		NodeTemplate: dataplanev1.NodeTemplate{
			AnsibleSSHPrivateKeySecret: "dataplane-ansible-ssh-private-key-secret",
		},
		Nodes: map[string]dataplanev1.NodeSection{
			"edpm-compute-node-set": {
				HostName: "edpm-bm-compute-1",
			},
		},
	}
}

func CreateDataplaneDeployment(name types.NamespacedName, spec dataplanev1.OpenStackDataPlaneDeploymentSpec) *dataplanev1.OpenStackDataPlaneDeployment {
	instance := DefaultDataplaneDeploymentTemplate(name, spec)
	err := k8sClient.Create(ctx, instance)
	Expect(err).NotTo(HaveOccurred())

	return instance
}

func DefaultDataPlaneDeploymentSpec() dataplanev1.OpenStackDataPlaneDeploymentSpec {
	return dataplanev1.OpenStackDataPlaneDeploymentSpec{
		NodeSets: []string{
			"edpm-compute-nodeset",
		},
	}
}

func DefaultDataPlaneNoNodeSetSpec() dataplanev1.OpenStackDataPlaneNodeSetSpec {
	return dataplanev1.OpenStackDataPlaneNodeSetSpec{
		PreProvisioned: true,
		NodeTemplate: dataplanev1.NodeTemplate{
			AnsibleSSHPrivateKeySecret: "dataplane-ansible-ssh-private-key-secret",
		},
		Nodes: map[string]dataplanev1.NodeSection{},
	}
}

func DefaultDataplaneNodeSetTemplate(name types.NamespacedName, spec dataplanev1.OpenStackDataPlaneNodeSetSpec) *dataplanev1.OpenStackDataPlaneNodeSet {
	return &dataplanev1.OpenStackDataPlaneNodeSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "dataplane.openstack.org/v1beta1",
			Kind:       "OpenStackDataPlaneNodeSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name.Name,
			Namespace: name.Namespace,
		},
		Spec: spec,
	}
}

func DefaultDataplaneDeploymentTemplate(name types.NamespacedName, spec dataplanev1.OpenStackDataPlaneDeploymentSpec) *dataplanev1.OpenStackDataPlaneDeployment {
	return &dataplanev1.OpenStackDataPlaneDeployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "dataplane.openstack.org/v1beta1",
			Kind:       "OpenStackDataPlaneDeployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name.Name,
			Namespace: name.Namespace,
		},
		Spec: spec,
	}
}

func GetDataplaneDeployment(name types.NamespacedName) *dataplanev1.OpenStackDataPlaneDeployment {
	instance := &dataplanev1.OpenStackDataPlaneDeployment{}
	Eventually(func(g Gomega) error {
		g.Expect(k8sClient.Get(ctx, name, instance)).Should(Succeed())
		return nil
	}, timeout, interval).Should(Succeed())
	return instance
}

func GetDataplaneNodeSet(name types.NamespacedName) *dataplanev1.OpenStackDataPlaneNodeSet {
	instance := &dataplanev1.OpenStackDataPlaneNodeSet{}
	Eventually(func(g Gomega) error {
		g.Expect(k8sClient.Get(ctx, name, instance)).Should(Succeed())
		return nil
	}, timeout, interval).Should(Succeed())
	return instance
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

// Create an OpenStackDataPlaneService with a given NamespacedName, assert on success
func CreateDataplaneService(name types.NamespacedName) *unstructured.Unstructured {
	raw := DefaultDataplaneService(name)

	return th.CreateUnstructured(raw)
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

func DeleteNamespace(name string) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	Expect(k8sClient.Delete(ctx, ns)).Should(Succeed())
}

func CreateSSHSecret(name types.NamespacedName) *corev1.Secret {
	return th.CreateSecret(
		types.NamespacedName{Namespace: name.Namespace, Name: name.Name},
		map[string][]byte{
			"ssh-privatekey": []byte("blah"),
		},
	)
}

func DataplaneConditionGetter(name types.NamespacedName) condition.Conditions {
	instance := GetDataplaneNodeSet(name)
	return instance.Status.Conditions
}

func DataplaneDeploymentConditionGetter(name types.NamespacedName) condition.Conditions {
	instance := GetDataplaneDeployment(name)
	return instance.Status.Conditions
}
