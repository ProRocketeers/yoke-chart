package resources

import (
	"testing"

	"github.com/ProRocketeers/yoke-chart/schema"
	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yokecd/yoke/pkg/flight"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/utils/ptr"
)

func TestRBAC(t *testing.T) {
	type CaseConfig struct {
		ValuesTransform func(*DeploymentValues)
		Asserts         func(*testing.T, []flight.Resource)
	}

	cases := map[string]func() CaseConfig{
		"can render additional Role + RoleBinding if specified": func() CaseConfig {
			rule := rbacv1.PolicyRule{
				APIGroups: []string{"v1"},
				Resources: []string{"pods"},
				Verbs:     []string{"get"},
			}

			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.ServiceAccount = &schema.ServiceAccount{
						AdditionalRole: &schema.ServiceAccountRole{
							Rules: []rbacv1.PolicyRule{rule},
						},
					}
				},
				Asserts: func(t *testing.T, r []flight.Resource) {
					require.Len(t, r, 2)

					roleName := "service--component--test--role"
					roleBindingName := "service--component--test--role-binding"

					role, ok := findResource[*rbacv1.Role](r, "Role", roleName)
					require.Truef(t, ok, "role %v not found", roleName)
					assert.Contains(t, role.Rules, rule)

					roleBinding, ok := findResource[*rbacv1.RoleBinding](r, "RoleBinding", roleBindingName)
					require.Truef(t, ok, "role binding %v not found", roleBindingName)

					assert.Equal(t, "service--component--test", roleBinding.Subjects[0].Name)
					assert.Equal(t, roleName, roleBinding.RoleRef.Name)
				},
			}
		},
		"can render additional ClusterRole + ClusterRoleBinding if specified": func() CaseConfig {
			rule := rbacv1.PolicyRule{
				APIGroups: []string{"v1"},
				Resources: []string{"pods"},
				Verbs:     []string{"get"},
			}

			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.ServiceAccount = &schema.ServiceAccount{
						AdditionalClusterRole: &schema.ServiceAccountRole{
							Rules: []rbacv1.PolicyRule{rule},
						},
					}
				},
				Asserts: func(t *testing.T, r []flight.Resource) {
					require.Len(t, r, 2)

					roleName := "service--component--test--cluster-role"
					roleBindingName := "service--component--test--cluster-role-binding"

					role, ok := findResource[*rbacv1.ClusterRole](r, "ClusterRole", roleName)
					require.Truef(t, ok, "cluster role %v not found", roleName)
					assert.Contains(t, role.Rules, rule)

					roleBinding, ok := findResource[*rbacv1.ClusterRoleBinding](r, "ClusterRoleBinding", roleBindingName)
					require.Truef(t, ok, "cluster role binding %v not found", roleBindingName)

					assert.Equal(t, "service--component--test", roleBinding.Subjects[0].Name)
					assert.Equal(t, roleName, roleBinding.RoleRef.Name)
				},
			}
		},
		"can override the name of the (Cluster)Role(Binding)": func() CaseConfig {
			rule := rbacv1.PolicyRule{
				APIGroups: []string{"v1"},
				Resources: []string{"pods"},
				Verbs:     []string{"get"},
			}
			name := "my-rbac"

			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.ServiceAccount = &schema.ServiceAccount{
						AdditionalClusterRole: &schema.ServiceAccountRole{
							Name:  ptr.To(name),
							Rules: []rbacv1.PolicyRule{rule},
						},
					}
				},
				Asserts: func(t *testing.T, r []flight.Resource) {
					require.Len(t, r, 2)

					_, ok := findResource[*rbacv1.ClusterRole](r, "ClusterRole", name)
					require.Truef(t, ok, "cluster role %v not found", name)
					_, ok = findResource[*rbacv1.ClusterRoleBinding](r, "ClusterRoleBinding", name)
					require.Truef(t, ok, "cluster role binding %v not found", name)
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
	}

	for testName, makeConfig := range cases {
		t.Run(testName, func(t *testing.T) {
			values := DeploymentValues{}
			copier.CopyWithOption(&values, &base, copier.Option{DeepCopy: true})

			config := makeConfig()
			config.ValuesTransform(&values)

			_, create := CreateRBAC(values)
			resources, err := create(values)
			if err != nil {
				t.Errorf("error during test setup: %v", err)
			}

			config.Asserts(t, resources)
		})
	}

	t.Run("should not create any RBAC if not specified", func(t *testing.T) {
		values := DeploymentValues{}
		copier.CopyWithOption(&values, &base, copier.Option{DeepCopy: true})
		values.ServiceAccount = &schema.ServiceAccount{
			Annotations: map[string]string{
				"some": "annotation",
			},
		}

		shouldCreate, _ := CreateRBAC(values)
		assert.False(t, shouldCreate)
	})
}
