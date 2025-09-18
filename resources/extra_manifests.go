package resources

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

func CreateExtraManifests(values DeploymentValues) (bool, ResourceCreator) {
	return len(values.ExtraManifests) > 0, func(values DeploymentValues) ([]unstructured.Unstructured, error) {
		return values.ExtraManifests, nil
	}
}
