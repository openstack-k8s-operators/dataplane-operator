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

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	dataplanev1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/lib-common/modules/certmanager"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/secret"
)

// EnsureTLSCerts generates  a secret containing all the certificates for the relevant service
// This secret will be mounted by the ansibleEE pod as an extra mount when the service is deployed.
func EnsureTLSCerts(ctx context.Context, helper *helper.Helper,
	instance *dataplanev1.OpenStackDataPlaneNodeSet,
	allHostnames map[string][]string,
	allIPs map[string][]string,
	serviceName string) (ctrl.Result, error) {

	certsData := map[string][]byte{}

	// for each node in the nodeset, issue all the TLS certs needed based on the
	// ips or DNS Names
	for nodeName := range instance.Spec.Nodes {
		var dnsNames []string
		var secretName string
		var certName string
		var certSecret *corev1.Secret = nil
		var err error
		var result ctrl.Result

		// TODO(alee) decide if we want to use other labels
		// For now we just add the hostname so we can select all the certs on one node
		labels := map[string]string{
			"hostname": nodeName,
		}

		dnsNames = allHostnames[nodeName]
		// ips = allIPs[nodeName]

		switch serviceName {
		default:
			// The default case provides a cert with all the dns names for the host.
			// This will probably be sufficient for most services.  If a service needs
			// a different kind of cert (for example, containing ips, or using a different
			// issuer)  then add a case for the service in this switch statement

			secretName = "cert-" + nodeName
			certSecret, _, err = secret.GetSecret(ctx, helper, secretName, instance.Namespace)
			if err != nil {
				if !k8serrors.IsNotFound(err) {
					err = fmt.Errorf("Error retrieving secret %s - %w", secretName, err)
					return ctrl.Result{}, err
				}

				certName = secretName
				duration := ptr.To(time.Hour * 24 * 365)
				certSecret, result, err = certmanager.EnsureCert(ctx, helper, certmanager.RootCAIssuerInternalLabel,
					certName, duration, dnsNames, nil, labels)
				if err != nil {
					return ctrl.Result{}, err
				} else if (result != ctrl.Result{}) {
					return result, nil
				}
			}
		}

		// TODO(alee) Add an owner reference to the secret so it can be monitored
		// We'll do this once stuggi adds a function to do this in libcommon

		// To use this cert, add it to the relevant service data
		// TODO(alee) We only need the cert and key.  The cacert will come from another label
		for key, value := range certSecret.Data {
			certsData[nodeName+"-"+key] = value
		}
	}

	// create a secret to hold the certs for the service
	serviceCertsSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetServiceCertsSecretName(instance, serviceName),
			Namespace: instance.Namespace,
		},
		Data: certsData,
	}
	_, result, err := secret.CreateOrPatchSecret(ctx, helper, instance, serviceCertsSecret)
	if err != nil {
		err = fmt.Errorf("Error creating certs secret for %s - %w", serviceName, err)
		return ctrl.Result{}, err
	} else if result != controllerutil.OperationResultNone {
		return ctrl.Result{RequeueAfter: time.Second * 5}, nil
	}

	return ctrl.Result{}, nil
}

// GetServiceCertsSecretName - return name of secret to be mounted in ansibleEE which contains
// all the TLS certs for the relevant service
// The convention we use here is "<nodeset.name>-<service>-certs", so for example,
// openstack-epdm-nova-certs.
func GetServiceCertsSecretName(instance *dataplanev1.OpenStackDataPlaneNodeSet, serviceName string) string {
	return fmt.Sprintf("%s-%s-certs", instance.Name, serviceName)
}
