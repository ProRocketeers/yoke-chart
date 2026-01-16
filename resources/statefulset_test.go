package resources

import (
	"testing"

	"github.com/ProRocketeers/yoke-chart/schema"
	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

func TestStatefulSet(t *testing.T) {
	type CaseConfig struct {
		ValuesTransform func(*DeploymentValues)
		Asserts         func(*testing.T, *appsv1.StatefulSet, *corev1.Service)
	}

	// a function to be able to create a closure and share variables to compare against etc.
	cases := map[string]func() CaseConfig{
		"renders a headless service": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Containers[0].Ports = []schema.Port{
						{Port: 80},
						{Port: 8080},
					}
					dv.Kind = "StatefulSet"
					dv.StatefulSet = &appsv1.StatefulSetSpec{}
				},
				Asserts: func(t *testing.T, sts *appsv1.StatefulSet, s *corev1.Service) {
					require.NotEmpty(t, s)

					assert.Equal(t, s.Name, "service--component--test-headless")
					assert.Equal(t, sts.Spec.ServiceName, "service--component--test-headless")

					assert.Subset(t, s.Spec.Selector, map[string]string{
						"app": "service--component--test",
					})

					assert.Equal(t, s.Spec.Type, corev1.ServiceTypeClusterIP)
					assert.Equal(t, s.Spec.ClusterIP, corev1.ClusterIPNone)
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
		"uses a proper selector": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Kind = "StatefulSet"
					dv.StatefulSet = &appsv1.StatefulSetSpec{}
				},
				Asserts: func(t *testing.T, sts *appsv1.StatefulSet, s *corev1.Service) {
					assert.Subset(t, sts.Spec.Selector.MatchLabels, map[string]string{
						"app": "service--component--test",
					})
				},
			}
		},
		"allows setting any value in Kube StatefulSetSpec": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Kind = "StatefulSet"
					dv.StatefulSet = &appsv1.StatefulSetSpec{
						PodManagementPolicy: appsv1.ParallelPodManagement,
						Ordinals: &appsv1.StatefulSetOrdinals{
							Start: 4,
						},
					}
				},
				Asserts: func(t *testing.T, sts *appsv1.StatefulSet, s *corev1.Service) {
					assert.Equal(t, sts.Spec.PodManagementPolicy, appsv1.ParallelPodManagement)
					assert.Equal(t, sts.Spec.Ordinals.Start, int32(4))
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

			_, create := CreateStatefulSet(values)
			resources, err := create(values)
			if err != nil {
				t.Errorf("error during test setup: %v", err)
			}
			statefulSet := fromUnstructuredOrPanic[*appsv1.StatefulSet](resources[0])
			service := fromUnstructuredOrPanic[*corev1.Service](resources[1])

			config.Asserts(t, statefulSet, service)
		})
	}
}
