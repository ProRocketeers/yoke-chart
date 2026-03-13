package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNetworkPolicies(t *testing.T) {
	commonMetadata := Metadata{
		Namespace:   "ns",
		Service:     "service",
		Component:   "component",
		Environment: "test",
	}

	t.Run("renders a single NetworkPolicy", func(t *testing.T) {
		values := DeploymentValues{
			Metadata: commonMetadata,
			NetworkPolicies: map[string]networkingv1.NetworkPolicySpec{
				"deny-all": {
					PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress},
					PodSelector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "myapp",
						},
					},
				},
			},
		}

		shouldCreate, createFn := CreateNetworkPolicies(values)

		require.True(t, shouldCreate)

		resources, err := createFn(values)
		require.NoError(t, err)

		np := fromUnstructuredOrPanic[*networkingv1.NetworkPolicy](resources[0])

		assert.Equal(t, "service--component--test-deny-all", np.Name)
		assert.Subset(t, np.Spec.PolicyTypes, []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress})
		assert.Subset(t, np.Spec.PodSelector.MatchLabels, map[string]string{"app": "myapp"})
	})

	t.Run("renders multiple NetworkPolicies", func(t *testing.T) {
		values := DeploymentValues{
			Metadata: commonMetadata,
			NetworkPolicies: map[string]networkingv1.NetworkPolicySpec{
				"deny-all": {
					PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress},
					PodSelector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "myapp",
						},
					},
				},
				"allow-some": {
					PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
					PodSelector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "myapp",
						},
					},
				},
			},
		}

		shouldCreate, createFn := CreateNetworkPolicies(values)

		require.True(t, shouldCreate)

		resources, err := createFn(values)
		require.NoError(t, err)

		require.Len(t, resources, 2)

		findResourceOrFail[*networkingv1.NetworkPolicy](t, resources, "NetworkPolicy", "service--component--test-deny-all")
		findResourceOrFail[*networkingv1.NetworkPolicy](t, resources, "NetworkPolicy", "service--component--test-allow-some")
	})
}
