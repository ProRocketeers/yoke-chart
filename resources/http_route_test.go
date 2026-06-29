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
		"defaults parentRef group and kind when unset": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.HTTPRoutes = map[string]schema.HTTPRoute{
						"primary": {
							HTTPRouteSpec: gatewayv1.HTTPRouteSpec{
								CommonRouteSpec: gatewayv1.CommonRouteSpec{
									ParentRefs: []gatewayv1.ParentReference{
										{Name: "my-gateway"},
									},
								},
							},
						},
					}
				},
				Asserts: func(t *testing.T, routes []*gatewayv1.HTTPRoute) {
					ref := routes[0].Spec.ParentRefs[0]
					assert.Equal(t, ptr.To(gatewayv1.Group("gateway.networking.k8s.io")), ref.Group)
					assert.Equal(t, ptr.To(gatewayv1.Kind("Gateway")), ref.Kind)
				},
			}
		},
		"preserves explicit parentRef group and kind": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.HTTPRoutes = map[string]schema.HTTPRoute{
						"primary": {
							HTTPRouteSpec: gatewayv1.HTTPRouteSpec{
								CommonRouteSpec: gatewayv1.CommonRouteSpec{
									ParentRefs: []gatewayv1.ParentReference{
										{
											Name:  "my-mesh-service",
											Group: ptr.To(gatewayv1.Group("")),
											Kind:  ptr.To(gatewayv1.Kind("Service")),
										},
									},
								},
							},
						},
					}
				},
				Asserts: func(t *testing.T, routes []*gatewayv1.HTTPRoute) {
					ref := routes[0].Spec.ParentRefs[0]
					assert.Equal(t, ptr.To(gatewayv1.Group("")), ref.Group)
					assert.Equal(t, ptr.To(gatewayv1.Kind("Service")), ref.Kind)
				},
			}
		},
		"defaults backendRef group, kind, and weight when unset": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.HTTPRoutes = map[string]schema.HTTPRoute{
						"primary": {
							HTTPRouteSpec: gatewayv1.HTTPRouteSpec{
								Rules: []gatewayv1.HTTPRouteRule{
									{
										BackendRefs: []gatewayv1.HTTPBackendRef{
											{
												BackendRef: gatewayv1.BackendRef{
													BackendObjectReference: gatewayv1.BackendObjectReference{
														Name: "my-svc",
														Port: ptr.To(gatewayv1.PortNumber(8080)),
													},
												},
											},
										},
									},
								},
							},
						},
					}
				},
				Asserts: func(t *testing.T, routes []*gatewayv1.HTTPRoute) {
					ref := routes[0].Spec.Rules[0].BackendRefs[0]
					assert.Equal(t, ptr.To(gatewayv1.Group("")), ref.Group)
					assert.Equal(t, ptr.To(gatewayv1.Kind("Service")), ref.Kind)
					assert.Equal(t, ptr.To(int32(1)), ref.Weight)
				},
			}
		},
		"defaults path type and value when path is set without them": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.HTTPRoutes = map[string]schema.HTTPRoute{
						"primary": {
							HTTPRouteSpec: gatewayv1.HTTPRouteSpec{
								Rules: []gatewayv1.HTTPRouteRule{
									{
										Matches: []gatewayv1.HTTPRouteMatch{
											{
												Path: &gatewayv1.HTTPPathMatch{},
											},
										},
									},
								},
							},
						},
					}
				},
				Asserts: func(t *testing.T, routes []*gatewayv1.HTTPRoute) {
					path := routes[0].Spec.Rules[0].Matches[0].Path
					assert.Equal(t, ptr.To(gatewayv1.PathMatchPathPrefix), path.Type)
					assert.Equal(t, ptr.To("/"), path.Value)
				},
			}
		},
		"does not set path defaults when match has no path": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.HTTPRoutes = map[string]schema.HTTPRoute{
						"primary": {
							HTTPRouteSpec: gatewayv1.HTTPRouteSpec{
								Rules: []gatewayv1.HTTPRouteRule{
									{
										Matches: []gatewayv1.HTTPRouteMatch{
											{
												Method: ptr.To(gatewayv1.HTTPMethodGet),
											},
										},
									},
								},
							},
						},
					}
				},
				Asserts: func(t *testing.T, routes []*gatewayv1.HTTPRoute) {
					assert.Nil(t, routes[0].Spec.Rules[0].Matches[0].Path)
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
