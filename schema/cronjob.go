package schema

import (
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Cronjob struct {
	Container `json:",inline"`

	Name              string            `json:"name" validate:"required"`
	Schedule          string            `json:"schedule" validate:"required"`
	MainContainerName *string           `json:"mainContainerName,omitempty"`
	InitContainers    []InitContainer   `json:"initContainers,omitempty" validate:"dive"`
	Volumes           map[string]Volume `json:"volumes,omitempty" validate:"dive"`
	PodMonitor        *PodMonitor       `json:"podMonitor"`

	CronJobAnnotations map[string]string      `json:"cronJobAnnotations,omitempty"`
	CronJobLabels      map[string]string      `json:"cronJobLabels,omitempty"`
	JobAnnotations     map[string]string      `json:"jobAnnotations,omitempty"`
	JobLabels          map[string]string      `json:"jobLabels,omitempty"`
	PodAnnotations     map[string]string      `json:"podAnnotations,omitempty"`
	PodLabels          map[string]string      `json:"podLabels,omitempty"`
	PodSecurityContext *v1.PodSecurityContext `json:"podSecurityContext,omitempty"`

	SchedulingConfig        `json:",inline"`
	CronJobAdditionalFields `json:",inline"`
}

type CronJobAdditionalFields struct {
	// some CronJobSpec fields
	Suspend                    *bool                      `json:"suspend,omitempty"`
	TimeZone                   *string                    `json:"timeZone,omitempty"`
	ConcurrencyPolicy          *batchv1.ConcurrencyPolicy `json:"concurrencyPolicy,omitempty"`
	StartingDeadlineSeconds    *int64                     `json:"startingDeadlineSeconds,omitempty"`
	SuccessfulJobsHistoryLimit *int32                     `json:"successfulJobsHistoryLimit,omitempty"`
	FailedJobsHistoryLimit     *int32                     `json:"failedJobsHistoryLimit,omitempty"`
	// and some JobSpec fields
	ActiveDeadlineSeconds   *int64                    `json:"activeDeadlineSeconds,omitempty"`
	BackoffLimit            *int32                    `json:"backoffLimit,omitempty"`
	CompletionMode          *batchv1.CompletionMode   `json:"completionMode,omitempty"`
	Completions             *int32                    `json:"completions,omitempty"`
	Parallelism             *int32                    `json:"parallelism,omitempty"`
	PodFailurePolicy        *batchv1.PodFailurePolicy `json:"podFailurePolicy,omitempty"`
	Selector                *metav1.LabelSelector     `json:"selector,omitempty"`
	TTLSecondsAfterFinished *int32                    `json:"ttlSecondsAfterFinished,omitempty"`
}
