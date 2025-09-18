package resources

import (
	"fmt"

	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func CreatePDB(values DeploymentValues) (bool, ResourceCreator) {
	return values.PodDisruptionBudget != nil, func(values DeploymentValues) ([]unstructured.Unstructured, error) {
		spec := values.PodDisruptionBudget
		if spec.MinAvailable != nil && spec.MaxUnavailable != nil {
			return []unstructured.Unstructured{}, fmt.Errorf("you cannot specify both 'minAvailable' and 'maxUnavailable' in a PodDisruptionBudget")
		}
		pdb := policyv1.PodDisruptionBudget{
			TypeMeta: metav1.TypeMeta{
				APIVersion: policyv1.SchemeGroupVersion.Identifier(),
				Kind:       "PodDisruptionBudget",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceName(values.Metadata),
				Namespace: values.Metadata.Namespace,
				Labels:    commonLabels(values.Metadata),
			},
			Spec: *spec,
		}
		pdb.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"app": serviceName(values.Metadata),
			},
		}
		u, err := toUnstructured(&pdb)
		if err != nil {
			return []unstructured.Unstructured{}, err
		}
		return u, nil
	}
}
