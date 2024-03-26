#!/usr/bin/env bash
set -ex

BAREMETAL=docs/assemblies/samples/ref_example-OpenStackDataPlaneNodeSet-CR-for-bare-metal-nodes.adoc
FOOTER=$(sed '0,/----/d' $BAREMETAL | sed -e '0,/----/d')
sed -i '/----/q' $BAREMETAL
NODESET=$(oc kustomize --load-restrictor LoadRestrictionsNone examples/baremetal | yq ' select(.kind == "OpenStackDataPlaneNodeSet")')
COUNT=1
CALLOUTS=(
    "env"
    "services"
    "baremetalSetTemplate"
    "edpm-compute-0"
    "networkAttachments"
    "nodeTemplate"
    "ansibleSSHPrivateKeySecret"
    "networks"
    "ansibleUser"
    "ansibleVars"
    "edpm_network_config_template"
)
for callout in "${CALLOUTS[@]}"
do
  NODESET=$(echo -e "$NODESET" | sed -e "/$callout:/ s/$/ #<$COUNT>/")
  COUNT=$((COUNT + 1))
done
echo -e "$NODESET" >> $BAREMETAL
echo -e "----\n$FOOTER" >> $BAREMETAL


PREPROVISIONED=docs/assemblies/samples/ref_example-OpenStackDataPlaneNodeSet-CR-for-preprovisioned-nodes.adoc
FOOTER=$(sed '0,/----/d' $PREPROVISIONED | sed -e '0,/----/d')
sed -i '/----/q' $PREPROVISIONED
NODESET=$(oc kustomize --load-restrictor LoadRestrictionsNone examples/preprovisioned | yq ' select(.kind == "OpenStackDataPlaneNodeSet")')
COUNT=1
CALLOUTS=(
    "env"
    "services"
    "preProvisioned"
    "edpm-compute-0"
    "networkAttachments"
    "nodeTemplate"
    "ansibleSSHPrivateKeySecret"
    "ansibleUser"
    "ansibleVars"
    "edpm_network_config_template"
)
for callout in "${CALLOUTS[@]}"
do
  NODESET=$(echo -e "$NODESET" | sed -e "/$callout:/ s/$/ #<$COUNT>/")
  COUNT=$((COUNT + 1))
done
echo -e "$NODESET" >> $PREPROVISIONED
echo -e "----\n$FOOTER" >> $PREPROVISIONED
