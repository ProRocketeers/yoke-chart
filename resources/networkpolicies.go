package resources

import (
	"fmt"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func CreateNetworkPolicies(values DeploymentValues) (bool, ResourceCreator) {
	return len(values.NetworkPolicies) > 0, func(values DeploymentValues) ([]unstructured.Unstructured, error) {
		var resources []unstructured.Unstructured
		for name, spec := range values.NetworkPolicies {
			np := networkingv1.NetworkPolicy{
				TypeMeta: metav1.TypeMeta{
					APIVersion: networkingv1.SchemeGroupVersion.Identifier(),
					Kind:       "NetworkPolicy",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-%s", serviceName(values.Metadata), name),
					Namespace: values.Metadata.Namespace,
					Labels:    commonLabels(values.Metadata),
				},
				Spec: spec,
			}
			u, err := toUnstructured(&np)
			if err != nil {
				return []unstructured.Unstructured{}, err
			}
			resources = append(resources, u...)
		}
		return resources, nil
	}
}
