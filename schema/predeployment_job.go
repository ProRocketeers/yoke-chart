package schema

import (
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PreDeploymentJob struct {
	Container `json:",inline"`

	MainContainerName  *string                `json:"mainContainerName,omitempty"`
	InitContainers     []InitContainer        `json:"initContainers,omitempty" validate:"dive"`
	Volumes            map[string]Volume      `json:"volumes,omitempty" validate:"dive"`
	Annotations        map[string]string      `json:"annotations,omitempty"`
	PodAnnotations     map[string]string      `json:"podAnnotations,omitempty"`
	Labels             map[string]string      `json:"labels,omitempty"`
	PodLabels          map[string]string      `json:"podLabels,omitempty"`
	PodMonitor         *PodMonitor            `json:"podMonitor"`
	PodSecurityContext *v1.PodSecurityContext `json:"podSecurityContext,omitempty"`

	SchedulingConfig `json:",inline"`
	JobSpec          `json:",inline"`
}

// k8s JobSpec, without the `template` field
// not the most up-to-date, but in compliance with the helm chart
type JobSpec struct {
	ActiveDeadlineSeconds   *int64                    `json:"activeDeadlineSeconds,omitempty"`
	BackoffLimit            *int32                    `json:"backoffLimit,omitempty"`
	CompletionMode          *batchv1.CompletionMode   `json:"completionMode,omitempty"`
	Completions             *int32                    `json:"completions,omitempty"`
	Parallelism             *int32                    `json:"parallelism,omitempty"`
	PodFailurePolicy        *batchv1.PodFailurePolicy `json:"podFailurePolicy,omitempty"`
	Selector                *metav1.LabelSelector     `json:"selector,omitempty"`
	Suspend                 *bool                     `json:"suspend,omitempty"`
	TTLSecondsAfterFinished *int32                    `json:"ttlSecondsAfterFinished,omitempty"`
}
