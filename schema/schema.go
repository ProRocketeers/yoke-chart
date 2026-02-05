package schema

import (
	"fmt"
	"strconv"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	postgres "github.com/ProRocketeers/yoke-chart/resources/postgresql"
	es "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	yaml "github.com/goccy/go-yaml"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type InputValues struct {
	Metadata  `json:",inline"`
	Container `json:",inline"`

	MainContainerName   *string                           `json:"mainContainerName,omitempty"`
	ReplicaCount        *int                              `json:"replicaCount,omitempty"`
	Autoscaling         *HorizontalPodAutoscaler          `json:"autoscaling,omitempty"`
	Strategy            *appsv1.DeploymentStrategy        `json:"strategy,omitempty"`
	PodDisruptionBudget *policyv1.PodDisruptionBudgetSpec `json:"podDisruptionBudget,omitempty"`
	InitContainers      []InitContainer                   `json:"initContainers,omitempty" validate:"dive"`
	Ingress             *Ingress                          `json:"ingress,omitempty"`
	HTTPRoute           *HTTPRoute                        `json:"httpRoute,omitempty"`
	Volumes             map[string]Volume                 `json:"volumes,omitempty" validate:"dive"`
	Sidecars            map[string]Container              `json:"sidecars,omitempty" validate:"dive"`
	PreDeploymentJob    *PreDeploymentJob                 `json:"preDeploymentJob,omitempty"`
	ServiceAccount      *ServiceAccount                   `json:"serviceAccount,omitempty"`
	DB                  *Database                         `json:"db,omitempty"`
	Cronjobs            []Cronjob                         `json:"cronjobs,omitempty" validate:"dive"`
	ConfigMaps          map[string]map[string]string      `json:"configMaps"`
	ServiceMonitor      *ServiceMonitor                   `json:"serviceMonitor"`

	Annotations    map[string]string `json:"annotations,omitempty"`
	PodAnnotations map[string]string `json:"podAnnotations,omitempty"`
	Labels         map[string]string `json:"labels,omitempty"`
	PodLabels      map[string]string `json:"podLabels,omitempty"`

	NodeSelector map[string]string   `json:"nodeSelector,omitempty"`
	Tolerations  []corev1.Toleration `json:"tolerations,omitempty" validate:"dive"`
	Affinity     *corev1.Affinity    `json:"affinity,omitempty"`

	ExtraManifests []map[string]interface{} `json:"extraManifests,omitempty"`

	Kind        *string                 `json:"kind,omitempty"`
	StatefulSet *appsv1.StatefulSetSpec `json:"statefulSet,omitempty"`
}

type Image struct {
	Repository              string             `json:"repository" validate:"required"`
	Tag                     *string            `json:"tag,omitempty"`
	PullPolicy              *corev1.PullPolicy `json:"pullPolicy,omitempty"`
	PullSecrets             []string           `json:"pullSecrets,omitempty"`
	InheritMainContainerTag *bool              `json:"inheritMainContainerTag,omitempty"`
}

type Port struct {
	Port          int     `json:"port" validate:"required"`
	ContainerPort *int    `json:"containerPort,omitempty"`
	Expose        *bool   `json:"expose,omitempty"`
	Name          *string `json:"name"`
}

// basically HorizontalPodAutoscalerSpec without the `scaleTargetRef` field
type HorizontalPodAutoscaler struct {
	MinReplicas *int32                                         `json:"minReplicas,omitempty"`
	MaxReplicas int32                                          `json:"maxReplicas" validate:"required"`
	Metrics     []autoscalingv2.MetricSpec                     `json:"metrics,omitempty"`
	Behavior    *autoscalingv2.HorizontalPodAutoscalerBehavior `json:"behavior,omitempty"`
}

type Metadata struct {
	Namespace   string `json:"namespace" validate:"required"`
	Service     string `json:"service" validate:"required"`
	Component   string `json:"component" validate:"required"`
	Environment string `json:"environment" validate:"required"`
}

