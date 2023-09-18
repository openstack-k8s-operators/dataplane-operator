package functional

import (
	"github.com/onsi/gomega"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func GetDataplaneNodeSet(name types.NamespacedName) *dataplanev1.OpenStackDataPlaneNodeSet {
	instance := &dataplanev1.OpenStackDataPlaneNodeSet{}
	gomega.Eventually(func(g gomega.Gomega) error {
		g.Expect(k8sClient.Get(ctx, name, instance)).Should(Succeed())
		return nil
	}, timeout, interval).Should(Succeed())
	return instance
}

func DeleteNamespace(name string) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	gomega.Expect(k8sClient.Delete(ctx, ns)).Should(gomega.Succeed())
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
