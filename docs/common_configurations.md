# Common Configurations

This page documents some of the common configurations that can be enabled
through ansible variables.  The ansible variables that affect the configuration
of the ansible executions are set in the `ansibleVars` field on the dataplane
resources.

The full set of ansible variables available for configuration are documented
within each role in the
[edpm-ansible](https://github.com/openstack-k8s-operators/edpm-ansible/tree/main/roles)
repository.

## Initial bootstrap command

**Variable**: `edpm_bootstrap_command`
**Type**: `string`
**Role**: [edpm_bootstrap](https://github.com/openstack-k8s-operators/edpm-ansible/tree/main/roles/edpm_bootstrap)

The `edpm_bootstrap_command` variable can be used to pass a shell command(s) that
will be executed as early as possible in the deployment as part of the
`configure-network` service. If the `services` list is customized with services
that execute prior to `configure-network` then the command(s) specified by
`edpm_bootstrap_command` would run after the custom services.

### Using `edpm_bootstrap_command` for system registration

`edpm_bootstrap_command` can be used to perform system registration in order to
enable needed package repositories. Choose a registration method (either Portal
or Satellite) and refer to the provided links below for instructions to create
the registration commands.

#### Red Hat Customer Portal registration

The registration commands for the Red Hat Customer Portal are documented at
<https://access.redhat.com/solutions/253273>.

#### Red Hat Satellite registration

The registration commands for Red Hat Satellite version 6.13 are documented at
<https://access.redhat.com/documentation/en-us/red_hat_satellite/6.13/html-single/managing_hosts/index#Registering_Hosts_to_Server_managing-hosts>.

If not using Satellite version 6.13, then refer to the specific version of the
documentation for the version of Satellite that is in use.

## Network Isolation

Network Isolation refers to the practice of separating network traffic by
function, and configuring the networks on dataplane nodes. Nodes will need
connectivity to various control plane services running on OCP. These services
may be bound to different networks. Each of those networks needs to be
configured as required on dataplane nodes.

For further details on the network architecture of the control plane, see
<https://github.com/openstack-k8s-operators/docs/blob/main/networking.md>.

### Configuring networking with edpm_network_config

The
[edpm_network_config](https://github.com/openstack-k8s-operators/edpm-ansible/tree/main/roles/edpm_network_config)
ansible role is responsible for configuring networking on dataplane nodes.

The `edpm_network_config_template` variable specifies the template that
describes the networking configuration to be applied. The
template itself also contains variables that can be used to customize the
networking configuration for a specific node (IP addresses, interface names,
routes, etc). Templates provided with the edpm_network_config role are at
<https://github.com/openstack-k8s-operators/edpm-ansible/tree/main/roles/edpm_network_config/templates>.

Custom templates can also be used, but they must be avaialable to ansible in
the ansible-runner image used by the `configure-network` service. Use the
[`ExtraMounts`](../composable_services/#using-extramounts) field to mount custom
content into the ansible-runner image.

The following is an example
[`ansibleVars`](http://127.0.0.1:8000/dataplane-operator/openstack_dataplanerole/#nodesection)
field that shows defining the variables that configure the
`edpm_network_config` role.

    ansibleVars:
      edpm_network_config_template: |
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
      ctlplane_ip: 192.168.122.100
      internal_api_ip: 172.17.0.100
      storage_ip: 172.18.0.100
      tenant_ip: 172.19.0.100
      fqdn_internal_api: edpm-compute-0.example.com

This configuration would be applied by the
[`configure-network`](../composable_services/#dataplane-operator-provided-services) service when
it's executed.

### Network attachment definitions

The
[`NetworkAttachmentDefinition`](https://github.com/openstack-k8s-operators/docs/blob/main/networking.md#network-attachment-definitions) resource is used to describe how pods can be attached to different networks. Network attachment definitions can be specified on the [`OpenStackDataPlaneRole`](openstack_dataplanerole.md) and [`OpenStackDataPlaneNode`](openstack_dataplanenode.md) resources using the `NetworkAttachments` field.

The network attachments are used to describe which networks will be connected
to the pod that is running ansible-runner. They do not enable networks on the
dataplane nodes themselves. For example, adding the `internalapi` network
attachment to `NetworkAttachments` means the ansible-runner pod will be
connected to the `internalapi` network. This can enable scenarios where ansible
needs to connect to different networks.
