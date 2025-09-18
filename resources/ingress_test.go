package resources

import (
	"testing"

	"github.com/ProRocketeers/yoke-chart/schema"
	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/assert"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/utils/ptr"
)

func TestIngress(t *testing.T) {
	type CaseConfig struct {
		ValuesTransform func(*DeploymentValues)
		Asserts         func(*testing.T, *networkingv1.Ingress)
	}

	cases := map[string]func() CaseConfig{
		"renders predefined defaults with the simple flag": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Ingress = &schema.Ingress{
						Enabled: ptr.To(true),
						Simple:  ptr.To(true),
						Variant: schema.SimpleIngress{
							Host: "ingress-host.com",
						},
					}
				},
				Asserts: func(t *testing.T, i *networkingv1.Ingress) {
					assert.Subset(t, i.Annotations, map[string]string{
						"kubernetes.io/ingress.class":                             "nginx",
						"traefik.ingress.kubernetes.io/router.entrypoints":        "websecure",
						"traefik.ingress.kubernetes.io/router.tls":                "true",
						"traefik.ingress.kubernetes.io/router.tls.certresolver":   "static",
						"traefik.ingress.kubernetes.io/router.tls.domains.0.main": "ingress-host.com",
					})
					assert.Contains(t, i.Spec.Rules, networkingv1.IngressRule{
						Host: "ingress-host.com",
						IngressRuleValue: networkingv1.IngressRuleValue{
							HTTP: &networkingv1.HTTPIngressRuleValue{
								Paths: []networkingv1.HTTPIngressPath{
									{
										Path:     "/",
										PathType: ptr.To(networkingv1.PathTypeImplementationSpecific),
										Backend: networkingv1.IngressBackend{
											Service: &networkingv1.IngressServiceBackend{
												Name: "service--component--test",
												Port: networkingv1.ServiceBackendPort{
													Number: int32(8080),
												},
											},
										},
									},
								},
							},
						},
					})
				},
			}
		},
		"renders the spec when not using simple config": func() CaseConfig {
			rule := networkingv1.IngressRule{
				Host: "my-ingress-host.com",
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{
							{
								Path:     "/",
								PathType: ptr.To(networkingv1.PathTypeImplementationSpecific),
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: "service--component--test",
										Port: networkingv1.ServiceBackendPort{
											Number: int32(8080),
										},
									},
								},
							},
						},
					},
				},
			}
			tls := networkingv1.IngressTLS{
				SecretName: "my-secret-tls",
				Hosts:      []string{"my-ingress-host.com"},
			}
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Ingress = &schema.Ingress{
						Enabled: ptr.To(true),
						Simple:  ptr.To(false),
						Variant: schema.FullIngress{
							Annotations: map[string]string{
								"my-annotation": "foo",
							},
							IngressSpec: networkingv1.IngressSpec{
								Rules:            []networkingv1.IngressRule{rule},
								IngressClassName: ptr.To("ic"),
								TLS:              []networkingv1.IngressTLS{tls},
							},
						},
					}
				},
				Asserts: func(t *testing.T, i *networkingv1.Ingress) {
					assert.Subset(t, i.Annotations, map[string]string{
						"my-annotation": "foo",
					})
					assert.Contains(t, i.Spec.Rules, rule)
				},
			}
		},
		"properly renders Homepage annotation with simple config": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Ingress = &schema.Ingress{
						Enabled: ptr.To(true),
						Simple:  ptr.To(true),
						Homepage: map[string]string{
							"enabled": "true",
							"href":    "https://ingress-host.com",
							"icon":    "circle",
						},
						Variant: schema.SimpleIngress{
							Host: "ingress-host.com",
						},
					}
				},
				Asserts: func(t *testing.T, i *networkingv1.Ingress) {
					assert.Subset(t, i.Annotations, map[string]string{
						"gethomepage.dev/enabled": "true",
						"gethomepage.dev/href":    "https://ingress-host.com",
						"gethomepage.dev/icon":    "circle",
					})
				},
			}
		},
		"properly renders Homepage annotation with full config": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Ingress = &schema.Ingress{
						Enabled: ptr.To(true),
						Simple:  ptr.To(false),
						Homepage: map[string]string{
							"enabled": "true",
							"href":    "https://my-ingress-host.com",
							"icon":    "circle",
						},
						Variant: schema.FullIngress{
							Annotations: map[string]string{
								"my-annotation": "foo",
							},
							IngressSpec: networkingv1.IngressSpec{
								Rules: []networkingv1.IngressRule{
									{
										Host: "my-ingress-host.com",
										IngressRuleValue: networkingv1.IngressRuleValue{
											HTTP: &networkingv1.HTTPIngressRuleValue{
												Paths: []networkingv1.HTTPIngressPath{
													{
														Path:     "/",
														PathType: ptr.To(networkingv1.PathTypeImplementationSpecific),
														Backend: networkingv1.IngressBackend{
															Service: &networkingv1.IngressServiceBackend{
																Name: "service--component--test",
																Port: networkingv1.ServiceBackendPort{
																	Number: int32(8080),
																},
															},
														},
													},
												},
											},
										},
									},
								},
								IngressClassName: ptr.To("ic"),
								TLS: []networkingv1.IngressTLS{
									{
										SecretName: "my-secret-tls",
										Hosts:      []string{"my-ingress-host.com"},
									},
								},
							},
						},
					}
				},
				Asserts: func(t *testing.T, i *networkingv1.Ingress) {
					assert.Subset(t, i.Annotations, map[string]string{
						"gethomepage.dev/enabled": "true",
						"gethomepage.dev/href":    "https://my-ingress-host.com",
						"gethomepage.dev/icon":    "circle",
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

			_, create := CreateIngress(values)
			resources, err := create(values)
			if err != nil {
				t.Errorf("error during test setup: %v", err)
			}
			ingress := resources[0].(*networkingv1.Ingress)

			config.Asserts(t, ingress)
		})
	}

	t.Run("doesn't render when ingress is not explicitly enabled", func(t *testing.T) {
		values := DeploymentValues{}
		copier.CopyWithOption(&values, &base, copier.Option{DeepCopy: true})
		values.Ingress = &schema.Ingress{
			Enabled: ptr.To(false),
		}

		shouldCreate, _ := CreateIngress(values)

		assert.False(t, shouldCreate)
	})
}
