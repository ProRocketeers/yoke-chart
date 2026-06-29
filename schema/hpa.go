package schema

import "k8s.io/api/autoscaling/v2"

// basically HorizontalPodAutoscalerSpec without the `scaleTargetRef` field
type HorizontalPodAutoscaler struct {
	MinReplicas *int32                              `json:"minReplicas,omitempty"`
	MaxReplicas int32                               `json:"maxReplicas" validate:"required"`
	Metrics     []v2.MetricSpec                     `json:"metrics,omitempty"`
	Behavior    *v2.HorizontalPodAutoscalerBehavior `json:"behavior,omitempty"`
}
