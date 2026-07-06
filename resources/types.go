package resources

import (
	"github.com/ProRocketeers/yoke-chart/schema"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// meant as a place to put various parsed/prepared types, separate from the input schema

// *mostly* a copy of `InputValues` (starts with deep copy after all), but there are differences which is why this is copied instead
type DeploymentValues struct {
	Metadata   Metadata
	Containers []Container

	ReplicaCount        int
	Autoscaling         *schema.HorizontalPodAutoscaler
	Strategy            *appsv1.DeploymentStrategy
	PodDisruptionBudget *policyv1.PodDisruptionBudgetSpec
	InitContainers      []Container
	Ingress             *schema.Ingress
	HTTPRoutes          map[string]schema.HTTPRoute
	NetworkPolicies     map[string]networkingv1.NetworkPolicySpec
	Volumes             map[string]schema.Volume
	PreDeploymentJob    *PreDeploymentJob
	ServiceAccount      *schema.ServiceAccount
	DB                  *schema.Database
	Cronjobs            []Cronjob
	ConfigMaps          map[string]map[string]string
	ServiceMonitor      *schema.ServiceMonitor
	Service             ServiceConfig

	Annotations    map[string]string
	PodAnnotations map[string]string
	Labels         map[string]string
	PodLabels      map[string]string

	SchedulingConfig schema.SchedulingConfig
	PodSpec          *corev1.PodSpec

	ExtraManifests []unstructured.Unstructured

	Kind            string
	StatefulSetSpec *appsv1.StatefulSetSpec
	DeploymentSpec  *appsv1.DeploymentSpec
}

type Metadata struct {
	Namespace   string
	Service     string
	Component   string
	Environment string
}

type ServiceConfig struct {
	Type        corev1.ServiceType
	Annotations map[string]string
	Labels      map[string]string
	RawSpec     *corev1.ServiceSpec
}

type Container struct {
	Name            string
	Image           Image
	Args            []string
	Command         []string
	Ports           []schema.Port
	Envs            map[string]string
	EnvsRaw         []corev1.EnvVar
	KubeSecrets     map[string]schema.SecretMapping
	ExternalSecrets []schema.ExternalSecretDefinition
	Resources       *corev1.ResourceRequirements
	ReadinessProbe  *corev1.Probe
	LivenessProbe   *corev1.Probe
	StartupProbe    *corev1.Probe
	Lifecycle       *corev1.Lifecycle
	ContainerSpec   *corev1.Container
}

type Image struct {
	Repository  string
	Tag         *string
	PullPolicy  *corev1.PullPolicy
	PullSecrets []string
}

type PreDeploymentJob struct {
	Metadata         Metadata
	Container        Container
	InitContainers   []Container
	Volumes          map[string]schema.Volume
	Annotations      map[string]string
	PodAnnotations   map[string]string
	Labels           map[string]string
	PodLabels        map[string]string
	PodMonitor       *schema.PodMonitor
	PodSpec          *corev1.PodSpec
	SchedulingConfig schema.SchedulingConfig

	JobSpec *batchv1.JobSpec
}

type Cronjob struct {
	Metadata       Metadata
	Name           string
	Schedule       string
	Container      Container
	InitContainers []Container
	Volumes        map[string]schema.Volume
	PodMonitor     *schema.PodMonitor
	PodSpec        *corev1.PodSpec

	CronJobAnnotations map[string]string
	CronJobLabels      map[string]string
	JobAnnotations     map[string]string
	JobLabels          map[string]string
	PodAnnotations     map[string]string
	PodLabels          map[string]string
	SchedulingConfig   schema.SchedulingConfig

	CronJobSpec *batchv1.CronJobSpec
	JobSpec     *batchv1.JobSpec
}

// common interface of the Pods from Deployment, Job and CronJobs
type PodValues struct {
	ImagePullSecrets []corev1.LocalObjectReference
	InitContainers   []Container
	Metadata         Metadata
	Containers       []Container
	Volumes          map[string]schema.Volume
	SchedulingConfig schema.SchedulingConfig
	RawPodSpec       *corev1.PodSpec
}

type PodValuesExtractor interface {
	GetPodValues() PodValues
}

func getPullSecrets(containerArrays ...[]Container) []corev1.LocalObjectReference {
	pullSecretsMap := map[string]bool{}
	// essentially turn all the pull secrets into a Set, getting rid of duplicates
	for _, containers := range containerArrays {
		for _, container := range containers {
			for _, secret := range container.Image.PullSecrets {
				pullSecretsMap[secret] = true
			}
		}
	}
	ret := []corev1.LocalObjectReference{}
	for secret := range sortedMap(pullSecretsMap) {
		ret = append(ret, corev1.LocalObjectReference{Name: secret})
	}
	return ret
}

func (v *DeploymentValues) GetPodValues() PodValues {
	// pod values for the main deployment
	return PodValues{
		ImagePullSecrets: getPullSecrets(v.Containers, v.InitContainers),
		InitContainers:   v.InitContainers,
		Metadata:         v.Metadata,
		Containers:       v.Containers,
		Volumes:          v.Volumes,
		SchedulingConfig: v.SchedulingConfig,
		RawPodSpec:       v.PodSpec,
	}
}

func (v *PreDeploymentJob) GetPodValues() PodValues {
	// pod values for the pre deployment job
	return PodValues{
		ImagePullSecrets: getPullSecrets([]Container{v.Container}, v.InitContainers),
		InitContainers:   v.InitContainers,
		Metadata:         v.Metadata,
		Containers:       []Container{v.Container},
		Volumes:          v.Volumes,
		SchedulingConfig: v.SchedulingConfig,
		RawPodSpec:       v.PodSpec,
	}
}

func (v *Cronjob) GetPodValues() PodValues {
	// you guessed it..
	return PodValues{
		ImagePullSecrets: getPullSecrets([]Container{v.Container}, v.InitContainers),
		InitContainers:   v.InitContainers,
		Metadata:         v.Metadata,
		Containers:       []Container{v.Container},
		Volumes:          v.Volumes,
		SchedulingConfig: v.SchedulingConfig,
		RawPodSpec:       v.PodSpec,
	}
}

// ResourceCategory identifies which logical part of the chart a resource belongs to, used to
// group resources into Outputs. Not just Kind: some categories share a Kind (e.g. the
// pre-deployment job's PodMonitor vs. a cronjob's PodMonitor) while others (Workload) can be
// one of several Kinds (Deployment or StatefulSet) depending on config.
type ResourceCategory string

const (
	CategoryWorkload                ResourceCategory = "Workload"
	CategoryHeadlessService         ResourceCategory = "HeadlessService"
	CategoryService                 ResourceCategory = "Service"
	CategoryIngress                 ResourceCategory = "Ingress"
	CategoryServiceAccount          ResourceCategory = "ServiceAccount"
	CategoryPreDeploymentJob        ResourceCategory = "PreDeploymentJob"
	CategoryHPA                     ResourceCategory = "HPA"
	CategoryPDB                     ResourceCategory = "PDB"
	CategoryDB                      ResourceCategory = "DB"
	CategoryRole                    ResourceCategory = "Role"
	CategoryRoleBinding             ResourceCategory = "RoleBinding"
	CategoryClusterRole             ResourceCategory = "ClusterRole"
	CategoryClusterRoleBinding      ResourceCategory = "ClusterRoleBinding"
	CategoryServiceMonitor          ResourceCategory = "ServiceMonitor"
	CategoryPreDeploymentPodMonitor ResourceCategory = "PreDeploymentPodMonitor"
	CategoryHTTPRoutes              ResourceCategory = "HTTPRoutes"
	CategoryNetworkPolicies         ResourceCategory = "NetworkPolicies"
	CategoryConfigMaps              ResourceCategory = "ConfigMaps"
	CategoryPVCs                    ResourceCategory = "PVCs"
	CategoryCronjobs                ResourceCategory = "Cronjobs"
	CategoryCronjobPodMonitors      ResourceCategory = "CronjobPodMonitors"
	CategoryExternalSecrets         ResourceCategory = "ExternalSecrets"
)

// NamedResource pairs a created object with its logical Category and, for map-keyed resources
// (e.g. the HTTPRoute name), its Key. Key is empty for singular resources.
type NamedResource struct {
	Category ResourceCategory
	Key      string
	Object   unstructured.Unstructured
}

type ResourceCreator func(DeploymentValues) ([]NamedResource, error)
