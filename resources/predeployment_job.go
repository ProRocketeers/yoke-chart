package resources

import (
	"fmt"
	"maps"

	"github.com/yokecd/yoke/pkg/flight"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreatePreDeploymentJob(values DeploymentValues) (bool, ResourceCreator) {
	return values.PreDeploymentJob != nil, func(values DeploymentValues) ([]flight.Resource, error) {
		j := values.PreDeploymentJob
		annotations := map[string]string{
			"helm.sh/hook":        "pre-install, pre-upgrade",
			"helm-sh/hook-weight": "-5",
		}
		maps.Copy(annotations, j.Annotations)

		podSpec, err := createPodSpec(j, values)
		if err != nil {
			return []flight.Resource{}, fmt.Errorf("error creating pod for pre-deployment job: %v", err)
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
						Labels:      j.PodLabels,
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
		return []flight.Resource{&job}, nil
	}
}
