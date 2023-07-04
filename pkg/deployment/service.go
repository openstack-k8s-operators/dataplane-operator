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
	"os"
	"path"
	"strings"

	yaml "gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	dataplanev1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	dataplaneutil "github.com/openstack-k8s-operators/dataplane-operator/pkg/util"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	ansibleeev1 "github.com/openstack-k8s-operators/openstack-ansibleee-operator/api/v1alpha1"
)

// ServiceYAML struct for service YAML unmarshalling
type ServiceYAML struct {
	Kind     string
	Metadata yaml.Node
	Spec     yaml.Node
}

// DeployService service deployment
func DeployService(ctx context.Context, helper *helper.Helper, obj client.Object, sshKeySecret string, inventoryConfigMap string, aeeSpec dataplanev1.AnsibleEESpec, foundService dataplanev1.OpenStackDataPlaneService) error {
	role := ansibleeev1.Role{}
	if foundService.Spec.Role != nil {
		role = *foundService.Spec.Role
	}
	err := dataplaneutil.AnsibleExecution(ctx, helper, obj, foundService.Spec.Label, sshKeySecret, inventoryConfigMap, foundService.Spec.Play, role, aeeSpec)
	if err != nil {
		helper.GetLogger().Error(err, fmt.Sprintf("Unable to execute Ansible for %s", foundService.Name))
		return err
	}

	return nil

}

// GetServices returns the list of services for the node's role
// Note that these are not inherited from NodeTemplate.
func GetServices(instance *dataplanev1.OpenStackDataPlaneNode,
	instanceRole *dataplanev1.OpenStackDataPlaneRole) []string {
	return instanceRole.Spec.Services
}

// GetService return service
func GetService(ctx context.Context, helper *helper.Helper, service string) (dataplanev1.OpenStackDataPlaneService, error) {
	client := helper.GetClient()
	beforeObj := helper.GetBeforeObject()
	namespace := beforeObj.GetNamespace()
	foundService := &dataplanev1.OpenStackDataPlaneService{}
	err := client.Get(ctx, types.NamespacedName{Name: service, Namespace: namespace}, foundService)
	return *foundService, err
}

// EnsureServices - ensure the OpenStackDataPlaneServices exist
func EnsureServices(ctx context.Context, helper *helper.Helper, instance *dataplanev1.OpenStackDataPlaneRole) error {
	servicesPath, found := os.LookupEnv("OPERATOR_SERVICES")
	if !found {
		servicesPath = "config/services"
		os.Setenv("OPERATOR_SERVICES", servicesPath)
		util.LogForObject(
			helper, "OPERATOR_SERVICES not set in env when reconciling ", instance,
			"defaulting to ", servicesPath)
	}

	helper.GetLogger().Info("Ensuring services", "servicesPath", servicesPath)
	services, err := os.ReadDir(servicesPath)
	if err != nil {
		return err
	}

	for _, service := range services {

		servicePath := path.Join(servicesPath, service.Name())

		if !strings.HasSuffix(service.Name(), ".yaml") {
			helper.GetLogger().Info("Skipping ensuring service from file without .yaml suffix", "file", service.Name())
			continue
		}

		data, _ := os.ReadFile(servicePath)
		var serviceObj ServiceYAML
		err = yaml.Unmarshal(data, &serviceObj)
		if err != nil {
			helper.GetLogger().Info("Service YAML file Unmarshal error", "service YAML file", servicePath)
			return err
		}

		if serviceObj.Kind != "OpenStackDataPlaneService" {
			helper.GetLogger().Info("Skipping ensuring service since kind is not OpenStackDataPlaneService", "file", servicePath, "Kind", serviceObj.Kind)
			continue
		}

		serviceObjMeta := &metav1.ObjectMeta{}
		err = serviceObj.Metadata.Decode(serviceObjMeta)
		if err != nil {
			helper.GetLogger().Info("Service Metadata decode error")
			return err
		}

		roleContainsService := false
		for _, roleServiceName := range instance.Spec.Services {
			if roleServiceName == serviceObjMeta.Name {
				roleContainsService = true
				break
			}
		}
		if !roleContainsService {
			helper.GetLogger().Info("Skipping ensure service since it is not a service on this role", "service", serviceObjMeta.Name)
			continue
		}

		serviceObjSpec := &dataplanev1.OpenStackDataPlaneServiceSpec{}
		err = serviceObj.Spec.Decode(serviceObjSpec)
		if err != nil {
			helper.GetLogger().Info("Service Spec decode error")
			return err
		}

		ensureService := &dataplanev1.OpenStackDataPlaneService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceObjMeta.Name,
				Namespace: instance.Namespace,
			},
		}
		_, err = controllerutil.CreateOrPatch(ctx, helper.GetClient(), ensureService, func() error {
			if len(instance.Spec.OpenStackAnsibleEERunnerImage) > 0 {
				serviceObjSpec.OpenStackAnsibleEERunnerImage = instance.Spec.OpenStackAnsibleEERunnerImage
			}
			ensureService.Spec = *serviceObjSpec
			return nil
		})
		if err != nil {
			return fmt.Errorf("Error ensuring service: %w", err)
		}
	}

	return nil
}
