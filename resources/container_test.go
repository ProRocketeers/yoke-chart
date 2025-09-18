package resources

import (
	"testing"

	"github.com/ProRocketeers/yoke-chart/schema"
	es "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/google/go-cmp/cmp"
	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

func TestContainer(t *testing.T) {
	type CaseConfig struct {
		ValuesTransform func(*DeploymentValues)
		Asserts         func(*testing.T, *appsv1.Deployment)
	}

	// a function to be able to create a closure and share variables to compare against etc.
	cases := map[string]func() CaseConfig{
		"can specify image pull policy": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Containers[0].Image.PullPolicy = ptr.To(corev1.PullAlways)
				},
				Asserts: func(t *testing.T, d *appsv1.Deployment) {
					assert.Equal(t, corev1.PullAlways, d.Spec.Template.Spec.Containers[0].ImagePullPolicy)
				},
			}
		},
		"default image pull policy is IfNotPresent": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {},
				Asserts: func(t *testing.T, d *appsv1.Deployment) {
					assert.Equal(t, corev1.PullIfNotPresent, d.Spec.Template.Spec.Containers[0].ImagePullPolicy)
				},
			}
		},
		"can specify args": func() CaseConfig {
			args := []string{"npm", "run", "start"}
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Containers[0].Args = args
				},
				Asserts: func(t *testing.T, d *appsv1.Deployment) {
					assert.Equal(t, args, d.Spec.Template.Spec.Containers[0].Args)
				},
			}
		},
		"can specify command": func() CaseConfig {
			command := []string{"npm", "run", "start"}
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Containers[0].Command = command
				},
				Asserts: func(t *testing.T, d *appsv1.Deployment) {
					assert.Equal(t, command, d.Spec.Template.Spec.Containers[0].Command)
				},
			}
		},
		"renders classic key-value envs": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Containers[0].Envs = map[string]string{
						"MY_ENV": "foo",
					}
				},
				Asserts: func(t *testing.T, d *appsv1.Deployment) {
					assert.Equal(t, corev1.EnvVar{Name: "MY_ENV", Value: "foo"}, d.Spec.Template.Spec.Containers[0].Env[0])
				},
			}
		},
		"renders partial export from Kube secret": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Containers[0].KubeSecrets = map[string]schema.SecretMapping{
						"my-secret-name": {
							"MY_ENV": ptr.To("foo"),
						},
					}
				},
				Asserts: func(t *testing.T, d *appsv1.Deployment) {
					assert.Equal(t, corev1.EnvVar{
						Name: "MY_ENV",
						ValueFrom: &corev1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "my-secret-name",
								},
								Key: "foo",
							},
						},
					}, d.Spec.Template.Spec.Containers[0].Env[0])
				},
			}
		},
		"renders full export from Kube secret": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Containers[0].KubeSecrets = map[string]schema.SecretMapping{
						"my-secret-name": nil,
					}
				},
				Asserts: func(t *testing.T, d *appsv1.Deployment) {
					assert.Equal(t, corev1.EnvFromSource{
						SecretRef: &corev1.SecretEnvSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "my-secret-name",
							},
						},
					}, d.Spec.Template.Spec.Containers[0].EnvFrom[0])
				},
			}
		},
		"renders export from Vault secrets": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Containers[0].ExternalSecrets = []schema.ExternalSecretDefinition{
						{
							SecretStore: es.SecretStoreRef{
								Name: "vault",
								Kind: "ClusterSecretStore",
							},
							Mapping: map[string]schema.SecretMapping{
								"path/to/secret": {
									"MY_ENV": ptr.To("MY-SECRET"),
								},
							},
						},
					}
				},
				Asserts: func(t *testing.T, d *appsv1.Deployment) {
					assert.Equal(t, corev1.EnvFromSource{
						SecretRef: &corev1.SecretEnvSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "service--component--test--vault--path-to-secret",
							},
						},
					}, d.Spec.Template.Spec.Containers[0].EnvFrom[0])
				},
			}
		},
		"renders raw envs": func() CaseConfig {
			env := corev1.EnvVar{
				Name: "CPU_REQUEST",
				ValueFrom: &corev1.EnvVarSource{
					ResourceFieldRef: &corev1.ResourceFieldSelector{
						ContainerName: "main",
						Resource:      "requests.cpu",
					},
				},
			}
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Containers[0].EnvsRaw = []corev1.EnvVar{env}
				},
				Asserts: func(t *testing.T, d *appsv1.Deployment) {
					partialEqual(t, env, d.Spec.Template.Spec.Containers[0].Env[0], cmp.Options{})
				},
			}
		},
		"renders ports": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Containers[0].Ports = []schema.Port{
						{Port: 80},
						{Port: 8085, ContainerPort: ptr.To(8086)},
						{Port: 8087, Name: ptr.To("metrics")},
					}
				},
				Asserts: func(t *testing.T, d *appsv1.Deployment) {
					assert.Contains(t, d.Spec.Template.Spec.Containers[0].Ports, corev1.ContainerPort{ContainerPort: 80})
					assert.Contains(t, d.Spec.Template.Spec.Containers[0].Ports, corev1.ContainerPort{ContainerPort: 8086})
					assert.Contains(t, d.Spec.Template.Spec.Containers[0].Ports, corev1.ContainerPort{ContainerPort: 8087, Name: "metrics"})
				},
			}
		},
		"renders lifecycle": func() CaseConfig {
			cmd := []string{"/bin/sh", "-c", `sleep "2"`}
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Containers[0].Lifecycle = &corev1.Lifecycle{
						PreStop: &corev1.LifecycleHandler{
							Exec: &corev1.ExecAction{
								Command: cmd,
							},
						},
					}
				},
				Asserts: func(t *testing.T, d *appsv1.Deployment) {
					assert.Equal(t, cmd, d.Spec.Template.Spec.Containers[0].Lifecycle.PreStop.Exec.Command)
				},
			}
		},
		"renders resources": func() CaseConfig {
			resources := corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("100Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("200Mi"),
				},
			}
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Containers[0].Resources = &resources
				},
				Asserts: func(t *testing.T, d *appsv1.Deployment) {
					assert.Equal(t, resources, d.Spec.Template.Spec.Containers[0].Resources)
				},
			}
		},
		"renders readinessProbeRaw": func() CaseConfig {
			probe := corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Port: intstr.FromInt(8085),
						Host: "/",
						Path: "/pong",
					},
				},
			}
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Containers[0].ReadinessProbe = &probe
				},
				Asserts: func(t *testing.T, d *appsv1.Deployment) {
					assert.Equal(t, probe.HTTPGet.Port, d.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Port)
					assert.Equal(t, probe.HTTPGet.Host, d.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Host)
					assert.Equal(t, probe.HTTPGet.Path, d.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Path)
				},
			}
		},
		"renders livenessProbe": func() CaseConfig {
			probe := corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Port: intstr.FromInt(80),
						Host: "/",
						Path: "/pong",
					},
				},
				InitialDelaySeconds: int32(5),
				PeriodSeconds:       int32(10),
				TimeoutSeconds:      int32(5),
				FailureThreshold:    int32(3),
				SuccessThreshold:    int32(1),
			}
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Containers[0].LivenessProbe = &probe
				},
				Asserts: func(t *testing.T, d *appsv1.Deployment) {
					assert.Equal(t, probe.HTTPGet.Port, d.Spec.Template.Spec.Containers[0].LivenessProbe.HTTPGet.Port)
					assert.Equal(t, probe.HTTPGet.Host, d.Spec.Template.Spec.Containers[0].LivenessProbe.HTTPGet.Host)
					assert.Equal(t, probe.HTTPGet.Path, d.Spec.Template.Spec.Containers[0].LivenessProbe.HTTPGet.Path)
					assert.Equal(t, probe.InitialDelaySeconds, d.Spec.Template.Spec.Containers[0].LivenessProbe.InitialDelaySeconds)
					assert.Equal(t, probe.PeriodSeconds, d.Spec.Template.Spec.Containers[0].LivenessProbe.PeriodSeconds)
					assert.Equal(t, probe.TimeoutSeconds, d.Spec.Template.Spec.Containers[0].LivenessProbe.TimeoutSeconds)
					assert.Equal(t, probe.FailureThreshold, d.Spec.Template.Spec.Containers[0].LivenessProbe.FailureThreshold)
					assert.Equal(t, probe.SuccessThreshold, d.Spec.Template.Spec.Containers[0].LivenessProbe.SuccessThreshold)
				},
			}
		},
		"doesn't render volume mount if it's NOT mounted in the container": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.InitContainers = []Container{
						{
							Name: "init-container",
							Image: Image{
								Repository: "image_repository_init",
								Tag:        ptr.To("foo"),
							},
						},
					}
					dv.Volumes = map[string]schema.Volume{
						"my-volume": {
							Type: schema.VolumeTypeStandardTmpfs,
							Mounts: map[string]schema.VolumeMount{
								"init-container": {
									ContainerPath: "/my-volume",
								},
							},
							// empty, but fails the test without it
							Variant: schema.StandardVolume{},
						},
					}
				},
				Asserts: func(t *testing.T, d *appsv1.Deployment) {
					// not mounted in the main container
					assert.Empty(t, d.Spec.Template.Spec.Containers[0].VolumeMounts)

					require.NotEmpty(t, d.Spec.Template.Spec.InitContainers[0].VolumeMounts)
					assert.Equal(t, corev1.VolumeMount{
						Name:             "my-volume",
						ReadOnly:         false,
						MountPath:        "/my-volume",
						MountPropagation: ptr.To(corev1.MountPropagationHostToContainer),
					}, d.Spec.Template.Spec.InitContainers[0].VolumeMounts[0])
				},
			}
		},
	}

	// instead of unit testing the `createContainer` with a bit more complicated parameters
	// we're testing Deployment with a single Container
	base := DeploymentValues{
		Metadata: Metadata{
			Namespace:   "ns",
			Service:     "service",
			Component:   "component",
			Environment: "test",
		},
		Containers: []Container{
			{
				Name: "main",
				Image: Image{
					Repository: "image_repository",
					Tag:        ptr.To("image_tag"),
				},
			},
		},
	}

	for testName, makeConfig := range cases {
		t.Run(testName, func(t *testing.T) {
			values := DeploymentValues{}
			copier.CopyWithOption(&values, &base, copier.Option{DeepCopy: true})

			config := makeConfig()
			config.ValuesTransform(&values)

			_, create := CreateDeployment(values)
			resources, err := create(values)
			if err != nil {
				t.Errorf("errpr during test setup: %v", err)
			}
			deployment := fromUnstructuredOrPanic[*appsv1.Deployment](resources[0])

			config.Asserts(t, deployment)
		})
	}
}