type Container struct {
	Image Image `json:"image" validate:"required"`

	Args            []string                     `json:"args,omitempty"`
	Command         []string                     `json:"command,omitempty"`
	Ports           []Port                       `json:"ports" validate:"dive"`
	Envs            map[string]string            `json:"envs,omitempty"`
	EnvsRaw         []corev1.EnvVar              `json:"envsRaw,omitempty" validate:"dive"`
	KubeSecrets     map[string]SecretMapping     `json:"kubeSecrets,omitempty" validate:"dive"`
	ExternalSecrets []ExternalSecretDefinition   `json:"externalSecrets,omitempty" validate:"dive"`
	Resources       *corev1.ResourceRequirements `json:"resources,omitempty"`
	ReadinessProbe  *corev1.Probe                `json:"readinessProbe,omitempty"`
	LivenessProbe   *corev1.Probe                `json:"livenessProbe,omitempty"`
	Lifecycle       *corev1.Lifecycle            `json:"lifecycle,omitempty"`
}

type SecretMapping = map[string]*string

type ExternalSecretDefinition struct {
	SecretStore     es.SecretStoreRef        `json:"secretStore" validate:"required"`
	RefreshInterval *metav1.Duration         `json:"refreshInterval"`
	Mapping         map[string]SecretMapping `json:"mapping" validate:"min=1,dive"`
}

type InitContainer struct {
	Container `json:",inline"`

	Name string `json:"name" validate:"required"`
}

type Ingress struct {
	Enabled     *bool             `json:"enabled" validate:"required"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`

	networkingv1.IngressSpec `json:",inline"`
}

type HTTPRoute struct {
	Enabled     *bool             `json:"enabled" validate:"required"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`

	gatewayv1.HTTPRouteSpec `json:",inline"`
}

type PreDeploymentJob struct {
	Container `json:",inline"`

	MainContainerName *string           `json:"mainContainerName,omitempty"`
	InitContainers    []InitContainer   `json:"initContainers,omitempty" validate:"dive"`
	Volumes           map[string]Volume `json:"volumes,omitempty" validate:"dive"`
	Annotations       map[string]string `json:"annotations,omitempty"`
	PodAnnotations    map[string]string `json:"podAnnotations,omitempty"`
	Labels            map[string]string `json:"labels,omitempty"`
	PodLabels         map[string]string `json:"podLabels,omitempty"`
	PodMonitor        *PodMonitor       `json:"podMonitor"`

	JobSpec `json:",inline"`
}

// k8s JobSpec, without the `template` field
// not the most up-to-date, but in compliance with the helm chart
type JobSpec struct {
	ActiveDeadlineSeconds   *int64                    `json:"activeDeadlineSeconds,omitempty"`
	BackoffLimit            *int32                    `json:"backoffLimit,omitempty"`
	CompletionMode          *batchv1.CompletionMode   `json:"completionMode,omitempty"`
	Completions             *int32                    `json:"completions,omitempty"`
	Parallelism             *int32                    `json:"parallelism,omitempty"`
	PodFailurePolicy        *batchv1.PodFailurePolicy `json:"podFailurePolicy,omitempty"`
	Selector                *metav1.LabelSelector     `json:"selector,omitempty"`
	Suspend                 *bool                     `json:"suspend,omitempty"`
	TTLSecondsAfterFinished *int32                    `json:"ttlSecondsAfterFinished,omitempty"`
}

type ServiceAccount struct {
	Annotations           map[string]string   `json:"annotations,omitempty"`
	AdditionalRole        *ServiceAccountRole `json:"additionalRole,omitempty"`
	AdditionalClusterRole *ServiceAccountRole `json:"additionalClusterRole,omitempty"`
}

type ServiceAccountRole struct {
	Name  *string             `json:"name,omitempty"`
	Rules []rbacv1.PolicyRule `json:"rules" validate:"required"`
}

type Database struct {
	Enabled          *bool                         `json:"enabled" validate:"required"`
	ClusterName      string                        `json:"clusterName" validate:"required"`
	Replicas         int                           `json:"replicas" validate:"required"`
	Version          int                           `json:"version" validate:"required"`
	Size             string                        `json:"size" validate:"required"`
	StorageClass     string                        `json:"storageClass" validate:"required"`
	Backup           *bool                         `json:"backup,omitempty"`
	Users            map[string]postgres.UserFlags `json:"users" validate:"required"`
	Databases        map[string]string             `json:"databases" validate:"required"`
	AdditionalConfig *postgres.PostgresSpec        `json:"additionalConfig,omitempty"`
}

