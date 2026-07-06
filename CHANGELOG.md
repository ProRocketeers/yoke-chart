# Changelog

Here will be documented any important changes.

Format is loosely based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the Yoke Flight ("chart") tries to follow [semver](https://semver.org/spec/v2.0.0.html).

For clarity we use following emoji for the changes:
- added = :star: `:star:` 
- changed = :pencil2: `:pencil2:` (:boom: `:boom:` for breaking changes)
- fixed = :hammer_and_wrench: `:hammer_and_wrench:`
- deprecated = :warning: `:warning:`
- removed = :x: `:x:`
- security = :lock: `:lock:`

## [unreleased]

## [1.11.0] - 2026-07-06

### :boom: BREAKING CHANGES
- `PreDeploymentJob` and `CronJob` extra config fields have been separated into their own respective properties and are now full unchecked overrides (see below)
  - table of what properties moved where:

    | Previously (flat on `cronjobs[]` / `preDeploymentJob`)                                                                                                          | Moved to                                                |
    |-----------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------------------------------------|
    | CronJob only fields - `timeZone`, `concurrencyPolicy`, `suspend`, `startingDeadlineSeconds`, `successfulJobsHistoryLimit`, `failedJobsHistoryLimit`             | `cronjobs[].cronJobSpec.*`                              |
    | Job fields - `activeDeadlineSeconds`, `backoffLimit`, `completionMode`, `completions`, `parallelism`, `podFailurePolicy`, `selector`, `ttlSecondsAfterFinished` | `cronjobs[].jobSpec.*` and `preDeploymentJob.jobSpec.*` |
    | `preDeploymentJob.suspend` (distinct field from `CronJobSpec.Suspend` above)                                                                                    | `preDeploymentJob.jobSpec.suspend`                      |

  - in addition to these fields, some new fields were "added" (more specifically, just exposed) - e.g. `JobSpec.(successPolicy|podReplacementPolicy|...)`
- `statefulSet` property (used for specifying additional `StatefulSet` properties) is now **renamed** to `statefulSetSpec` (to align with other override fields)
  - also, the behavior changed - previously it was taken to be the "source", while the chart overridden `selector`, `template`, `serviceName` and `replicas` (to ensure correctness)
  - now, again in unity with other override fields, this property is taken to be a full override over the opinionated defaults
- `securityContext` and `podSecurityContext` (on all levels/places) - now **REMOVED**, in favor of the new `podSpec` and `containerSpec` fields

### :star: Added
- ability to specify `startupProbe` - same types as other probes, but run before readiness/liveness probes; very useful for slow-starting containers
- ability to specify `dataSource` or `dataSourceRef` on new persistent volumes (same shape as in `PersistentVolumeSpec`)
  ```yaml
  volumes:
    data:
      type: persistent
      existing: false
      # ...
      dataSource:
        name: volume-snapshot
        kind: VolumeSnapshot
        apiGroup: snapshot.storage.k8s.io
  ```
- ability to override `creationPolicy` and `deletionPolicy` on `ExternalSecrets` (defaults to `Owner` and `Delete` respectively)
  ```yaml
  externalSecrets:
    - secretStore: {...}
      creationPolicy: Orphan
      deletionPolicy: Retain
      mapping: {}
  ```
- ability to specify `topologySpreadConstraints` and `priorityClassName` (on root level and also in `cronjobs[].*` and `preDeploymentJob.*` - on Pod level) extending the scheduling configs

#### Overrides
- the chart is generally *opinionated*, with reasonable defaults for majority of situations, however we wanted to provide customizability
- added properties that are meant to be catch-all, **UNCHECKED** overrides at every level
- these are meant to be overrides that are layered **AFTER** the chart templates the objects
- these merges are **NOT CHECKED**, and can result in misconfiguration and broken deployments - that's on you as a user
- specific added fields:
  - `deploymentSpec` / `statefulSetSpec`
  - `podSpec` - for main workload's `Pod`
    - also in other places - `preDeploymentJob.podSpec`, `cronjobs[].podSpec`
  - `containerSpec` - for main workload's main container
    - also on every other Container specification:
    - `sidecars.containerSpec`
    - `initContainers[].containerSpec`
    - `preDeploymentJob.containerSpec` - pre-deployment Job's main container
    - `preDeploymentJob.initContainers[].containerSpec`
    - `cronjobs[].containerSpec` - CronJob's main container
    - `cronjobs[].initContainers[].containerSpec`
  - `serviceConfig` - overrides for the main workload's `Service`
    - the fields here are actually inlined, so its fields such as `Type` or `SessionAffinity` are directly on `serviceConfig`
  - `preDeploymentJob.jobSpec`
    - this was previously inlined, with only selected fields, now a full override
  - `cronjobs[].jobSpec` and `cronjobs[].cronJobSpec`
    - also previously an inlined combination of fields, now separate in its own fields, again a full override

### :hammer_and_wrench: Fixed
- `HorizontalPodAutoscaler` now properly attaches to `StatefulSet` if `kind: StatefulSet`

## [1.10.0] - 2026-07-06

### :star: Added
#### Templates in `extraManifests`
- previous implementation of `extraManifests` more or less returned, but with better options and some extra guardrails
- now, any **string**, **leaf** node/property (e.g. string values in arrays, string properties) can be templated with `{{ }}` Go templates
- e.g.
```yaml
extraManifests:
  - apiVersion: gateway.kgateway.dev/v1alpha1
    kind: TrafficPolicy
    metadata:
      name: proxy-body-size
      namespace: "{{ .Values.Metadata.Namespace }}"
    spec:
      targetRefs:
        - group: gateway.networking.k8s.io
          kind: HTTPRoute
          name: "{{ .Outputs.HTTPRoutes.main.Name }}"
      buffer:
        maxRequestSize: 50Mi
```
- this is very useful for referencing either values used in the chart itself (`.Values`), or the actual created resources by the chart (we only collect name and namespace) - `.Outputs`, without having to know the actual patterns or templates for the names
##### Guardrails
- all control flow nodes such as `{{if}}`, `{{range}}`, `{{with}}`, `{{template}}`, `{{define}}`, `{{block}}`, `{{break}}` or `{{continue}}`  are rejected and an error is thrown
- due to technical constraints, only string nodes can be templated (not booleans or numbers)
##### Functions
- you can use functions in the templates and compose them into pipelines with `|`
- currently available are all functions from Go's `text/template`, plus a deterministic subset of Sprig's functions
  - some Sprig functions are excluded even beyond Sprig's own "hermetic" labeling, since a few of those still depend on the clock or on randomness internally (e.g. `ago`, `randInt`, `genPrivateKey`) - rendering must stay reproducible, or ArgoCD would see a diff (or worse) on every sync
  - excluded entirely (non-deterministic - reads the clock, randomness, environment or network): `now`, `date`, `dateInZone`, `dateModify`, `date_in_zone`, `date_modify`, `htmlDate`, `htmlDateInZone`, `ago`, `randAlphaNum`, `randAlpha`, `randAscii`, `randNumeric`, `randBytes`, `randInt`, `shuffle`, `uuidv4`, `env`, `expandenv`, `getHostByName`, `bcrypt`, `htpasswd`, `genPrivateKey`, `genCA`, `genCAWithKey`, `genSelfSignedCert`, `genSelfSignedCertWithKey`, `genSignedCert`, `genSignedCertWithKey`, `encryptAES`

| Category                     | Functions                                                                                                                                                                                                                                                                                                                                                                                                       |
|------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Go `text/template` built-ins | `and`, `or`, `not`, `eq`, `ne`, `lt`, `le`, `gt`, `ge`, `len`, `index`, `slice`, `call`, `print`, `printf`, `println`, `html`, `js`, `urlquery`                                                                                                                                                                                                                                                                 |
| Chart-specific               | `serviceName(metadata)`                                                                                                                                                                                                                                                                                                                                                                                         |
| Strings                      | `abbrev`, `abbrevboth`, `trunc`, `trim`, `upper`, `lower`, `title`, `untitle`, `substr`, `repeat`, `trimAll`, `trimSuffix`, `trimPrefix`, `nospace`, `initials`, `swapcase`, `snakecase`, `camelcase`, `kebabcase`, `wrap`, `wrapWith`, `contains`, `hasPrefix`, `hasSuffix`, `quote`, `squote`, `cat`, `indent`, `nindent`, `replace`, `plural`, `sha1sum`, `sha256sum`, `sha512sum`, `adler32sum`, `toString` |
| String lists                 | `split`, `splitList`, `splitn`, `toStrings`, `join`, `sortAlpha`                                                                                                                                                                                                                                                                                                                                                |
| Integer/float math           | `atoi`, `int`, `int64`, `float64`, `seq`, `toDecimal`, `add1`, `add`, `sub`, `div`, `mod`, `mul`, `add1f`, `addf`, `subf`, `divf`, `mulf`, `max`, `min`, `maxf`, `minf`, `ceil`, `floor`, `round`, `biggest`                                                                                                                                                                                                    |
| Dates (fixed input only)     | `duration`, `durationRound`, `toDate`, `mustToDate`, `mustDateModify`, `unixEpoch`                                                                                                                                                                                                                                                                                                                              |
| Defaults / logic             | `default`, `empty`, `coalesce`, `all`, `any`, `ternary`, `compact`, `mustCompact`                                                                                                                                                                                                                                                                                                                               |
| Encoding                     | `b64enc`, `b64dec`, `b32enc`, `b32dec`, `toJson`, `toPrettyJson`, `toRawJson`, `fromJson`, `mustToJson`, `mustToPrettyJson`, `mustToRawJson`, `mustFromJson`                                                                                                                                                                                                                                                    |
| Lists                        | `list`, `tuple`, `first`, `mustFirst`, `rest`, `mustRest`, `last`, `mustLast`, `initial`, `mustInitial`, `reverse`, `mustReverse`, `uniq`, `mustUniq`, `without`, `mustWithout`, `has`, `mustHas`, `slice`, `mustSlice`, `concat`, `append`/`push`, `mustAppend`/`mustPush`, `prepend`, `mustPrepend`, `chunk`, `mustChunk`                                                                                     |
| Dictionaries                 | `dict`, `get`, `set`, `unset`, `hasKey`, `pluck`, `keys`, `pick`, `omit`, `merge`, `mergeOverwrite`, `mustMerge`, `mustMergeOverwrite`, `values`, `dig`, `deepCopy`, `mustDeepCopy`                                                                                                                                                                                                                             |
| Reflection                   | `typeOf`, `typeIs`, `typeIsLike`, `kindOf`, `kindIs`, `deepEqual`                                                                                                                                                                                                                                                                                                                                               |
| Paths                        | `base`, `dir`, `clean`, `ext`, `isAbs`, `osBase`, `osClean`, `osDir`, `osExt`, `osIsAbs`                                                                                                                                                                                                                                                                                                                        |
| Crypto (deterministic only)  | `derivePassword`, `decryptAES`                                                                                                                                                                                                                                                                                                                                                                                  |
| SemVer                       | `semver`, `semverCompare`                                                                                                                                                                                                                                                                                                                                                                                       |
| Regex                        | `regexMatch`, `mustRegexMatch`, `regexFindAll`, `mustRegexFindAll`, `regexFind`, `mustRegexFind`, `regexReplaceAll`, `mustRegexReplaceAll`, `regexReplaceAllLiteral`, `mustRegexReplaceAllLiteral`, `regexSplit`, `mustRegexSplit`, `regexQuoteMeta`                                                                                                                                                            |
| URLs                         | `urlParse`, `urlJoin`                                                                                                                                                                                                                                                                                                                                                                                           |
| Flow control                 | `fail`                                                                                                                                                                                                                                                                                                                                                                                                          |

##### Values
- the root context `.` only contains `.Values` (the deployment values, after passing the `setup.go`, **NOT** input values), and `.Outputs` (see below)
- every field is either a single `{Name, Namespace, Kind}` reference, or (for resources that can be created multiple times) a map of those keyed the same way you named them in your values - singular fields are absent (`nil`) if that resource wasn't created, map fields are always present but empty if nothing of that kind was created

| `.Outputs` Field                     | Shape                                   | References                                                                  |
|--------------------------------------|-----------------------------------------|-----------------------------------------------------------------------------|
| `Workload`                           | single                                  | the main `Deployment` or `StatefulSet`                                      |
| `HeadlessService`                    | single                                  | the headless `Service` created alongside a `StatefulSet`                    |
| `Service`                            | single                                  | the main `Service`                                                          |
| `Ingress`                            | single                                  | the `Ingress`                                                               |
| `ServiceAccount`                     | single                                  | the `ServiceAccount`                                                        |
| `PreDeploymentJob`                   | single                                  | the pre-deployment `Job`                                                    |
| `HPA`                                | single                                  | the `HorizontalPodAutoscaler`                                               |
| `PDB`                                | single                                  | the `PodDisruptionBudget`                                                   |
| `DB`                                 | single                                  | the `postgresql` resource                                                   |
| `Role` / `RoleBinding`               | single                                  | `serviceAccount.additionalRole`'s `Role`/`RoleBinding`                      |
| `ClusterRole` / `ClusterRoleBinding` | single                                  | `serviceAccount.additionalClusterRole`'s `ClusterRole`/`ClusterRoleBinding` |
| `ServiceMonitor`                     | single                                  | the `ServiceMonitor`                                                        |
| `PreDeploymentPodMonitor`            | single                                  | the pre-deployment job's `PodMonitor`                                       |
| `HTTPRoutes`                         | map, keyed by `httpRoutes.<name>`       | each `HTTPRoute`                                                            |
| `NetworkPolicies`                    | map, keyed by `networkPolicies.<name>`  | each `NetworkPolicy`                                                        |
| `ConfigMaps`                         | map, keyed by `configMaps.<name>`       | each `ConfigMap`                                                            |
| `PVCs`                               | map, keyed by the generated PVC name    | each `PersistentVolumeClaim`                                                |
| `Cronjobs`                           | map, keyed by `cronjobs[].name`         | each `CronJob`                                                              |
| `CronjobPodMonitors`                 | map, keyed by `cronjobs[].name`         | each cronjob's `PodMonitor`                                                 |
| `ExternalSecrets`                    | map, keyed by the generated secret name | each `ExternalSecret`                                                       |

### :hammer_and_wrench: Fixed
- version 1.9.0 forgot to update the chart's version (that gets rendered in labels)

## [1.9.0] - 2026-07-05

### :star: Added
- volumes can now be mounted **multiple times** into the same container (e.g. at different `containerPath` or `volumePath`)
  - this change is _backwards compatible_ and the chart supports both input shapes:
    ```yaml
    volume:
      storage:
        type: persistent
        # ...
        mounts:
          main:
            containerPath: /data
          sidecar:
            - containerPath: /data
            - containerPath: /setup
              volumePath: setup
        
    ```
- volume mount's `readOnly` can now be overridden (defaults to `true` for Secret, ConfigMap, Downward API and Projected volumes, `false` otherwise)
- volume mount's `mountPropagation` can now be overridden (defaults to `HostToContainer` where `readOnly = false`)

### :hammer_and_wrench: Fixed
- volume mount's `volumePath` (mounted as `subPath`) is no longer silently ignored for `secret`/`configMap` volumes - it can now be used to mount a single file into an existing directory
- container names in `volumes.*.mounts` are now actually validated if they exist and throw error otherwise

## [1.8.2] - 2026-06-30

### :pencil2: Changed
- partially reverted the breaking change introduced in **1.8.0** - its quality of life upgrade was not worth breaking the backward compatibility
  - introduced an adapter between old `httpRoute` values and the new `httpRoutes` values
  - `httpRoute` can now still be used, it will now behave as if it was `httpRoutes.main`
    - this means that it will change the previous HTTPRoute name, from `{service}--{component}--{env}` to `{service}--{component}--{env}-main`, with the same body
  - **both fields are now mutually exclusive** - it is assumed that if you want more routes, you will migrate to `httpRoutes`

## [1.8.1] - 2026-06-29

### :pencil2: Changed
- fixed the implementation of `httpRoutes` - now they automatically fill in the defaults for certain fields in `HTTPRouteSpec`
  - this changes nothing for values already specified, but clears up an unwanted diff in ArgoCD that was missing these values (they are otherwise added with admission webhooks)
  - affected fields (all scoped in `httpRoutes.*.`:
    - `parentRefs[].group` - default `gateway.networking.k8s.io`
    - `parentRefs[].kind` - default `Gateway`
    - `rules[].matches[].path.type` - default `PathPrefix`
    - `rules[].matches[].path.value` - default `/`
    - `rules[].backendRefs[].group` - default `""` (core API group)
    - `rules[].backendRefs[].kind` - default `Service`
    - `rules[].backendRefs[].weight` - default `1`
  - if your values for these properties are equal to these defaults, they can be safely removed from values
- _(internal)_ split the schema file into multiple files, so it's easier to maintain

## [1.8.0] - 2026-06-10

### :boom: BREAKING CHANGE
- extended creating of `HTTPRoutes`
  - you can now create multiple HTTPRoutes - `httpRoute` became `httpRoutes`
  - `httpRoute.enabled` got removed - now it's assumed if you specify it, you want to create it
  - the remaining shape of the configuration did not change
  - :warning: **How to migrate**
    - wrap your existing `httpRoute` config into a descriptive name, like `main`
    - change `httpRoute` to `httpRoutes`
    - **before**:
      ```yaml
      httpRoute:
        enabled: true
        parentRefs:
          - ...
        hostnames:
          - ...
        rules:
          - ...
      ```
    - **after**:
      ```yaml
      httpRoutes:
        main:
          parentRefs:
            - ...
          hostnames:
            - ...
          rules:
            - ...
      ```


## [1.7.0] - 2026-06-10

### :star: Added
- possibility to specify `nodeSelector`, `tolerations` and `affinity` on individual job levels (`preDeploymentJob` and `cronjobs[]`)
- you can now add annotations and labels to the main deployment's `Service` - `serviceConfig.(labels | annotations)`

### :boom: BREAKING CHANGE
- field `serviceType` was moved into `serviceConfig.type`. Behavior otherwise unchanged

### :pencil2: Changed
- order of merging labels is fixed, now user supplied labels have higher priority
  - :warning: **WARNING:** overriding `app` label will pretty much break everything as it's used for selectors

### :hammer_and_wrench: Fixed
- random small fixes, unhandled errors, code simplification

## [1.6.0] - 2026-04-14

### :star: Added
- added the possibility to specify `PodSecurityContext` (in case of main `Deployment`, `PreDeploymentJob` and `CronJobs`) and also on individual Container level - properties `podSecurityContext` and `securityContext` when applicable

## [1.5.0] - 2026-03-22

### :star: Added
- `NodePort` Service support
  - support for changing the main deployment's `Service` type - `serviceType` (default `ClusterIP`)
  - ability to specify a `nodePort` on any port - `ports[].nodePort` (must be in the range **30000-32767** and not used by any other service)
  - `serviceType` is automatically changed to `NodePort` if any port has `nodePort` specified and `serviceType` is not set or is `ClusterIP`
- added small `Makefile` for convenience
- added example of `networkPolicies` to [values.yaml](./values.yaml) from `1.4.0`

## [1.4.1] - 2026-03-18

### :pencil2: Changed
- validation for the mutual exclusivity of `ingress` and `httpRoute` was removed
  - now both can be specified at the same, since they use different mechanisms and controllers for routing traffic and having both of them is vital for certain part of migrating

## [1.4.0] - 2026-03-13
### :star: Added
- support for creating `NetworkPolicies`
  - these behave exactly like `configMaps`
  - e.g.
    ```yaml
    networkPolicies:
      deny-all:
        podSelector:
          matchLabels:
            app: foo
        policyTypes: [Ingress, Egress]
      allow-some:
        podSelector:
          matchLabels:
            app: foo
        policyTypes: [Ingress]
        ingress:
          - from:
            - namespaceSelector:
                matchLabels:
                  kubernetes.io/metadata.name: ingress-nginx
    ```
  - this will render 2 `NetworkPolicies` - `{service}--{component}--{env}-deny-all` and `{service}--{component}--{env}-allow-some` in the given namespace, and the value of each is the `NetworkPolicySpec`

## [1.3.0] - 2026-02-06

### :star: Added
- support for Gateway API's `HTTPRoute`
  - works the same as `ingress` - specify an `enabled` flag, optionally `annotations` and `labels`, and the rest is from the original `HTTPRouteSpec`
  - **NOTE:** obviously, both `ingress.enabled` and `httpRoute.enabled` cannot be `true` at the same time
  - e.g.
    ```yaml
    httpRoute:
      enabled: true
      # ... rest of the HTTPRoute spec
      parentRefs:
        - namespace: foo
          name: my-gateway
      hostnames:
        - my-service.prorocketeers.com
      rules:
        - matches:
            - path:
                value: /
          filters:
            - type: RequestHeaderModifier
              requestHeaderModifier:
                add:
                  - name: my-header
                    value: foo
          backendRefs:
            - name: my-service2
              port: 8080
    ```

## [1.2.0] - 2026-01-16

### :star: Added
- added possibility for rendering `StatefulSet` instead of `Deployment`:
  - to enable, specify:
    ```yaml
    kind: StatefulSet
    statefulSet:
      # any property in the official StatefulSetSpec
      podManagementPolicy: Parallel
    ```
  - this also automatically renders the headless `Service` that the `StatefulSet` requires

## [1.1.0] - 2025-10-08

### :boom: Breaking change
- fixed the length of the `ExternalSecrets` rendered by the Flight to 253 characters instead of previous 63
  - what changes?
    - if you happened to reuse an `ExternalSecret` by referencing the generated name, it was previously truncated to 63 characters
    - now you can use the full name as it was intended - `{service}--{component}--{env}--{secretStoreName}--{pathToSecret}`

## [1.0.0] - 2025--19

### :star: Moved the project to public GitHub repository! :rocket:

[unreleased]: https://github.com/ProRocketeers/yoke-chart/compare/1.11.0...HEAD
[1.11.0]: https://github.com/ProRocketeers/yoke-chart/compare/1.10.0...1.11.0
[1.10.0]: https://github.com/ProRocketeers/yoke-chart/compare/1.9.0...1.10.0
[1.9.0]: https://github.com/ProRocketeers/yoke-chart/compare/1.8.2...1.9.0
[1.8.2]: https://github.com/ProRocketeers/yoke-chart/compare/1.8.1...1.8.2
[1.8.1]: https://github.com/ProRocketeers/yoke-chart/compare/1.8.0...1.8.1
[1.8.0]: https://github.com/ProRocketeers/yoke-chart/compare/1.7.0...1.8.0
[1.7.0]: https://github.com/ProRocketeers/yoke-chart/compare/1.6.0...1.7.0
[1.6.0]: https://github.com/ProRocketeers/yoke-chart/compare/1.5.0...1.6.0
[1.5.0]: https://github.com/ProRocketeers/yoke-chart/compare/1.4.1...1.5.0
[1.4.1]: https://github.com/ProRocketeers/yoke-chart/compare/1.4.0...1.4.1
[1.4.0]: https://github.com/ProRocketeers/yoke-chart/compare/1.3.0...1.4.0
[1.3.0]: https://github.com/ProRocketeers/yoke-chart/compare/1.2.0...1.3.0
[1.2.0]: https://github.com/ProRocketeers/yoke-chart/compare/1.1.0...1.2.0
[1.1.0]: https://github.com/ProRocketeers/yoke-chart/compare/1.0.0...1.1.0
[1.0.0]: https://github.com/ProRocketeers/yoke-chart/tree/1.0.0
