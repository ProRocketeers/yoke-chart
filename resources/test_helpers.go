package resources

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func findResource[T metav1.Object](resources []unstructured.Unstructured, kind, name string) (T, bool) {
	var (
		zero  T
		value T
	)

	for i, res := range resources {
		gvk := res.GroupVersionKind()
		if gvk.Kind == kind && res.GetName() == name {
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(resources[i].Object, &value)
			if err == nil {
				return value, true
			}
		}
	}
	return zero, false
}

func findResourceOrFail[T metav1.Object](t *testing.T, r []unstructured.Unstructured, kind, name string) T {
	t.Helper()
	res, ok := findResource[T](r, kind, name)
	require.Truef(t, ok, "%v %v not found", kind, name)
	return res
}

func fromUnstructuredOrPanic[T metav1.Object](u unstructured.Unstructured) T {
	var value T
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &value)
	if err != nil {
		panic(err)
	}
	return value
}

func fromUnstructuredArrayOrPanic[T metav1.Object](objects []unstructured.Unstructured) (ret []T) {
	for _, obj := range objects {
		ret = append(ret, fromUnstructuredOrPanic[T](obj))
	}
	return
}

func partialEqual(t require.TestingT, expected, actual any, diffOpts cmp.Options, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}

	if cmp.Equal(expected, actual, diffOpts) {
		return
	}

	diff := cmp.Diff(expected, actual, diffOpts)
	assert.Fail(t, fmt.Sprintf("Not equal: \n"+
		"expected: %s\n"+
		"actual  : %s%s", expected, actual, diff), msgAndArgs...)
}

func partialContains[T any](t require.TestingT, array []T, element T, diffOpts cmp.Options, msgAndArgs ...any) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}

	for _, val := range array {
		if cmp.Equal(element, val, diffOpts) {
			return
		} else {
			fmt.Println(cmp.Diff(element, val, diffOpts))
		}
	}

	assert.Fail(t, fmt.Sprintf("Array item not found: \n"+
		"array: %v\n"+
		"item  : %v", array, element), msgAndArgs...)
}

type tHelper interface {
	Helper()
}
