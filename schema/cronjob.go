package schema

import (
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
)

type Cronjob struct {
	Container `json:",inline"`

	Name              string            `json:"name" validate:"required"`
	Schedule          string            `json:"schedule" validate:"required"`
	MainContainerName *string           `json:"mainContainerName,omitempty"`
	InitContainers    []InitContainer   `json:"initContainers,omitempty" validate:"dive"`
	Volumes           map[string]Volume `json:"volumes,omitempty" validate:"dive"`
	PodMonitor        *PodMonitor       `json:"podMonitor"`

	CronJobAnnotations map[string]string `json:"cronJobAnnotations,omitempty"`
	CronJobLabels      map[string]string `json:"cronJobLabels,omitempty"`
	JobAnnotations     map[string]string `json:"jobAnnotations,omitempty"`
	JobLabels          map[string]string `json:"jobLabels,omitempty"`
	PodAnnotations     map[string]string `json:"podAnnotations,omitempty"`
	PodLabels          map[string]string `json:"podLabels,omitempty"`
	PodSpec            *v1.PodSpec       `json:"podSpec,omitempty"`

	SchedulingConfig `json:",inline"`

	// OPTIONAL - full `CronJobSpec`/`JobSpec` as specified by Kubernetes, split in two since they
	// both have a `suspend` field. The Flight builds `schedule`/`jobTemplate`/`template` itself, then
	// layers these on top - so anything you set here (including those) wins if you explicitly set it,
	// otherwise the built value is kept
	CronJobSpec *batchv1.CronJobSpec `json:"cronJobSpec,omitempty"`
	JobSpec     *batchv1.JobSpec     `json:"jobSpec,omitempty"`
}
