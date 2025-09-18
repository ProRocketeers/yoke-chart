package resources

import (
	"testing"

	"github.com/ProRocketeers/yoke-chart/schema"
	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

func TestPVC(t *testing.T) {
	type CaseConfig struct {
		ValuesTransform func(*DeploymentValues)
		Asserts         func(*testing.T, *corev1.PersistentVolumeClaim)
	}

	cases := map[string]func() CaseConfig{
		"will render a PVC for new persistent volumes": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Volumes = map[string]schema.Volume{
						"pvc": {
							Type: schema.VolumeTypePersistent,
							Mounts: map[string]schema.VolumeMount{
								"main": {
									ContainerPath: "/pvc",
								},
							},
							Variant: schema.PersistentVolume{
								Existing: ptr.To(false),
								Variant: schema.PersistentVolumeNew{
									StorageClassName: "my-sc",
									Size:             "5Gi",
								},
							},
						},
					}
				},
				Asserts: func(t *testing.T, pvc *corev1.PersistentVolumeClaim) {
					assert.Equal(t, "service--component--test--pvc", pvc.Name)
					assert.Equal(t, ptr.To("my-sc"), pvc.Spec.StorageClassName)
					assert.Equal(t, resource.MustParse("5Gi"), pvc.Spec.Resources.Requests[corev1.ResourceStorage])
					// defaults
					assert.Equal(t, []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}, pvc.Spec.AccessModes)
					assert.Equal(t, ptr.To(corev1.PersistentVolumeFilesystem), pvc.Spec.VolumeMode)
				},
			}
		},
		"can override accessModes": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Volumes = map[string]schema.Volume{
						"pvc": {
							Type: schema.VolumeTypePersistent,
							Mounts: map[string]schema.VolumeMount{
								"main": {
									ContainerPath: "/pvc",
								},
							},
							Variant: schema.PersistentVolume{
								Existing: ptr.To(false),
								Variant: schema.PersistentVolumeNew{
									AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
									StorageClassName: "my-sc",
									Size:             "5Gi",
								},
							},
						},
					}
				},
				Asserts: func(t *testing.T, pvc *corev1.PersistentVolumeClaim) {
					assert.Equal(t, []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}, pvc.Spec.AccessModes)
				},
			}
		},
		"can override volumeMode": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Volumes = map[string]schema.Volume{
						"pvc": {
							Type: schema.VolumeTypePersistent,
							Mounts: map[string]schema.VolumeMount{
								"main": {
									ContainerPath: "/pvc",
								},
							},
							Variant: schema.PersistentVolume{
								Existing: ptr.To(false),
								Variant: schema.PersistentVolumeNew{
									VolumeMode:       ptr.To(corev1.PersistentVolumeBlock),
									StorageClassName: "my-sc",
									Size:             "5Gi",
								},
							},
						},
					}
				},
				Asserts: func(t *testing.T, pvc *corev1.PersistentVolumeClaim) {
					assert.Equal(t, ptr.To(corev1.PersistentVolumeBlock), pvc.Spec.VolumeMode)
				},
			}
		},
	}

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
			},
		},
	}

	for testName, makeConfig := range cases {
		t.Run(testName, func(t *testing.T) {
			values := DeploymentValues{}
			copier.CopyWithOption(&values, &base, copier.Option{DeepCopy: true})

			config := makeConfig()
			config.ValuesTransform(&values)

			_, create := CreatePVCs(values)
			resources, err := create(values)
			if err != nil {
				t.Errorf("error during test setup: %v", err)
			}
			pvc := resources[0].(*corev1.PersistentVolumeClaim)

			config.Asserts(t, pvc)
		})
	}

	t.Run("should not create any PVCs if not specified", func(t *testing.T) {
		values := DeploymentValues{}
		copier.CopyWithOption(&values, &base, copier.Option{DeepCopy: true})

		shouldCreate, _ := CreatePVCs(values)
		assert.False(t, shouldCreate)
	})
}
