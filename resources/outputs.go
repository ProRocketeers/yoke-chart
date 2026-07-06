package resources

// Ref is a stable reference to a resource created by the chart, read directly off the object
// that was actually created rather than re-derived from a naming formula, so it can never drift
// out of sync (e.g. a Role's name being overridden via `AdditionalRole.Name`).
type Ref struct {
	Name      string
	Namespace string
	Kind      string
}

// Outputs exposes references to every resource the chart creates, keyed the same way the user
// referred to it in their input (e.g. the HTTPRoute's map key), for use in extraManifests
// templating. Singular fields are nil if that resource wasn't created.
type Outputs struct {
	Workload                *Ref
	HeadlessService         *Ref
	Service                 *Ref
	Ingress                 *Ref
	ServiceAccount          *Ref
	PreDeploymentJob        *Ref
	HPA                     *Ref
	PDB                     *Ref
	DB                      *Ref
	Role                    *Ref
	RoleBinding             *Ref
	ClusterRole             *Ref
	ClusterRoleBinding      *Ref
	ServiceMonitor          *Ref
	PreDeploymentPodMonitor *Ref

	HTTPRoutes         map[string]Ref
	NetworkPolicies    map[string]Ref
	ConfigMaps         map[string]Ref
	PVCs               map[string]Ref
	Cronjobs           map[string]Ref
	CronjobPodMonitors map[string]Ref
	ExternalSecrets    map[string]Ref
}

func BuildOutputs(resources []NamedResource) Outputs {
	outputs := Outputs{
		HTTPRoutes:         map[string]Ref{},
		NetworkPolicies:    map[string]Ref{},
		ConfigMaps:         map[string]Ref{},
		PVCs:               map[string]Ref{},
		Cronjobs:           map[string]Ref{},
		CronjobPodMonitors: map[string]Ref{},
		ExternalSecrets:    map[string]Ref{},
	}

	for _, r := range resources {
		ref := Ref{
			Name:      r.Object.GetName(),
			Namespace: r.Object.GetNamespace(),
			Kind:      r.Object.GetKind(),
		}

		switch r.Category {
		case CategoryWorkload:
			outputs.Workload = &ref
		case CategoryHeadlessService:
			outputs.HeadlessService = &ref
		case CategoryService:
			outputs.Service = &ref
		case CategoryIngress:
			outputs.Ingress = &ref
		case CategoryServiceAccount:
			outputs.ServiceAccount = &ref
		case CategoryPreDeploymentJob:
			outputs.PreDeploymentJob = &ref
		case CategoryHPA:
			outputs.HPA = &ref
		case CategoryPDB:
			outputs.PDB = &ref
		case CategoryDB:
			outputs.DB = &ref
		case CategoryRole:
			outputs.Role = &ref
		case CategoryRoleBinding:
			outputs.RoleBinding = &ref
		case CategoryClusterRole:
			outputs.ClusterRole = &ref
		case CategoryClusterRoleBinding:
			outputs.ClusterRoleBinding = &ref
		case CategoryServiceMonitor:
			outputs.ServiceMonitor = &ref
		case CategoryPreDeploymentPodMonitor:
			outputs.PreDeploymentPodMonitor = &ref
		case CategoryHTTPRoutes:
			outputs.HTTPRoutes[r.Key] = ref
		case CategoryNetworkPolicies:
			outputs.NetworkPolicies[r.Key] = ref
		case CategoryConfigMaps:
			outputs.ConfigMaps[r.Key] = ref
		case CategoryPVCs:
			outputs.PVCs[r.Key] = ref
		case CategoryCronjobs:
			outputs.Cronjobs[r.Key] = ref
		case CategoryCronjobPodMonitors:
			outputs.CronjobPodMonitors[r.Key] = ref
		case CategoryExternalSecrets:
			outputs.ExternalSecrets[r.Key] = ref
		}
	}

	return outputs
}
