package resources

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/ptr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func CreateHttpRoutes(values DeploymentValues) (bool, ResourceCreator) {
	return len(values.HTTPRoutes) > 0, func(values DeploymentValues) ([]unstructured.Unstructured, error) {
		var resources []unstructured.Unstructured
		for name, route := range values.HTTPRoutes {
			httpRoute := gatewayv1.HTTPRoute{
				TypeMeta: metav1.TypeMeta{
					APIVersion: gatewayv1.SchemeGroupVersion.Identifier(),
					Kind:       "HTTPRoute",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:        fmt.Sprintf("%s-%s", serviceName(values.Metadata), name),
					Namespace:   values.Metadata.Namespace,
					Annotations: route.Annotations,
					Labels:      withCommonLabels(route.Labels, values.Metadata),
				},
				Spec: withHTTPRouteDefaults(route.HTTPRouteSpec),
			}
			u, err := toUnstructured(&httpRoute)
			if err != nil {
				return []unstructured.Unstructured{}, err
			}
			resources = append(resources, u...)
		}
		return resources, nil
	}
}

// withHTTPRouteDefaults fills in the same field defaults that Kubernetes applies via CRD defaulting
// webhooks, so that ArgoCD doesn't flag spurious diffs between the desired and live state.
func withHTTPRouteDefaults(spec gatewayv1.HTTPRouteSpec) gatewayv1.HTTPRouteSpec {
	for i := range spec.ParentRefs {
		if spec.ParentRefs[i].Group == nil {
			spec.ParentRefs[i].Group = ptr.To(gatewayv1.Group("gateway.networking.k8s.io"))
		}
		if spec.ParentRefs[i].Kind == nil {
			spec.ParentRefs[i].Kind = ptr.To(gatewayv1.Kind("Gateway"))
		}
	}

	for i := range spec.Rules {
		for j := range spec.Rules[i].BackendRefs {
			if spec.Rules[i].BackendRefs[j].Group == nil {
				spec.Rules[i].BackendRefs[j].Group = ptr.To(gatewayv1.Group(""))
			}
			if spec.Rules[i].BackendRefs[j].Kind == nil {
				spec.Rules[i].BackendRefs[j].Kind = ptr.To(gatewayv1.Kind("Service"))
			}
			if spec.Rules[i].BackendRefs[j].Weight == nil {
				spec.Rules[i].BackendRefs[j].Weight = ptr.To(int32(1))
			}
		}

		for j := range spec.Rules[i].Matches {
			if spec.Rules[i].Matches[j].Path != nil {
				if spec.Rules[i].Matches[j].Path.Type == nil {
					spec.Rules[i].Matches[j].Path.Type = ptr.To(gatewayv1.PathMatchPathPrefix)
				}
				if spec.Rules[i].Matches[j].Path.Value == nil {
					spec.Rules[i].Matches[j].Path.Value = ptr.To("/")
				}
			}
		}
	}

	return spec
}
