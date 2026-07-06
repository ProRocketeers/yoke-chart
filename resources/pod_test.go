package resources

import (
	"testing"

	"github.com/ProRocketeers/yoke-chart/schema"
	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

func TestCreatePodSpec(t *testing.T) {
	type CaseConfig struct {
		ValuesTransform func(*DeploymentValues)
		Asserts         func(*testing.T, corev1.PodSpec, error)
	}

	cases := map[string]func() CaseConfig{
		"fails when a volume mount references a container that doesn't exist": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Volumes = map[string]schema.Volume{
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
				},
				Asserts: func(t *testing.T, podSpec corev1.PodSpec, err error) {
					require.Error(t, err)
					assert.ErrorContains(t, err, "payments-api-config")
					assert.ErrorContains(t, err, "mian")
				},
			}
		},
		"succeeds when a volume mount references an init container": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.InitContainers = []Container{
						{
							Name: "migrate",
							Image: Image{
								Repository: "migrate_repository",
								Tag:        ptr.To("migrate_tag"),
							},
						},
					}
					dv.Volumes = map[string]schema.Volume{
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
				},
				Asserts: func(t *testing.T, podSpec corev1.PodSpec, err error) {
					require.NoError(t, err)
				},
			}
		},
		"topologySpreadConstraints and priorityClassName from SchedulingConfig reach the pod spec": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.SchedulingConfig = schema.SchedulingConfig{
						PriorityClassName: ptr.To("high-priority"),
						TopologySpreadConstraints: []corev1.TopologySpreadConstraint{
							{MaxSkew: 1, TopologyKey: "zone"},
						},
					}
				},
				Asserts: func(t *testing.T, podSpec corev1.PodSpec, err error) {
					require.NoError(t, err)
					assert.Equal(t, "high-priority", podSpec.PriorityClassName)
					require.Len(t, podSpec.TopologySpreadConstraints, 1)
					assert.Equal(t, "zone", podSpec.TopologySpreadConstraints[0].TopologyKey)
				},
			}
		},
		"podSpec merges in fields with no dedicated field, like terminationGracePeriodSeconds": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.PodSpec = &corev1.PodSpec{
						TerminationGracePeriodSeconds: ptr.To(int64(120)),
						DNSPolicy:                     corev1.DNSNone,
					}
				},
				Asserts: func(t *testing.T, podSpec corev1.PodSpec, err error) {
					require.NoError(t, err)
					assert.Equal(t, ptr.To(int64(120)), podSpec.TerminationGracePeriodSeconds)
					assert.Equal(t, corev1.DNSNone, podSpec.DNSPolicy)
					// chart-derived fields survive untouched since podSpec didn't set them
					assert.Equal(t, "payments-api--component--test", podSpec.ServiceAccountName)
				},
			}
		},
		"podSpec overrides SchedulingConfig's priorityClassName when both are set": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.SchedulingConfig = schema.SchedulingConfig{
						PriorityClassName: ptr.To("low-priority"),
					}
					dv.PodSpec = &corev1.PodSpec{
						PriorityClassName: "high-priority",
					}
				},
				Asserts: func(t *testing.T, podSpec corev1.PodSpec, err error) {
					require.NoError(t, err)
					assert.Equal(t, "high-priority", podSpec.PriorityClassName)
				},
			}
		},
		"podSpec can override chart-built containers - no protected fields": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.PodSpec = &corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "replaced"},
						},
					}
				},
				Asserts: func(t *testing.T, podSpec corev1.PodSpec, err error) {
					require.NoError(t, err)
					require.Len(t, podSpec.Containers, 1)
					assert.Equal(t, "replaced", podSpec.Containers[0].Name)
				},
			}
		},
	}

	base := DeploymentValues{
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

	for testName, makeConfig := range cases {
		t.Run(testName, func(t *testing.T) {
			values := DeploymentValues{}
			copier.CopyWithOption(&values, &base, copier.Option{DeepCopy: true})

			config := makeConfig()
			config.ValuesTransform(&values)

			podSpec, err := createPodSpec(&values, values)

			config.Asserts(t, podSpec, err)
		})
	}
}
