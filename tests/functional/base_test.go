package functional

import (
	"github.com/onsi/gomega"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	dataplanev1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
)

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
		Spec: dataplanev1.OpenStackDataPlaneRoleSpec{
			DeployStrategy: dataplanev1.DeployStrategySection{
				Deploy: false,
			},
		},
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
			Deploy: false,
		},
		Roles: map[string]dataplanev1.OpenStackDataPlaneRoleSpec{
			"edpm-compute-no-nodes": {
				Services: []string{"configure-network", "validate-network", "install-os", "configure-os", "run-os"},
				NodeTemplate: dataplanev1.NodeSection{
					AnsibleSSHPrivateKeySecret: "dataplane-ansible-ssh-private-key-secret",
				},
			},
		},
	}
}

func DataPlaneWithNodeSpec() dataplanev1.OpenStackDataPlaneSpec {
	customNicConfigTemplate := `
            ---
            network_config:
            - type: interface
              name: nic2
              mtu: 1500
              addresses:
                - ip_netmask:
                    {{ ctlplane_ip }}/{{ ctlplane_subnet_cidr }}
            - type: ovs_bridge
              name: {{ neutron_physical_bridge_name }}
              mtu: 1500
              use_dhcp: false
              dns_servers: {{ ctlplane_dns_nameservers }}
              domain: []
              addresses:
              - ip_netmask: {{ lookup('vars', networks_lower["External"] ~ '_ip') }}/{{ lookup('vars', networks_lower["External"] ~ '_cidr') }}
              routes: [{'ip_netmask': '0.0.0.0/0', 'next_hop': '192.168.1.254'}]
              members:
              - type: interface
                name: nic1
                mtu: 1500
                # force the MAC address of the bridge to this interface
                primary: true
              - type: vlan
                mtu: 1500
                vlan_id: 20
                addresses:
                - ip_netmask:
                    172.17.0.101/24
                routes: []
              - type: vlan
                mtu: 1500
                vlan_id: 25
                addresses:
                - ip_netmask:
                    172.18.0.101/24
                routes: []
              - type: vlan
                mtu: 1500
                vlan_id: 22
                addresses:
                - ip_netmask:
                    172.19.0.101/24
                routes: []
	`
	return dataplanev1.OpenStackDataPlaneSpec{
		DeployStrategy: dataplanev1.DeployStrategySection{
			Deploy: false,
		},
		Roles: map[string]dataplanev1.OpenStackDataPlaneRoleSpec{
			"edpm-compute": {
				Services: []string{"configure-network", "validate-network", "install-os", "configure-os", "run-os"},
				NodeTemplate: dataplanev1.NodeSection{
					AnsibleSSHPrivateKeySecret: "dataplane-ansible-ssh-private-key-secret",
				},
			},
		},
		Nodes: map[string]dataplanev1.OpenStackDataPlaneNodeSpec{
			"edpm-compute-0": {
				Node: dataplanev1.NodeSection{
					NetworkConfig: dataplanev1.NetworkConfigSection{
						Template: customNicConfigTemplate,
					},
					AnsibleSSHPrivateKeySecret: "ssh-key-secret",
				},
				Role: "edpm-compute",
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
