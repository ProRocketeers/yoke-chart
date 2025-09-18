package resources

import (
	"testing"

	"github.com/ProRocketeers/yoke-chart/resources/postgresql"
	"github.com/ProRocketeers/yoke-chart/schema"
	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/assert"
	"k8s.io/utils/ptr"
)

func TestDB(t *testing.T) {
	type CaseConfig struct {
		ValuesTransform func(*DeploymentValues)
		Asserts         func(*testing.T, *postgresql.Postgresql)
	}

	cases := map[string]func() CaseConfig{
		"should render minimal required values properly": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {},
				Asserts: func(t *testing.T, p *postgresql.Postgresql) {
					assert.Equal(t, "service-db", p.Name)
					assert.Equal(t, "ns", p.Spec.TeamID)
					assert.Equal(t, "15", p.Spec.PostgresqlParam.PgVersion)
					assert.Equal(t, int32(1), p.Spec.NumberOfInstances)
					assert.Equal(t, postgresql.Volume{
						Size:         "5Gi",
						StorageClass: "my-sc",
					}, p.Spec.Volume)
					assert.Equal(t, false, p.Spec.EnableLogicalBackup)
					assert.Equal(t, map[string]postgresql.UserFlags{
						"owner": {"superuser", "createdb"},
					}, p.Spec.Users)
					assert.Equal(t, map[string]string{
						"main": "owner",
					}, p.Spec.Databases)
					assert.Equal(t, &postgresql.Resources{
						ResourceRequests: postgresql.ResourceDescription{
							CPU:    ptr.To("100m"),
							Memory: ptr.To("100Mi"),
						},
						ResourceLimits: postgresql.ResourceDescription{
							CPU:    ptr.To("1"),
							Memory: ptr.To("500Mi"),
						},
					}, p.Spec.Resources)
				},
			}
		},
		"allows to enable backups": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.DB.Backup = ptr.To(true)
				},
				Asserts: func(t *testing.T, p *postgresql.Postgresql) {
					assert.Equal(t, true, p.Spec.EnableLogicalBackup)
				},
			}
		},
		"allows additional config to be passed": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.DB.AdditionalConfig = &postgresql.PostgresSpec{
						Resources: &postgresql.Resources{
							ResourceRequests: postgresql.ResourceDescription{
								CPU:    ptr.To("200m"),
								Memory: ptr.To("200Mi"),
							},
							ResourceLimits: postgresql.ResourceDescription{
								CPU:    ptr.To("2"),
								Memory: ptr.To("1Gi"),
							},
						},
						DockerImage: "my-docker-image:latest",
					}
				},
				Asserts: func(t *testing.T, p *postgresql.Postgresql) {
					assert.Equal(t, &postgresql.Resources{
						ResourceRequests: postgresql.ResourceDescription{
							CPU:    ptr.To("200m"),
							Memory: ptr.To("200Mi"),
						},
						ResourceLimits: postgresql.ResourceDescription{
							CPU:    ptr.To("2"),
							Memory: ptr.To("1Gi"),
						},
					}, p.Spec.Resources)
					assert.Equal(t, "my-docker-image:latest", p.Spec.DockerImage)
				},
			}
		},
	}

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
		DB: &schema.Database{
			Enabled:      ptr.To(true),
			ClusterName:  "service-db",
			Replicas:     1,
			Version:      15,
			StorageClass: "my-sc",
			Size:         "5Gi",
			Users: map[string]postgresql.UserFlags{
				"owner": {"superuser", "createdb"},
			},
			Databases: map[string]string{
				"main": "owner",
			},
		},
	}

	for testName, makeConfig := range cases {
		t.Run(testName, func(t *testing.T) {
			values := DeploymentValues{}
			copier.CopyWithOption(&values, &base, copier.Option{DeepCopy: true})

			config := makeConfig()
			config.ValuesTransform(&values)

			_, create := CreateDB(values)
			resources, err := create(values)
			if err != nil {
				t.Errorf("error during test setup: %v", err)
			}
			db := resources[0].(*postgresql.Postgresql)

			config.Asserts(t, db)
		})
	}

	t.Run("should not create a DB if not specified", func(t *testing.T) {
		values := DeploymentValues{}
		copier.CopyWithOption(&values, &base, copier.Option{DeepCopy: true})
		values.DB = nil

		shouldCreate, _ := CreateDB(values)
		assert.False(t, shouldCreate)
	})
	t.Run("should not render manifest if 'db' is specified but disabled", func(t *testing.T) {
		values := DeploymentValues{}
		copier.CopyWithOption(&values, &base, copier.Option{DeepCopy: true})
		values.DB.Enabled = ptr.To(false)

		shouldCreate, _ := CreateDB(values)
		assert.False(t, shouldCreate)
	})

}
