package resources

import (
	"fmt"

	"github.com/ProRocketeers/yoke-chart/schema"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

func createContainer(c Container, podValues PodValues) (corev1.Container, error) {
	envs, envFrom := getEnvs(c, podValues.Metadata)
	container := corev1.Container{
		Name:            c.Name,
		Image:           fmt.Sprintf("%v:%v", c.Image.Repository, *c.Image.Tag),
		ImagePullPolicy: corev1.PullIfNotPresent,
		Args:            c.Args,
		Command:         c.Command,
		EnvFrom:         envFrom,
		Env:             envs,
		Ports:           getPorts(c),
		Lifecycle:       c.Lifecycle,
		ReadinessProbe:  c.ReadinessProbe,
		LivenessProbe:   c.LivenessProbe,
		VolumeMounts:    getVolumeMounts(c, podValues),
	}
	if c.Image.PullPolicy != nil {
		container.ImagePullPolicy = *c.Image.PullPolicy
	}
	if c.Resources != nil {
		container.Resources = *c.Resources
	}
	return container, nil
}

func getEnvs(c Container, metadata Metadata) ([]corev1.EnvVar, []corev1.EnvFromSource) {
	envs := []corev1.EnvVar{}
	for name, value := range sortedMap(c.Envs) {
		envs = append(envs, corev1.EnvVar{
			Name:  name,
			Value: value,
		})
	}
	envsFrom := []corev1.EnvFromSource{}
	for secretName, secretMapping := range sortedMap(c.KubeSecrets) {
		// when mounting the whole secret
		if secretMapping == nil {
			envsFrom = append(envsFrom, corev1.EnvFromSource{
				SecretRef: &corev1.SecretEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: secretName,
					},
				},
			})
		} else {
			for envName, secretKey := range sortedMap(secretMapping) {
				env := corev1.EnvVar{
					Name: envName,
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: secretName,
							},
							Key: envName,
						},
					},
				}
				if secretKey != nil {
					env.ValueFrom.SecretKeyRef.Key = *secretKey
				}
				envs = append(envs, env)
			}
		}
	}
	for secretPath := range sortedMap(c.VaultSecrets) {
		envsFrom = append(envsFrom, corev1.EnvFromSource{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: vaultSecretName(secretPath, metadata),
				},
			},
		})
	}
	envs = append(envs, c.EnvsRaw...)
	return envs, envsFrom
}

func getPorts(c Container) []corev1.ContainerPort {
	ports := []corev1.ContainerPort{}
	for _, port := range c.Ports {
		p := corev1.ContainerPort{
			ContainerPort: int32(port.Port),
		}
		if port.ContainerPort != nil {
			p.ContainerPort = int32(*port.ContainerPort)
		}
		if port.Name != nil {
			p.Name = *port.Name
		}
		ports = append(ports, p)
	}
	return ports
}

func getVolumeMounts(c Container, podValues PodValues) []corev1.VolumeMount {
	mounts := []corev1.VolumeMount{}

	for volumeName, volumeInput := range sortedMap(podValues.Volumes) {
		if mount, ok := volumeInput.Mounts[c.Name]; ok {
			m := corev1.VolumeMount{
				Name:      volumeName,
				MountPath: mount.ContainerPath,
			}

			isReadonly := volumeInput.Type == schema.VolumeTypeSecret || volumeInput.Type == schema.VolumeTypeConfigMap

			if isReadonly {
				m.ReadOnly = true
			} else {
				m.ReadOnly = false
				m.MountPropagation = ptr.To(corev1.MountPropagationHostToContainer)
				if mount.VolumePath != nil {
					m.SubPath = *mount.VolumePath
				}
			}
			mounts = append(mounts, m)
		}
	}
	return mounts
}
