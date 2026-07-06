package schema

import (
	"fmt"
	"strconv"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	es "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	yaml "github.com/goccy/go-yaml"
)

type InputValues struct {
	Metadata  `json:",inline"`
	Container `json:",inline"`

	MainContainerName   *string                                   `json:"mainContainerName,omitempty"`
	ReplicaCount        *int                                      `json:"replicaCount,omitempty"`
	Autoscaling         *HorizontalPodAutoscaler                  `json:"autoscaling,omitempty"`
	Strategy            *appsv1.DeploymentStrategy                `json:"strategy,omitempty"`
	PodDisruptionBudget *policyv1.PodDisruptionBudgetSpec         `json:"podDisruptionBudget,omitempty"`
	InitContainers      []InitContainer                           `json:"initContainers,omitempty" validate:"dive"`
	Ingress             *Ingress                                  `json:"ingress,omitempty"`
	HTTPRoute           *HTTPRoute                                `json:"httpRoute,omitempty"`
	HTTPRoutes          map[string]HTTPRoute                      `json:"httpRoutes,omitempty" validate:"dive"`
	NetworkPolicies     map[string]networkingv1.NetworkPolicySpec `json:"networkPolicies"`
	Volumes             map[string]Volume                         `json:"volumes,omitempty" validate:"dive"`
	Sidecars            map[string]Container                      `json:"sidecars,omitempty" validate:"dive"`
	PreDeploymentJob    *PreDeploymentJob                         `json:"preDeploymentJob,omitempty"`
	ServiceAccount      *ServiceAccount                           `json:"serviceAccount,omitempty"`
	DB                  *Database                                 `json:"db,omitempty"`
	Cronjobs            []Cronjob                                 `json:"cronjobs,omitempty" validate:"dive"`
	ConfigMaps          map[string]map[string]string              `json:"configMaps"`
	ServiceMonitor      *ServiceMonitor                           `json:"serviceMonitor"`

	ServiceConfig *ServiceConfig `json:"serviceConfig,omitempty"`

	Annotations    map[string]string `json:"annotations,omitempty"`
	PodAnnotations map[string]string `json:"podAnnotations,omitempty"`
	Labels         map[string]string `json:"labels,omitempty"`
	PodLabels      map[string]string `json:"podLabels,omitempty"`

	SchedulingConfig `json:",inline"`
	PodSpec          *corev1.PodSpec `json:"podSpec,omitempty"`

	ExtraManifests []map[string]interface{} `json:"extraManifests,omitempty"`

	Kind            *string                 `json:"kind,omitempty"`
	StatefulSetSpec *appsv1.StatefulSetSpec `json:"statefulSetSpec,omitempty"`
	DeploymentSpec  *appsv1.DeploymentSpec  `json:"deploymentSpec,omitempty"`
}

type SchedulingConfig struct {
	NodeSelector              map[string]string                 `json:"nodeSelector,omitempty"`
	Tolerations               []corev1.Toleration               `json:"tolerations,omitempty" validate:"dive"`
	Affinity                  *corev1.Affinity                  `json:"affinity,omitempty"`
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty" validate:"dive"`
	PriorityClassName         *string                           `json:"priorityClassName,omitempty"`
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
	NodePort      *int32  `json:"nodePort"`
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
	ContainerSpec   *corev1.Container            `json:"containerSpec,omitempty"`
}

type SecretMapping = map[string]*string

type ExternalSecretDefinition struct {
	SecretStore     es.SecretStoreRef        `json:"secretStore" validate:"required"`
	RefreshInterval *metav1.Duration         `json:"refreshInterval"`
	Mapping         map[string]SecretMapping `json:"mapping" validate:"min=1,dive"`

	// OPTIONAL - defaults to `Owner`/`Delete` (today's behavior) when unset
	CreationPolicy *es.ExternalSecretCreationPolicy `json:"creationPolicy,omitempty"`
	DeletionPolicy *es.ExternalSecretDeletionPolicy `json:"deletionPolicy,omitempty"`
}

type InitContainer struct {
	Container `json:",inline"`

	Name string `json:"name" validate:"required"`
}

type ServiceConfig struct {
	Annotations map[string]string `json:"annotations,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`

	// OPTIONAL - escape hatch: full Kubernetes `ServiceSpec`, inlined so `serviceConfig.type` etc. keep
	// working directly. The Flight builds `selector`/`type`/`ports` itself first, then layers this on
	// top - same merge semantics as `containerSpec`/`podSpec`/etc.
	corev1.ServiceSpec `json:",inline"`
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
