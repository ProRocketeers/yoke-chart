package resources

import (
	"fmt"
	"maps"

	"dario.cat/mergo"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreatePreDeploymentJob(values DeploymentValues) (bool, ResourceCreator) {
	return values.PreDeploymentJob != nil, func(values DeploymentValues) ([]NamedResource, error) {
		j := values.PreDeploymentJob
		annotations := map[string]string{
			"helm.sh/hook":        "pre-install, pre-upgrade",
			"helm-sh/hook-weight": "-5",
		}
		maps.Copy(annotations, j.Annotations)

		podSpec, err := createPodSpec(j, values)
		if err != nil {
			return nil, fmt.Errorf("error creating pod for pre-deployment job: %v", err)
		}
		podSpec.RestartPolicy = corev1.RestartPolicyNever

		jobSpec := batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: j.PodAnnotations,
					Labels: func() map[string]string {
						m := map[string]string{}
						maps.Copy(m, j.PodLabels)
						if j.PodMonitor != nil && *j.PodMonitor.Enabled {
							m["app"] = preDeploymentJobName(j.Metadata)
							m["prometheus-scrape"] = "true"
						}
						return m
					}(),
				},
				Spec: podSpec,
			},
		}
		if j.JobSpec != nil {
			if err := mergo.Merge(&jobSpec, *j.JobSpec, mergo.WithOverride); err != nil {
				return nil, fmt.Errorf("merging raw jobSpec for pre-deployment job: %v", err)
			}
		}

		job := batchv1.Job{
			TypeMeta: metav1.TypeMeta{
				APIVersion: batchv1.SchemeGroupVersion.Identifier(),
				Kind:       "Job",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        preDeploymentJobName(j.Metadata),
				Namespace:   j.Metadata.Namespace,
				Labels:      withCommonLabels(j.Labels, j.Metadata),
				Annotations: annotations,
			},
			Spec: jobSpec,
		}
		u, err := toUnstructured(&job)
		if err != nil {
			return nil, err
		}
		return []NamedResource{{Category: CategoryPreDeploymentJob, Object: u[0]}}, nil
	}
}
