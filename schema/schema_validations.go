package schema

import "fmt"

func CustomValidations(values InputValues) error {
	// here will be any arbitrary custom validations that are difficult or impossible to express otherwise

	// 1. Kind should either be `null`, or `Deployment` or `StatefulSet`
	if err := validateKindValue(values); err != nil {
		return err
	}
	return nil
}

func validateKindValue(values InputValues) error {
	if values.Kind != nil && *values.Kind != "Deployment" && *values.Kind != "StatefulSet" {
		return fmt.Errorf("invalid kind %v", *values.Kind)
	}
	return nil
}
