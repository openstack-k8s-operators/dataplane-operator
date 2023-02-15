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

	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Deploy(obj client.Object, ctx context.Context, helper *helper.Helper, sshKeySecret string, inventoryConfigMap string) error {

	// ConfigureNetwork
	err := ConfigureNetwork(obj, ctx, helper, sshKeySecret, inventoryConfigMap)
	if err != nil {
		util.LogErrorForObject(helper, err, fmt.Sprintf("Unable to configure network for %s", obj.GetName()), obj)
		return err
	}

	return nil
}
