// Package v1 holds a copy of the ExternalSecret API types from
// github.com/external-secrets/external-secrets v0.18.1 (apis/externalsecrets/v1/externalsecret_types.go).
//
// Importing the upstream package links controller-runtime and client-go's full scheme into the
// wasm module (86MB vs 18MB), which every yokecd render pays for in memory. Only the plain types
// are needed to build manifests, so they are vendored here. Refresh the copy when bumping
// external-secrets.
package v1

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var SchemeGroupVersion = schema.GroupVersion{Group: "external-secrets.io", Version: "v1"}

// DeepCopyObject satisfies runtime.Object.
// ponytail: JSON round-trip instead of generated deepcopy; it is only used to satisfy the interface.
func (in *ExternalSecret) DeepCopyObject() runtime.Object {
	out := new(ExternalSecret)
	data, _ := json.Marshal(in)
	_ = json.Unmarshal(data, out)
	return out
}
