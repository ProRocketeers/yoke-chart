package resources

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func CreateHttpRoute(values DeploymentValues) (bool, ResourceCreator) {
	enabled := values.HTTPRoute != nil && *values.HTTPRoute.Enabled
	return enabled, func(values DeploymentValues) ([]unstructured.Unstructured, error) {
		httpRoute := gatewayv1.HTTPRoute{
			TypeMeta: metav1.TypeMeta{
				APIVersion: gatewayv1.SchemeGroupVersion.Identifier(),
				Kind:       "HTTPRoute",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        serviceName(values.Metadata),
				Namespace:   values.Metadata.Namespace,
				Annotations: values.HTTPRoute.Annotations,
				Labels:      withCommonLabels(values.HTTPRoute.Labels, values.Metadata),
			},
			Spec: values.HTTPRoute.HTTPRouteSpec,
		}
		u, err := toUnstructured(&httpRoute)
		if err != nil {
			return []unstructured.Unstructured{}, err
		}
		return u, nil
	}
}
