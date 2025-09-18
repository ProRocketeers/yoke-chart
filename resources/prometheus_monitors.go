package resources

import (
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func CreatePrometheusMonitors(values DeploymentValues) (bool, ResourceCreator) {
	create := func() bool {
		if values.ServiceMonitor != nil && *values.ServiceMonitor.Enabled {
			return true
		}
		if values.PreDeploymentJob != nil && values.PreDeploymentJob.PodMonitor != nil && *values.PreDeploymentJob.PodMonitor.Enabled {
			return true
		}
		for _, cronjob := range values.Cronjobs {
			if cronjob.PodMonitor != nil && *cronjob.PodMonitor.Enabled {
				return true
			}
		}
		return false
	}()
	return create, func(values DeploymentValues) ([]unstructured.Unstructured, error) {
		resources := []unstructured.Unstructured{}
		if values.ServiceMonitor != nil && *values.ServiceMonitor.Enabled {
			sm := monitoringv1.ServiceMonitor{
				TypeMeta: metav1.TypeMeta{
					APIVersion: monitoringv1.SchemeGroupVersion.Identifier(),
					Kind:       "ServiceMonitor",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      serviceName(values.Metadata),
					Namespace: values.Metadata.Namespace,
					Labels:    commonLabels(values.Metadata),
				},
				Spec: monitoringv1.ServiceMonitorSpec{
					NamespaceSelector: monitoringv1.NamespaceSelector{
						MatchNames: []string{values.Metadata.Namespace},
					},
					Selector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app":               serviceName(values.Metadata),
							"prometheus-scrape": "true",
						},
					},
					Endpoints: values.ServiceMonitor.Endpoints,
				},
			}
			u, err := toUnstructured(&sm)
			if err != nil {
				return []unstructured.Unstructured{}, err
			}
			resources = append(resources, u...)
		}
		if values.PreDeploymentJob != nil && values.PreDeploymentJob.PodMonitor != nil && *values.PreDeploymentJob.PodMonitor.Enabled {
			pm := monitoringv1.PodMonitor{
				TypeMeta: metav1.TypeMeta{
					APIVersion: monitoringv1.SchemeGroupVersion.Identifier(),
					Kind:       "PodMonitor",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      preDeploymentJobName(values.Metadata),
					Namespace: values.Metadata.Namespace,
					Labels:    commonLabels(values.Metadata),
				},
				Spec: monitoringv1.PodMonitorSpec{
					NamespaceSelector: monitoringv1.NamespaceSelector{
						MatchNames: []string{values.Metadata.Namespace},
					},
					Selector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app":               preDeploymentJobName(values.Metadata),
							"prometheus-scrape": "true",
						},
					},
					PodMetricsEndpoints: values.PreDeploymentJob.PodMonitor.Endpoints,
				},
			}
			u, err := toUnstructured(&pm)
			if err != nil {
				return []unstructured.Unstructured{}, err
			}
			resources = append(resources, u...)
		}
		for _, cronjob := range values.Cronjobs {
			if cronjob.PodMonitor != nil && *cronjob.PodMonitor.Enabled {
				pm := monitoringv1.PodMonitor{
					TypeMeta: metav1.TypeMeta{
						APIVersion: monitoringv1.SchemeGroupVersion.Identifier(),
						Kind:       "PodMonitor",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      cronjobName(cronjob),
						Namespace: values.Metadata.Namespace,
						Labels:    commonLabels(values.Metadata),
					},
					Spec: monitoringv1.PodMonitorSpec{
						NamespaceSelector: monitoringv1.NamespaceSelector{
							MatchNames: []string{values.Metadata.Namespace},
						},
						Selector: metav1.LabelSelector{
							MatchLabels: map[string]string{
								"app":               cronjobName(cronjob),
								"prometheus-scrape": "true",
							},
						},
						PodMetricsEndpoints: cronjob.PodMonitor.Endpoints,
					},
				}
				u, err := toUnstructured(&pm)
				if err != nil {
					return []unstructured.Unstructured{}, err
				}
				resources = append(resources, u...)
			}
		}
		return resources, nil
	}
}
