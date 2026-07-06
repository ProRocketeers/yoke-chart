package resources

import (
	"fmt"
	"strings"

	"github.com/ProRocketeers/yoke-chart/schema"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/ptr"
)

func PrepareDeploymentValues(input schema.InputValues) (DeploymentValues, error) {
	values := DeploymentValues{
		ReplicaCount:        1,
		Autoscaling:         input.Autoscaling,
		Strategy:            input.Strategy,
		PodDisruptionBudget: input.PodDisruptionBudget,
		Ingress:             input.Ingress,
		HTTPRoutes:          resolveHttpRoutes(input),
		NetworkPolicies:     input.NetworkPolicies,
		Volumes:             input.Volumes,
		ServiceAccount:      input.ServiceAccount,
		Service:             ServiceConfig{Type: corev1.ServiceTypeClusterIP},
		DB:                  input.DB,
		Annotations:         input.Annotations,
		PodAnnotations:      input.PodAnnotations,
		Labels:              input.Labels,
		PodLabels:           input.PodLabels,
		SchedulingConfig:    input.SchedulingConfig,
		PodSecurityContext:  input.PodSecurityContext,
		PodSpec:             input.PodSpec,
		ConfigMaps:          input.ConfigMaps,
		ExtraManifests:      []unstructured.Unstructured{},
		ServiceMonitor:      input.ServiceMonitor,
		Kind:                "Deployment",
		StatefulSetSpec:     input.StatefulSetSpec,
		DeploymentSpec:      input.DeploymentSpec,

		Metadata: Metadata{
			Namespace:   input.Metadata.Namespace,
			Service:     input.Metadata.Service,
			Component:   input.Metadata.Component,
			Environment: input.Metadata.Environment,
		},
	}
	if input.ReplicaCount != nil {
		values.ReplicaCount = *input.ReplicaCount
	}

	if input.Kind != nil {
		values.Kind = *input.Kind
	}

	if input.ServiceConfig != nil {
		if input.ServiceConfig.Type != nil {
			values.Service.Type = *input.ServiceConfig.Type
		}
		values.Service.Annotations = input.ServiceConfig.Annotations
		values.Service.Labels = input.ServiceConfig.Labels
	}

	if containers, err := getDeploymentContainers(input); err != nil {
		return DeploymentValues{}, fmt.Errorf("error while preparing deployment containers: %v", err)
	} else {
		values.Containers = containers
	}

	if initContainers, err := getInitContainers(input); err != nil {
		return DeploymentValues{}, fmt.Errorf("error while preparing init containers: %v", err)
	} else {
		values.InitContainers = initContainers
	}

	// check for main deployment containers and if at least 1 of their ports has a NodePort, override the service type
	for _, container := range values.Containers {
		for _, port := range container.Ports {
			// override only if service type is ClusterIP (if manually set to NodePort or LoadBalancer, that is allowed)
			if port.NodePort != nil && values.Service.Type == corev1.ServiceTypeClusterIP {
				values.Service.Type = corev1.ServiceTypeNodePort
				break
			}
		}
	}

	if input.PreDeploymentJob != nil {
		if preDeploymentJob, err := getPreDeploymentJob(input); err != nil {
			return DeploymentValues{}, fmt.Errorf("error while preparing pre-deployment job: %v", err)
		} else {
			values.PreDeploymentJob = &preDeploymentJob
		}
	}

	if len(input.Cronjobs) > 0 {
		if cronjobs, err := getCronjobs(input); err != nil {
			return DeploymentValues{}, fmt.Errorf("error while preparing cronjobs: %v", err)
		} else {
			values.Cronjobs = cronjobs
		}
	}

	for _, raw := range input.ExtraManifests {
		values.ExtraManifests = append(values.ExtraManifests, unstructured.Unstructured{Object: raw})
	}

	return values, nil
}

