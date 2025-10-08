package resources

import (
	"fmt"
	"iter"
	"maps"
	"slices"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
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

func secretName(secretPath, secretStoreName string, metadata Metadata) string {
	path := strings.Clone(secretPath)
	path = strings.ReplaceAll(path, "/", "-")

	name := fmt.Sprintf("%s--%s--%s", serviceName(metadata), secretStoreName, path)
	targetLength := min(len(name), 253)
	name = name[:targetLength]
	name = strings.TrimSuffix(name, "-")
	return name
}

func preDeploymentJobName(metadata Metadata) string {
	// TODO: add something to make it unique?? chart had `Release.Revision`
	return fmt.Sprintf("%s--pre-deploy", serviceName(metadata))
}

func cronjobName(cronjob Cronjob) string {
	return fmt.Sprintf("%s--%s", cronjob.Name, cronjob.Metadata.Environment)
}

func toUnstructured(objects ...runtime.Object) ([]unstructured.Unstructured, error) {
	var (
		ret    []unstructured.Unstructured
		errors []string
	)

	for _, obj := range objects {
		o, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			errors = append(errors, err.Error())
		} else {
			// drop the `status` field as that's never needed for creating objects and causes diff
			delete(o, "status")
			ret = append(ret, unstructured.Unstructured{Object: o})
		}
	}

	if len(errors) > 0 {
		return []unstructured.Unstructured{}, fmt.Errorf("found %v errors: %v", len(errors), errors)
	}

	return ret, nil
}
