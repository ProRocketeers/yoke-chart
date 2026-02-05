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

[unreleased]: https://github.com/ProRocketeers/yoke-chart/compare/1.3.0...HEAD
[1.3.0]: https://github.com/ProRocketeers/yoke-chart/compare/1.2.0...1.3.0
[1.2.0]: https://github.com/ProRocketeers/yoke-chart/compare/1.1.0...1.2.0
[1.1.0]: https://github.com/ProRocketeers/yoke-chart/compare/1.0.0...1.1.0
[1.0.0]: https://github.com/ProRocketeers/yoke-chart/tree/1.0.0
