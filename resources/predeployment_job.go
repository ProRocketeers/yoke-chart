package resources

import (
	"fmt"
	"maps"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func CreatePreDeploymentJob(values DeploymentValues) (bool, ResourceCreator) {
	return values.PreDeploymentJob != nil, func(values DeploymentValues) ([]unstructured.Unstructured, error) {
		j := values.PreDeploymentJob
		annotations := map[string]string{
			"helm.sh/hook":        "pre-install, pre-upgrade",
			"helm-sh/hook-weight": "-5",
		}
		maps.Copy(annotations, j.Annotations)

		podSpec, err := createPodSpec(j, values)
		if err != nil {
			return []unstructured.Unstructured{}, fmt.Errorf("error creating pod for pre-deployment job: %v", err)
		}
		podSpec.RestartPolicy = corev1.RestartPolicyNever

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
			Spec: batchv1.JobSpec{
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

				ActiveDeadlineSeconds:   j.ActiveDeadlineSeconds,
				BackoffLimit:            j.BackoffLimit,
				CompletionMode:          j.CompletionMode,
				Completions:             j.Completions,
				Parallelism:             j.Parallelism,
				PodFailurePolicy:        j.PodFailurePolicy,
				Selector:                j.Selector,
				Suspend:                 j.Suspend,
				TTLSecondsAfterFinished: j.TTLSecondsAfterFinished,
			},
		}
		u, err := toUnstructured(&job)
		if err != nil {
			return []unstructured.Unstructured{}, err
		}
		return u, nil
	}
}
