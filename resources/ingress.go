package resources

import (
	"maps"

	"github.com/ProRocketeers/yoke-chart/schema"
	"github.com/yokecd/yoke/pkg/flight"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func CreateIngress(values DeploymentValues) (bool, ResourceCreator) {
	enabled := values.Ingress != nil && *values.Ingress.Enabled == true
	return enabled, func(values DeploymentValues) ([]flight.Resource, error) {
		annotations, spec := prepareIngressValues(values)
		ingress := networkingv1.Ingress{
			TypeMeta: metav1.TypeMeta{
				APIVersion: networkingv1.SchemeGroupVersion.Identifier(),
				Kind:       "Ingress",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        serviceName(values.Metadata),
				Namespace:   values.Metadata.Namespace,
				Annotations: annotations,
				Labels:      commonLabels(values.Metadata),
			},
			Spec: spec,
		}
		return []flight.Resource{&ingress}, nil
	}
}

func prepareIngressValues(values DeploymentValues) (map[string]string, networkingv1.IngressSpec) {
	if values.Ingress.Simple == nil || *values.Ingress.Simple == true {
		return getSimpleIngressValues(values)
	}
	annotations := map[string]string{}
	v := values.Ingress.Variant.(schema.FullIngress)
	for name, value := range sortedMap(values.Ingress.Homepage) {
		annotations["gethomepage.dev/"+name] = value
	}
	maps.Copy(annotations, v.Annotations)
	return annotations, v.IngressSpec
}

func getSimpleIngressValues(values DeploymentValues) (annotations map[string]string, spec networkingv1.IngressSpec) {
	mainPort := values.Containers[0].Ports[0].Port
	host := values.Ingress.Variant.(schema.SimpleIngress).Host

	annotations = map[string]string{
		"kubernetes.io/ingress.class":                             "nginx",
		"traefik.ingress.kubernetes.io/router.entrypoints":        "websecure",
		"traefik.ingress.kubernetes.io/router.tls":                "true",
		"traefik.ingress.kubernetes.io/router.tls.certresolver":   "static",
		"traefik.ingress.kubernetes.io/router.tls.domains.0.main": host,
	}

	for name, value := range sortedMap(values.Ingress.Homepage) {
		annotations["gethomepage.dev/"+name] = value
	}

	return annotations, networkingv1.IngressSpec{
		Rules: []networkingv1.IngressRule{
			{
				Host: host,
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{
							{
								Path:     "/",
								PathType: ptr.To(networkingv1.PathTypeImplementationSpecific),
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: serviceName(values.Metadata),
										Port: networkingv1.ServiceBackendPort{
											Number: int32(mainPort),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
