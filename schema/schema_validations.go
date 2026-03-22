package schema

import "fmt"

func CustomValidations(values InputValues) error {
	// here will be any arbitrary custom validations that are difficult or impossible to express otherwise

	// 1. Kind should either be `null`, or `Deployment` or `StatefulSet`
	if err := validateKindValue(values); err != nil {
		return err
	}

	// 2. NodePorts (if specified) must be between 30000 and 32767
	if err := validateNodePortRange(values); err != nil {
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

func validateNodePortRange(values InputValues) error {
	for _, port := range values.Ports {
		if port.NodePort != nil && (*port.NodePort < 30000 || *port.NodePort > 32767) {
			return fmt.Errorf("node port on port %d must be between 30000 and 32767", port.Port)
		}
	}
	for name, sidecar := range values.Sidecars {
		for _, port := range sidecar.Ports {
			if port.NodePort != nil && (*port.NodePort < 30000 || *port.NodePort > 32767) {
				return fmt.Errorf("node port on port %d of sidecar %s must be between 30000 and 32767", port.Port, name)
			}
		}
	}
	return nil
}
