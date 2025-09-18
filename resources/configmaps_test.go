package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestConfigMaps(t *testing.T) {
	commonMetadata := Metadata{
		Namespace:   "ns",
		Service:     "service",
		Component:   "component",
		Environment: "test",
	}

	t.Run("renders a single ConfigMap", func(t *testing.T) {
		values := DeploymentValues{
			Metadata: commonMetadata,
			ConfigMaps: map[string]map[string]string{
				"foo": {
					"value": "baz",
				},
			},
		}

		shouldCreate, createFn := CreateConfigMaps(values)

		require.True(t, shouldCreate)

		resources, err := createFn(values)
		require.NoError(t, err)

		cm := fromUnstructuredOrPanic[*corev1.ConfigMap](resources[0])

		assert.Equal(t, "service--component--test-foo", cm.Name)
		assert.Equal(t, map[string]string{"value": "baz"}, cm.Data)
	})

	t.Run("renders multiple ConfigMaps", func(t *testing.T) {
		values := DeploymentValues{
			Metadata: commonMetadata,
			ConfigMaps: map[string]map[string]string{
				"foo": {
					"value": "baz",
				},
				"bar": {
					"some": "value",
				},
			},
		}

		shouldCreate, createFn := CreateConfigMaps(values)

		require.True(t, shouldCreate)

		resources, err := createFn(values)
		require.NoError(t, err)

		require.Len(t, resources, 2)

		findResourceOrFail[*corev1.ConfigMap](t, resources, "ConfigMap", "service--component--test-foo")
		findResourceOrFail[*corev1.ConfigMap](t, resources, "ConfigMap", "service--component--test-bar")
	})
}
