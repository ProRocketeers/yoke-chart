package schema

import (
	networkingv1 "k8s.io/api/networking/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type Ingress struct {
	Enabled     *bool             `json:"enabled" validate:"required"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`

	networkingv1.IngressSpec `json:",inline"`
}

type HTTPRoute struct {
	Annotations map[string]string `json:"annotations,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`

	gatewayv1.HTTPRouteSpec `json:",inline"`
}
