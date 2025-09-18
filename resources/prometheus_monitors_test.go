package resources

import (
	"testing"

	"github.com/ProRocketeers/yoke-chart/schema"
	"github.com/jinzhu/copier"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/ptr"
)

func TestPrometheusMonitors(t *testing.T) {
	type CaseConfig struct {
		ValuesTransform func(*DeploymentValues)
		Asserts         func(*testing.T, []unstructured.Unstructured)
	}

	cases := map[string]CaseConfig{
		"renders service monitor properly": {
			ValuesTransform: func(dv *DeploymentValues) {
				dv.ServiceMonitor = &schema.ServiceMonitor{
					Enabled: ptr.To(true),
					Endpoints: []monitoringv1.Endpoint{
						{
							Port:            "metrics",
							Path:            "/metrics",
							Scheme:          "http",
							Interval:        monitoringv1.Duration("15s"),
							ScrapeTimeout:   monitoringv1.Duration("10s"),
							HonorLabels:     false,
							HonorTimestamps: ptr.To(true),
						},
					},
				}
			},
			Asserts: func(t *testing.T, r []unstructured.Unstructured) {
				sm := findResourceOrFail[*monitoringv1.ServiceMonitor](t, r, "ServiceMonitor", "service--component--test")

				assert.Equal(t, "ns", sm.Namespace)
				assert.Equal(t, "ns", sm.Spec.NamespaceSelector.MatchNames[0])

				assert.Subset(t, sm.Spec.Selector.MatchLabels, map[string]string{
					"app":               "service--component--test",
					"prometheus-scrape": "true",
				})

				assert.Equal(t, "metrics", sm.Spec.Endpoints[0].Port)
				assert.Equal(t, "/metrics", sm.Spec.Endpoints[0].Path)
				assert.Equal(t, "http", sm.Spec.Endpoints[0].Scheme)
				assert.Equal(t, monitoringv1.Duration("15s"), sm.Spec.Endpoints[0].Interval)
				assert.Equal(t, monitoringv1.Duration("10s"), sm.Spec.Endpoints[0].ScrapeTimeout)
				assert.Equal(t, false, sm.Spec.Endpoints[0].HonorLabels)
				assert.Equal(t, ptr.To(true), sm.Spec.Endpoints[0].HonorTimestamps)
			},
		},
		"renders pre-deployment job pod monitor properly": {
			ValuesTransform: func(dv *DeploymentValues) {
				dv.PreDeploymentJob = &PreDeploymentJob{
					Metadata: Metadata{
						Namespace:   "ns",
						Service:     "service",
						Component:   "component",
						Environment: "test",
					},
					Container: Container{
						Name: "main",
						Image: Image{
							Repository: "job_image_repository",
							Tag:        ptr.To("job_image_tag"),
						},
					},
					PodMonitor: &schema.PodMonitor{
						Enabled: ptr.To(true),
						Endpoints: []monitoringv1.PodMetricsEndpoint{
							{
								Port:            ptr.To("metrics"),
								Path:            "/metrics",
								Scheme:          "http",
								Interval:        monitoringv1.Duration("15s"),
								ScrapeTimeout:   monitoringv1.Duration("10s"),
								HonorLabels:     false,
								HonorTimestamps: ptr.To(true),
							},
						},
					},
				}
			},
			Asserts: func(t *testing.T, r []unstructured.Unstructured) {
				pm := findResourceOrFail[*monitoringv1.PodMonitor](t, r, "PodMonitor", "service--component--test--pre-deploy")

				assert.Equal(t, "ns", pm.Namespace)
				assert.Equal(t, "ns", pm.Spec.NamespaceSelector.MatchNames[0])

				assert.Subset(t, pm.Spec.Selector.MatchLabels, map[string]string{
					"app":               "service--component--test--pre-deploy",
					"prometheus-scrape": "true",
				})

				assert.Equal(t, ptr.To("metrics"), pm.Spec.PodMetricsEndpoints[0].Port)
				assert.Equal(t, "/metrics", pm.Spec.PodMetricsEndpoints[0].Path)
				assert.Equal(t, "http", pm.Spec.PodMetricsEndpoints[0].Scheme)
				assert.Equal(t, monitoringv1.Duration("15s"), pm.Spec.PodMetricsEndpoints[0].Interval)
				assert.Equal(t, monitoringv1.Duration("10s"), pm.Spec.PodMetricsEndpoints[0].ScrapeTimeout)
				assert.Equal(t, false, pm.Spec.PodMetricsEndpoints[0].HonorLabels)
				assert.Equal(t, ptr.To(true), pm.Spec.PodMetricsEndpoints[0].HonorTimestamps)
			},
		},
		"renders cronjob pod monitor properly": {
			ValuesTransform: func(dv *DeploymentValues) {
				dv.Cronjobs = []Cronjob{
					{
						Metadata: Metadata{
							Namespace:   "ns",
							Service:     "service",
							Component:   "component",
							Environment: "test",
						},
						Name:     "cronjob",
						Schedule: "* * * * *",
						Container: Container{
							Name: "main",
							Image: Image{
								Repository: "cronjob_image_repository",
								Tag:        ptr.To("cronjob_image_tag"),
							},
						},
						PodMonitor: &schema.PodMonitor{
							Enabled: ptr.To(true),
							Endpoints: []monitoringv1.PodMetricsEndpoint{
								{
									Port:            ptr.To("metrics"),
									Path:            "/metrics",
									Scheme:          "http",
									Interval:        monitoringv1.Duration("15s"),
									ScrapeTimeout:   monitoringv1.Duration("10s"),
									HonorLabels:     false,
									HonorTimestamps: ptr.To(true),
								},
							},
						},
					},
				}
			},
			Asserts: func(t *testing.T, r []unstructured.Unstructured) {
				pm := findResourceOrFail[*monitoringv1.PodMonitor](t, r, "PodMonitor", "cronjob--test")

				assert.Equal(t, "ns", pm.Namespace)
				assert.Equal(t, "ns", pm.Spec.NamespaceSelector.MatchNames[0])

				assert.Subset(t, pm.Spec.Selector.MatchLabels, map[string]string{
					"app":               "cronjob--test",
					"prometheus-scrape": "true",
				})

				assert.Equal(t, ptr.To("metrics"), pm.Spec.PodMetricsEndpoints[0].Port)
				assert.Equal(t, "/metrics", pm.Spec.PodMetricsEndpoints[0].Path)
				assert.Equal(t, "http", pm.Spec.PodMetricsEndpoints[0].Scheme)
				assert.Equal(t, monitoringv1.Duration("15s"), pm.Spec.PodMetricsEndpoints[0].Interval)
				assert.Equal(t, monitoringv1.Duration("10s"), pm.Spec.PodMetricsEndpoints[0].ScrapeTimeout)
				assert.Equal(t, false, pm.Spec.PodMetricsEndpoints[0].HonorLabels)
				assert.Equal(t, ptr.To(true), pm.Spec.PodMetricsEndpoints[0].HonorTimestamps)
			},
		},
		"does not render service monitor if not specified - nil config": {
			ValuesTransform: func(dv *DeploymentValues) {},
			Asserts: func(t *testing.T, r []unstructured.Unstructured) {
				_, found := findResource[*monitoringv1.ServiceMonitor](r, "ServiceMonitor", "service--component--test")
				assert.False(t, found, "Service monitor found when it shouldn't exist")
			},
		},
		"does not render service monitor if not specified - disabled": {
			ValuesTransform: func(dv *DeploymentValues) {
				dv.ServiceMonitor = &schema.ServiceMonitor{
					Enabled:   ptr.To(false),
					Endpoints: []monitoringv1.Endpoint{},
				}
			},
			Asserts: func(t *testing.T, r []unstructured.Unstructured) {
				_, found := findResource[*monitoringv1.ServiceMonitor](r, "ServiceMonitor", "service--component--test")
				assert.False(t, found, "Service monitor found when it shouldn't exist")
			},
		},
		"does not render pod monitor for PDJ if not specified - nil config": {
			ValuesTransform: func(dv *DeploymentValues) {
				dv.PreDeploymentJob = &PreDeploymentJob{
					Metadata: Metadata{
						Namespace:   "ns",
						Service:     "service",
						Component:   "component",
						Environment: "test",
					},
					Container: Container{
						Name: "main",
						Image: Image{
							Repository: "job_image_repository",
							Tag:        ptr.To("job_image_tag"),
						},
					},
				}
			},
			Asserts: func(t *testing.T, r []unstructured.Unstructured) {
				_, found := findResource[*monitoringv1.PodMonitor](r, "PodMonitor", "service--component--test--pre-deploy")
				assert.False(t, found, "Pod monitor found when it shouldn't exist")
			},
		},
		"does not render pod monitor for PDJ if not specified - disabled": {
			ValuesTransform: func(dv *DeploymentValues) {
				dv.PreDeploymentJob = &PreDeploymentJob{
					Metadata: Metadata{
						Namespace:   "ns",
						Service:     "service",
						Component:   "component",
						Environment: "test",
					},
					Container: Container{
						Name: "main",
						Image: Image{
							Repository: "job_image_repository",
							Tag:        ptr.To("job_image_tag"),
						},
					},
					PodMonitor: &schema.PodMonitor{
						Enabled:   ptr.To(false),
						Endpoints: []monitoringv1.PodMetricsEndpoint{},
					},
				}
			},
			Asserts: func(t *testing.T, r []unstructured.Unstructured) {
				_, found := findResource[*monitoringv1.PodMonitor](r, "PodMonitor", "service--component--test--pre-deploy")
				assert.False(t, found, "Pod monitor found when it shouldn't exist")
			},
		},
		"does not render pod monitor for cronjob if not specified - nil config": {
			ValuesTransform: func(dv *DeploymentValues) {
				dv.Cronjobs = []Cronjob{
					{
						Metadata: Metadata{
							Namespace:   "ns",
							Service:     "service",
							Component:   "component",
							Environment: "test",
						},
						Name:     "cronjob",
						Schedule: "* * * * *",
						Container: Container{
							Name: "main",
							Image: Image{
								Repository: "cronjob_image_repository",
								Tag:        ptr.To("cronjob_image_tag"),
							},
						},
					},
				}
			},
			Asserts: func(t *testing.T, r []unstructured.Unstructured) {
				_, found := findResource[*monitoringv1.PodMonitor](r, "PodMonitor", "cronjob--test")
				assert.False(t, found, "Pod monitor found when it shouldn't exist")
			},
		},
		"does not render pod monitor for cronjob if not specified - disabled": {
			ValuesTransform: func(dv *DeploymentValues) {
				dv.Cronjobs = []Cronjob{
					{
						Metadata: Metadata{
							Namespace:   "ns",
							Service:     "service",
							Component:   "component",
							Environment: "test",
						},
						Name:     "cronjob",
						Schedule: "* * * * *",
						Container: Container{
							Name: "main",
							Image: Image{
								Repository: "cronjob_image_repository",
								Tag:        ptr.To("cronjob_image_tag"),
							},
						},
						PodMonitor: &schema.PodMonitor{
							Enabled:   ptr.To(false),
							Endpoints: []monitoringv1.PodMetricsEndpoint{},
						},
					},
				}
			},
			Asserts: func(t *testing.T, r []unstructured.Unstructured) {
				_, found := findResource[*monitoringv1.PodMonitor](r, "PodMonitor", "cronjob--test")
				assert.False(t, found, "Pod monitor found when it shouldn't exist")
			},
		},
	}

	base := DeploymentValues{
		Metadata: Metadata{
			Namespace:   "ns",
			Service:     "service",
			Component:   "component",
			Environment: "test",
		},
		Containers: []Container{
			{
				Name: "main",
				Image: Image{
					Repository: "image_repository",
					Tag:        ptr.To("image_tag"),
				},
			},
		},
	}

	for testName, config := range cases {
		t.Run(testName, func(t *testing.T) {
			values := DeploymentValues{}
			copier.CopyWithOption(&values, &base, copier.Option{DeepCopy: true})

			config.ValuesTransform(&values)

			_, create := CreatePrometheusMonitors(values)
			resources, err := create(values)
			if err != nil {
				t.Errorf("error during test setup: %v", err)
			}

			config.Asserts(t, resources)
		})
	}
}
