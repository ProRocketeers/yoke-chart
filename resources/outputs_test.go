package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func namedResource(category ResourceCategory, key, kind, name, namespace string) NamedResource {
	return NamedResource{
		Category: category,
		Key:      key,
		Object: unstructured.Unstructured{Object: map[string]interface{}{
			"kind": kind,
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
		}},
	}
}

func TestBuildOutputs(t *testing.T) {
	type CaseConfig struct {
		Resources []NamedResource
		Asserts   func(*testing.T, Outputs)
	}

	cases := map[string]CaseConfig{
		"groups singular resources by category, reading Name/Namespace/Kind off the object": {
			Resources: []NamedResource{
				namedResource(CategoryWorkload, "", "Deployment", "svc", "ns"),
				namedResource(CategoryService, "", "Service", "svc", "ns"),
			},
			Asserts: func(t *testing.T, outputs Outputs) {
				require.NotNil(t, outputs.Workload)
				assert.Equal(t, Ref{Name: "svc", Namespace: "ns", Kind: "Deployment"}, *outputs.Workload)
				require.NotNil(t, outputs.Service)
				assert.Equal(t, Ref{Name: "svc", Namespace: "ns", Kind: "Service"}, *outputs.Service)
				assert.Nil(t, outputs.Ingress)
			},
		},
		"reflects a name override instead of re-deriving it": {
			// e.g. ServiceAccount.AdditionalRole.Name overriding the default role name formula:
			// Outputs must reflect whatever name actually ended up on the created object.
			Resources: []NamedResource{
				namedResource(CategoryRole, "", "Role", "my-custom-role-name", "ns"),
			},
			Asserts: func(t *testing.T, outputs Outputs) {
				require.NotNil(t, outputs.Role)
				assert.Equal(t, "my-custom-role-name", outputs.Role.Name)
			},
		},
		"groups map-keyed resources by category and key": {
			Resources: []NamedResource{
				namedResource(CategoryHTTPRoutes, "main", "HTTPRoute", "service-main", "ns"),
				namedResource(CategoryHTTPRoutes, "internal", "HTTPRoute", "service-internal", "ns"),
			},
			Asserts: func(t *testing.T, outputs Outputs) {
				assert.Equal(t, Ref{Name: "service-main", Namespace: "ns", Kind: "HTTPRoute"}, outputs.HTTPRoutes["main"])
				assert.Equal(t, Ref{Name: "service-internal", Namespace: "ns", Kind: "HTTPRoute"}, outputs.HTTPRoutes["internal"])
			},
		},
		"map-keyed categories are non-nil but empty when nothing of that category was created": {
			Resources: nil,
			Asserts: func(t *testing.T, outputs Outputs) {
				assert.NotNil(t, outputs.HTTPRoutes)
				assert.Empty(t, outputs.HTTPRoutes)
				assert.NotNil(t, outputs.ConfigMaps)
				assert.Empty(t, outputs.ConfigMaps)
			},
		},
	}

	for testName, config := range cases {
		t.Run(testName, func(t *testing.T) {
			config.Asserts(t, BuildOutputs(config.Resources))
		})
	}
}
