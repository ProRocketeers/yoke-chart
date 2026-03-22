package resources

import (
	"testing"

	"github.com/ProRocketeers/yoke-chart/schema"
	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

func TestSetup(t *testing.T) {
	type CaseConfig struct {
		ValuesTransform func(*schema.InputValues)
		Asserts         func(*testing.T, DeploymentValues, error)
	}

	cases := map[string]func() CaseConfig{
		"uses default replica count = 1": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(iv *schema.InputValues) {},
				Asserts: func(t *testing.T, dv DeploymentValues, err error) {
					require.Nil(t, err)
					require.NotZero(t, dv)

					assert.Equal(t, 1, dv.ReplicaCount)
				},
			}
		},
		"uses default Kind = Deployment": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(iv *schema.InputValues) {},
				Asserts: func(t *testing.T, dv DeploymentValues, err error) {
					require.Nil(t, err)
					require.NotZero(t, dv)

					assert.Equal(t, "Deployment", dv.Kind)
				},
			}
		},
		"uses default ServiceType = ClusterIP": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(iv *schema.InputValues) {},
				Asserts: func(t *testing.T, dv DeploymentValues, err error) {
					require.Nil(t, err)
					require.NotZero(t, dv)

					assert.Equal(t, corev1.ServiceTypeClusterIP, dv.ServiceType)
				},
			}
		},
		"can override ServiceType": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(iv *schema.InputValues) {
					iv.ServiceType = ptr.To(corev1.ServiceTypeNodePort)
				},
				Asserts: func(t *testing.T, dv DeploymentValues, err error) {
					require.Nil(t, err)
					require.NotZero(t, dv)

					assert.Equal(t, corev1.ServiceTypeNodePort, dv.ServiceType)
				},
			}
		},
		"automatically overrides ServiceType if node port is specified on any port and service type is ClusterIP": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(iv *schema.InputValues) {
					iv.Container.Ports = []schema.Port{
						{
							Port:     80,
							NodePort: ptr.To(int32(31000)),
						},
					}
				},
				Asserts: func(t *testing.T, dv DeploymentValues, err error) {
					require.Nil(t, err)
					require.NotZero(t, dv)

					assert.Equal(t, corev1.ServiceTypeNodePort, dv.ServiceType)
				},
			}
		},
		"does not override ServiceType if node port is specified on any port and service type is not ClusterIP": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(iv *schema.InputValues) {
					iv.ServiceType = ptr.To(corev1.ServiceTypeLoadBalancer)
					iv.Container.Ports = []schema.Port{
						{
							Port:     80,
							NodePort: ptr.To(int32(31000)),
						},
					}
				},
				Asserts: func(t *testing.T, dv DeploymentValues, err error) {
					require.Nil(t, err)
					require.NotZero(t, dv)

					assert.Equal(t, corev1.ServiceTypeLoadBalancer, dv.ServiceType)
				},
			}
		},
		"can override default replica count": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(iv *schema.InputValues) {},
				Asserts: func(t *testing.T, dv DeploymentValues, err error) {
					require.Nil(t, err)
					require.NotZero(t, dv)

					assert.Equal(t, 1, dv.ReplicaCount)
				},
			}
		},
		"main container - must have image tag": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(iv *schema.InputValues) {
					iv.Container.Image.Tag = nil
				},
				Asserts: func(t *testing.T, dv DeploymentValues, err error) {
					// not validating the error message as that can change
					assert.NotNil(t, err)
				},
			}
		},
		"main container - must have at least 1 port": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(iv *schema.InputValues) {
					iv.Container.Ports = []schema.Port{}
				},
				Asserts: func(t *testing.T, dv DeploymentValues, err error) {
					assert.NotNil(t, err)
				},
			}
		},
		"main container - allows to override container name": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(iv *schema.InputValues) {
					iv.MainContainerName = ptr.To("app")
				},
				Asserts: func(t *testing.T, dv DeploymentValues, err error) {
					assert.Equal(t, "app", dv.Containers[0].Name)
				},
			}
		},
		"sidecar - must have either image tag, or inherit the main container tag": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(iv *schema.InputValues) {
					iv.Sidecars = map[string]schema.Container{
						"side": {
							Image: schema.Image{
								Repository: "sidecar_repository",
							},
						},
					}
				},
				Asserts: func(t *testing.T, dv DeploymentValues, err error) {
					assert.NotNil(t, err)
				},
			}
		},
		"sidecar - can inherit main container tag": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(iv *schema.InputValues) {
					iv.Sidecars = map[string]schema.Container{
						"side": {
							Image: schema.Image{
								Repository:              "sidecar_repository",
								InheritMainContainerTag: ptr.To(true),
							},
						},
					}
				},
				Asserts: func(t *testing.T, dv DeploymentValues, err error) {
					assert.Equal(t, ptr.To("image_tag"), dv.Containers[1].Image.Tag)
				},
			}
		},
		"init container - must have either image tag, or inherit the main container tag": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(iv *schema.InputValues) {
					iv.InitContainers = []schema.InitContainer{
						{
							Name: "init",
							Container: schema.Container{
								Image: schema.Image{
									Repository: "sidecar_repository",
								},
							},
						},
					}
				},
				Asserts: func(t *testing.T, dv DeploymentValues, err error) {
					assert.NotNil(t, err)
				},
			}
		},
		"init container - can inherit main container tag": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(iv *schema.InputValues) {
					iv.InitContainers = []schema.InitContainer{
						{
							Name: "init",
							Container: schema.Container{
								Image: schema.Image{
									Repository:              "sidecar_repository",
									InheritMainContainerTag: ptr.To(true),
								},
							},
						},
					}
				},
				Asserts: func(t *testing.T, dv DeploymentValues, err error) {
					assert.Equal(t, ptr.To("image_tag"), dv.InitContainers[0].Image.Tag)
				},
			}
		},
		"PDJ main container - must have either image tag, or inherit the main container tag": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(iv *schema.InputValues) {
					iv.PreDeploymentJob = &schema.PreDeploymentJob{
						Container: schema.Container{
							Image: schema.Image{
								Repository: "pdj_repository",
							},
						},
					}
				},
				Asserts: func(t *testing.T, dv DeploymentValues, err error) {
					assert.NotNil(t, err)
				},
			}
		},
		"PDJ main container - can inherit main container tag": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(iv *schema.InputValues) {
					iv.PreDeploymentJob = &schema.PreDeploymentJob{
						Container: schema.Container{
							Image: schema.Image{
								Repository:              "pdj_repository",
								InheritMainContainerTag: ptr.To(true),
							},
						},
					}
				},
				Asserts: func(t *testing.T, dv DeploymentValues, err error) {
					assert.Nil(t, err)
					assert.Equal(t, ptr.To("image_tag"), dv.PreDeploymentJob.Container.Image.Tag)
				},
			}
		},
		"PDJ main container - allows to override container name": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(iv *schema.InputValues) {
					iv.PreDeploymentJob = &schema.PreDeploymentJob{
						MainContainerName: ptr.To("app"),
						Container: schema.Container{
							Image: schema.Image{
								Repository:              "pdj_repository",
								InheritMainContainerTag: ptr.To(true),
							},
						},
					}
				},
				Asserts: func(t *testing.T, dv DeploymentValues, err error) {
					assert.Equal(t, "app", dv.PreDeploymentJob.Container.Name)
				},
			}
		},
		"PDJ init container - must have either image tag, or inherit the main container tag": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(iv *schema.InputValues) {
					iv.PreDeploymentJob = &schema.PreDeploymentJob{
						Container: schema.Container{
							Image: schema.Image{
								Repository:              "pdj_repository",
								InheritMainContainerTag: ptr.To(true),
							},
						},
						InitContainers: []schema.InitContainer{
							{
								Name: "init",
								Container: schema.Container{
									Image: schema.Image{
										Repository: "sidecar_repository",
									},
								},
							},
						},
					}
				},
				Asserts: func(t *testing.T, dv DeploymentValues, err error) {
					assert.NotNil(t, err)
				},
			}
		},
		"PDJ init container - can inherit main container tag": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(iv *schema.InputValues) {
					iv.PreDeploymentJob = &schema.PreDeploymentJob{
						Container: schema.Container{
							Image: schema.Image{
								Repository:              "pdj_repository",
								InheritMainContainerTag: ptr.To(true),
							},
						},
						InitContainers: []schema.InitContainer{
							{
								Name: "init",
								Container: schema.Container{
									Image: schema.Image{
										Repository:              "sidecar_repository",
										InheritMainContainerTag: ptr.To(true),
									},
								},
							},
						},
					}
				},
				Asserts: func(t *testing.T, dv DeploymentValues, err error) {
					assert.Equal(t, ptr.To("image_tag"), dv.PreDeploymentJob.InitContainers[0].Image.Tag)
				},
			}
		},
		"cronjob main container - must have either image tag, or inherit the main container tag": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(iv *schema.InputValues) {
					iv.Cronjobs = []schema.Cronjob{
						{
							Name:     "cronjob",
							Schedule: "* * * * *",
							Container: schema.Container{
								Image: schema.Image{
									Repository: "pdj_repository",
								},
							},
						},
					}
				},
				Asserts: func(t *testing.T, dv DeploymentValues, err error) {
					assert.NotNil(t, err)
				},
			}
		},
		"cronjob main container - can inherit main container tag": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(iv *schema.InputValues) {
					iv.Cronjobs = []schema.Cronjob{
						{
							Name:     "cronjob",
							Schedule: "* * * * *",
							Container: schema.Container{
								Image: schema.Image{
									Repository:              "pdj_repository",
									InheritMainContainerTag: ptr.To(true),
								},
							},
						},
					}
				},
				Asserts: func(t *testing.T, dv DeploymentValues, err error) {
					assert.Nil(t, err)
					assert.Equal(t, ptr.To("image_tag"), dv.Cronjobs[0].Container.Image.Tag)
				},
			}
		},
		"cronjob main container - allows to override container name": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(iv *schema.InputValues) {
					iv.Cronjobs = []schema.Cronjob{
						{
							Name:              "cronjob",
							Schedule:          "* * * * *",
							MainContainerName: ptr.To("app"),
							Container: schema.Container{
								Image: schema.Image{
									Repository:              "pdj_repository",
									InheritMainContainerTag: ptr.To(true),
								},
							},
						},
					}
				},
				Asserts: func(t *testing.T, dv DeploymentValues, err error) {
					assert.Nil(t, err)
					assert.Equal(t, "app", dv.Cronjobs[0].Container.Name)
				},
			}
		},
		"cronjob init container - must have either image tag, or inherit the main container tag": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(iv *schema.InputValues) {
					iv.Cronjobs = []schema.Cronjob{
						{
							Name:     "cronjob",
							Schedule: "* * * * *",
							Container: schema.Container{
								Image: schema.Image{
									Repository:              "pdj_repository",
									InheritMainContainerTag: ptr.To(true),
								},
							},
							InitContainers: []schema.InitContainer{
								{
									Name: "init",
									Container: schema.Container{
										Image: schema.Image{
											Repository: "init_repository",
										},
									},
								},
							},
						},
					}
				},
				Asserts: func(t *testing.T, dv DeploymentValues, err error) {
					assert.NotNil(t, err)
				},
			}
		},
		"cronjob init container - can inherit main container tag": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(iv *schema.InputValues) {
					iv.Cronjobs = []schema.Cronjob{
						{
							Name:     "cronjob",
							Schedule: "* * * * *",
							Container: schema.Container{
								Image: schema.Image{
									Repository:              "pdj_repository",
									InheritMainContainerTag: ptr.To(true),
								},
							},
							InitContainers: []schema.InitContainer{
								{
									Name: "init",
									Container: schema.Container{
										Image: schema.Image{
											Repository:              "init_repository",
											InheritMainContainerTag: ptr.To(true),
										},
									},
								},
							},
						},
					}
				},
				Asserts: func(t *testing.T, dv DeploymentValues, err error) {
					assert.Nil(t, err)
					assert.Equal(t, ptr.To("image_tag"), dv.Cronjobs[0].InitContainers[0].Image.Tag)
				},
			}
		},
	}

	base := schema.InputValues{
		Metadata: schema.Metadata{
			Namespace:   "ns",
			Service:     "service",
			Component:   "component",
			Environment: "test",
		},
		Container: schema.Container{
			Image: schema.Image{
				Repository: "image_repository",
				Tag:        ptr.To("image_tag"),
			},
			Ports: []schema.Port{{Port: 8080}},
		},
	}

	for testName, makeConfig := range cases {
		t.Run(testName, func(t *testing.T) {
			values := schema.InputValues{}
			copier.CopyWithOption(&values, &base, copier.Option{DeepCopy: true})

			config := makeConfig()
			config.ValuesTransform(&values)

			deploymentValues, err := PrepareDeploymentValues(values)
			config.Asserts(t, deploymentValues, err)
		})
	}
}
