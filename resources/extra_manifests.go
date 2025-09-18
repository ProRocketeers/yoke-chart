package resources

import "github.com/yokecd/yoke/pkg/flight"

func CreateExtraManifests(values DeploymentValues) (bool, ResourceCreator) {
	return len(values.ExtraManifests) > 0, func(values DeploymentValues) ([]flight.Resource, error) {
		resources := []flight.Resource{}
		for _, m := range values.ExtraManifests {
			resources = append(resources, &m)
		}
		return resources, nil
	}
}