type Cronjob struct {
	Container `json:",inline"`

	Name              string            `json:"name" validate:"required"`
	Schedule          string            `json:"schedule" validate:"required"`
	MainContainerName *string           `json:"mainContainerName,omitempty"`
	InitContainers    []InitContainer   `json:"initContainers,omitempty" validate:"dive"`
	Volumes           map[string]Volume `json:"volumes,omitempty" validate:"dive"`
	PodMonitor        *PodMonitor       `json:"podMonitor"`

	CronJobAnnotations map[string]string `json:"cronJobAnnotations,omitempty"`
	CronJobLabels      map[string]string `json:"cronJobLabels,omitempty"`
	JobAnnotations     map[string]string `json:"jobAnnotations,omitempty"`
	JobLabels          map[string]string `json:"jobLabels,omitempty"`
	PodAnnotations     map[string]string `json:"podAnnotations,omitempty"`
	PodLabels          map[string]string `json:"podLabels,omitempty"`

	// some CronJobSpec fields
	Suspend                    *bool                      `json:"suspend,omitempty"`
	TimeZone                   *string                    `json:"timeZone,omitempty"`
	ConcurrencyPolicy          *batchv1.ConcurrencyPolicy `json:"concurrencyPolicy,omitempty"`
	StartingDeadlineSeconds    *int64                     `json:"startingDeadlineSeconds,omitempty"`
	SuccessfulJobsHistoryLimit *int32                     `json:"successfulJobsHistoryLimit,omitempty"`
	FailedJobsHistoryLimit     *int32                     `json:"failedJobsHistoryLimit,omitempty"`
	// and some JobSpec fields
	ActiveDeadlineSeconds   *int64                    `json:"activeDeadlineSeconds,omitempty"`
	BackoffLimit            *int32                    `json:"backoffLimit,omitempty"`
	CompletionMode          *batchv1.CompletionMode   `json:"completionMode,omitempty"`
	Completions             *int32                    `json:"completions,omitempty"`
	Parallelism             *int32                    `json:"parallelism,omitempty"`
	PodFailurePolicy        *batchv1.PodFailurePolicy `json:"podFailurePolicy,omitempty"`
	Selector                *metav1.LabelSelector     `json:"selector,omitempty"`
	TTLSecondsAfterFinished *int32                    `json:"ttlSecondsAfterFinished,omitempty"`
}

type ServiceMonitor struct {
	Enabled   *bool                   `json:"enabled" validate:"required"`
	Endpoints []monitoringv1.Endpoint `json:"endpoints" validate:"required,min=1"`
}

type PodMonitor struct {
	Enabled   *bool                             `json:"enabled" validate:"required"`
	Endpoints []monitoringv1.PodMetricsEndpoint `json:"endpoints" validate:"required,min=1"`
}

// ------------ discriminated unions handling
// ------------ volumes
type VolumeType string

const (
	VolumeTypeStandardTmpfs VolumeType = "tmpfs"
	VolumeTypeStandardLocal VolumeType = "local"
	VolumeTypeRaw           VolumeType = "raw"
	VolumeTypePersistent    VolumeType = "persistent"
	VolumeTypeSecret        VolumeType = "secret"
	VolumeTypeConfigMap     VolumeType = "configMap"
)

type Volume struct {
	Type   VolumeType             `json:"type" validate:"required"`
	Mounts map[string]VolumeMount `json:"mounts" validate:"required,dive"`

	Variant VolumeVariant `json:"-"`
}

type VolumeMount struct {
	ContainerPath string  `json:"containerPath" validate:"required"`
	VolumePath    *string `json:"volumePath,omitempty"`
}

// Go doesn't have a type union, like `StandardVolume | RawVolume | ...`
// but while putting `interface{}` or `any` works, you could assign anything to that field even if it doesn't make sense
// putting an interface "marks" the struct as being allowed to be used in the field
type VolumeVariant interface {
	// the function can be a noop and not do anything, it just has to be implemented
	// since Go doesn't have any `implements` keyword, and is duck typed instead
	IsVolumeVariant()
}

type StandardVolume struct {
	// type: `tmpfs` or `local`
}

func (StandardVolume) IsVolumeVariant() {}

type RawVolume struct {
	// type: `raw`
	Spec corev1.VolumeSource `json:"spec" validate:"required"`
}

func (RawVolume) IsVolumeVariant() {}

type PersistentVolume struct {
	// type: `persistent`
	// `required` only validates if the value is not "zero", which for booleans is `false`
	// but `false` here is a valid value, so we have to make it a pointer to actually validate its presence
	Existing *bool `json:"existing" validate:"required"`

	// again splitting into variants, so we can validate each separately
	Variant PersistentVolumeVariant `json:"-"`
}

func (PersistentVolume) IsVolumeVariant() {}

