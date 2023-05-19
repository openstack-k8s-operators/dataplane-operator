package deployment

import (
	"testing"

	. "github.com/onsi/gomega"
	"k8s.io/utils/pointer"

	dataplanev1beta1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
)

func TestMergeNovaTemplates(t *testing.T) {
	RegisterTestingT(t)

	t.Run("No override", func(t *testing.T) {
		node := dataplanev1beta1.NovaTemplate{
			CustomServiceConfig: "foo=bar",
			Deploy:              nil,
		}

		role := dataplanev1beta1.NovaTemplate{
			CellName: "cell2",
			Deploy:   pointer.Bool(false),
		}

		merged, err := mergeNovaTemplates(node, role)

		Expect(err).NotTo(HaveOccurred())
		Expect(merged.CellName).To(Equal("cell2"))
		Expect(merged.CustomServiceConfig).To(Equal("foo=bar"))
		Expect(*merged.Deploy).To(BeFalse())
	})

	t.Run("Override", func(t *testing.T) {
		node := dataplanev1beta1.NovaTemplate{
			CustomServiceConfig: "foo=bar",
		}

		role := dataplanev1beta1.NovaTemplate{
			CustomServiceConfig: "baz=boo",
		}

		merged, err := mergeNovaTemplates(node, role)

		Expect(err).NotTo(HaveOccurred())
		Expect(merged.CustomServiceConfig).To(Equal("foo=bar"))
	})
	t.Run("Deploy override", func(t *testing.T) {
		node := dataplanev1beta1.NovaTemplate{
			Deploy: pointer.Bool(true),
		}

		role := dataplanev1beta1.NovaTemplate{
			Deploy: pointer.Bool(false),
		}

		merged, err := mergeNovaTemplates(node, role)

		Expect(err).NotTo(HaveOccurred())
		Expect(*merged.Deploy).To(BeTrue())
	})

	t.Run("Deploy only in node", func(t *testing.T) {
		node := dataplanev1beta1.NovaTemplate{
			Deploy: pointer.Bool(true),
		}

		role := dataplanev1beta1.NovaTemplate{
			Deploy: nil,
		}

		merged, err := mergeNovaTemplates(node, role)

		Expect(err).NotTo(HaveOccurred())
		Expect(*merged.Deploy).To(BeTrue())
	})
}
