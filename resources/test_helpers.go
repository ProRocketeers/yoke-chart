package resources

import (
	"github.com/yokecd/yoke/pkg/flight"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func findResource[T metav1.Object](resources []flight.Resource, kind, name string) (T, bool) {
	var zero T

	for i, res := range resources {
		gvk := res.GroupVersionKind()
		if gvk.Kind == kind {
			if r, ok := res.(T); ok {
				if r.GetName() == name {
					return resources[i].(T), true
				}
			}
		}
	}
	return zero, false
}
