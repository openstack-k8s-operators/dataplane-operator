package functional

import (
	"github.com/onsi/gomega"
	. "github.com/onsi/gomega"
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

func DefaultDataplaneRoleNoNodesTemplate(name types.NamespacedName) *dataplanev1.OpenStackDataPlaneRole {
	return &dataplanev1.OpenStackDataPlaneRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "dataplane.openstack.org/v1beta1",
			Kind:       "OpenStackDataPlaneRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "edpm-compute-no-nodes",
			Namespace: name.Namespace,
		},
		Spec: dataplanev1.OpenStackDataPlaneRoleSpec{},
	}
}

func CreateDataplaneRoleNoNodes(name types.NamespacedName) *dataplanev1.OpenStackDataPlaneRole {
	instance := DefaultDataplaneRoleNoNodesTemplate(name)
	err := k8sClient.Create(ctx, instance)
	Expect(err).NotTo(HaveOccurred())

	return instance
}

func DefaultDataPlaneSpec() dataplanev1.OpenStackDataPlaneSpec {
	return dataplanev1.OpenStackDataPlaneSpec{
		DeployStrategy: dataplanev1.DeployStrategySection{
			AnsibleTags: "",
		},
		Roles: map[string]dataplanev1.OpenStackDataPlaneRoleSpec{
			"edpm-compute-no-nodes": {
				Services: []string{"configure-network", "validate-network", "install-os", "configure-os", "run-os"},
				NodeTemplate: dataplanev1.NodeTemplate{
					AnsibleSSHPrivateKeySecret: "dataplane-ansible-ssh-private-key-secret",
				},
			},
		},
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

func GetDataplaneRole(name types.NamespacedName) *dataplanev1.OpenStackDataPlaneRole {
	instance := &dataplanev1.OpenStackDataPlaneRole{}
	gomega.Eventually(func(g gomega.Gomega) error {
		g.Expect(k8sClient.Get(ctx, name, instance)).Should(Succeed())
		return nil
	}, timeout, interval).Should(Succeed())
	return instance
}
