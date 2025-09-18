package resources

import (
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func CreateIngress(values DeploymentValues) (bool, ResourceCreator) {
	enabled := values.Ingress != nil && *values.Ingress.Enabled
	return enabled, func(values DeploymentValues) ([]unstructured.Unstructured, error) {
		ingress := networkingv1.Ingress{
			TypeMeta: metav1.TypeMeta{
				APIVersion: networkingv1.SchemeGroupVersion.Identifier(),
				Kind:       "Ingress",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        serviceName(values.Metadata),
				Namespace:   values.Metadata.Namespace,
				Annotations: values.Ingress.Annotations,
				Labels:      withCommonLabels(values.Ingress.Labels, values.Metadata),
			},
			Spec: values.Ingress.IngressSpec,
		}
		u, err := toUnstructured(&ingress)
		if err != nil {
			return []unstructured.Unstructured{}, err
		}
		return u, nil
	}
}
