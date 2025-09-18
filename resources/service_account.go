package resources

import (
	"github.com/yokecd/yoke/pkg/flight"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateServiceAccount(values DeploymentValues) (bool, ResourceCreator) {
	return true, func(values DeploymentValues) ([]flight.Resource, error) {
		sa := corev1.ServiceAccount{
			TypeMeta: metav1.TypeMeta{
				APIVersion: corev1.SchemeGroupVersion.Identifier(),
				Kind:       "ServiceAccount",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceName(values.Metadata),
				Namespace: values.Metadata.Namespace,
				Labels:    commonLabels(values.Metadata),
			},
		}
		if values.ServiceAccount != nil {
			sa.ObjectMeta.Annotations = values.ServiceAccount.Annotations
		}
		return []flight.Resource{&sa}, nil
	}
}
