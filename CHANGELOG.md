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

## [1.0.0] - 2025-09-19

### :star: Moved the project to public GitHub repository! :rocket:

[unreleased]: https://github.com/ProRocketeers/yoke-chart/compare/1.9.0...HEAD
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
