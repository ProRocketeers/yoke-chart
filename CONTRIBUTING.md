# Contributing
In case anyone wanted to make any changes, here's the general idea how it works.

## How it works
The entrypoint is `main.go`. The application **must** be designed so that it reads YAML or JSON input from standard input (since it's a Yoke chart, it will be passed this input likely from the ArgoCD CMP plugin container on `stdin`). The app's goal is to take this input, do whatever with it, and output an array of Kubernetes manifests in JSON format on `stdout` (or write errors to `stderr` and exit with nonzero code).

The way we've implemented it is we first read the values from `stdin` and unmarshal it from YAML into an input struct `schema.InputValues` (in `schema/schema.go`). This struct represents the *input*, it's not the format of data individual functions work with. In this file, there are YAML/JSON mappings, types, sometimes custom unmarshalling logic (mainly for "polymorphic" types, such as Volumes or Ingress).

After this parsing, we run it through a validator that checks mainly required values (based on struct tags), but it can also check a fuckton of other things if needed. Then we run it through a function that transforms this input into structure better suited for processing - `resources.DeploymentValues`:

- a lot of properties are passed *as-is*, like `autoscaling`, `strategy`, `podDisruptionBudget`, `ingress` etc.
- what happens here is mostly preparing the containers - in the main deployment, Job or CronJobs
  - image validations
  - but mostly just reorganizing the data a little

This structure is then an input to functions that return the Kube manifests. Since those often times depend on a condition to be met, there's a pattern estabilished:

- the top level function like `CreateIngress` takes the `DeploymentValues` and returns 2 values:
  - boolean value indicating if the resource(s) should even be created
  - function that again takes the `DeploymentValues` and returns the actual resources
- we then take these functions, run them one by one and if they "should create" something, then actually call the function and collect the resources

The resources are then encoded as JSON onto `stdout`.

## Maintenance
### Testing
Each important or frequently used thing is properly unit tested (in Go, there's a convention of test files being next to the tested file). 

Any added feature should have some tests attached, and made sure to not break (or fix) whatever was working before.
### Docs
- keep `CHANGELOG` up to date
- the `values.yaml` here represents the input values in the format the users will write (equivalent of `schema.go`, just more readable), that should be kept up-to-date
- if possible, maintain `README` but it's not that high priority

### Releasing
1. Update the `Version` variable in `resources/setup.go` to your **new** desired version
2. Adequately update `CHANGELOG` and `values.yaml` with an example/explanation
3. Commit, push, in Gitlab, create tag with the same version you used
4. That's it, Gitlab CI will unit test the package, compile it and push it as OCI artifact into Harbor - `yoke/base-chart` (under the specified version + `latest` tag)