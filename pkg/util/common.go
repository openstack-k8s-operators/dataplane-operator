package util

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CheckDeployProtectionAnnotation checks to see if the node has the deploy protection annotation set. If this is set,
// we will block the execution of Ansible tasks until it is cleared.
func CheckDeployProtectionAnnotation(instance client.Object, annotation map[string]string) (annotations map[string]string, keyExists bool) {
	annotations = instance.GetAnnotations()
	for k := range annotations {
		_, keyExists := annotation[k]
		return annotations, keyExists
	}
	return annotations, false
}
