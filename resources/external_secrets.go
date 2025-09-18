package resources

import (
	"fmt"
	"slices"
	"strings"
	"time"

	es "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func CreateExternalSecrets(values DeploymentValues) (bool, ResourceCreator) {
	create := false
	containers := getAllContainers(values)
	for _, container := range containers {
		if len(container.ExternalSecrets) > 0 {
			create = true
			break
		}
	}
	return create, func(values DeploymentValues) ([]unstructured.Unstructured, error) {
		resources := []unstructured.Unstructured{}
		containers := getAllContainers(values)
		if hasDuplicateExternalSecrets(containers, values.Metadata) {
			return []unstructured.Unstructured{}, fmt.Errorf("duplicate external secret paths in multiple containers")
		}
		for _, container := range containers {
			for _, definition := range container.ExternalSecrets {
				for secretPath, secretMapping := range sortedMap(definition.Mapping) {
					secretName := secretName(secretPath, definition.SecretStore.Name, values.Metadata)

					secret := es.ExternalSecret{
						TypeMeta: metav1.TypeMeta{
							APIVersion: es.SchemeGroupVersion.Identifier(),
							Kind:       "ExternalSecret",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      secretName,
							Namespace: values.Metadata.Namespace,
							Labels: withCommonLabels(map[string]string{
								"container": container.Name,
							}, values.Metadata),
						},
						Spec: es.ExternalSecretSpec{
							RefreshInterval: func() *metav1.Duration {
								if definition.RefreshInterval != nil {
									return definition.RefreshInterval
								}
								return &metav1.Duration{Duration: 1 * time.Minute}
							}(),
							SecretStoreRef: definition.SecretStore,
						},
					}

					// fetching the entire secret
					if secretMapping == nil {
						secret.Spec.DataFrom = []es.ExternalSecretDataFromRemoteRef{
							{
								Extract: &es.ExternalSecretDataRemoteRef{
									Key: secretPath,
								},
							},
						}
						secret.Spec.Target = es.ExternalSecretTarget{
							Name:           secretName,
							CreationPolicy: es.CreatePolicyOwner,
							DeletionPolicy: es.DeletionPolicyDelete,
						}
					} else {
						// or just part of it
						remoteRefs := []es.ExternalSecretData{}
						templateData := map[string]string{}

						for envName, vaultKey := range sortedMap(secretMapping) {
							property := envName
							if vaultKey != nil {
								property = *vaultKey
							}

							r := es.ExternalSecretData{
								RemoteRef: es.ExternalSecretDataRemoteRef{
									Key:                secretPath,
									Property:           property,
									ConversionStrategy: es.ExternalSecretConversionDefault,
									DecodingStrategy:   es.ExternalSecretDecodeNone,
									MetadataPolicy:     es.ExternalSecretMetadataPolicyNone,
								},
								SecretKey: strings.ToLower(envName),
							}

							remoteRefs = append(remoteRefs, r)
							templateData[envName] = fmt.Sprintf("{{ .%s }}", strings.ToLower(envName))
						}

						secret.Spec.Data = remoteRefs
						secret.Spec.Target = es.ExternalSecretTarget{
							Name:           secretName,
							CreationPolicy: es.CreatePolicyOwner,
							DeletionPolicy: es.DeletionPolicyDelete,
							Template: &es.ExternalSecretTemplate{
								Type:          corev1.SecretTypeOpaque,
								EngineVersion: es.TemplateEngineV2,
								MergePolicy:   es.MergePolicyReplace,
								Data:          templateData,
							},
						}
					}
					u, err := toUnstructured(&secret)
					if err != nil {
						return []unstructured.Unstructured{}, err
					}
					resources = append(resources, u...)
				}
			}
		}
		return resources, nil
	}
}

func getAllContainers(values DeploymentValues) []Container {
	allContainers := []Container{}
	allContainers = append(allContainers, values.Containers...)
	allContainers = append(allContainers, values.InitContainers...)
	if values.PreDeploymentJob != nil {
		allContainers = append(allContainers, values.PreDeploymentJob.Container)
		allContainers = append(allContainers, values.PreDeploymentJob.InitContainers...)
	}
	for _, c := range values.Cronjobs {
		allContainers = append(allContainers, c.Container)
		allContainers = append(allContainers, c.InitContainers...)
	}
	return allContainers
}

func hasDuplicateExternalSecrets(containers []Container, metadata Metadata) bool {
	var usedNames []string
	for _, container := range containers {
		for _, definition := range container.ExternalSecrets {
			for path := range definition.Mapping {
				secretName := secretName(path, definition.SecretStore.Name, metadata)
				if slices.Contains(usedNames, secretName) {
					return true
				}
				usedNames = append(usedNames, secretName)
			}
		}
	}
	return false
}
