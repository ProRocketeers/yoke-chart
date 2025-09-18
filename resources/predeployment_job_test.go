package resources

import (
	"testing"

	"github.com/ProRocketeers/yoke-chart/schema"
	"github.com/google/go-cmp/cmp"
	"github.com/jinzhu/copier"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestPreDeploymentJob(t *testing.T) {
	type CaseConfig struct {
		ValuesTransform func(*DeploymentValues)
		Asserts         func(*testing.T, *batchv1.Job)
	}

	cases := map[string]func() CaseConfig{
		"renders a job if specified": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {},
				Asserts: func(t *testing.T, j *batchv1.Job) {
					require.NotNil(t, j)
					assert.Equal(t, "service--component--test--pre-deploy", j.Name)
				},
			}
		},
		"accepts Kubernetes JobSpec overrides": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.PreDeploymentJob.JobSpec.ActiveDeadlineSeconds = ptr.To(int64(300))
					dv.PreDeploymentJob.JobSpec.BackoffLimit = ptr.To(int32(8))
					dv.PreDeploymentJob.JobSpec.CompletionMode = ptr.To(batchv1.IndexedCompletion)
					dv.PreDeploymentJob.JobSpec.Completions = ptr.To(int32(3))
					dv.PreDeploymentJob.JobSpec.Parallelism = ptr.To(int32(5))
					dv.PreDeploymentJob.JobSpec.Selector = &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"label": "value",
						},
					}
					dv.PreDeploymentJob.JobSpec.Suspend = ptr.To(false)
					dv.PreDeploymentJob.JobSpec.TTLSecondsAfterFinished = ptr.To(int32(20))
				},
				Asserts: func(t *testing.T, j *batchv1.Job) {
					assert.Equal(t, ptr.To(int64(300)), j.Spec.ActiveDeadlineSeconds)
					assert.Equal(t, ptr.To(int32(8)), j.Spec.BackoffLimit)
					assert.Equal(t, ptr.To(batchv1.IndexedCompletion), j.Spec.CompletionMode)
					assert.Equal(t, ptr.To(int32(3)), j.Spec.Completions)
					assert.Equal(t, ptr.To(int32(5)), j.Spec.Parallelism)
					assert.Equal(t, map[string]string{"label": "value"}, j.Spec.Selector.MatchLabels)
					assert.Equal(t, ptr.To(false), j.Spec.Suspend)
					assert.Equal(t, ptr.To(int32(20)), j.Spec.TTLSecondsAfterFinished)
				},
			}
		},
		"supports init containers": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.PreDeploymentJob.InitContainers = []Container{
						{
							Name: "init-1",
							Image: Image{
								Repository: "init1_repository",
								Tag:        ptr.To("init1_tag"),
							},
						},
						{
							Name: "init-2",
							Image: Image{
								Repository: "init2_repository",
								Tag:        ptr.To("init2_tag"),
							},
						},
					}
				},
				Asserts: func(t *testing.T, j *batchv1.Job) {
					require.Len(t, j.Spec.Template.Spec.InitContainers, 2)
					assert.Equal(t, "init-1", j.Spec.Template.Spec.InitContainers[0].Name)
					assert.Equal(t, "init1_repository:init1_tag", j.Spec.Template.Spec.InitContainers[0].Image)
					assert.Equal(t, "init-2", j.Spec.Template.Spec.InitContainers[1].Name)
					assert.Equal(t, "init2_repository:init2_tag", j.Spec.Template.Spec.InitContainers[1].Image)
				},
			}
		},
		"supports volumes": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.PreDeploymentJob.Volumes = map[string]schema.Volume{
						"secretVolume": {
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
				Asserts: func(t *testing.T, j *batchv1.Job) {
					partialContains(t, j.Spec.Template.Spec.Volumes, corev1.Volume{
						Name: "secretVolume",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName:  "mySecret",
								DefaultMode: ptr.To(int32(0444)),
								Items:       nil,
							},
						},
					}, cmp.Options{})
					partialContains(t, j.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
						Name:      "secretVolume",
						MountPath: "/secret",
						ReadOnly:  true,
					}, cmp.Options{})
				},
			}
		},
		"renders user defined job/pod annotations/labels": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.PreDeploymentJob.Annotations = map[string]string{
						"job-annotation": "foo",
					}
					dv.PreDeploymentJob.Labels = map[string]string{
						"job-label": "foo",
					}
					dv.PreDeploymentJob.PodAnnotations = map[string]string{
						"pod-annotation": "bar",
					}
					dv.PreDeploymentJob.PodLabels = map[string]string{
						"pod-label": "bar",
					}
				},
				Asserts: func(t *testing.T, j *batchv1.Job) {
					require.Contains(t, j.Annotations, "job-annotation")
					assert.Equal(t, "foo", j.Annotations["job-annotation"])

					require.Contains(t, j.Labels, "job-label")
					assert.Equal(t, "foo", j.Labels["job-label"])

					require.Contains(t, j.Spec.Template.Annotations, "pod-annotation")
					assert.Equal(t, "bar", j.Spec.Template.Annotations["pod-annotation"])

					require.Contains(t, j.Spec.Template.Labels, "pod-label")
					assert.Equal(t, "bar", j.Spec.Template.Labels["pod-label"])
				},
			}
		},
		"renders scrape labels if pod monitor is enabled": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.PreDeploymentJob.PodMonitor = &schema.PodMonitor{
						Enabled:   ptr.To(true),
						Endpoints: []monitoringv1.PodMetricsEndpoint{},
					}
				},
				Asserts: func(t *testing.T, j *batchv1.Job) {
					assert.Subset(t, j.Spec.Template.Labels, map[string]string{
						"app":               "service--component--test--pre-deploy",
						"prometheus-scrape": "true",
					})
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
		PreDeploymentJob: &PreDeploymentJob{
			Metadata: Metadata{
				Namespace:   "ns",
				Service:     "service",
				Component:   "component",
				Environment: "test",
			},
			Container: Container{
				Name: "main",
				Image: Image{
					Repository: "job_image_repository",
					Tag:        ptr.To("job_image_tag"),
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

			_, create := CreatePreDeploymentJob(values)
			resources, err := create(values)
			if err != nil {
				t.Errorf("error during test setup: %v", err)
			}

			config.Asserts(t, fromUnstructuredOrPanic[*batchv1.Job](resources[0]))
		})
	}

	t.Run("doesn't render when PDJ is not specified", func(t *testing.T) {
		values := DeploymentValues{}
		copier.CopyWithOption(&values, &base, copier.Option{DeepCopy: true})
		values.PreDeploymentJob = nil

		shouldCreate, _ := CreatePreDeploymentJob(values)

		assert.False(t, shouldCreate)
	})
}
