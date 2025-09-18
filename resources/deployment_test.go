package resources

import (
	"testing"

	"github.com/ProRocketeers/yoke-chart/schema"
	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

func TestDeployment(t *testing.T) {
	type CaseConfig struct {
		ValuesTransform func(*DeploymentValues)
		Asserts         func(*testing.T, *appsv1.Deployment)
	}

	// a function to be able to create a closure and share variables to compare against etc.
	cases := map[string]func() CaseConfig{
		"allows overriding of deployment strategy": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Strategy = &appsv1.DeploymentStrategy{
						Type: appsv1.RecreateDeploymentStrategyType,
					}
				},
				Asserts: func(t *testing.T, d *appsv1.Deployment) {
					assert.Equal(t, appsv1.RecreateDeploymentStrategyType, d.Spec.Strategy.Type)
				},
			}
		},
		"renders image pull secrets from all containers": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Containers[0].Image.PullSecrets = []string{"pull-secret-one"}
					dv.Containers = append(dv.Containers, Container{
						Name: "foo",
						Image: Image{
							Repository:  "sidecar_repository",
							Tag:         ptr.To("sidecar_tag"),
							PullSecrets: []string{"pull-secret-two"},
						},
					})
					dv.InitContainers = []Container{
						{
							Name: "init",
							Image: Image{
								Repository:  "init_repository",
								Tag:         ptr.To("init_tag"),
								PullSecrets: []string{"pull-secret-three"},
							},
						},
					}
				},
				Asserts: func(t *testing.T, d *appsv1.Deployment) {
					assert.Len(t, d.Spec.Template.Spec.ImagePullSecrets, 3)
					assert.Contains(t, d.Spec.Template.Spec.ImagePullSecrets, corev1.LocalObjectReference{
						Name: "pull-secret-one",
					})
					assert.Contains(t, d.Spec.Template.Spec.ImagePullSecrets, corev1.LocalObjectReference{
						Name: "pull-secret-two",
					})
					assert.Contains(t, d.Spec.Template.Spec.ImagePullSecrets, corev1.LocalObjectReference{
						Name: "pull-secret-three",
					})
				},
			}
		},
		"renders init containers": func() CaseConfig {
			name := "init"
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.InitContainers = []Container{
						{
							Name: name,
							Image: Image{
								Repository: "init_repository",
								Tag:        ptr.To("init_tag"),
							},
						},
					}
				},
				Asserts: func(t *testing.T, d *appsv1.Deployment) {
					require.Len(t, d.Spec.Template.Spec.InitContainers, 1)
					assert.Equal(t, name, d.Spec.Template.Spec.InitContainers[0].Name)
				},
			}
		},
		"renders optional values if specified: nodeSelector, affinity, tolerations": func() CaseConfig {
			nodeSelector := map[string]string{
				"node": "value",
			}
			affinity := corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									{
										Key:      "topology.kubernetes.io/zone",
										Operator: corev1.NodeSelectorOpIn,
										Values:   []string{"east1"},
									},
								},
							},
						},
					},
				},
			}
			toleration := corev1.Toleration{
				Key:               "node.kubernetes.io/unreachable",
				Operator:          corev1.TolerationOpExists,
				Effect:            corev1.TaintEffectNoExecute,
				TolerationSeconds: ptr.To(int64(6000)),
			}
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.NodeSelector = nodeSelector
					dv.Affinity = &affinity
					dv.Tolerations = []corev1.Toleration{toleration}
				},
				Asserts: func(t *testing.T, d *appsv1.Deployment) {
					require.Contains(t, d.Spec.Template.Spec.NodeSelector, "node")
					assert.Equal(t, "value", d.Spec.Template.Spec.NodeSelector["node"])

					assert.Equal(t, affinity.NodeAffinity, d.Spec.Template.Spec.Affinity.NodeAffinity)
					assert.Contains(t, d.Spec.Template.Spec.Tolerations, toleration)
				},
			}
		},
		"properly renders secret volume": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Volumes = map[string]schema.Volume{
						"my-secret-volume": {
							Type: schema.VolumeTypeSecret,
							Mounts: map[string]schema.VolumeMount{
								"main": {
									ContainerPath: "/secret",
								},
							},
							Variant: schema.SecretVolume{
								SecretName: "mySecret",
							},
						},
					}
				},
				Asserts: func(t *testing.T, d *appsv1.Deployment) {
					assert.Contains(t, d.Spec.Template.Spec.Volumes, corev1.Volume{
						Name: "my-secret-volume",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName:  "mySecret",
								DefaultMode: ptr.To(int32(0444)),
								Items:       []corev1.KeyToPath{},
							},
						},
					})
					assert.Contains(t, d.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
						Name:      "my-secret-volume",
						MountPath: "/secret",
						ReadOnly:  true,
					})
				},
			}
		},
		"properly renders configmap volume, including items": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Volumes = map[string]schema.Volume{
						"my-cm-volume": {
							Type: schema.VolumeTypeConfigMap,
							Mounts: map[string]schema.VolumeMount{
								"main": {
									ContainerPath: "/cm",
								},
							},
							Variant: schema.ConfigMapVolume{
								ConfigMapName: "my-cm",
								Items: map[string]*string{
									"foo.cfg":     ptr.To("key"),
									"another.cfg": nil,
								},
							},
						},
					}
				},
				Asserts: func(t *testing.T, d *appsv1.Deployment) {
					volume := d.Spec.Template.Spec.Volumes[0]

					assert.Equal(t, "my-cm-volume", volume.Name)
					assert.Equal(t, "my-cm", volume.VolumeSource.ConfigMap.Name)
					assert.Equal(t, ptr.To(int32(0444)), volume.ConfigMap.DefaultMode)
					assert.Contains(t, volume.VolumeSource.ConfigMap.Items, corev1.KeyToPath{Key: "key", Path: "foo.cfg"})
					assert.Contains(t, volume.VolumeSource.ConfigMap.Items, corev1.KeyToPath{Key: "another.cfg", Path: "another.cfg"})

					assert.Contains(t, d.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
						Name:      "my-cm-volume",
						MountPath: "/cm",
						ReadOnly:  true,
					})
				},
			}
		},
		"properly renders raw volume": func() CaseConfig {
			volumeName := "my-raw-volume"
			spec := corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: volumeName,
					},
					Items: []corev1.KeyToPath{
						{Key: "config-file", Path: "./config-file"},
					},
				},
			}
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Volumes = map[string]schema.Volume{
						volumeName: {
							Type: schema.VolumeTypeRaw,
							Mounts: map[string]schema.VolumeMount{
								"main": {
									ContainerPath: "/raw",
								},
							},
							Variant: schema.RawVolume{
								Spec: spec,
							},
						},
					}
				},
				Asserts: func(t *testing.T, d *appsv1.Deployment) {
					assert.Contains(t, d.Spec.Template.Spec.Volumes, corev1.Volume{
						Name:         volumeName,
						VolumeSource: spec,
					})
					assert.Contains(t, d.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
						Name:             volumeName,
						MountPath:        "/raw",
						ReadOnly:         false,
						MountPropagation: ptr.To(corev1.MountPropagationHostToContainer),
					})
				},
			}
		},
		"properly renders existing persistent volume": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Volumes = map[string]schema.Volume{
						"existing-persistent": {
							Type: schema.VolumeTypePersistent,
							Mounts: map[string]schema.VolumeMount{
								"main": {
									ContainerPath: "/persistent-existing",
								},
							},
							Variant: schema.PersistentVolume{
								Existing: ptr.To(true),
								Variant: schema.PersistentVolumeExisting{
									PvcName: "my-pvc",
								},
							},
						},
					}
				},
				Asserts: func(t *testing.T, d *appsv1.Deployment) {
					assert.Contains(t, d.Spec.Template.Spec.Volumes, corev1.Volume{
						Name: "existing-persistent",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: "my-pvc",
							},
						},
					})
					assert.Contains(t, d.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
						Name:             "existing-persistent",
						MountPath:        "/persistent-existing",
						ReadOnly:         false,
						MountPropagation: ptr.To(corev1.MountPropagationHostToContainer),
					})
				},
			}
		},
		"properly renders nonexisting persistent volume": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Volumes = map[string]schema.Volume{
						"nonexisting-persistent": {
							Type: schema.VolumeTypePersistent,
							Mounts: map[string]schema.VolumeMount{
								"main": {
									ContainerPath: "/persistent-nonexisting",
								},
							},
							Variant: schema.PersistentVolume{
								Existing: ptr.To(false),
								Variant: schema.PersistentVolumeNew{
									Size:             "2Gi",
									StorageClassName: "my-sc",
								},
							},
						},
					}
				},
				Asserts: func(t *testing.T, d *appsv1.Deployment) {
					assert.Contains(t, d.Spec.Template.Spec.Volumes, corev1.Volume{
						Name: "nonexisting-persistent",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: "service--component--test--nonexisting-persistent",
							},
						},
					})
					assert.Contains(t, d.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
						Name:             "nonexisting-persistent",
						MountPath:        "/persistent-nonexisting",
						ReadOnly:         false,
						MountPropagation: ptr.To(corev1.MountPropagationHostToContainer),
					})
				},
			}
		},
		"properly renders default tmpfs volume": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Volumes = map[string]schema.Volume{
						"tmpfs-volume": {
							Type: schema.VolumeTypeStandardTmpfs,
							Mounts: map[string]schema.VolumeMount{
								"main": {
									ContainerPath: "/tmpfs",
								},
							},
							Variant: schema.StandardVolume{},
						},
					}
				},
				Asserts: func(t *testing.T, d *appsv1.Deployment) {
					assert.Contains(t, d.Spec.Template.Spec.Volumes, corev1.Volume{
						Name: "tmpfs-volume",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{
								Medium: corev1.StorageMediumMemory,
							},
						},
					})
					assert.Contains(t, d.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
						Name:             "tmpfs-volume",
						MountPath:        "/tmpfs",
						ReadOnly:         false,
						MountPropagation: ptr.To(corev1.MountPropagationHostToContainer),
					})
				},
			}
		},
		"properly renders local volume": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Volumes = map[string]schema.Volume{
						"local-volume": {
							Type: schema.VolumeTypeStandardLocal,
							Mounts: map[string]schema.VolumeMount{
								"main": {
									ContainerPath: "/local",
								},
							},
							Variant: schema.StandardVolume{},
						},
					}
				},
				Asserts: func(t *testing.T, d *appsv1.Deployment) {
					assert.Contains(t, d.Spec.Template.Spec.Volumes, corev1.Volume{
						Name: "local-volume",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{
								Medium: corev1.StorageMediumDefault,
							},
						},
					})
					assert.Contains(t, d.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
						Name:             "local-volume",
						MountPath:        "/local",
						ReadOnly:         false,
						MountPropagation: ptr.To(corev1.MountPropagationHostToContainer),
					})
				},
			}
		},
		"renders container image tags as pod annotations as well as user pod annotations": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.PodAnnotations = map[string]string{
						"my-annotation": "foo",
					}
				},
				Asserts: func(t *testing.T, d *appsv1.Deployment) {
					require.Contains(t, d.Spec.Template.Annotations, "my-annotation")
					assert.Equal(t, "foo", d.Spec.Template.Annotations["my-annotation"])
					assert.Equal(t, "image_tag", d.Spec.Template.Annotations["container-main-image-tag"])
				},
			}
		},
		"renders user defined deployment/pod annotations/labels": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Annotations = map[string]string{
						"deployment-annotation": "foo",
					}
					dv.Labels = map[string]string{
						"deployment-label": "foo",
					}
					dv.PodAnnotations = map[string]string{
						"pod-annotation": "bar",
					}
					dv.PodLabels = map[string]string{
						"pod-label": "bar",
					}
				},
				Asserts: func(t *testing.T, d *appsv1.Deployment) {
					require.Contains(t, d.Annotations, "deployment-annotation")
					assert.Equal(t, "foo", d.Annotations["deployment-annotation"])

					require.Contains(t, d.Labels, "deployment-label")
					assert.Equal(t, "foo", d.Labels["deployment-label"])

					require.Contains(t, d.Spec.Template.Annotations, "pod-annotation")
					assert.Equal(t, "bar", d.Spec.Template.Annotations["pod-annotation"])

					require.Contains(t, d.Spec.Template.Labels, "pod-label")
					assert.Equal(t, "bar", d.Spec.Template.Labels["pod-label"])
				},
			}
		},
		"automatically uses created service account": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {},
				Asserts: func(t *testing.T, d *appsv1.Deployment) {
					assert.Equal(t, "service--component--test", d.Spec.Template.Spec.ServiceAccountName)
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

			_, create := CreateDeployment(values)
			resources, err := create(values)
			if err != nil {
				t.Errorf("errpr during test setup: %v", err)
			}
			deployment := resources[0].(*appsv1.Deployment)

			config.Asserts(t, deployment)
		})
	}
}