func resolveHttpRoutes(input schema.InputValues) map[string]schema.HTTPRoute {
	// validated to be mutually exclusive
	if input.HTTPRoute != nil {
		return map[string]schema.HTTPRoute{
			"main": *input.HTTPRoute,
		}
	}
	return input.HTTPRoutes
}

func getDeploymentContainers(input schema.InputValues) ([]Container, error) {
	// validate main container image
	if input.Image.Tag == nil {
		return []Container{}, fmt.Errorf("main container must have `image.tag` set")
	}
	if len(input.Ports) == 0 {
		return []Container{}, fmt.Errorf("main container must have at least one port")
	}
	containers := []Container{
		convertContainer(input.Container, input.MainContainerName, ptr.To("main")),
	}
	for sidecarName, sidecarInput := range sortedMap(input.Sidecars) {
		if err := validateAndSetSideContainerImage(&sidecarInput.Image, &input.Image); err != nil {
			return []Container{}, fmt.Errorf("error validating sidecar '%v': %v", sidecarName, err)
		}
		containers = append(containers, convertContainer(sidecarInput, ptr.To(sidecarName)))
	}
	return containers, nil
}

func getInitContainers(input schema.InputValues) ([]Container, error) {
	containers := []Container{}
	for _, initContainerInput := range input.InitContainers {
		if err := validateAndSetSideContainerImage(&initContainerInput.Image, &input.Image); err != nil {
			return []Container{}, fmt.Errorf("error validating init container '%v': %v", initContainerInput.Name, err)
		}
		containers = append(containers, convertContainer(initContainerInput.Container, ptr.To(initContainerInput.Name)))
	}
	return containers, nil
}

func validateAndSetSideContainerImage(targetImage, mainImage *schema.Image) error {
	imageTagIsEmpty := targetImage.Tag == nil || strings.TrimSpace(*targetImage.Tag) == ""
	inheritTag := targetImage.InheritMainContainerTag != nil && *targetImage.InheritMainContainerTag
	if !inheritTag && imageTagIsEmpty {
		return fmt.Errorf("side container must have either `image.tag` set or `image.inheritMainContainerTag: true`")
	}
	if inheritTag {
		targetImage.Tag = mainImage.Tag
	}
	return nil
}

func getPreDeploymentJob(input schema.InputValues) (PreDeploymentJob, error) {
	if err := validateAndSetSideContainerImage(&input.PreDeploymentJob.Image, &input.Image); err != nil {
		return PreDeploymentJob{}, fmt.Errorf("error validating pre-deployment job main container: %v", err)
	}

	job := PreDeploymentJob{
		Container: convertContainer(input.PreDeploymentJob.Container, input.PreDeploymentJob.MainContainerName, ptr.To("main")),
		Metadata: Metadata{
			Namespace:   input.Metadata.Namespace,
			Service:     input.Metadata.Service,
			Component:   input.Metadata.Component,
			Environment: input.Metadata.Environment,
		},
		PodMonitor:         input.PreDeploymentJob.PodMonitor,
		Volumes:            input.PreDeploymentJob.Volumes,
		Annotations:        input.PreDeploymentJob.Annotations,
		Labels:             input.PreDeploymentJob.Labels,
		PodAnnotations:     input.PreDeploymentJob.PodAnnotations,
		PodLabels:          input.PreDeploymentJob.PodLabels,
		PodSecurityContext: input.PreDeploymentJob.PodSecurityContext,
		PodSpec:            input.PreDeploymentJob.PodSpec,
		SchedulingConfig:   input.PreDeploymentJob.SchedulingConfig,
		JobSpec:            input.PreDeploymentJob.JobSpec,
	}

	// init containers
	initContainers := []Container{}
	for _, initContainerInput := range input.PreDeploymentJob.InitContainers {
		if err := validateAndSetSideContainerImage(&initContainerInput.Image, &input.Image); err != nil {
			return PreDeploymentJob{}, fmt.Errorf("error validating pre-deployment job's init container '%v': %v", initContainerInput.Name, err)
		}
		initContainers = append(initContainers, convertContainer(initContainerInput.Container, ptr.To(initContainerInput.Name)))
	}
	job.InitContainers = initContainers
	return job, nil
}

