package schema

import "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

type ServiceMonitor struct {
	Enabled   *bool         `json:"enabled" validate:"required"`
	Endpoints []v1.Endpoint `json:"endpoints" validate:"required,min=1"`
}

type PodMonitor struct {
	Enabled   *bool                   `json:"enabled" validate:"required"`
	Endpoints []v1.PodMetricsEndpoint `json:"endpoints" validate:"required,min=1"`
}
