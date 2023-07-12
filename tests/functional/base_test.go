package functional

import (
	"github.com/onsi/gomega"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	dataplanev1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
)

func DefaultDataplaneNoNodesTemplate(name types.NamespacedName) *dataplanev1.OpenStackDataPlane {
	return &dataplanev1.OpenStackDataPlane{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "dataplane.openstack.org/v1beta1",
			Kind:       "OpenStackDataPlane",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name.Name,
			Namespace: name.Namespace,
		},
		Spec: dataplanev1.OpenStackDataPlaneSpec{},
	}
}

func CreateDataplaneNoNodes(name types.NamespacedName) *dataplanev1.OpenStackDataPlane {
	instance := DefaultDataplaneNoNodesTemplate(name)
	err := k8sClient.Create(ctx, instance)
	Expect(err).NotTo(HaveOccurred())

	return instance
}

func DefaultDataPlaneSpec() dataplanev1.OpenStackDataPlaneSpec {
	return dataplanev1.OpenStackDataPlaneSpec{
		DeployStrategy: dataplanev1.DeployStrategySection{
			Deploy: true,
		},
		NodeSets: []string{
			"edpm-compute-node-set",
		},
	}
}

func CreateDataplaneNodeSet(name types.NamespacedName, spec dataplanev1.OpenStackDataPlaneNodeSetSpec) *dataplanev1.OpenStackDataPlaneNodeSet {
	instance := DefaultDataplaneNodeSetTemplate(name, spec)
	err := k8sClient.Create(ctx, instance)
	Expect(err).NotTo(HaveOccurred())

	return instance
}

func DefaultDataPlaneNodeSetSpec() dataplanev1.OpenStackDataPlaneNodeSetSpec {
	return dataplanev1.OpenStackDataPlaneNodeSetSpec{
		DeployStrategy: dataplanev1.DeployStrategySection{
			Deploy: false,
		},
		DataPlane:      "openstack-dataplane",
		PreProvisioned: false,
		NodeTemplate: dataplanev1.NodeTemplate{
			Nodes: map[string]dataplanev1.NodeSection{
				"edpm-compute-node-set": {
					HostName: "edpm-bm-compute-1",
				},
			},
			Ansible: dataplanev1.AnsibleOpts{
				AnsibleSSHPrivateKeySecret: "dataplane-ansible-ssh-private-key-secret",
			},
		},
	}
}

func DefaultDataplaneNodeSetTemplate(name types.NamespacedName, spec dataplanev1.OpenStackDataPlaneNodeSetSpec) *dataplanev1.OpenStackDataPlaneNodeSet {
	return &dataplanev1.OpenStackDataPlaneNodeSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "dataplane.openstack.org/v1beta1",
			Kind:       "OpenStackDataPlane",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name.Name,
			Namespace: name.Namespace,
			Labels: map[string]string{
				"openstackdataplane": "dataplane-test",
			},
		},
		Spec: spec,
	}
}

func DefaultDataPlane(name types.NamespacedName, spec dataplanev1.OpenStackDataPlaneSpec) *dataplanev1.OpenStackDataPlane {
	return &dataplanev1.OpenStackDataPlane{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "dataplane.openstack.org/v1beta1",
			Kind:       "OpenStackDataPlane",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name.Name,
			Namespace: name.Namespace,
		},
		Spec: spec,
	}
}

func CreateDataPlane(name types.NamespacedName, spec dataplanev1.OpenStackDataPlaneSpec) *dataplanev1.OpenStackDataPlane {
	instance := DefaultDataPlane(name, spec)
	err := k8sClient.Create(ctx, instance)
	Expect(err).NotTo(HaveOccurred())

	return instance
}

func GetDataplane(name types.NamespacedName) *dataplanev1.OpenStackDataPlane {
	instance := &dataplanev1.OpenStackDataPlane{}
	gomega.Eventually(func(g gomega.Gomega) error {
		g.Expect(k8sClient.Get(ctx, name, instance)).Should(Succeed())
		return nil
	}, timeout, interval).Should(Succeed())
	return instance
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

func CreateSSHSecret(namespace string, name string) *corev1.Secret {
	return th.CreateSecret(
		types.NamespacedName{Namespace: namespace, Name: "ssh-key-secret"},
		map[string][]byte{
			"ssh-privatekey": []byte("blah"),
		},
	)
}
