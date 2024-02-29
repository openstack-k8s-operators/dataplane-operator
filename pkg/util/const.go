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

const (
	// AnsibleExecutionServiceNameLen max length for the ansibleEE service name prefix
	AnsibleExecutionServiceNameLen = 53
	SubscriptionPlay               = `
- hosts: all
  strategy: linear
  tasks:
    - name: subscription-manager register
      become: true
      no_log: true
      when: ansible_facts.distribution == 'RedHat'
      ansible.builtin.shell: |
        set -euxo pipefail
        subscription-manager register --username {{ lookup('ansible.builtin.env', 'SECRET_USERNAME') }} --password {{ lookup('ansible.builtin.env', 'SECRET_PASSWORD') }}
`
)