type PersistentVolumeVariant interface{ IsPersistentVolumeVariant() }

type PersistentVolumeExisting struct {
	PvcName string `json:"pvcName" validate:"required"`
}

func (PersistentVolumeExisting) IsPersistentVolumeVariant() {}

type PersistentVolumeNew struct {
	// type: `persistent`
	AccessModes      []corev1.PersistentVolumeAccessMode `json:"accessModes" validate:"required"`
	Size             string                              `json:"size,omitempty"  validate:"required"`
	StorageClassName string                              `json:"storageClassName,omitempty"  validate:"required"`
	VolumeMode       *corev1.PersistentVolumeMode        `json:"volumeMode,omitempty"`
}

func (PersistentVolumeNew) IsPersistentVolumeVariant() {}

type SecretVolume struct {
	// type: `secret`
	Mode       *int32             `json:"mode"`
	SecretName string             `json:"secretName" validate:"required"`
	Items      map[string]*string `json:"items,omitempty"`
}

func (SecretVolume) IsVolumeVariant() {}

type ConfigMapVolume struct {
	// type: `configMap`
	Mode          *int32             `json:"mode"`
	ConfigMapName string             `json:"configMapName" validate:"required"`
	Items         map[string]*string `json:"items,omitempty"`
}

func (ConfigMapVolume) IsVolumeVariant() {}

// YAML package uses reflection to look for this interface on the given types and use these instead of the default behavior
func (v *Volume) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// `v` is not unmarshaled at this point yet, just created with zero values

	// type alias => identical in structure but doesn't copy methods => doesn't lead to recursion
	type alias Volume
	var a alias

	if err := unmarshal(&a); err != nil {
		return err
	}

	// initialize the fields from `a` which copies the parsed fields
	*v = Volume(a)

	switch v.Type {
	case VolumeTypeStandardTmpfs:
		fallthrough
	case VolumeTypeStandardLocal:
		var variant StandardVolume
		if err := unmarshal(&variant); err != nil {
			return err
		}
		v.Variant = variant
	case VolumeTypeRaw:
		var variant RawVolume
		if err := unmarshal(&variant); err != nil {
			return err
		}
		v.Variant = variant
	case VolumeTypePersistent:
		var persistentVariant PersistentVolume
		if err := unmarshal(&persistentVariant); err != nil {
			return err
		}

		if persistentVariant.Existing == nil {
			return fmt.Errorf("persistent volume missing 'existing' field")
		}
		if *persistentVariant.Existing {
			var variant PersistentVolumeExisting
			if err := unmarshal(&variant); err != nil {
				return err
			}
			persistentVariant.Variant = variant
		} else {
			var variant PersistentVolumeNew
			if err := unmarshal(&variant); err != nil {
				return err
			}
			if _, err := resource.ParseQuantity(variant.Size); err != nil {
				return fmt.Errorf("invalid volume size: %v", err)
			}
			persistentVariant.Variant = variant
		}
		v.Variant = persistentVariant
	case VolumeTypeSecret:
		var variant SecretVolume
		if err := unmarshal(&variant); err != nil {
			return err
		}
		v.Variant = variant
	case VolumeTypeConfigMap:
		var variant ConfigMapVolume
		if err := unmarshal(&variant); err != nil {
			return err
		}
		v.Variant = variant
	default:
		return fmt.Errorf("unknown volume type: %s", v.Type)
	}

	return nil
}

func init() {
	// both `intstr.IntOrString` and `resource.Quantity` are fields that can have either int or string with specific formats
	// in k8s YAML parser, they have their own YAML parsers so we have to somewhat duplicate that here as well
	yaml.RegisterCustomUnmarshaler(func(ios *intstr.IntOrString, b []byte) error {
		s := string(b)
		s = strings.TrimSpace(s)
		s = strings.Trim(s, `"`)

		if i, err := strconv.Atoi(s); err == nil {
			*ios = intstr.FromInt(i)
			return nil
		}
		*ios = intstr.FromString(s)
		return nil
	})

	yaml.RegisterCustomUnmarshaler(func(q *resource.Quantity, b []byte) error {
		s := string(b)
		s = strings.TrimSpace(s)
		s = strings.Trim(s, `"`)

		if quantity, err := resource.ParseQuantity(s); err != nil {
			return fmt.Errorf("error while parsing '%v' as resource quantity: %v", s, err)
		} else {
			*q = quantity
		}
		return nil
	})
}
