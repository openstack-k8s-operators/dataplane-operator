/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package deployment

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/exp/slices"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	certmgrv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	dataplanev1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	infranetworkv1 "github.com/openstack-k8s-operators/infra-operator/apis/network/v1beta1"
	"github.com/openstack-k8s-operators/lib-common/modules/certmanager"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/secret"
)

// EnsureTLSCerts generates  a secret containing all the certificates for the relevant service
// This secret will be mounted by the ansibleEE pod as an extra mount when the service is deployed.
func EnsureTLSCerts(ctx context.Context, helper *helper.Helper,
	instance *dataplanev1.OpenStackDataPlaneNodeSet,
	allHostnames map[string]map[infranetworkv1.NetNameStr]string,
	allIPs map[string]map[infranetworkv1.NetNameStr]string,
	service dataplanev1.OpenStackDataPlaneService,
) (*ctrl.Result, error) {
	certsData := map[string][]byte{}

	// for each node in the nodeset, issue all the TLS certs needed based on the
	// ips or DNS Names
	for nodeName := range instance.Spec.Nodes {
		var dnsNames map[infranetworkv1.NetNameStr]string
		var ipsMap map[infranetworkv1.NetNameStr]string
		var hosts []string
		var ips []string
		var issuer string
		var secretName string
		var certSecret *corev1.Secret = nil
		var err error
		var result ctrl.Result

		// TODO(alee) decide if we want to use other labels
		// For now we just add the hostname so we can select all the certs on one node
		labels := map[string]string{
			"hostname": nodeName,
			"service":  service.Name,
		}
		secretName = "cert-" + service.Name + "-" + nodeName

		dnsNames = allHostnames[nodeName]
		ipsMap = allIPs[nodeName]

		// Create the hosts and ips lists
		if service.Spec.TLSCert.Networks == nil {
			hosts = make([]string, 0, len(dnsNames))
			for _, host := range dnsNames {
				hosts = append(hosts, host)
			}
			ips = make([]string, 0, len(ipsMap))
			for _, ip := range ipsMap {
				ips = append(ips, ip)
			}
		} else {
			hosts = make([]string, 0, len(service.Spec.TLSCert.Networks))
			for _, network := range service.Spec.TLSCert.Networks {
				hosts = append(hosts, dnsNames[network])
			}
			ips = make([]string, 0, len(service.Spec.TLSCert.Networks))
			for _, network := range service.Spec.TLSCert.Networks {
				ips = append(ips, ipsMap[network])
			}
		}

		if !slices.Contains(service.Spec.TLSCert.Contents, DNSNamesStr) {
			hosts = nil
		}
		if !slices.Contains(service.Spec.TLSCert.Contents, IPValuesStr) {
			ips = nil
		}

		if service.Spec.TLSCert.Issuer == "" {
			// by default, use the internal root CA
			issuer = certmanager.RootCAIssuerInternalLabel
		} else {
			issuer = service.Spec.TLSCert.Issuer
		}

		certSecret, result, err = GetTLSNodeCert(ctx, helper, instance, secretName,
			issuer, labels, hosts, ips, nil)

		// handle cert request errors
		if (err != nil) || (result != ctrl.Result{}) {
			return &result, err
		}
		// TODO(alee) Add an owner reference to the secret so it can be monitored
		// We'll do this once stuggi adds a function to do this in libcommon

		// To use this cert, add it to the relevant service data
		certsData[nodeName+"-tls.key"] = certSecret.Data["tls.key"]
		certsData[nodeName+"-tls.crt"] = certSecret.Data["tls.crt"]
	}

	// create a secret to hold the certs for the service
	serviceCertsSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetServiceCertsSecretName(instance, service.Name),
			Namespace: instance.Namespace,
		},
		Data: certsData,
	}
	_, result, err := secret.CreateOrPatchSecret(ctx, helper, instance, serviceCertsSecret)
	if err != nil {
		err = fmt.Errorf("error creating certs secret for %s - %w", service.Name, err)
		return &ctrl.Result{}, err
	} else if result != controllerutil.OperationResultNone {
		return &ctrl.Result{RequeueAfter: time.Second * 5}, nil
	}

	return &ctrl.Result{}, nil
}

// GetTLSNodeCert creates or retrieves the cert for a node for a given service
func GetTLSNodeCert(ctx context.Context, helper *helper.Helper,
	instance *dataplanev1.OpenStackDataPlaneNodeSet,
	secretName string, issuer string,
	labels map[string]string,
	hostnames []string, ips []string, usages []certmgrv1.KeyUsage,
) (*corev1.Secret, ctrl.Result, error) {
	certSecret, _, err := secret.GetSecret(ctx, helper, secretName, instance.Namespace)
	var result ctrl.Result
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			err = fmt.Errorf("error retrieving secret %s - %w", secretName, err)
			return nil, ctrl.Result{}, err
		}

		duration := ptr.To(time.Hour * 24 * 365)
		request := certmanager.CertificateRequest{
			IssuerName:  issuer,
			CertName:    secretName,
			Duration:    duration,
			Hostnames:   hostnames,
			Ips:         ips,
			Annotations: nil,
			Labels:      labels,
			Usages:      usages,
		}

		certSecret, result, err = certmanager.EnsureCert(ctx, helper, request)
		if err != nil {
			return nil, ctrl.Result{}, err
		} else if (result != ctrl.Result{}) {
			return nil, result, nil
		}
	}
	return certSecret, ctrl.Result{}, nil
}

// GetServiceCertsSecretName - return name of secret to be mounted in ansibleEE which contains
// all the TLS certs for the relevant service
// The convention we use here is "<nodeset.name>-<service>-certs", so for example,
// openstack-epdm-nova-certs.
func GetServiceCertsSecretName(instance *dataplanev1.OpenStackDataPlaneNodeSet, serviceName string) string {
	return fmt.Sprintf("%s-%s-certs", instance.Name, serviceName)
}
