package resources

import (
	"fmt"

	"github.com/yokecd/yoke/pkg/flight"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateCronjobs(values DeploymentValues) (bool, ResourceCreator) {
	return len(values.Cronjobs) > 0, func(values DeploymentValues) ([]flight.Resource, error) {
		cronjobs := []flight.Resource{}

		for _, c := range values.Cronjobs {
			podSpec, err := createPodSpec(&c, values)
			if err != nil {
				return []flight.Resource{}, fmt.Errorf("error creating pod for cronjob '%v': %v", c.Name, err)
			}

			podSpec.RestartPolicy = corev1.RestartPolicyOnFailure

			cronjob := batchv1.CronJob{
				TypeMeta: metav1.TypeMeta{
					APIVersion: batchv1.SchemeGroupVersion.Identifier(),
					Kind:       "CronJob",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:        fmt.Sprintf("%s--%s", c.Name, c.Metadata.Environment),
					Namespace:   c.Metadata.Namespace,
					Labels:      withCommonLabels(c.CronJobLabels, c.Metadata),
					Annotations: c.CronJobAnnotations,
				},
				Spec: batchv1.CronJobSpec{
					JobTemplate: batchv1.JobTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: c.JobAnnotations,
							Labels:      withCommonLabels(c.JobLabels, c.Metadata),
						},
						Spec: batchv1.JobSpec{
							Template: corev1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Annotations: c.PodAnnotations,
									Labels:      c.PodLabels,
								},
								Spec: podSpec,
							},

							ActiveDeadlineSeconds:   c.ActiveDeadlineSeconds,
							BackoffLimit:            c.BackoffLimit,
							CompletionMode:          c.CompletionMode,
							Completions:             c.Completions,
							Parallelism:             c.Parallelism,
							PodFailurePolicy:        c.PodFailurePolicy,
							Selector:                c.Selector,
							TTLSecondsAfterFinished: c.TTLSecondsAfterFinished,
						},
					},
					Schedule:                   c.Schedule,
					Suspend:                    c.Suspend,
					TimeZone:                   c.TimeZone,
					ConcurrencyPolicy:          batchv1.AllowConcurrent,
					StartingDeadlineSeconds:    c.StartingDeadlineSeconds,
					SuccessfulJobsHistoryLimit: c.SuccessfulJobsHistoryLimit,
					FailedJobsHistoryLimit:     c.FailedJobsHistoryLimit,
				},
			}
			if c.ConcurrencyPolicy != nil {
				cronjob.Spec.ConcurrencyPolicy = *c.ConcurrencyPolicy
			}
			cronjobs = append(cronjobs, &cronjob)
		}
		return cronjobs, nil
	}
}
