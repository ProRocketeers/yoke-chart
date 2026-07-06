package resources

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// RenderExtraManifests templates each extraManifests entry's leaf string values against values
// and outputs, and returns the rendered, ready-to-apply objects. Unlike every other resource,
// extraManifests aren't tagged into Outputs: they're user-authored, so nothing else in the chart
// could ever need to reference one by name.
func RenderExtraManifests(values DeploymentValues, outputs Outputs) ([]unstructured.Unstructured, error) {
	if len(values.ExtraManifests) == 0 {
		return nil, nil
	}

	ctx := TemplateContext{Values: values, Outputs: outputs}

	rendered := make([]unstructured.Unstructured, 0, len(values.ExtraManifests))
	for i, manifest := range values.ExtraManifests {
		out, err := templateLeafValues(manifest.Object, ctx)
		if err != nil {
			return nil, fmt.Errorf("error templating extraManifests[%d]: %w", i, err)
		}
		rendered = append(rendered, unstructured.Unstructured{Object: out.(map[string]interface{})})
	}
	return rendered, nil
}
