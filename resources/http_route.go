package resources

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
				Spec: route.HTTPRouteSpec,
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
