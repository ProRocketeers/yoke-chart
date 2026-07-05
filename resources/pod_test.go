package resources

import (
	"testing"

	"github.com/ProRocketeers/yoke-chart/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
)

func TestVolumeMountTargetValidation(t *testing.T) {
	base := func() DeploymentValues {
		return DeploymentValues{
			Metadata: Metadata{
				Namespace:   "ns",
				Service:     "payments-api",
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
				},
			},
		}
	}

	t.Run("fails when a volume mount references a container that doesn't exist", func(t *testing.T) {
		values := base()
		values.Volumes = map[string]schema.Volume{
			"payments-api-config": {
				Type: schema.VolumeTypeConfigMap,
				Mounts: map[string]schema.VolumeMountList{
					// typo: the actual container is named "main"
					"mian": {
						{ContainerPath: "/etc/payments-api"},
					},
				},
				Variant: schema.ConfigMapVolume{
					ConfigMapName: "payments-api-config",
				},
			},
		}

		_, create := CreateDeployment(values)
		_, err := create(values)
		require.Error(t, err)
		assert.ErrorContains(t, err, "payments-api-config")
		assert.ErrorContains(t, err, "mian")
	})

	t.Run("succeeds when a volume mount references an init container", func(t *testing.T) {
		values := base()
		values.InitContainers = []Container{
			{
				Name: "migrate",
				Image: Image{
					Repository: "migrate_repository",
					Tag:        ptr.To("migrate_tag"),
				},
			},
		}
		values.Volumes = map[string]schema.Volume{
			"payments-api-config": {
				Type: schema.VolumeTypeConfigMap,
				Mounts: map[string]schema.VolumeMountList{
					"migrate": {
						{ContainerPath: "/etc/payments-api"},
					},
				},
				Variant: schema.ConfigMapVolume{
					ConfigMapName: "payments-api-config",
				},
			},
		}

		_, create := CreateDeployment(values)
		_, err := create(values)
		require.NoError(t, err)
	})
}
