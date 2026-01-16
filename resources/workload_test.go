package resources

import (
	"testing"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
)

func TestMainWorkload(t *testing.T) {
	commonMetadata := Metadata{
		Namespace:   "ns",
		Service:     "service",
		Component:   "component",
		Environment: "test",
	}

	t.Run("will render a Deployment if kind = Deployment", func(t *testing.T) {
		values := DeploymentValues{
			Metadata: commonMetadata,
			Kind:     "Deployment",
		}

		shouldCreate, createFn := CreateMainWorkload(values)

		require.True(t, shouldCreate)

		resources, err := createFn(values)
		require.NoError(t, err)

		findResourceOrFail[*appsv1.Deployment](t, resources, "Deployment", "service--component--test")
	})

	t.Run("will render a StatefulSet if kind = StatefulSet", func(t *testing.T) {
		values := DeploymentValues{
			Metadata:    commonMetadata,
			Kind:        "StatefulSet",
			StatefulSet: &appsv1.StatefulSetSpec{},
		}

		shouldCreate, createFn := CreateMainWorkload(values)

		require.True(t, shouldCreate)

		resources, err := createFn(values)
		require.NoError(t, err)

		findResourceOrFail[*appsv1.StatefulSet](t, resources, "StatefulSet", "service--component--test")
	})
}
