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
	"reflect"
	"sort"
	"strconv"
	"strings"
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

// Helper function to create the data structure that will be used to store the secrets.
func createSecretsDataStructure(secretMaxSize int,
	certsData map[string][]byte,
) []map[string][]byte {

	ci := []map[string][]byte{}

	keys := []string{}
	for k := range certsData {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	totalSize := secretMaxSize
	var cur *map[string][]byte
	// Going 3 by 3 to include CA, crt and key, in the same secret.
	for k := 0; k < len(keys)-1; k += 3 {
		szCa := len(certsData[keys[k]]) + len(keys[k])
		szCrt := len(certsData[keys[k+1]]) + len(keys[k+1])
		szKey := len(certsData[keys[k+2]]) + len(keys[k+2])
		sz := szCa + szCrt + szKey
		if (totalSize + sz) > secretMaxSize {
			i := len(ci)
			ci = append(ci, make(map[string][]byte))
			cur = &ci[i]
			totalSize = 0
		}
		totalSize += sz
		(*cur)[keys[k]] = certsData[keys[k]]
		(*cur)[keys[k+1]] = certsData[keys[k+1]]
		(*cur)[keys[k+2]] = certsData[keys[k+2]]
	}

	return ci
}

// EnsureTLSCerts generates secrets containing all the certificates for the relevant service
// These secrets will be mounted by the ansibleEE pod as an extra mount when the service is deployed.
func EnsureTLSCerts(ctx context.Context, helper *helper.Helper,
	instance *dataplanev1.OpenStackDataPlaneNodeSet,
	allHostnames map[string]map[infranetworkv1.NetNameStr]string,
	allIPs map[string]map[infranetworkv1.NetNameStr]string,
	service dataplanev1.OpenStackDataPlaneService,
) (*ctrl.Result, error) {
	certsData := map[string][]byte{}
	secretMaxSize := instance.Spec.SecretMaxSize

	// for each node in the nodeset, issue all the TLS certs needed based on the
	// ips or DNS Names
	for nodeName, node := range instance.Spec.Nodes {
		var dnsNames map[infranetworkv1.NetNameStr]string
		var ipsMap map[infranetworkv1.NetNameStr]string
		var hosts []string
		var ips []string
		var issuer *certmgrv1.Issuer
		var issuerLabelSelector map[string]string
		var certName string
		var certSecret *corev1.Secret
		var err error
		var result ctrl.Result

		// TODO(alee) decide if we want to use other labels
		// For now we just add the hostname so we can select all the certs on one node
		hostName := node.HostName
		labels := map[string]string{
			HostnameLabel: hostName,
			ServiceLabel:  service.Name,
			NodeSetLabel:  instance.Name,
		}
		certName = service.Name + "-" + hostName

		dnsNames = allHostnames[hostName]
		ipsMap = allIPs[hostName]

		dnsNamesInCert := slices.Contains(service.Spec.TLSCert.Contents, DNSNamesStr)
		ipValuesInCert := slices.Contains(service.Spec.TLSCert.Contents, IPValuesStr)

		// Create the hosts and ips lists
		if dnsNamesInCert {
			if len(service.Spec.TLSCert.Networks) == 0 {
				hosts = make([]string, 0, len(dnsNames))
				for _, host := range dnsNames {
					hosts = append(hosts, host)
				}
			} else {
				hosts = make([]string, 0, len(service.Spec.TLSCert.Networks))
				for _, network := range service.Spec.TLSCert.Networks {
					certNetwork := strings.ToLower(string(network))
					hosts = append(hosts, dnsNames[infranetworkv1.NetNameStr(certNetwork)])
				}
			}
		}
		if ipValuesInCert {
			if len(service.Spec.TLSCert.Networks) == 0 {
				ips = make([]string, 0, len(ipsMap))
				for _, ip := range ipsMap {
					ips = append(ips, ip)
				}
			} else {
				ips = make([]string, 0, len(service.Spec.TLSCert.Networks))
				for _, network := range service.Spec.TLSCert.Networks {
					certNetwork := strings.ToLower(string(network))
					ips = append(ips, ipsMap[infranetworkv1.NetNameStr(certNetwork)])
				}
			}
		}

		if service.Spec.TLSCert.Issuer == "" {
			// by default, use the internal root CA
			issuerLabelSelector = map[string]string{certmanager.RootCAIssuerInternalLabel: ""}
		} else {
			issuerLabelSelector = map[string]string{service.Spec.TLSCert.Issuer: ""}
		}

		issuer, err = certmanager.GetIssuerByLabels(ctx, helper, instance.Namespace, issuerLabelSelector)
		if err != nil {
			helper.GetLogger().Info("Error retrieving issuer by label", "issuerLabelSelector", issuerLabelSelector)
			return &result, err
		}

		// TODO: paramaterize usage
		certSecret, result, err = GetTLSNodeCert(ctx, helper, instance, certName,
			issuer.Name, labels, hosts, ips, service.Spec.TLSCert.KeyUsages)

		// handle cert request errors
		if (err != nil) || (result != ctrl.Result{}) {
			return &result, err
		}
		// TODO(alee) Add an owner reference to the secret so it can be monitored
		// We'll do this once stuggi adds a function to do this in libcommon

		// NOTE: we are assuming that there will always be a ctlplane network
		// that means if you are not using network isolation with multiple networks
		// you should still need to have a ctlplane network at a minimum to use tls-e
		basename := allHostnames[nodeName][CtlPlaneNetwork]
		// in case the control plane network is not present we will fall back to the
		// hostname, and log a warning.
		field := reflect.ValueOf(basename)
		if field.IsZero() {
			basename = hostName
			helper.GetLogger().Error(fmt.Errorf(
				"control plane network not found for node %s, falling back to hostname", nodeName),
				"tls-e requires a control plane network to be present")
		}
		// To use this cert, add it to the relevant service data
		certsData[basename+"-tls.key"] = certSecret.Data["tls.key"]
		certsData[basename+"-tls.crt"] = certSecret.Data["tls.crt"]
		certsData[basename+"-ca.crt"] = certSecret.Data["ca.crt"]
	}

	// Calculate number of secrets to create
	ci := createSecretsDataStructure(secretMaxSize, certsData)

	labels := map[string]string{
		"numberOfSecrets": strconv.Itoa(len(ci)),
	}
	// create secrets to hold the certs for the services
	for i := range ci {
		labels["secretNumber"] = strconv.Itoa(i)
		serviceCertsSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      GetServiceCertsSecretName(instance, service.Name, i),
				Namespace: instance.Namespace,
				Labels:    labels,
			},
			Data: ci[i],
		}
		_, result, err := secret.CreateOrPatchSecret(ctx, helper, instance, serviceCertsSecret)
		if err != nil {
			err = fmt.Errorf("error creating certs secret for %s - %w", service.Name, err)
			return &ctrl.Result{}, err
		} else if result != controllerutil.OperationResultNone {
			return &ctrl.Result{RequeueAfter: time.Second * 5}, nil
		}
	}

	return &ctrl.Result{}, nil
}

