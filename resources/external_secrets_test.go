package resources

import (
	"slices"
	"testing"

	"github.com/ProRocketeers/yoke-chart/schema"
	es "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
)

func TestExternalSecrets(t *testing.T) {
	type CaseConfig struct {
		ValuesTransform func(*DeploymentValues)
		Asserts         func(*testing.T, []*es.ExternalSecret)
	}

	cases := map[string]func() CaseConfig{
		"renders basic external secret mapping": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Containers[0].VaultSecrets = map[string]schema.SecretMapping{
						"path/to/secret": {
							"MY_ENV":       ptr.To("MY-SECRET"),
							"MY_OTHER_ENV": nil,
						},
					}
				},
				Asserts: func(t *testing.T, secrets []*es.ExternalSecret) {
					require.Len(t, secrets, 1)

					assert.Equal(t, "service--component--test--vault--path-to-secret", secrets[0].Name)
					assert.Equal(t, "vault-test", secrets[0].Spec.SecretStoreRef.Name)
					require.Len(t, secrets[0].Spec.Data, 2)

					assert.Contains(t, secrets[0].Spec.Data, es.ExternalSecretData{
						SecretKey: "my_env",
						RemoteRef: es.ExternalSecretDataRemoteRef{
							Key:      "path/to/secret",
							Property: "MY-SECRET",
							// defaults
							ConversionStrategy: es.ExternalSecretConversionDefault,
							DecodingStrategy:   es.ExternalSecretDecodeNone,
							MetadataPolicy:     es.ExternalSecretMetadataPolicyNone,
						},
					})
					assert.Contains(t, secrets[0].Spec.Data, es.ExternalSecretData{
						SecretKey: "my_other_env",
						RemoteRef: es.ExternalSecretDataRemoteRef{
							Key:      "path/to/secret",
							Property: "MY_OTHER_ENV",
							// defaults
							ConversionStrategy: es.ExternalSecretConversionDefault,
							DecodingStrategy:   es.ExternalSecretDecodeNone,
							MetadataPolicy:     es.ExternalSecretMetadataPolicyNone,
						},
					})
					assert.Subset(t, secrets[0].Spec.Target.Template.Data, map[string]string{
						"MY_ENV":       "{{ .my_env }}",
						"MY_OTHER_ENV": "{{ .my_other_env }}",
					})
				},
			}
		},
		"properly renders full secret mapping": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Containers[0].VaultSecrets = map[string]schema.SecretMapping{
						"path/to/secret": nil,
					}
				},
				Asserts: func(t *testing.T, secrets []*es.ExternalSecret) {
					require.Len(t, secrets, 1)

					assert.Equal(t, "vault-test", secrets[0].Spec.SecretStoreRef.Name)

					require.Len(t, secrets[0].Spec.DataFrom, 1)
					assert.Equal(t, "path/to/secret", secrets[0].Spec.DataFrom[0].Extract.Key)
				},
			}
		},
		"properly renders multiple external secrets ": func() CaseConfig {
			return CaseConfig{
				ValuesTransform: func(dv *DeploymentValues) {
					dv.Containers[0].VaultSecrets = map[string]schema.SecretMapping{
						"path/to/secret": {
							"MY_ENV": ptr.To("MY-SECRET"),
						},
						"path/to/other/secret": {
							"MY_OTHER_ENV": ptr.To("THEIR-SECRET"),
						},
					}
				},
				Asserts: func(t *testing.T, secrets []*es.ExternalSecret) {
					require.Len(t, secrets, 2)

					firstSecretI := findSecretIndexByName(secrets, "service--component--test--vault--path-to-secret")
					secondSecretI := findSecretIndexByName(secrets, "service--component--test--vault--path-to-other-secret")

					assert.True(t, firstSecretI >= 0, "secret for path/to/secret not found")
					assert.True(t, secondSecretI >= 0, "secret for path/to/other/secret not found")
				},
			}
		},
	}

	base := DeploymentValues{
		Metadata: Metadata{
			Namespace:   "ns",
			Service:     "service",
			Component:   "component",
			Environment: "test",
		},
		Containers: []Container{
			{
				Name: "main",
				Image: Image{
					Repository: "image_repository",
					Tag:        ptr.To("image_tag"),
				},
			},
		},
	}

	for testName, makeConfig := range cases {
		t.Run(testName, func(t *testing.T) {
			values := DeploymentValues{}
			copier.CopyWithOption(&values, &base, copier.Option{DeepCopy: true})

			config := makeConfig()
			config.ValuesTransform(&values)

			shouldCreate, create := CreateExternalSecrets(values)
			if !shouldCreate {
				t.Error("error during test setup: shouldCreate returned false")
			}
			resources, err := create(values)
			if err != nil {
				t.Errorf("error during test setup: %v", err)
			}
			secrets := []*es.ExternalSecret{}
			for i := range resources {
				if s, ok := resources[i].(*es.ExternalSecret); ok {
					secrets = append(secrets, s)
				} else {
					t.Error("error while retyping external secrets in test setup")
				}
			}
			config.Asserts(t, secrets)
		})
	}

	t.Run("returns error when there are multiple mappings from the same path", func(t *testing.T) {
		values := DeploymentValues{}
		copier.CopyWithOption(&values, &base, copier.Option{DeepCopy: true})

		values.Containers[0].VaultSecrets = map[string]schema.SecretMapping{
			"path/to/secret": {
				"MY_ENV": ptr.To("MY-SECRET"),
			},
			"path/to/other/secret": {
				"MY_OTHER_ENV": ptr.To("THEIR-SECRET"),
			},
		}
		values.InitContainers = []Container{
			{
				Name: "init",
				Image: Image{
					Repository: "image_repository",
					Tag:        ptr.To("image_tag"),
				},
				VaultSecrets: map[string]schema.SecretMapping{
					"path/to/secret": {
						"MY_ENV": ptr.To("MY-SECRET"),
					},
				},
			},
		}
		shouldCreate, create := CreateExternalSecrets(values)
		if !shouldCreate {
			t.Error("error during test setup: shouldCreate returned false")
		}
		_, err := create(values)
		require.Error(t, err)
	})
}

func findSecretIndexByName(secrets []*es.ExternalSecret, name string) int {
	return slices.IndexFunc(secrets, func(s *es.ExternalSecret) bool {
		return s.Name == name
	})
}
