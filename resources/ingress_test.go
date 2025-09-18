package resources

import (
	"testing"

	"github.com/ProRocketeers/yoke-chart/schema"
	"github.com/google/go-cmp/cmp"
	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/assert"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/utils/ptr"
)

func TestIngress(t *testing.T) {
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

	t.Run("renders the spec as is, including annotations and labels", func(t *testing.T) {
		values := DeploymentValues{}
		copier.CopyWithOption(&values, &base, copier.Option{DeepCopy: true})

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
		values.Ingress = &schema.Ingress{
			Enabled: ptr.To(true),
			Annotations: map[string]string{
				"my-annotation": "foo",
			},
			Labels: map[string]string{
				"my-label": "bar",
			},
			IngressSpec: networkingv1.IngressSpec{
				Rules:            []networkingv1.IngressRule{rule},
				IngressClassName: ptr.To("ic"),
				TLS:              []networkingv1.IngressTLS{tls},
			},
		}

		_, create := CreateIngress(values)
		resources, err := create(values)
		if err != nil {
			t.Errorf("error during test setup: %v", err)
		}

		ingress := fromUnstructuredOrPanic[*networkingv1.Ingress](resources[0])

		assert.Subset(t, ingress.Annotations, map[string]string{
			"my-annotation": "foo",
		})
		assert.Subset(t, ingress.Labels, map[string]string{
			"my-label": "bar",
		})
		partialContains(t, ingress.Spec.Rules, rule, cmp.Options{})
		partialContains(t, ingress.Spec.TLS, tls, cmp.Options{})
	})
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
