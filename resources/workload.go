package resources

func CreateMainWorkload(values DeploymentValues) (bool, ResourceCreator) {
	if values.Kind == "Deployment" {
		return CreateDeployment(values)
	}
	return CreateStatefulSet(values)
}
