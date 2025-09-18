package resources

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func CreateHPA(values DeploymentValues) (bool, ResourceCreator) {
	return values.Autoscaling != nil, func(values DeploymentValues) ([]unstructured.Unstructured, error) {
		a := values.Autoscaling
		if a.MinReplicas != nil && *a.MinReplicas > a.MaxReplicas {
			return []unstructured.Unstructured{}, fmt.Errorf("autoscaling 'maxReplicas' cannot be lower than 'minReplicas' (or 1 by default)")
		}
		hpa := autoscalingv2.HorizontalPodAutoscaler{
			TypeMeta: metav1.TypeMeta{
				APIVersion: autoscalingv2.SchemeGroupVersion.Identifier(),
				Kind:       "HorizontalPodAutoscaler",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceName(values.Metadata),
				Namespace: values.Metadata.Namespace,
				Labels:    commonLabels(values.Metadata),
			},
			Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
				ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
					APIVersion: appsv1.SchemeGroupVersion.Identifier(),
					Kind:       "Deployment",
					Name:       serviceName(values.Metadata),
				},
				MinReplicas: a.MinReplicas,
				MaxReplicas: a.MaxReplicas,
				Metrics:     a.Metrics,
				Behavior:    a.Behavior,
			},
		}
		u, err := toUnstructured(&hpa)
		if err != nil {
			return []unstructured.Unstructured{}, err
		}
		return u, nil
	}
}
