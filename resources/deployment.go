package resources

import (
	"fmt"
	"maps"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/ptr"
)

func CreateDeployment(values DeploymentValues) (bool, ResourceCreator) {
	return true, func(values DeploymentValues) ([]unstructured.Unstructured, error) {
		podAnnotations := map[string]string{}
		maps.Copy(podAnnotations, values.PodAnnotations)

		for _, container := range values.Containers {
			podAnnotations["container-"+container.Name+"-image-tag"] = *container.Image.Tag
		}

		podSpec, err := createPodSpec(&values, values)
		if err != nil {
			return nil, fmt.Errorf("error creating deployment pod spec: %v", err)
		}

		deployment := appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				APIVersion: appsv1.SchemeGroupVersion.Identifier(),
				Kind:       "Deployment",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        serviceName(values.Metadata),
				Namespace:   values.Metadata.Namespace,
				Annotations: values.Annotations,
				Labels:      withCommonLabels(values.Labels, values.Metadata),
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": serviceName(values.Metadata),
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: podAnnotations,
						Labels:      withCommonLabels(values.PodLabels, values.Metadata),
					},
					Spec: podSpec,
				},
			},
		}
		if values.Autoscaling == nil {
			deployment.Spec.Replicas = ptr.To(int32(values.ReplicaCount))
		}
		if values.Strategy != nil {
			deployment.Spec.Strategy = *values.Strategy
		}
		u, err := toUnstructured(&deployment)
		if err != nil {
			return []unstructured.Unstructured{}, err
		}
		return u, nil
	}
}