func getCronjobs(input schema.InputValues) ([]Cronjob, error) {
	cronjobs := []Cronjob{}
	for i := 0; i < len(input.Cronjobs); i++ {
		if err := validateAndSetSideContainerImage(&input.Cronjobs[i].Image, &input.Image); err != nil {
			return []Cronjob{}, fmt.Errorf("error validating cronjob '%v' main container: %v", input.Cronjobs[i].Name, err)
		}

		cronjob := Cronjob{
			Container: convertContainer(input.Cronjobs[i].Container, input.Cronjobs[i].MainContainerName, ptr.To("main")),
			Metadata: Metadata{
				Namespace:   input.Metadata.Namespace,
				Service:     input.Metadata.Service,
				Component:   input.Metadata.Component,
				Environment: input.Metadata.Environment,
			},
			Name:               input.Cronjobs[i].Name,
			Schedule:           input.Cronjobs[i].Schedule,
			Volumes:            input.Cronjobs[i].Volumes,
			PodMonitor:         input.Cronjobs[i].PodMonitor,
			PodSecurityContext: input.Cronjobs[i].PodSecurityContext,
			PodSpec:            input.Cronjobs[i].PodSpec,

			CronJobAnnotations: input.Cronjobs[i].CronJobAnnotations,
			CronJobLabels:      input.Cronjobs[i].CronJobLabels,
			JobAnnotations:     input.Cronjobs[i].JobAnnotations,
			JobLabels:          input.Cronjobs[i].JobLabels,
			PodAnnotations:     input.Cronjobs[i].PodAnnotations,
			PodLabels:          input.Cronjobs[i].PodLabels,
			SchedulingConfig:   input.Cronjobs[i].SchedulingConfig,
			CronJobSpec:        input.Cronjobs[i].CronJobSpec,
			JobSpec:            input.Cronjobs[i].JobSpec,
		}

		// init containers
		initContainers := []Container{}
		for _, initContainerInput := range input.Cronjobs[i].InitContainers {
			if err := validateAndSetSideContainerImage(&initContainerInput.Image, &input.Image); err != nil {
				return []Cronjob{}, fmt.Errorf("error validating cronjob's '%v' init container '%v': %v", input.Cronjobs[i].Name, initContainerInput.Name, err)
			}
			initContainers = append(initContainers, convertContainer(initContainerInput.Container, ptr.To(initContainerInput.Name)))
		}
		cronjob.InitContainers = initContainers
		cronjobs = append(cronjobs, cronjob)
	}
	return cronjobs, nil
}

func convertContainer(container schema.Container, names ...*string) Container {
	name := ""
	// takes the first non-nil and non-empty name from the variadic names
	// usage: container name overrides, but provide a default if the override is nil
	for _, n := range names {
		if n != nil {
			if s := strings.TrimSpace(*n); s != "" {
				name = s
				break
			}
		}
	}

	return Container{
		Name: name,
		Image: Image{
			Repository:  container.Image.Repository,
			Tag:         container.Image.Tag, // already should have proper image tag, respecting the inherit flag
			PullPolicy:  container.Image.PullPolicy,
			PullSecrets: container.Image.PullSecrets,
		},
		Args:            container.Args,
		Command:         container.Command,
		Ports:           container.Ports,
		Envs:            container.Envs,
		EnvsRaw:         container.EnvsRaw,
		KubeSecrets:     container.KubeSecrets,
		ExternalSecrets: container.ExternalSecrets,
		Resources:       container.Resources,
		ReadinessProbe:  container.ReadinessProbe,
		LivenessProbe:   container.LivenessProbe,
		Lifecycle:       container.Lifecycle,
		SecurityContext: container.SecurityContext,
		ContainerSpec:   container.ContainerSpec,
	}
}
