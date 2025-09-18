package resources

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func CreateConfigMaps(values DeploymentValues) (bool, ResourceCreator) {
	return len(values.ConfigMaps) > 0, func(values DeploymentValues) ([]unstructured.Unstructured, error) {
		resources := []unstructured.Unstructured{}
		for name, contents := range values.ConfigMaps {
			cm := corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					APIVersion: corev1.SchemeGroupVersion.Identifier(),
					Kind:       "ConfigMap",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-%s", serviceName(values.Metadata), name),
					Namespace: values.Metadata.Namespace,
					Labels:    commonLabels(values.Metadata),
				},
				Data: contents,
			}
			u, err := toUnstructured(&cm)
			if err != nil {
				return []unstructured.Unstructured{}, err
			}
			resources = append(resources, u...)
		}
		return resources, nil
	}
}
