package resources

import (
	"testing"

	"github.com/ProRocketeers/yoke-chart/schema"
	"github.com/google/go-cmp/cmp"
	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
					dv.SchedulingConfig = schema.SchedulingConfig{
						NodeSelector: nodeSelector,
						Affinity:     &affinity,
						Tolerations:  []corev1.Toleration{toleration},
					}
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
							Mounts: map[string]schema.VolumeMountList{
								"main": {
									{ContainerPath: "/secret"},
								},
							},
							Variant: schema.SecretVolume{
								SecretName: "mySecret",
							},
						},
					}
				},
				Asserts: func(t *testing.T, d *appsv1.Deployment) {
					partialContains(t, d.Spec.Template.Spec.Volumes, corev1.Volume{
						Name: "my-secret-volume",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName:  "mySecret",
								DefaultMode: ptr.To(int32(0444)),
								Items:       nil,
							},
						},
					}, cmp.Options{})
					partialContains(t, d.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
						Name:      "my-secret-volume",
						MountPath: "/secret",
						ReadOnly:  true,
					}, cmp.Options{})
				},
			}
		},
		"subPath places a single secret key at an exact path without exposing the rest of the secret": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Volumes = map[string]schema.Volume{
						"payments-api-tls": {
							Type: schema.VolumeTypeSecret,
							Mounts: map[string]schema.VolumeMountList{
								"main": {
									{ContainerPath: "/etc/ssl/certs/payments-api.crt", VolumePath: ptr.To("tls.crt")},
								},
							},
							Variant: schema.SecretVolume{
								SecretName: "payments-api-tls",
							},
						},
					}
				},
				Asserts: func(t *testing.T, d *appsv1.Deployment) {
					partialContains(t, d.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
						Name:      "payments-api-tls",
						MountPath: "/etc/ssl/certs/payments-api.crt",
						SubPath:   "tls.crt",
						ReadOnly:  true,
					}, cmp.Options{})
				},
			}
		},
		"properly renders configmap volume, including items": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Volumes = map[string]schema.Volume{
						"my-cm-volume": {
							Type: schema.VolumeTypeConfigMap,
							Mounts: map[string]schema.VolumeMountList{
								"main": {
									{ContainerPath: "/cm"},
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
		"supports mounting the same volume into a container more than once, at different subPaths": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Volumes = map[string]schema.Volume{
						"payments-api-config": {
							Type: schema.VolumeTypeConfigMap,
							Mounts: map[string]schema.VolumeMountList{
								"main": {
									{ContainerPath: "/etc/payments-api/app.yaml", VolumePath: ptr.To("app.yaml")},
									{ContainerPath: "/etc/payments-api/logging.yaml", VolumePath: ptr.To("logging.yaml")},
								},
							},
							Variant: schema.ConfigMapVolume{
								ConfigMapName: "payments-api-config",
							},
						},
					}
				},
				Asserts: func(t *testing.T, d *appsv1.Deployment) {
					mounts := d.Spec.Template.Spec.Containers[0].VolumeMounts
					require.Len(t, mounts, 2)
					assert.Contains(t, mounts, corev1.VolumeMount{
						Name:      "payments-api-config",
						MountPath: "/etc/payments-api/app.yaml",
						SubPath:   "app.yaml",
						ReadOnly:  true,
					})
					assert.Contains(t, mounts, corev1.VolumeMount{
						Name:      "payments-api-config",
						MountPath: "/etc/payments-api/logging.yaml",
						SubPath:   "logging.yaml",
						ReadOnly:  true,
					})
				},
			}
		},
		"properly renders raw volume (eg projected)": func() CaseConfig {
			// `raw` is the escape hatch for volume sources without a dedicated type of their own -
			// a projected volume (merging a TLS Secret and an app ConfigMap into one mount) is a
			// realistic example of that, and it should default to read-only just like `secret`/`configMap` do
			volumeName := "merged-app-config"
			spec := corev1.VolumeSource{
				Projected: &corev1.ProjectedVolumeSource{
					Sources: []corev1.VolumeProjection{
						{
							Secret: &corev1.SecretProjection{
								LocalObjectReference: corev1.LocalObjectReference{Name: "app-tls"},
								Items: []corev1.KeyToPath{
									{Key: "tls.crt", Path: "tls/tls.crt"},
									{Key: "tls.key", Path: "tls/tls.key"},
								},
							},
						},
						{
							ConfigMap: &corev1.ConfigMapProjection{
								LocalObjectReference: corev1.LocalObjectReference{Name: "app-config"},
								Items: []corev1.KeyToPath{
									{Key: "app.yaml", Path: "config/app.yaml"},
								},
							},
						},
					},
				},
			}
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Volumes = map[string]schema.Volume{
						volumeName: {
							Type: schema.VolumeTypeRaw,
							Mounts: map[string]schema.VolumeMountList{
								"main": {
									{ContainerPath: "/etc/app/merged"},
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
						Name:      volumeName,
						MountPath: "/etc/app/merged",
						ReadOnly:  true,
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
							Mounts: map[string]schema.VolumeMountList{
								"main": {
									{ContainerPath: "/persistent-existing"},
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
							Mounts: map[string]schema.VolumeMountList{
								"main": {
									{ContainerPath: "/persistent-nonexisting"},
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
							Mounts: map[string]schema.VolumeMountList{
								"main": {
									{ContainerPath: "/tmpfs"},
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
							Mounts: map[string]schema.VolumeMountList{
								"main": {
									{ContainerPath: "/local"},
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
		"readOnly can be overridden per mount": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Containers = append(dv.Containers, Container{
						Name: "cache-warmer",
						Image: Image{
							Repository: "cache_warmer_repository",
							Tag:        ptr.To("cache_warmer_tag"),
						},
					})
					dv.Volumes = map[string]schema.Volume{
						"shared-cache": {
							Type: schema.VolumeTypePersistent,
							Mounts: map[string]schema.VolumeMountList{
								// main writes to the cache, cache-warmer only ever needs to read it
								"main": {
									{ContainerPath: "/var/cache/app"},
								},
								"cache-warmer": {
									{ContainerPath: "/var/cache/app", ReadOnly: ptr.To(true)},
								},
							},
							Variant: schema.PersistentVolume{
								Existing: ptr.To(true),
								Variant: schema.PersistentVolumeExisting{
									PvcName: "payments-api-cache",
								},
							},
						},
					}
				},
				Asserts: func(t *testing.T, d *appsv1.Deployment) {
					assert.Contains(t, d.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
						Name:             "shared-cache",
						MountPath:        "/var/cache/app",
						ReadOnly:         false,
						MountPropagation: ptr.To(corev1.MountPropagationHostToContainer),
					})
					assert.Contains(t, d.Spec.Template.Spec.Containers[1].VolumeMounts, corev1.VolumeMount{
						Name:      "shared-cache",
						MountPath: "/var/cache/app",
						ReadOnly:  true,
					})
				},
			}
		},
		"mountPropagation can be explicitly overridden": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Containers = append(dv.Containers, Container{
						Name: "log-shipper",
						Image: Image{
							Repository: "log_shipper_repository",
							Tag:        ptr.To("log_shipper_tag"),
						},
					})
					dv.Volumes = map[string]schema.Volume{
						"scratch": {
							Type: schema.VolumeTypeStandardLocal,
							Mounts: map[string]schema.VolumeMountList{
								// log-shipper mounts additional filesystems under this path at runtime,
								// and needs those to propagate back so the main container can see them too
								"log-shipper": {
									{ContainerPath: "/var/scratch", MountPropagation: ptr.To(corev1.MountPropagationBidirectional)},
								},
							},
							Variant: schema.StandardVolume{},
						},
					}
				},
				Asserts: func(t *testing.T, d *appsv1.Deployment) {
					assert.Contains(t, d.Spec.Template.Spec.Containers[1].VolumeMounts, corev1.VolumeMount{
						Name:             "scratch",
						MountPath:        "/var/scratch",
						ReadOnly:         false,
						MountPropagation: ptr.To(corev1.MountPropagationBidirectional),
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
		"deploymentSpec merges in fields with no dedicated field, like minReadySeconds": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.DeploymentSpec = &appsv1.DeploymentSpec{
						MinReadySeconds:      10,
						RevisionHistoryLimit: ptr.To(int32(3)),
					}
				},
				Asserts: func(t *testing.T, d *appsv1.Deployment) {
					assert.Equal(t, int32(10), d.Spec.MinReadySeconds)
					assert.Equal(t, ptr.To(int32(3)), d.Spec.RevisionHistoryLimit)
					// chart-derived fields survive untouched since deploymentSpec didn't set them
					assert.Equal(t, "service--component--test", d.Spec.Selector.MatchLabels["app"])
				},
			}
		},
		"deploymentSpec can override selector/replicas - no protected fields": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.DeploymentSpec = &appsv1.DeploymentSpec{
						Replicas: ptr.To(int32(7)),
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"app": "replaced"},
						},
					}
				},
				Asserts: func(t *testing.T, d *appsv1.Deployment) {
					assert.Equal(t, ptr.To(int32(7)), d.Spec.Replicas)
					assert.Equal(t, "replaced", d.Spec.Selector.MatchLabels["app"])
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
			deployment := fromUnstructuredOrPanic[*appsv1.Deployment](resources[0])

			config.Asserts(t, deployment)
		})
	}
}
