// Package v1 holds a copy of the ExternalSecret API types from
// github.com/external-secrets/external-secrets v0.18.1
// (apis/externalsecrets/v1/externalsecret_types.go). The file externalsecret_types.go is
// byte-identical to upstream; register.go is handwritten.
//
// It is vendored because importing the upstream package links controller-runtime and client-go's
// full kubernetes/scheme into the wasm module: the .wasm went from 91MB to 45MB and yokecd-svr
// host memory per render from 152MiB to 79MiB without it.
//
// To update to a new upstream version:
//
//  1. Download the file for the new tag, replacing the copy (if upstream moved the file or the
//     API version, find the new path in the repo first):
//
//     curl -fsSL https://raw.githubusercontent.com/external-secrets/external-secrets/<tag>/apis/externalsecrets/v1/externalsecret_types.go -o externalsecrets/v1/externalsecret_types.go
//
//  2. Update the version mentioned in this comment.
//
//  3. Run `go build ./... && go test ./...`. If the new file references types defined in other
//     upstream files, copy just those types into this package; never import the upstream package.
//
//  4. Check that upstream deps are not linked into the module; this must print nothing:
//
//     GOOS=wasip1 GOARCH=wasm go list -deps . | grep -E 'controller-runtime|client-go|external-secrets/external-secrets'
//
//  5. Optionally build with `make` and confirm chart.wasm is still ~45MB, not ~90MB.
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
