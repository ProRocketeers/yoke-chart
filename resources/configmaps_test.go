package resources

import (
	"slices"
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

		cm := resources[0].(*corev1.ConfigMap)

		assert.Equal(t, "service--component--test-foo", cm.Name)
		assert.Equal(t, map[string]string{"value": "baz"}, cm.Data)
	})

	t.Run("renders multiple ConfigMaps", func(t *testing.T) {
		findConfigMapIndexByName := func(cms []*corev1.ConfigMap, name string) int {
			return slices.IndexFunc(cms, func(s *corev1.ConfigMap) bool {
				return s.Name == name
			})
		}

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

		configMaps := []*corev1.ConfigMap{}
		for i := range resources {
			if c, ok := resources[i].(*corev1.ConfigMap); ok {
				configMaps = append(configMaps, c)
			} else {
				t.Error("error while retyping configmaps in test setup")
			}
		}
		assert.NotEqual(t, -1, findConfigMapIndexByName(configMaps, "service--component--test-foo"))
		assert.NotEqual(t, -1, findConfigMapIndexByName(configMaps, "service--component--test-bar"))
	})
}
