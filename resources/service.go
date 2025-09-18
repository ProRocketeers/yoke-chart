package resources

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func CreateService(values DeploymentValues) (bool, ResourceCreator) {
	return true, func(values DeploymentValues) ([]unstructured.Unstructured, error) {
		service := corev1.Service{
			TypeMeta: metav1.TypeMeta{
				APIVersion: corev1.SchemeGroupVersion.Identifier(),
				Kind:       "Service",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceName(values.Metadata),
				Namespace: values.Metadata.Namespace,
				Labels: func() map[string]string {
					labels := commonLabels(values.Metadata)
					if values.ServiceMonitor != nil && *values.ServiceMonitor.Enabled {
						labels["prometheus-scrape"] = "true"
					}
					return labels
				}(),
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{
					"app": serviceName(values.Metadata),
				},
				Type:  corev1.ServiceTypeClusterIP,
				Ports: getServicePorts(values),
			},
		}
		u, err := toUnstructured(&service)
		if err != nil {
			return []unstructured.Unstructured{}, err
		}
		return u, nil
	}
}

func getServicePorts(values DeploymentValues) []corev1.ServicePort {
	ports := []corev1.ServicePort{}

	for i, container := range values.Containers {
		for j, port := range container.Ports {
			p := corev1.ServicePort{
				Protocol:   corev1.ProtocolTCP,
				Port:       int32(port.Port),
				TargetPort: intstr.FromInt(port.Port),
			}
			if port.ContainerPort != nil {
				p.TargetPort = intstr.FromInt(*port.ContainerPort)
			}
			if i == 0 && j == 0 {
				// main container, first port => "main" port
				p.Name = "main-port"
			} else {
				p.Name = fmt.Sprintf("other-port-%s-%d", container.Name, j)
			}
			if port.Name != nil {
				p.Name = *port.Name
			}
			if port.Expose == nil || *port.Expose {
				ports = append(ports, p)
			}
		}
	}
	return ports
}
