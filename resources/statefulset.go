package resources

import (
	"fmt"
	"maps"
	"strings"

	"dario.cat/mergo"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func CreateStatefulSet(values DeploymentValues) (bool, ResourceCreator) {
	return true, func(values DeploymentValues) ([]NamedResource, error) {
		podAnnotations := map[string]string{}
		maps.Copy(podAnnotations, values.PodAnnotations)

		for _, container := range values.Containers {
			podAnnotations["container-"+container.Name+"-image-tag"] = *container.Image.Tag
		}

		podSpec, err := createPodSpec(&values, values)
		if err != nil {
			return nil, fmt.Errorf("error creating deployment pod spec: %v", err)
		}

		headlessSvc := corev1.Service{
			TypeMeta: metav1.TypeMeta{
				APIVersion: corev1.SchemeGroupVersion.Identifier(),
				Kind:       "Service",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      headlessServiceName(values.Metadata),
				Namespace: values.Metadata.Namespace,
				Labels:    commonLabels(values.Metadata),
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{
					"app": serviceName(values.Metadata),
				},
				Type:      corev1.ServiceTypeClusterIP,
				ClusterIP: corev1.ClusterIPNone,
				Ports:     getServicePorts(values),
			},
		}

		statefulSet := appsv1.StatefulSet{
			TypeMeta: metav1.TypeMeta{
				APIVersion: appsv1.SchemeGroupVersion.Identifier(),
				Kind:       "StatefulSet",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        serviceName(values.Metadata),
				Namespace:   values.Metadata.Namespace,
				Annotations: values.Annotations,
				Labels:      withCommonLabels(values.Labels, values.Metadata),
			},
			Spec: appsv1.StatefulSetSpec{
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
				ServiceName: headlessServiceName(values.Metadata),
			},
		}

		if values.Autoscaling == nil {
			statefulSet.Spec.Replicas = ptr.To(int32(values.ReplicaCount))
		}

		if err := mergo.Merge(&statefulSet.Spec, *values.StatefulSetSpec, mergo.WithOverride); err != nil {
			return nil, fmt.Errorf("merging raw statefulSet spec: %v", err)
		}

		u, err := toUnstructured(&statefulSet, &headlessSvc)
		if err != nil {
			return nil, err
		}
		return []NamedResource{
			{Category: CategoryWorkload, Object: u[0]},
			{Category: CategoryHeadlessService, Object: u[1]},
		}, nil
	}
}

func headlessServiceName(metadata Metadata) string {
	s := fmt.Sprintf("%s-headless", serviceName(metadata))
	return strings.TrimSpace(s)
}
