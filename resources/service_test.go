package resources

import (
	"testing"

	"github.com/ProRocketeers/yoke-chart/schema"
	"github.com/jinzhu/copier"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

func TestService(t *testing.T) {
	type CaseConfig struct {
		ValuesTransform func(*DeploymentValues)
		Asserts         func(*testing.T, *corev1.Service)
	}

	// a function to be able to create a closure and share variables to compare against etc.
	cases := map[string]func() CaseConfig{
		"has the same name as deployment": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {},
				Asserts: func(t *testing.T, s *corev1.Service) {
					assert.Equal(t, "service--component--test", s.Name)
				},
			}
		},
		"renders proper port names": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Containers[0].Ports = []schema.Port{
						{Port: 80},
						{Port: 8080},
					}
				},
				Asserts: func(t *testing.T, s *corev1.Service) {
					assert.Contains(t, s.Spec.Ports, corev1.ServicePort{
						Name:       "main-port",
						Protocol:   corev1.ProtocolTCP,
						Port:       int32(80),
						TargetPort: intstr.FromInt(80),
					})
					assert.Contains(t, s.Spec.Ports, corev1.ServicePort{
						Name:       "other-port-main-1",
						Protocol:   corev1.ProtocolTCP,
						Port:       int32(8080),
						TargetPort: intstr.FromInt(8080),
					})
				},
			}
		},
		"allows to override port names": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Containers[0].Ports = []schema.Port{
						{Port: 80, Name: ptr.To("new-main")},
						{Port: 8080, Name: ptr.To("new-side-port")},
					}
				},
				Asserts: func(t *testing.T, s *corev1.Service) {
					assert.Contains(t, s.Spec.Ports, corev1.ServicePort{
						Name:       "new-main",
						Protocol:   corev1.ProtocolTCP,
						Port:       int32(80),
						TargetPort: intstr.FromInt(80),
					})
					assert.Contains(t, s.Spec.Ports, corev1.ServicePort{
						Name:       "new-side-port",
						Protocol:   corev1.ProtocolTCP,
						Port:       int32(8080),
						TargetPort: intstr.FromInt(8080),
					})
				},
			}
		},
		"allows to specify container port": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Containers[0].Ports = []schema.Port{
						{Port: 80},
						{Port: 8080, ContainerPort: ptr.To(8085)},
					}
				},
				Asserts: func(t *testing.T, s *corev1.Service) {
					assert.Contains(t, s.Spec.Ports, corev1.ServicePort{
						Name:       "main-port",
						Protocol:   corev1.ProtocolTCP,
						Port:       int32(80),
						TargetPort: intstr.FromInt(80),
					})
					assert.Contains(t, s.Spec.Ports, corev1.ServicePort{
						Name:       "other-port-main-1",
						Protocol:   corev1.ProtocolTCP,
						Port:       int32(8080),
						TargetPort: intstr.FromInt(8085),
					})
				},
			}
		},
		"allows to not expose a port": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Containers[0].Ports = []schema.Port{
						{Port: 80},
						{Port: 8080, Expose: ptr.To(false)},
					}
				},
				Asserts: func(t *testing.T, s *corev1.Service) {
					assert.Len(t, s.Spec.Ports, 1)
				},
			}
		},
		"renders scrape label if service monitor is enabled": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.ServiceMonitor = &schema.ServiceMonitor{
						Enabled:   ptr.To(true),
						Endpoints: []monitoringv1.Endpoint{},
					}
				},
				Asserts: func(t *testing.T, s *corev1.Service) {
					assert.Subset(t, s.Labels, map[string]string{
						"app":               "service--component--test",
						"prometheus-scrape": "true",
					})
				},
			}
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

	for testName, makeConfig := range cases {
		t.Run(testName, func(t *testing.T) {
			values := DeploymentValues{}
			copier.CopyWithOption(&values, &base, copier.Option{DeepCopy: true})

			config := makeConfig()
			config.ValuesTransform(&values)

			_, create := CreateService(values)
			resources, err := create(values)
			if err != nil {
				t.Errorf("error during test setup: %v", err)
			}

			config.Asserts(t, fromUnstructuredOrPanic[*corev1.Service](resources[0]))
		})
	}
}
