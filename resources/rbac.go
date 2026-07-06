package resources

import (
	"fmt"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateRBAC(values DeploymentValues) (bool, ResourceCreator) {
	create := values.ServiceAccount != nil && (values.ServiceAccount.AdditionalRole != nil || values.ServiceAccount.AdditionalClusterRole != nil)
	return create, func(values DeploymentValues) ([]NamedResource, error) {
		resources := []NamedResource{}
		sa := values.ServiceAccount

		if sa.AdditionalRole != nil {
			roleName := fmt.Sprintf("%s--role", serviceName(values.Metadata))
			roleBindingName := fmt.Sprintf("%s--role-binding", serviceName(values.Metadata))
			if sa.AdditionalRole.Name != nil {
				roleName = *sa.AdditionalRole.Name
				roleBindingName = *sa.AdditionalRole.Name
			}
			role := rbacv1.Role{
				TypeMeta: metav1.TypeMeta{
					APIVersion: rbacv1.SchemeGroupVersion.Identifier(),
					Kind:       "Role",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      roleName,
					Namespace: values.Metadata.Namespace,
				},
				Rules: sa.AdditionalRole.Rules,
			}
			roleBinding := rbacv1.RoleBinding{
				TypeMeta: metav1.TypeMeta{
					APIVersion: rbacv1.SchemeGroupVersion.Identifier(),
					Kind:       "RoleBinding",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      roleBindingName,
					Namespace: values.Metadata.Namespace,
				},
				Subjects: []rbacv1.Subject{
					{
						Kind:      "ServiceAccount",
						Name:      serviceName(values.Metadata),
						Namespace: values.Metadata.Namespace,
					},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: rbacv1.SchemeGroupVersion.Group,
					Kind:     "Role",
					Name:     roleName,
				},
			}
			u, err := toUnstructured(&role, &roleBinding)
			if err != nil {
				return nil, err
			}
			resources = append(resources,
				NamedResource{Category: CategoryRole, Object: u[0]},
				NamedResource{Category: CategoryRoleBinding, Object: u[1]},
			)
		}

		if sa.AdditionalClusterRole != nil {
			roleName := fmt.Sprintf("%s--cluster-role", serviceName(values.Metadata))
			roleBindingName := fmt.Sprintf("%s--cluster-role-binding", serviceName(values.Metadata))
			if sa.AdditionalClusterRole.Name != nil {
				roleName = *sa.AdditionalClusterRole.Name
				roleBindingName = *sa.AdditionalClusterRole.Name
			}
			role := rbacv1.ClusterRole{
				TypeMeta: metav1.TypeMeta{
					APIVersion: rbacv1.SchemeGroupVersion.Identifier(),
					Kind:       "ClusterRole",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: roleName,
				},
				Rules: sa.AdditionalClusterRole.Rules,
			}
			roleBinding := rbacv1.ClusterRoleBinding{
				TypeMeta: metav1.TypeMeta{
					APIVersion: rbacv1.SchemeGroupVersion.Identifier(),
					Kind:       "ClusterRoleBinding",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: roleBindingName,
				},
				Subjects: []rbacv1.Subject{
					{
						Kind:      "ServiceAccount",
						Name:      serviceName(values.Metadata),
						Namespace: values.Metadata.Namespace,
					},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: rbacv1.SchemeGroupVersion.Group,
					Kind:     "ClusterRole",
					Name:     roleName,
				},
			}
			u, err := toUnstructured(&role, &roleBinding)
			if err != nil {
				return nil, err
			}
			resources = append(resources,
				NamedResource{Category: CategoryClusterRole, Object: u[0]},
				NamedResource{Category: CategoryClusterRoleBinding, Object: u[1]},
			)
		}

		return resources, nil
	}
}
