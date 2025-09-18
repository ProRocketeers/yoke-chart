package resources

import (
	"testing"

	"github.com/ProRocketeers/yoke-chart/schema"
	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestCronjob(t *testing.T) {
	type CaseConfig struct {
		ValuesTransform func(*DeploymentValues)
		Asserts         func(*testing.T, []*batchv1.CronJob)
	}

	// a function to be able to create a closure and share variables to compare against etc.
	cases := map[string]func() CaseConfig{
		"renders a cronjob": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {},
				Asserts: func(t *testing.T, cj []*batchv1.CronJob) {
					require.Len(t, cj, 1)
					assert.Equal(t, "cronjob--test", cj[0].Name)
				},
			}
		},
		"can render multiple cronjobs": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Cronjobs = append(dv.Cronjobs, Cronjob{
						Metadata: dv.Metadata,
						Name:     "cronjob2",
						Schedule: "0 0 * * *",
						Container: Container{
							Name: "main",
							Image: Image{
								Repository: "cronjob2_image_repository",
								Tag:        ptr.To("cronjob2_image_tag"),
							},
						},
					})
				},
				Asserts: func(t *testing.T, cj []*batchv1.CronJob) {
					require.Len(t, cj, 2)
					assert.Equal(t, "cronjob--test", cj[0].Name)
					assert.Equal(t, "cronjob2--test", cj[1].Name)
				},
			}
		},
		"accepts Kube CronJobSpec overrides": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Cronjobs[0].Suspend = ptr.To(true)
					dv.Cronjobs[0].TimeZone = ptr.To("Europe/Prague")
					dv.Cronjobs[0].ConcurrencyPolicy = ptr.To(batchv1.ForbidConcurrent)
					dv.Cronjobs[0].StartingDeadlineSeconds = ptr.To(int64(5))
					dv.Cronjobs[0].SuccessfulJobsHistoryLimit = ptr.To(int32(5))
					dv.Cronjobs[0].FailedJobsHistoryLimit = ptr.To(int32(2))
				},
				Asserts: func(t *testing.T, cj []*batchv1.CronJob) {
					assert.Equal(t, ptr.To(true), cj[0].Spec.Suspend)
					assert.Equal(t, ptr.To("Europe/Prague"), cj[0].Spec.TimeZone)
					assert.Equal(t, batchv1.ForbidConcurrent, cj[0].Spec.ConcurrencyPolicy)
					assert.Equal(t, ptr.To(int64(5)), cj[0].Spec.StartingDeadlineSeconds)
					assert.Equal(t, ptr.To(int32(5)), cj[0].Spec.SuccessfulJobsHistoryLimit)
					assert.Equal(t, ptr.To(int32(2)), cj[0].Spec.FailedJobsHistoryLimit)
				},
			}
		},
		"accepts Kube JobSpec overrides": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Cronjobs[0].ActiveDeadlineSeconds = ptr.To(int64(300))
					dv.Cronjobs[0].BackoffLimit = ptr.To(int32(8))
					dv.Cronjobs[0].CompletionMode = ptr.To(batchv1.IndexedCompletion)
					dv.Cronjobs[0].Completions = ptr.To(int32(3))
					dv.Cronjobs[0].Parallelism = ptr.To(int32(5))
					dv.Cronjobs[0].Selector = &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"label": "value",
						},
					}
					dv.Cronjobs[0].TTLSecondsAfterFinished = ptr.To(int32(20))
				},
				Asserts: func(t *testing.T, cj []*batchv1.CronJob) {
					assert.Equal(t, ptr.To(int64(300)), cj[0].Spec.JobTemplate.Spec.ActiveDeadlineSeconds)
					assert.Equal(t, ptr.To(int32(8)), cj[0].Spec.JobTemplate.Spec.BackoffLimit)
					assert.Equal(t, ptr.To(batchv1.IndexedCompletion), cj[0].Spec.JobTemplate.Spec.CompletionMode)
					assert.Equal(t, ptr.To(int32(3)), cj[0].Spec.JobTemplate.Spec.Completions)
					assert.Equal(t, ptr.To(int32(5)), cj[0].Spec.JobTemplate.Spec.Parallelism)
					assert.Equal(t, map[string]string{"label": "value"}, cj[0].Spec.JobTemplate.Spec.Selector.MatchLabels)
					assert.Equal(t, ptr.To(int32(20)), cj[0].Spec.JobTemplate.Spec.TTLSecondsAfterFinished)
				},
			}
		},
		"supports init containers": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Cronjobs[0].InitContainers = []Container{
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
				Asserts: func(t *testing.T, cj []*batchv1.CronJob) {
					require.Len(t, cj[0].Spec.JobTemplate.Spec.Template.Spec.InitContainers, 2)
					assert.Equal(t, "init-1", cj[0].Spec.JobTemplate.Spec.Template.Spec.InitContainers[0].Name)
					assert.Equal(t, "init1_repository:init1_tag", cj[0].Spec.JobTemplate.Spec.Template.Spec.InitContainers[0].Image)
					assert.Equal(t, "init-2", cj[0].Spec.JobTemplate.Spec.Template.Spec.InitContainers[1].Name)
					assert.Equal(t, "init2_repository:init2_tag", cj[0].Spec.JobTemplate.Spec.Template.Spec.InitContainers[1].Image)
				},
			}
		},
		"supports volumes": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Cronjobs[0].Volumes = map[string]schema.Volume{
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
				Asserts: func(t *testing.T, cj []*batchv1.CronJob) {
					assert.Contains(t, cj[0].Spec.JobTemplate.Spec.Template.Spec.Volumes, corev1.Volume{
						Name: "secretVolume",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName:  "mySecret",
								DefaultMode: ptr.To(int32(0444)),
								Items:       []corev1.KeyToPath{},
							},
						},
					})
					assert.Contains(t, cj[0].Spec.JobTemplate.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
						Name:      "secretVolume",
						MountPath: "/secret",
						ReadOnly:  true,
					})
				},
			}
		},
		"renders user defined cronjob/job/pod annotations/labels": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Cronjobs[0].CronJobAnnotations = map[string]string{
						"cronjob-annotation": "foo",
					}
					dv.Cronjobs[0].CronJobLabels = map[string]string{
						"cronjob-label": "foo",
					}
					dv.Cronjobs[0].JobAnnotations = map[string]string{
						"job-annotation": "bar",
					}
					dv.Cronjobs[0].JobLabels = map[string]string{
						"job-label": "bar",
					}
					dv.Cronjobs[0].PodAnnotations = map[string]string{
						"pod-annotation": "baz",
					}
					dv.Cronjobs[0].PodLabels = map[string]string{
						"pod-label": "baz",
					}
				},
				Asserts: func(t *testing.T, cj []*batchv1.CronJob) {
					require.Contains(t, cj[0].Annotations, "cronjob-annotation")
					assert.Equal(t, "foo", cj[0].Annotations["cronjob-annotation"])

					require.Contains(t, cj[0].Labels, "cronjob-label")
					assert.Equal(t, "foo", cj[0].Labels["cronjob-label"])

					require.Contains(t, cj[0].Spec.JobTemplate.Annotations, "job-annotation")
					assert.Equal(t, "bar", cj[0].Spec.JobTemplate.Annotations["job-annotation"])

					require.Contains(t, cj[0].Spec.JobTemplate.Labels, "job-label")
					assert.Equal(t, "bar", cj[0].Spec.JobTemplate.Labels["job-label"])

					require.Contains(t, cj[0].Spec.JobTemplate.Spec.Template.Annotations, "pod-annotation")
					assert.Equal(t, "baz", cj[0].Spec.JobTemplate.Spec.Template.Annotations["pod-annotation"])

					require.Contains(t, cj[0].Spec.JobTemplate.Spec.Template.Labels, "pod-label")
					assert.Equal(t, "baz", cj[0].Spec.JobTemplate.Spec.Template.Labels["pod-label"])
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
		Cronjobs: []Cronjob{
			{
				Metadata: Metadata{
					Namespace:   "ns",
					Service:     "service",
					Component:   "component",
					Environment: "test",
				},
				Name:     "cronjob",
				Schedule: "* * * * *",
				Container: Container{
					Name: "main",
					Image: Image{
						Repository: "cronjob_image_repository",
						Tag:        ptr.To("cronjob_image_tag"),
					},
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

			_, create := CreateCronjobs(values)
			resources, err := create(values)
			if err != nil {
				t.Errorf("error during test setup: %v", err)
			}
			cronjobs := []*batchv1.CronJob{}
			for i := range resources {
				if c, ok := resources[i].(*batchv1.CronJob); ok {
					cronjobs = append(cronjobs, c)
				} else {
					t.Error("error while retyping cronjobs in test setup")
				}
			}

			config.Asserts(t, cronjobs)
		})
	}

	t.Run("should not create any cronjobs if not specified", func(t *testing.T) {
		values := DeploymentValues{}
		copier.CopyWithOption(&values, &base, copier.Option{DeepCopy: true})
		values.Cronjobs = []Cronjob{}

		shouldCreate, _ := CreateCronjobs(values)
		assert.False(t, shouldCreate)
	})
}
