package resources

import (
	"testing"

	"github.com/ProRocketeers/yoke-chart/schema"
	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/ptr"
)

func TestRBAC(t *testing.T) {
	type CaseConfig struct {
		ValuesTransform func(*DeploymentValues)
		Asserts         func(*testing.T, []unstructured.Unstructured)
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
				Asserts: func(t *testing.T, r []unstructured.Unstructured) {
					require.Len(t, r, 2)

					roleName := "service--component--test--role"
					roleBindingName := "service--component--test--role-binding"

					role := findResourceOrFail[*rbacv1.Role](t, r, "Role", roleName)
					assert.Contains(t, role.Rules, rule)

					roleBinding := findResourceOrFail[*rbacv1.RoleBinding](t, r, "RoleBinding", roleBindingName)

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
				Asserts: func(t *testing.T, r []unstructured.Unstructured) {
					require.Len(t, r, 2)

					roleName := "service--component--test--cluster-role"
					roleBindingName := "service--component--test--cluster-role-binding"

					role := findResourceOrFail[*rbacv1.ClusterRole](t, r, "ClusterRole", roleName)
					assert.Contains(t, role.Rules, rule)

					roleBinding := findResourceOrFail[*rbacv1.ClusterRoleBinding](t, r, "ClusterRoleBinding", roleBindingName)

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
				Asserts: func(t *testing.T, r []unstructured.Unstructured) {
					require.Len(t, r, 2)

					findResourceOrFail[*rbacv1.ClusterRole](t, r, "ClusterRole", name)
					findResourceOrFail[*rbacv1.ClusterRoleBinding](t, r, "ClusterRoleBinding", name)
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
