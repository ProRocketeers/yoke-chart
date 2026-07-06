package resources

import (
	"fmt"
	"maps"

	"dario.cat/mergo"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateCronjobs(values DeploymentValues) (bool, ResourceCreator) {
	return len(values.Cronjobs) > 0, func(values DeploymentValues) ([]NamedResource, error) {
		resources := []NamedResource{}

		for _, c := range values.Cronjobs {
			podSpec, err := createPodSpec(&c, values)
			if err != nil {
				return nil, fmt.Errorf("error creating pod for cronjob '%v': %v", c.Name, err)
			}

			podSpec.RestartPolicy = corev1.RestartPolicyOnFailure

			jobSpec := batchv1.JobSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: c.PodAnnotations,
						Labels: func() map[string]string {
							m := map[string]string{}
							maps.Copy(m, c.PodLabels)
							if c.PodMonitor != nil && *c.PodMonitor.Enabled {
								m["app"] = cronjobName(c)
								m["prometheus-scrape"] = "true"
							}
							return m
						}(),
					},
					Spec: podSpec,
				},
			}
			if c.JobSpec != nil {
				if err := mergo.Merge(&jobSpec, *c.JobSpec, mergo.WithOverride); err != nil {
					return nil, fmt.Errorf("merging raw jobSpec for cronjob '%v': %v", c.Name, err)
				}
			}

			cronJobSpec := batchv1.CronJobSpec{
				Schedule: c.Schedule,
				JobTemplate: batchv1.JobTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: c.JobAnnotations,
						Labels:      withCommonLabels(c.JobLabels, c.Metadata),
					},
					Spec: jobSpec,
				},
				ConcurrencyPolicy: batchv1.AllowConcurrent,
			}
			if c.CronJobSpec != nil {
				if err := mergo.Merge(&cronJobSpec, *c.CronJobSpec, mergo.WithOverride); err != nil {
					return nil, fmt.Errorf("merging raw cronJobSpec for cronjob '%v': %v", c.Name, err)
				}
			}

			cronjob := batchv1.CronJob{
				TypeMeta: metav1.TypeMeta{
					APIVersion: batchv1.SchemeGroupVersion.Identifier(),
					Kind:       "CronJob",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:        cronjobName(c),
					Namespace:   c.Metadata.Namespace,
					Labels:      withCommonLabels(c.CronJobLabels, c.Metadata),
					Annotations: c.CronJobAnnotations,
				},
				Spec: cronJobSpec,
			}
			u, err := toUnstructured(&cronjob)
			if err != nil {
				return nil, err
			}
			resources = append(resources, NamedResource{Category: CategoryCronjobs, Key: c.Name, Object: u[0]})
		}
		return resources, nil
	}
}
