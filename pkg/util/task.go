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

package util

import (
	ansibleeev1alpha1 "github.com/openstack-k8s-operators/openstack-ansibleee-operator/api/v1alpha1"
)

// Task defines the required data for import_role tasks
type Task struct {
	Name          string
	RoleName      string
	RoleTasksFrom string
	When          string
	Tags          []string
}

// PopulateTasks creates Tasks
func PopulateTasks(tasks []Task) []ansibleeev1alpha1.Task {
	var populated []ansibleeev1alpha1.Task
	for _, task := range tasks {
		aeeTask := ansibleeev1alpha1.Task{
			Name: task.Name,
			ImportRole: ansibleeev1alpha1.ImportRole{
				Name: task.RoleName,
			},
			Tags: task.Tags,
		}
		if len(task.RoleTasksFrom) > 0 {
			aeeTask.ImportRole.TasksFrom = task.RoleTasksFrom
		}
		if len(task.When) > 0 {
			aeeTask.When = task.When
		}
		populated = append(populated, aeeTask)
	}
	return populated
}
