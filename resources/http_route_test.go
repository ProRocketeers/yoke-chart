package resources

import (
	"testing"

	"github.com/ProRocketeers/yoke-chart/schema"
	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/assert"
	"k8s.io/utils/ptr"
)

func TestHttpRoute(t *testing.T) {
	base := DeploymentValues{
		Metadata: Metadata{
			Namespace:   "ns",
			Service:     "service",
			Component:   "component",
			Environment: "test",
		},
		Containers: []Container{
			{
				Name: "main",
				Image: Image{
					Repository: "image_repository",
					Tag:        ptr.To("image_tag"),
				},
				Ports: []schema.Port{{Port: 8080}},
			},
		},
	}

	t.Run("doesn't render when httpRoute is not explicitly enabled", func(t *testing.T) {
		values := DeploymentValues{}
		copier.CopyWithOption(&values, &base, copier.Option{DeepCopy: true})
		values.HTTPRoute = &schema.HTTPRoute{
			Enabled: ptr.To(false),
		}

		shouldCreate, _ := CreateHttpRoute(values)

		assert.False(t, shouldCreate)
	})
}
