#!/bin/bash
set -ex

TMPDIR=${TMPDIR:-"/tmp/k8s-webhook-server/serving-certs"}
SKIP_CERT=${SKIP_CERT:-false}
CRC_IP=${CRC_IP:-$(/sbin/ip -o -4 addr list crc | awk '{print $4}' | cut -d/ -f1)}
FIREWALL_ZONE=${FIREWALL_ZONE:-"libvirt"}
SKIP_FIREWALL=${SKIP_FIREWALL:-false}

if [ "$SKIP_FIREWALL" = false ] ; then
    #Open 9443
    sudo firewall-cmd --zone=${FIREWALL_ZONE} --add-port=9443/tcp
    sudo firewall-cmd --runtime-to-permanent
fi

# Generate the certs and the ca bundle
if [ "$SKIP_CERT" = false ] ; then
    mkdir -p ${TMPDIR}
    rm -rf ${TMPDIR}/* || true

    openssl req -newkey rsa:2048 -days 3650 -nodes -x509 \
        -subj "/CN=${HOSTNAME}" \
        -addext "subjectAltName = IP:${CRC_IP}" \
        -keyout ${TMPDIR}/tls.key \
        -out ${TMPDIR}/tls.crt

    cat ${TMPDIR}/tls.crt ${TMPDIR}/tls.key | base64 -w 0 > ${TMPDIR}/bundle.pem

fi

CA_BUNDLE=`cat ${TMPDIR}/bundle.pem`

# Patch the webhook(s)
cat >> ${TMPDIR}/patch_webhook_configurations.yaml <<EOF_CAT
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: vopenstackdataplanenodeset.kb.io
webhooks:
- admissionReviewVersions:
  - v1
  clientConfig:
    caBundle: ${CA_BUNDLE}
    url: https://${CRC_IP}:9443/validate-dataplane-openstack-org-v1beta1-openstackdataplanenodeset
  failurePolicy: Fail
  matchPolicy: Equivalent
  name: vopenstackdataplanenodeset.kb.io
  objectSelector: {}
  rules:
  - apiGroups:
    - dataplane.openstack.org
    apiVersions:
    - v1beta1
    operations:
    - CREATE
    - UPDATE
    resources:
    - openstackdataplanenodesets
    scope: '*'
  sideEffects: None
  timeoutSeconds: 10
---
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: mopenstackdataplanenodeset.kb.io
webhooks:
- admissionReviewVersions:
  - v1
  clientConfig:
    caBundle: ${CA_BUNDLE}
    url: https://${CRC_IP}:9443/mutate-dataplane-openstack-org-v1beta1-openstackdataplanenodeset
  failurePolicy: Fail
  matchPolicy: Equivalent
  name: mopenstackdataplanenodeset.kb.io
  objectSelector: {}
  rules:
  - apiGroups:
    - dataplane.openstack.org
    apiVersions:
    - v1beta1
    operations:
    - CREATE
    - UPDATE
    resources:
    - openstackdataplanenodesets
    scope: '*'
  sideEffects: None
  timeoutSeconds: 10
---
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: vopenstackdataplanedeployment.kb.io
webhooks:
- admissionReviewVersions:
  - v1
  clientConfig:
    caBundle: ${CA_BUNDLE}
    url: https://${CRC_IP}:9443/validate-dataplane-openstack-org-v1beta1-openstackdataplanedeployment
  failurePolicy: Fail
  matchPolicy: Equivalent
  name: vopenstackdataplanedeployment.kb.io
  objectSelector: {}
  rules:
  - apiGroups:
    - dataplane.openstack.org
    apiVersions:
    - v1beta1
    operations:
    - CREATE
    - UPDATE
    resources:
    - openstackdataplanedeployments
    scope: '*'
  sideEffects: None
  timeoutSeconds: 10
---
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: mopenstackdataplanedeployment.kb.io
webhooks:
- admissionReviewVersions:
  - v1
  clientConfig:
    caBundle: ${CA_BUNDLE}
    url: https://${CRC_IP}:9443/mutate-dataplane-openstack-org-v1beta1-openstackdataplanedeployment
  failurePolicy: Fail
  matchPolicy: Equivalent
  name: mopenstackdataplanedeployment.kb.io
  objectSelector: {}
  rules:
  - apiGroups:
    - dataplane.openstack.org
    apiVersions:
    - v1beta1
    operations:
    - CREATE
    - UPDATE
    resources:
    - openstackdataplanedeployments
    scope: '*'
  sideEffects: None
  timeoutSeconds: 10
---
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: vopenstackdataplaneservice.kb.io
webhooks:
- admissionReviewVersions:
  - v1
  clientConfig:
    caBundle: ${CA_BUNDLE}
    url: https://${CRC_IP}:9443/validate-dataplane-openstack-org-v1beta1-openstackdataplaneservice
  failurePolicy: Fail
  matchPolicy: Equivalent
  name: vopenstackdataplaneservice.kb.io
  objectSelector: {}
  rules:
  - apiGroups:
    - dataplane.openstack.org
    apiVersions:
    - v1beta1
    operations:
    - CREATE
    - UPDATE
    resources:
    - openstackdataplaneservices
    scope: '*'
  sideEffects: None
  timeoutSeconds: 10
---
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: mopenstackdataplaneservice.kb.io
webhooks:
- admissionReviewVersions:
  - v1
  clientConfig:
    caBundle: ${CA_BUNDLE}
    url: https://${CRC_IP}:9443/mutate-dataplane-openstack-org-v1beta1-openstackdataplaneservice
  failurePolicy: Fail
  matchPolicy: Equivalent
  name: mopenstackdataplaneservice.kb.io
  objectSelector: {}
  rules:
  - apiGroups:
    - dataplane.openstack.org
    apiVersions:
    - v1beta1
    operations:
    - CREATE
    - UPDATE
    resources:
    - openstackdataplaneservices
    scope: '*'
  sideEffects: None
  timeoutSeconds: 10
EOF_CAT

oc apply -n openstack -f ${TMPDIR}/patch_webhook_configurations.yaml
