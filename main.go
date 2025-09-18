package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/ProRocketeers/yoke-chart/resources"
	"github.com/ProRocketeers/yoke-chart/schema"
	"github.com/go-playground/validator/v10"
	yaml "github.com/goccy/go-yaml"
	"github.com/yokecd/yoke/pkg/flight"
)

func main() {
	if err := run(); err != nil {
		// outputting to `stderr` actually shows the output in ArgoCD CMP plugin, `stdout` gets discarded
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	file := flag.String("file", "", "read from file instead of stdin (for debugging)")
	flag.Parse()

	var (
		values schema.InputValues
		err    error
	)
	if file != nil && *file != "" {
		f, err := os.Open(*file)
		if err != nil {
			return fmt.Errorf("error while reading file %v: %v", *file, err)
		}
		values, err = parseFromSource(f)
		if err != nil {
			return fmt.Errorf("error while parsing from file: %v", err)
		}
	} else {
		values, err = parseFromSource(os.Stdin)
		if err != nil {
			return fmt.Errorf("error while parsing from stdin: %v", err)
		}
	}

	deploymentValues, err := resources.PrepareDeploymentValues(values)

	if err != nil {
		return fmt.Errorf("error while preparing the deployment values: %v", err)
	}

	res, err := collectResources(
		deploymentValues,
		resources.CreateDeployment,
		resources.CreateService,
		resources.CreateIngress,
		resources.CreateServiceAccount,
		resources.CreatePVCs,
		resources.CreatePreDeploymentJob,
		resources.CreateCronjobs,
		resources.CreateExternalSecrets,
		resources.CreateHPA,
		resources.CreatePDB,
		resources.CreateDB,
		resources.CreateRBAC,
		resources.CreateConfigMaps,
		resources.CreateExtraManifests,
	)
	if err != nil {
		return fmt.Errorf("error while rendering the resources: %v", err)
	}

	return json.NewEncoder(os.Stdout).Encode(res)
}

func collectResources(values resources.DeploymentValues, creators ...func(resources.DeploymentValues) (bool, resources.ResourceCreator)) ([]flight.Resource, error) {
	resources := []flight.Resource{}
	for _, shouldCreateResource := range creators {
		if ok, create := shouldCreateResource(values); ok {
			if newResources, err := create(values); err != nil {
				return []flight.Resource{}, err
			} else {
				resources = append(resources, newResources...)
			}
		}

	}
	return resources, nil
}

func parseFromSource(r io.Reader) (schema.InputValues, error) {
	var values schema.InputValues
	bytes, err := io.ReadAll(r)
	if err != nil {
		return schema.InputValues{}, fmt.Errorf("stdin read error: %v", err)
	}
	if err := yaml.Unmarshal(bytes, &values); err != nil {
		return schema.InputValues{}, fmt.Errorf("unmarshalling error: %v", err)
	}
	// unmarshal doesn't validate fields being required (`string` vs `*string`), just parses the YAML into struct
	// to validate required fields or others, need the validator package too
	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(values); err != nil {
		return schema.InputValues{}, fmt.Errorf("validation error: %v", err)
	}
	return values, nil
}
