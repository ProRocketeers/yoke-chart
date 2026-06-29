package schema

import "k8s.io/api/rbac/v1"

type ServiceAccount struct {
	Annotations           map[string]string   `json:"annotations,omitempty"`
	AdditionalRole        *ServiceAccountRole `json:"additionalRole,omitempty"`
	AdditionalClusterRole *ServiceAccountRole `json:"additionalClusterRole,omitempty"`
}

type ServiceAccountRole struct {
	Name  *string         `json:"name,omitempty"`
	Rules []v1.PolicyRule `json:"rules" validate:"required"`
}
