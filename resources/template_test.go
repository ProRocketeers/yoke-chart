package resources

import (
	"testing"

	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestRenderExtraManifests(t *testing.T) {
	type CaseConfig struct {
		Manifest map[string]interface{}
		Outputs  Outputs
		Asserts  func(*testing.T, []unstructured.Unstructured, error)
	}

	base := DeploymentValues{
		Metadata: Metadata{
			Namespace:   "ns",
			Service:     "service",
			Component:   "component",
			Environment: "test",
		},
	}

	cases := map[string]CaseConfig{
		"leaves strings without template actions untouched": {
			Manifest: map[string]interface{}{
				"kind":     "ConfigMap",
				"metadata": map[string]interface{}{"name": "plain-name"},
			},
			Asserts: func(t *testing.T, rendered []unstructured.Unstructured, err error) {
				require.NoError(t, err)
				require.Len(t, rendered, 1)
				assert.Equal(t, "plain-name", rendered[0].GetName())
			},
		},
		"leaves non-string leaf values (int, bool, nil) untouched": {
			Manifest: map[string]interface{}{
				"kind":      "ConfigMap",
				"metadata":  map[string]interface{}{"name": "cm"},
				"immutable": true,
				"revision":  3,
				"note":      nil,
			},
			Asserts: func(t *testing.T, rendered []unstructured.Unstructured, err error) {
				require.NoError(t, err)
				require.Len(t, rendered, 1)
				assert.Equal(t, true, rendered[0].Object["immutable"])
				assert.Equal(t, 3, rendered[0].Object["revision"])
				assert.Nil(t, rendered[0].Object["note"])
			},
		},
		"substitutes a leaf value referencing Outputs": {
			Manifest: map[string]interface{}{
				"kind": "TrafficPolicy",
				"spec": map[string]interface{}{
					"targetRef": map[string]interface{}{
						"name": "{{ .Outputs.HTTPRoutes.main.Name }}",
						"kind": "HTTPRoute",
					},
				},
			},
			Outputs: Outputs{
				HTTPRoutes: map[string]Ref{
					"main": {Name: "service-main", Namespace: "ns", Kind: "HTTPRoute"},
				},
			},
			Asserts: func(t *testing.T, rendered []unstructured.Unstructured, err error) {
				require.NoError(t, err)
				require.Len(t, rendered, 1)
				spec := rendered[0].Object["spec"].(map[string]interface{})
				targetRef := spec["targetRef"].(map[string]interface{})
				assert.Equal(t, "service-main", targetRef["name"])
			},
		},
		"substitutes a leaf value referencing Values": {
			Manifest: map[string]interface{}{
				"kind": "ConfigMap",
				"metadata": map[string]interface{}{
					"name":      "cm",
					"namespace": "{{ .Values.Metadata.Namespace }}",
				},
			},
			Asserts: func(t *testing.T, rendered []unstructured.Unstructured, err error) {
				require.NoError(t, err)
				require.Len(t, rendered, 1)
				assert.Equal(t, "ns", rendered[0].GetNamespace())
			},
		},
		"supports the serviceName function": {
			Manifest: map[string]interface{}{
				"kind": "ConfigMap",
				"metadata": map[string]interface{}{
					"name": "{{ serviceName .Values.Metadata }}-extra",
				},
			},
			Asserts: func(t *testing.T, rendered []unstructured.Unstructured, err error) {
				require.NoError(t, err)
				require.Len(t, rendered, 1)
				assert.Equal(t, "service--component--test-extra", rendered[0].GetName())
			},
		},
		"supports curated sprig functions": {
			Manifest: map[string]interface{}{
				"kind": "ConfigMap",
				"metadata": map[string]interface{}{
					"name": `{{ upper "foo" | lower }}`,
				},
			},
			Asserts: func(t *testing.T, rendered []unstructured.Unstructured, err error) {
				require.NoError(t, err)
				require.Len(t, rendered, 1)
				assert.Equal(t, "foo", rendered[0].GetName())
			},
		},
		"rejects nonhermetic sprig functions like now/env": {
			Manifest: map[string]interface{}{
				"kind":     "ConfigMap",
				"metadata": map[string]interface{}{"name": "{{ now }}"},
			},
			Asserts: func(t *testing.T, rendered []unstructured.Unstructured, err error) {
				require.Error(t, err)
				assert.ErrorContains(t, err, "not defined")
			},
		},
		"rejects sprig functions that are non-deterministic despite Sprig's own hermetic labeling": {
			Manifest: map[string]interface{}{
				"kind":     "ConfigMap",
				"metadata": map[string]interface{}{"name": "{{ randInt 0 10 }}"},
			},
			Asserts: func(t *testing.T, rendered []unstructured.Unstructured, err error) {
				require.Error(t, err)
				assert.ErrorContains(t, err, "not defined")
			},
		},
		"recurses into nested maps and lists": {
			Manifest: map[string]interface{}{
				"kind": "Foo",
				"spec": map[string]interface{}{
					"rules": []interface{}{
						map[string]interface{}{
							"backendRefs": []interface{}{
								map[string]interface{}{
									"name": "{{ .Outputs.HTTPRoutes.main.Name }}",
								},
							},
						},
					},
				},
			},
			Outputs: Outputs{
				HTTPRoutes: map[string]Ref{"main": {Name: "service-main"}},
			},
			Asserts: func(t *testing.T, rendered []unstructured.Unstructured, err error) {
				require.NoError(t, err)
				spec := rendered[0].Object["spec"].(map[string]interface{})
				rules := spec["rules"].([]interface{})
				rule := rules[0].(map[string]interface{})
				backendRefs := rule["backendRefs"].([]interface{})
				backendRef := backendRefs[0].(map[string]interface{})
				assert.Equal(t, "service-main", backendRef["name"])
			},
		},
		"errors on a missing Outputs field instead of silently rendering blank": {
			Manifest: map[string]interface{}{
				"kind":     "ConfigMap",
				"metadata": map[string]interface{}{"name": "{{ .Outputs.NoSuchField }}"},
			},
			Asserts: func(t *testing.T, rendered []unstructured.Unstructured, err error) {
				require.Error(t, err)
			},
		},
		"rejects {{if}}": {
			Manifest: map[string]interface{}{
				"kind":     "ConfigMap",
				"metadata": map[string]interface{}{"name": "{{ if true }}a{{ end }}"},
			},
			Asserts: func(t *testing.T, rendered []unstructured.Unstructured, err error) {
				require.Error(t, err)
				assert.ErrorContains(t, err, "not supported")
			},
		},
		"rejects {{range}}": {
			Manifest: map[string]interface{}{
				"kind":     "ConfigMap",
				"metadata": map[string]interface{}{"name": "{{ range .Values.Cronjobs }}a{{ end }}"},
			},
			Asserts: func(t *testing.T, rendered []unstructured.Unstructured, err error) {
				require.Error(t, err)
				assert.ErrorContains(t, err, "not supported")
			},
		},
		"rejects {{with}}": {
			Manifest: map[string]interface{}{
				"kind":     "ConfigMap",
				"metadata": map[string]interface{}{"name": "{{ with .Values }}a{{ end }}"},
			},
			Asserts: func(t *testing.T, rendered []unstructured.Unstructured, err error) {
				require.Error(t, err)
				assert.ErrorContains(t, err, "not supported")
			},
		},
		"rejects {{define}}": {
			Manifest: map[string]interface{}{
				"kind":     "ConfigMap",
				"metadata": map[string]interface{}{"name": `{{ define "x" }}a{{ end }}`},
			},
			Asserts: func(t *testing.T, rendered []unstructured.Unstructured, err error) {
				require.Error(t, err)
				assert.ErrorContains(t, err, "not supported")
			},
		},
		"rejects {{block}}": {
			Manifest: map[string]interface{}{
				"kind":     "ConfigMap",
				"metadata": map[string]interface{}{"name": `{{ block "x" . }}a{{ end }}`},
			},
			Asserts: func(t *testing.T, rendered []unstructured.Unstructured, err error) {
				require.Error(t, err)
				assert.ErrorContains(t, err, "not supported")
			},
		},
	}

	for testName, config := range cases {
		t.Run(testName, func(t *testing.T) {
			values := DeploymentValues{}
			copier.CopyWithOption(&values, &base, copier.Option{DeepCopy: true})
			values.ExtraManifests = []unstructured.Unstructured{{Object: config.Manifest}}

			rendered, err := RenderExtraManifests(values, config.Outputs)
			config.Asserts(t, rendered, err)
		})
	}

	t.Run("returns nothing when there are no extraManifests", func(t *testing.T) {
		rendered, err := RenderExtraManifests(DeploymentValues{}, Outputs{})
		require.NoError(t, err)
		assert.Empty(t, rendered)
	})
}
