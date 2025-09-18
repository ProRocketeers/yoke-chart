package resources

import (
	"fmt"

	"github.com/yokecd/yoke/pkg/flight"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateConfigMaps(values DeploymentValues) (bool, ResourceCreator) {
	return len(values.ConfigMaps) > 0, func(values DeploymentValues) ([]flight.Resource, error) {
		resources := []flight.Resource{}
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
			resources = append(resources, &cm)
		}
		return resources, nil
	}
}
