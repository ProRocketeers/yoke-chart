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

[unreleased]: https://github.com/ProRocketeers/yoke-chart/compare/1.5.0...HEAD
[1.5.0]: https://github.com/ProRocketeers/yoke-chart/compare/1.4.1...1.5.0
[1.4.1]: https://github.com/ProRocketeers/yoke-chart/compare/1.4.0...1.4.1
[1.4.0]: https://github.com/ProRocketeers/yoke-chart/compare/1.3.0...1.4.0
[1.3.0]: https://github.com/ProRocketeers/yoke-chart/compare/1.2.0...1.3.0
[1.2.0]: https://github.com/ProRocketeers/yoke-chart/compare/1.1.0...1.2.0
[1.1.0]: https://github.com/ProRocketeers/yoke-chart/compare/1.0.0...1.1.0
[1.0.0]: https://github.com/ProRocketeers/yoke-chart/tree/1.0.0
