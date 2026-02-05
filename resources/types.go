package resources

import (
	"github.com/ProRocketeers/yoke-chart/schema"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	HTTPRoute           *schema.HTTPRoute
	Volumes             map[string]schema.Volume
	Sidecars            map[string]schema.Container
	PreDeploymentJob    *PreDeploymentJob
	ServiceAccount      *schema.ServiceAccount
	DB                  *schema.Database
	Cronjobs            []Cronjob
	ConfigMaps          map[string]map[string]string
	ServiceMonitor      *schema.ServiceMonitor

	Annotations    map[string]string
	PodAnnotations map[string]string
	Labels         map[string]string
	PodLabels      map[string]string

	NodeSelector map[string]string
	Tolerations  []corev1.Toleration
	Affinity     *corev1.Affinity

	ExtraManifests []unstructured.Unstructured

	Kind        string
	StatefulSet *appsv1.StatefulSetSpec
}

type Metadata struct {
	Namespace   string
	Service     string
	Component   string
	Environment string
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
	Lifecycle       *corev1.Lifecycle
}

type Image struct {
	Repository  string
	Tag         *string
	PullPolicy  *corev1.PullPolicy
	PullSecrets []string
}

type PreDeploymentJob struct {
	Metadata       Metadata
	Container      Container
	InitContainers []Container
	Volumes        map[string]schema.Volume
	Annotations    map[string]string
	PodAnnotations map[string]string
	Labels         map[string]string
	PodLabels      map[string]string
	PodMonitor     *schema.PodMonitor

	schema.JobSpec
}

type Cronjob struct {
	Metadata       Metadata
	Name           string
	Schedule       string
	Container      Container
	InitContainers []Container
	Volumes        map[string]schema.Volume
	PodMonitor     *schema.PodMonitor

	CronJobAnnotations map[string]string
	CronJobLabels      map[string]string
	JobAnnotations     map[string]string
	JobLabels          map[string]string
	PodAnnotations     map[string]string
	PodLabels          map[string]string

	// some CronJobSpec fields
	Suspend                    *bool
	TimeZone                   *string
	ConcurrencyPolicy          *batchv1.ConcurrencyPolicy
	StartingDeadlineSeconds    *int64
	SuccessfulJobsHistoryLimit *int32
	FailedJobsHistoryLimit     *int32
	// and some JobSpec fields
	ActiveDeadlineSeconds   *int64
	BackoffLimit            *int32
	CompletionMode          *batchv1.CompletionMode
	Completions             *int32
	Parallelism             *int32
	PodFailurePolicy        *batchv1.PodFailurePolicy
	Selector                *metav1.LabelSelector
	TTLSecondsAfterFinished *int32
}

// common interface of the Pods from Deployment, Job and CronJobs
type PodValues struct {
	ImagePullSecrets []corev1.LocalObjectReference
	InitContainers   []Container
	Metadata         Metadata
	Containers       []Container
	Volumes          map[string]schema.Volume
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
	for secret := range pullSecretsMap {
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
	}
}

type ResourceCreator func(DeploymentValues) ([]unstructured.Unstructured, error)