// GetTLSNodeCert creates or retrieves the cert for a node for a given service
func GetTLSNodeCert(ctx context.Context, helper *helper.Helper,
	instance *dataplanev1.OpenStackDataPlaneNodeSet,
	certName string, issuer string,
	labels map[string]string,
	hostnames []string, ips []string, usages []certmgrv1.KeyUsage,
) (*corev1.Secret, ctrl.Result, error) {
	secretName := "cert-" + certName
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
			CertName:    certName,
			Duration:    duration,
			Hostnames:   hostnames,
			Ips:         ips,
			Annotations: nil,
			Labels:      labels,
			Usages:      usages,
		}

		certSecret, result, err = certmanager.EnsureCert(ctx, helper, request, instance)
		if err != nil {
			return nil, ctrl.Result{}, err
		} else if (result != ctrl.Result{}) {
			return nil, result, nil
		}
	}
	return certSecret, ctrl.Result{}, nil
}

// GetServiceCertsSecretName - return name of secret to be mounted in ansibleEE which contains
// all the TLS certs that fit in a secret for the relevant service. The index variable is used
// to make the secret name unique.
// The convention we use here is "<nodeset.name>-<service>-certs-<index>", so for example,
// openstack-epdm-nova-certs-0.
func GetServiceCertsSecretName(instance *dataplanev1.OpenStackDataPlaneNodeSet, serviceName string, index int) string {
	return fmt.Sprintf("%s-%s-certs-%s", instance.Name, serviceName, strconv.Itoa(index))
}
