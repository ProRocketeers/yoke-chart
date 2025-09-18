package resources

import (
	"fmt"

	"github.com/yokecd/yoke/pkg/flight"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreatePDB(values DeploymentValues) (bool, ResourceCreator) {
	return values.PodDisruptionBudget != nil, func(values DeploymentValues) ([]flight.Resource, error) {
		spec := values.PodDisruptionBudget
		if spec.MinAvailable != nil && spec.MaxUnavailable != nil {
			return []flight.Resource{}, fmt.Errorf("you cannot specify both 'minAvailable' and 'maxUnavailable' in a PodDisruptionBudget")
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
		return []flight.Resource{&pdb}, nil
	}
}
