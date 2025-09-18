package resources

import (
	"iter"

	"github.com/ProRocketeers/yoke-chart/schema"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/ptr"
)

func CreatePVCs(values DeploymentValues) (bool, ResourceCreator) {
	return hasAnyPVC(values), func(values DeploymentValues) ([]unstructured.Unstructured, error) {
		resources := []unstructured.Unstructured{}
		for volumeName, volume := range newPersistentVolumes(values) {
			v := volume.Variant.(schema.PersistentVolume).Variant.(schema.PersistentVolumeNew)
			// already validated during unmarshaling, ignoring error
			sizeQuantity, _ := resource.ParseQuantity(v.Size)

			pvc := corev1.PersistentVolumeClaim{
				TypeMeta: metav1.TypeMeta{
					APIVersion: corev1.SchemeGroupVersion.Identifier(),
					Kind:       "PersistentVolumeClaim",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      pvcName(volumeName, values.Metadata),
					Namespace: values.Metadata.Namespace,
					Labels:    commonLabels(values.Metadata),
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					VolumeMode:  ptr.To(corev1.PersistentVolumeFilesystem),
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: sizeQuantity,
						},
					},
					StorageClassName: &v.StorageClassName,
				},
			}

			if len(v.AccessModes) > 0 {
				pvc.Spec.AccessModes = v.AccessModes
			}
			if v.VolumeMode != nil {
				pvc.Spec.VolumeMode = v.VolumeMode
			}

			u, err := toUnstructured(&pvc)
			if err != nil {
				return []unstructured.Unstructured{}, err
			}
			resources = append(resources, u...)
		}
		return resources, nil
	}
}

func hasAnyPVC(values DeploymentValues) bool {
	create := false
	// use the iterator *once* (just to check if we should even render anything, at least 1 PVC)
	newPersistentVolumes(values)(func(string, schema.Volume) bool {
		// stop after first yield
		create = true
		return false
	})
	return create
}

func shouldCreatePVC(volume schema.Volume) bool {
	if volume.Type == schema.VolumeTypePersistent {
		v := volume.Variant.(schema.PersistentVolume)
		if !*v.Existing {
			return true
		}
	}
	return false
}

func newPersistentVolumes(values DeploymentValues) iter.Seq2[string, schema.Volume] {
	return func(yield func(string, schema.Volume) bool) {
		// check all places from where we might want to create persistent volumes
		for volumeName, volume := range sortedMap(values.Volumes) {
			if shouldCreatePVC(volume) {
				if !yield(volumeName, volume) {
					return
				}
			}
		}
		if values.PreDeploymentJob != nil {
			for volumeName, volume := range sortedMap(values.PreDeploymentJob.Volumes) {
				if shouldCreatePVC(volume) {
					if !yield(volumeName, volume) {
						return
					}
				}
			}
		}
		for _, cronjob := range values.Cronjobs {
			for volumeName, volume := range sortedMap(cronjob.Volumes) {
				if shouldCreatePVC(volume) {
					if !yield(volumeName, volume) {
						return
					}
				}
			}
		}
	}
}
