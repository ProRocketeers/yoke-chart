package schema

import (
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
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
	PodSpec            *v1.PodSpec            `json:"podSpec,omitempty"`

	SchedulingConfig `json:",inline"`

	// OPTIONAL - full `JobSpec` as specified by Kubernetes. The Flight builds `template` itself
	// (from this job's container/volumes/podSpec/etc.), then layers this on top - so anything you
	// set here (including `template`) wins if you explicitly set it, otherwise the built value is kept
	JobSpec *batchv1.JobSpec `json:"jobSpec,omitempty"`
}
