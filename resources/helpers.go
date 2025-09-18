package resources

import (
	"fmt"
	"iter"
	"maps"
	"slices"
	"sort"
	"strings"
)

func sortedMap[T any](m map[string]T) iter.Seq2[string, T] {
	return func(yield func(string, T) bool) {
		keys := slices.Collect(maps.Keys(m))
		sort.Strings(keys)

		for _, key := range keys {
			if !yield(key, m[key]) {
				return
			}
		}
	}
}

func serviceName(metadata Metadata) string {
	s := fmt.Sprintf("%s--%s--%s", metadata.Service, metadata.Component, metadata.Environment)
	return strings.TrimSpace(s)
}

func commonLabels(metadata Metadata) map[string]string {
	return map[string]string{
		"app":                          serviceName(metadata),
		"namespace":                    metadata.Namespace,
		"service":                      metadata.Service,
		"component":                    metadata.Component,
		"environment":                  metadata.Environment,
		"yoke-flight-version":          Version,
		"app.kubernetes.io/managed-by": "yoke",
	}
}

func withCommonLabels(labels map[string]string, metadata Metadata) map[string]string {
	dst := map[string]string{}
	maps.Copy(dst, labels)
	maps.Copy(dst, commonLabels(metadata))
	return dst
}

func pvcName(volumeName string, metadata Metadata) string {
	return fmt.Sprintf("%s--%s", serviceName(metadata), volumeName)
}

func vaultSecretName(secretPath string, metadata Metadata) string {
	path := strings.Clone(secretPath)
	path = strings.ReplaceAll(path, "/", "-")

	name := fmt.Sprintf("%s--vault--%s", serviceName(metadata), path)
	targetLength := min(len(name), 63)
	name = name[:targetLength]
	name = strings.TrimSuffix(name, "-")
	return name
}

func vaultName(metadata Metadata) string {
	if metadata.Environment == "dev" || metadata.Environment == "test" {
		return "vault-test"
	} else {
		return "vault-prod"
	}
}

func preDeploymentJobName(metadata Metadata) string {
	// TODO: add something to make it unique?? chart had `Release.Revision`
	return fmt.Sprintf("%s--pre-deploy", serviceName(metadata))
}
