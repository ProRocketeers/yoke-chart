package resources

import (
	"fmt"

	"github.com/ProRocketeers/yoke-chart/schema"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

func createPodSpec(podValuesExtractor PodValuesExtractor, values DeploymentValues) (corev1.PodSpec, error) {
	podValues := podValuesExtractor.GetPodValues()
	initContainers, containers := []corev1.Container{}, []corev1.Container{}
	for _, initContainer := range podValues.InitContainers {
		if container, err := createContainer(initContainer, podValues); err != nil {
			return corev1.PodSpec{}, fmt.Errorf("creating init container '%v': %v", initContainer.Name, err)
		} else {
			initContainers = append(initContainers, container)
		}
	}
	for _, containerInput := range podValues.Containers {
		if container, err := createContainer(containerInput, podValues); err != nil {
			return corev1.PodSpec{}, fmt.Errorf("creating container '%v': %v", containerInput.Name, err)
		} else {
			containers = append(containers, container)
		}
	}
	volumes, err := prepareVolumes(podValues.Volumes, values.Metadata)
	if err != nil {
		return corev1.PodSpec{}, fmt.Errorf("preparing pod volumes: %v", err)
	}
	podSpec := corev1.PodSpec{
		ImagePullSecrets:   podValues.ImagePullSecrets,
		InitContainers:     initContainers,
		ServiceAccountName: serviceName(values.Metadata),
		Containers:         containers,
		NodeSelector:       values.NodeSelector,
		Affinity:           values.Affinity,
		Tolerations:        values.Tolerations,
		Volumes:            volumes,
	}
	return podSpec, nil
}

func prepareVolumes(volumes map[string]schema.Volume, metadata Metadata) ([]corev1.Volume, error) {
	volumesRet := []corev1.Volume{}

	for volumeName, volumeInput := range sortedMap(volumes) {
		source := corev1.VolumeSource{}

		switch v := volumeInput.Variant.(type) {
		case schema.SecretVolume:
			source.Secret = &corev1.SecretVolumeSource{
				SecretName:  v.SecretName,
				DefaultMode: ptr.To(int32(0444)),
			}
			if v.Mode != nil {
				source.Secret.DefaultMode = v.Mode
			}
			items := []corev1.KeyToPath{}
			for filePath, secretKey := range sortedMap(v.Items) {
				item := corev1.KeyToPath{
					Path: filePath,
				}
				if secretKey != nil {
					item.Key = *secretKey
				} else {
					item.Key = filePath
				}
				items = append(items, item)
			}
			source.Secret.Items = items
		case schema.ConfigMapVolume:
			source.ConfigMap = &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: v.ConfigMapName,
				},
				DefaultMode: ptr.To(int32(0444)),
			}
			if v.Mode != nil {
				source.ConfigMap.DefaultMode = v.Mode
			}
			items := []corev1.KeyToPath{}
			for filePath, secretKey := range sortedMap(v.Items) {
				item := corev1.KeyToPath{
					Path: filePath,
				}
				if secretKey != nil {
					item.Key = *secretKey
				} else {
					item.Key = filePath
				}
				items = append(items, item)
			}
			source.ConfigMap.Items = items
		case schema.RawVolume:
			source = v.Spec
		case schema.PersistentVolume:
			var claimName string
			if *v.Existing {
				claimName = v.Variant.(schema.PersistentVolumeExisting).PvcName
			} else {
				claimName = pvcName(volumeName, metadata)
			}
			source.PersistentVolumeClaim = &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: claimName,
			}
		case schema.StandardVolume:
			source.EmptyDir = &corev1.EmptyDirVolumeSource{}
			if volumeInput.Type == schema.VolumeTypeStandardTmpfs {
				source.EmptyDir.Medium = corev1.StorageMediumMemory
			} else {
				source.EmptyDir.Medium = corev1.StorageMediumDefault
			}
		default:
			return []corev1.Volume{}, fmt.Errorf("unknown volume type '%v' in volume '%v'", volumeInput.Type, volumeName)
		}

		volume := corev1.Volume{
			Name:         volumeName,
			VolumeSource: source,
		}
		volumesRet = append(volumesRet, volume)
	}
	return volumesRet, nil
}
