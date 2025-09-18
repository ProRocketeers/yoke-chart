package resources

import (
	"testing"

	"github.com/ProRocketeers/yoke-chart/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestServiceAccount(t *testing.T) {
	commonMetadata := Metadata{
		Namespace:   "ns",
		Service:     "service",
		Component:   "component",
		Environment: "test",
	}

	t.Run("always gets created", func(t *testing.T) {
		values := DeploymentValues{
			Metadata: commonMetadata,
		}

		shouldCreate, createFn := CreateServiceAccount(values)

		// `require` package = failed assert => marks test as failed and exits test
		require.True(t, shouldCreate)

		resources, err := createFn(values)
		require.NoError(t, err)

		sa := fromUnstructuredOrPanic[*corev1.ServiceAccount](resources[0])

		// `assert` package = failed assert => marks test as failed but continues
		assert.Equal(t, serviceName(values.Metadata), sa.Name)
		assert.Equal(t, values.Metadata.Namespace, sa.Namespace)
	})

	t.Run("renders annotations if specified", func(t *testing.T) {
		expected := map[string]string{
			"my-annotation": "foo",
		}
		values := DeploymentValues{
			Metadata: commonMetadata,
			ServiceAccount: &schema.ServiceAccount{
				Annotations: expected,
			},
		}

		_, create := CreateServiceAccount(values)
		resources, _ := create(values)

		sa := fromUnstructuredOrPanic[*corev1.ServiceAccount](resources[0])

		assert.Equal(t, expected, sa.Annotations)
	})
}
