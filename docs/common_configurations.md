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
