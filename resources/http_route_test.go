package resources

import (
	"testing"

	"github.com/ProRocketeers/yoke-chart/schema"
	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/assert"
	"k8s.io/utils/ptr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestHttpRoutes(t *testing.T) {
	type CaseConfig struct {
		ValuesTransform func(*DeploymentValues)
		Asserts         func(*testing.T, []*gatewayv1.HTTPRoute)
	}

	cases := map[string]func() CaseConfig{
		"doesn't render when httpRoutes is empty": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {},
				Asserts: func(t *testing.T, routes []*gatewayv1.HTTPRoute) {
					assert.Empty(t, routes)
				},
			}
		},
		"creates one route per map entry": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.HTTPRoutes = map[string]schema.HTTPRoute{
						"primary":   {},
						"secondary": {},
					}
				},
				Asserts: func(t *testing.T, routes []*gatewayv1.HTTPRoute) {
					assert.Len(t, routes, 2)
				},
			}
		},
		"names route using serviceName and map key": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.HTTPRoutes = map[string]schema.HTTPRoute{
						"primary": {},
					}
				},
				Asserts: func(t *testing.T, routes []*gatewayv1.HTTPRoute) {
					assert.Equal(t, "service--component--test-primary", routes[0].Name)
				},
			}
		},
		"propagates annotations and labels": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.HTTPRoutes = map[string]schema.HTTPRoute{
						"primary": {
							Annotations: map[string]string{"custom-annotation": "value"},
							Labels:      map[string]string{"custom-label": "value"},
						},
					}
				},
				Asserts: func(t *testing.T, routes []*gatewayv1.HTTPRoute) {
					assert.Equal(t, map[string]string{"custom-annotation": "value"}, routes[0].Annotations)
					assert.Subset(t, routes[0].Labels, map[string]string{"custom-label": "value"})
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
				Ports: []schema.Port{{Port: 8080}},
			},
		},
	}

	for testName, makeConfig := range cases {
		t.Run(testName, func(t *testing.T) {
			values := DeploymentValues{}
			copier.CopyWithOption(&values, &base, copier.Option{DeepCopy: true})

			config := makeConfig()
			config.ValuesTransform(&values)

			shouldCreate, create := CreateHttpRoutes(values)
			if !shouldCreate {
				config.Asserts(t, nil)
				return
			}

			resources, err := create(values)
			if err != nil {
				t.Errorf("error during test setup: %v", err)
			}

			config.Asserts(t, fromUnstructuredArrayOrPanic[*gatewayv1.HTTPRoute](resources))
		})
	}
}
