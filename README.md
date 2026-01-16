# Yoke "chart" (Flight)

## How to use
First, get the Yoke binary
```bash
go install github.com/yokecd/yoke/cmd/yoke@latest
```

### Testing
```bash
# run tests in current and all subdirectories
# `count=1` is there to run fresh tests and not return cached results
go test -count=1 -timeout 30s ./...

# either run natively
go run . < values.yaml

# or build and test the compiled version
GOOS=wasip1 GOARCH=wasm go build -o chart.wasm

# `dry` => doesn't actually deploy anything
# `cross-namespace` => allows to deploy across multiple namespaces (since we don't specify, the default is "default" namespace)
# `out` => allows for checking of the rendered YAMLs, will output into `./example`, grouped by namespace and GVK
# `"example"` => name of the release
yoke takeoff -dry -cross-namespace -out . example ./chart.wasm < values.yaml
```

### Releasing
1. Update the `Version` variable in `resources/schema.go` to your **new** desired version
2. Adequately update `CHANGELOG` / `README`
3. Commit, push, in GitHub, create tag with the same version you used
4. That's it, GitHub CI will unit test the package, compile it and push it as OCI artifact GitHub Container Registry (under the specified version + `latest` tag)

## How to use it in ArgoCD

### Setup
Because Yoke is integrated with ArgoCD as a CMP (Content Management Plugin), the ArgoCD deployment must be changed (a sidecar with the plugin must be added to repo server). Up to date instructions are [here](https://yokecd.github.io/docs/yokecd/#patch-an-existing-argocd-installation).

### Usage in `Applications`
YokeCD fully supports reading multiple Helm values files (whether JSON or YAML), also supports inlined `valuesObject` equivalent - see the [docs](https://yokecd.github.io/docs/yokecd/#input-map-parameter) with examples

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: my-app
  namespace: argocd
spec:
  destination:
    namespace: my-app
    server: https://kubernetes.default.svc
  project: default
  source:
    repoURL: https://github.com/myorg/my-repo.git
    path: .
    targetRevision: HEAD
    plugin:
      name: yokecd
      parameters:
        - name: wasm
          string: oci://ghcr.io/prorocketeers/yoke-chart:1.2.0
        - name: inputFiles
          # relative to the `source.path`
          array:
            - values.yaml
  syncPolicy:
    syncOptions:
      - CreateNamespace=true
```