package resources

import (
	"fmt"
	"strings"
	"text/template"
	"text/template/parse"

	sprig "github.com/Masterminds/sprig/v3"
)

// TemplateContext is exposed to extraManifests leaf-value templates as the template's ".".
type TemplateContext struct {
	Values  DeploymentValues
	Outputs Outputs
}

// additionalNonhermeticSprigFuncs plugs gaps in Sprig's own HermeticTxtFuncMap: these still read
// wall-clock time or crypto/math randomness internally, but Sprig's denylist doesn't cover them.
var additionalNonhermeticSprigFuncs = []string{
	"ago",                // compares its argument against time.Now()
	"shuffle", "randInt", // math/rand
	"bcrypt", "htpasswd", // random salt via crypto/rand
	"genPrivateKey", "genCA", "genCAWithKey",
	"genSelfSignedCert", "genSelfSignedCertWithKey",
	"genSignedCert", "genSignedCertWithKey", // key/cert generation via crypto/rand
	"encryptAES", // random IV via crypto/rand (decryptAES is fine, no randomness)
}

func templateFuncs() template.FuncMap {
	funcs := sprig.HermeticTxtFuncMap()
	for _, name := range additionalNonhermeticSprigFuncs {
		delete(funcs, name)
	}
	funcs["serviceName"] = serviceName
	return funcs
}

// renderLeafString renders s as a Go template if it contains a template action, otherwise
// returns it unchanged.
func renderLeafString(s string, ctx TemplateContext) (string, error) {
	if !strings.Contains(s, "{{") {
		return s, nil
	}

	tmpl, err := template.New("extraManifest").Funcs(templateFuncs()).Option("missingkey=error").Parse(s)
	if err != nil {
		return "", fmt.Errorf("error parsing template %q: %w", s, err)
	}

	if err := rejectControlFlow(tmpl); err != nil {
		return "", fmt.Errorf("in template %q: %w", s, err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return "", fmt.Errorf("error executing template %q: %w", s, err)
	}
	return buf.String(), nil
}

// rejectControlFlow enforces that extraManifests templates only ever substitute a scalar value,
// never restructure the surrounding manifest the way Helm's {{range}}/{{if}} over YAML text can.
func rejectControlFlow(t *template.Template) error {
	if len(t.Templates()) > 1 {
		return fmt.Errorf("{{define}}/{{block}} are not supported: templating is limited to a single leaf value, not document structure")
	}
	return rejectControlFlowNodes(t.Root.Nodes)
}

func rejectControlFlowNodes(nodes []parse.Node) error {
	for _, n := range nodes {
		switch n.(type) {
		case *parse.IfNode:
			return fmt.Errorf("{{if}} is not supported: templating is limited to a single leaf value, not document structure")
		case *parse.RangeNode:
			return fmt.Errorf("{{range}} is not supported: templating is limited to a single leaf value, not document structure")
		case *parse.WithNode:
			return fmt.Errorf("{{with}} is not supported: templating is limited to a single leaf value, not document structure")
		case *parse.TemplateNode:
			return fmt.Errorf("{{template}}/{{block}} are not supported: templating is limited to a single leaf value, not document structure")
		case *parse.BreakNode:
			return fmt.Errorf("{{break}} is not supported outside of {{range}}")
		case *parse.ContinueNode:
			return fmt.Errorf("{{continue}} is not supported outside of {{range}}")
		}
	}
	return nil
}

// templateLeafValues recursively walks an already-parsed YAML/JSON value (maps, slices, scalars)
// and renders any string leaf as a template, leaving the surrounding structure untouched. This is
// what keeps templating confined to leaf values: by the time this runs, the document is already
// fully parsed, so there's no raw text left for a template to restructure.
func templateLeafValues(v any, ctx TemplateContext) (any, error) {
	switch val := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, item := range val {
			rendered, err := templateLeafValues(item, ctx)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", k, err)
			}
			out[k] = rendered
		}
		return out, nil
	case []any:
		out := make([]any, len(val))
		for i, item := range val {
			rendered, err := templateLeafValues(item, ctx)
			if err != nil {
				return nil, fmt.Errorf("[%d]: %w", i, err)
			}
			out[i] = rendered
		}
		return out, nil
	case string:
		return renderLeafString(val, ctx)
	default:
		// Only strings are ever templated, by construction, not just by choice: `{{ }}` only
		// survives YAML parsing inside a quoted (i.e. already-string) scalar, since unquoted it
		// starts with `{`, a flow-mapping indicator. And even if it didn't, text/template.Execute
		// only gives back formatted text, never the pipeline's original typed Go value, so a
		// non-string leaf couldn't come back out as the same type anyway. Bools/numbers pass
		// through untouched below.
		return val, nil
	}
}
