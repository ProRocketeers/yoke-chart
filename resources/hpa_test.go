package resources

import (
	"testing"

	"github.com/ProRocketeers/yoke-chart/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"k8s.io/utils/ptr"
)

func TestHPA(t *testing.T) {
	commonMetadata := Metadata{
		Namespace:   "ns",
		Service:     "service",
		Component:   "component",
		Environment: "test",
	}

	baseAutoscaling := &schema.HorizontalPodAutoscaler{
		MaxReplicas: 5,
	}

	t.Run("targets a Deployment by default", func(t *testing.T) {
		values := DeploymentValues{
			Metadata:    commonMetadata,
			Kind:        "Deployment",
			Autoscaling: baseAutoscaling,
		}

		shouldCreate, createFn := CreateHPA(values)
		require.True(t, shouldCreate)

		resources, err := createFn(values)
		require.NoError(t, err)

		hpa := fromUnstructuredOrPanic[*autoscalingv2.HorizontalPodAutoscaler](resources[0])
		assert.Equal(t, "Deployment", hpa.Spec.ScaleTargetRef.Kind)
		assert.Equal(t, "service--component--test", hpa.Spec.ScaleTargetRef.Name)
	})

	t.Run("targets a StatefulSet when the workload kind is StatefulSet", func(t *testing.T) {
		values := DeploymentValues{
			Metadata:    commonMetadata,
			Kind:        "StatefulSet",
			Autoscaling: baseAutoscaling,
		}

		shouldCreate, createFn := CreateHPA(values)
		require.True(t, shouldCreate)

		resources, err := createFn(values)
		require.NoError(t, err)

		hpa := fromUnstructuredOrPanic[*autoscalingv2.HorizontalPodAutoscaler](resources[0])
		assert.Equal(t, "StatefulSet", hpa.Spec.ScaleTargetRef.Kind)
		assert.Equal(t, "service--component--test", hpa.Spec.ScaleTargetRef.Name)
	})

	t.Run("fails when maxReplicas is lower than minReplicas", func(t *testing.T) {
		values := DeploymentValues{
			Metadata: commonMetadata,
			Kind:     "Deployment",
			Autoscaling: &schema.HorizontalPodAutoscaler{
				MinReplicas: ptr.To(int32(10)),
				MaxReplicas: 5,
			},
		}

		_, createFn := CreateHPA(values)
		_, err := createFn(values)
		require.Error(t, err)
	})
}
