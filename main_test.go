package main

import (
	"strings"
	"testing"

	"github.com/ProRocketeers/yoke-chart/schema"
	"github.com/lithammer/dedent"
	"github.com/stretchr/testify/assert"
)

// meant for testing the parsing mechanism and custom validation logic etc.
func TestMain(t *testing.T) {
	type CaseConfig struct {
		// can contain arbitrary whitespace around it to make it pretty in code
		// but watch tabs/spaces => YAML can't handle tabs that are default indent in Go
		Input   string
		Asserts func(*testing.T, schema.InputValues, error)
	}

	cases := map[string]CaseConfig{
		"passes empty Kind": {
			Input: `
        namespace: foo
        service: foo
        component: bar
        environment: test

        image:
          repository: foo
          tag: bleh
      `,
			Asserts: func(t *testing.T, iv schema.InputValues, err error) {
				assert.NoError(t, err)
			},
		},
		"passes Kind = Deployment": {
			Input: `
        namespace: foo
        service: foo
        component: bar
        environment: test

        image:
          repository: foo
          tag: bleh

        kind: Deployment
      `,
			Asserts: func(t *testing.T, iv schema.InputValues, err error) {
				assert.NoError(t, err)
			},
		},
		"passes Kind = StatefulSet": {
			Input: `
        namespace: foo
        service: foo
        component: bar
        environment: test

        image:
          repository: foo
          tag: bleh

        kind: StatefulSet
      `,
			Asserts: func(t *testing.T, iv schema.InputValues, err error) {
				assert.NoError(t, err)
			},
		},
		"fails a non-enum Kind value": {
			Input: `
        namespace: foo
        service: foo
        component: bar
        environment: test

        image:
          repository: foo
          tag: bleh

        kind: foo
      `,
			Asserts: func(t *testing.T, iv schema.InputValues, err error) {
				assert.Error(t, err)
			},
		},
		"fails when both ingress and httpRoute are specified": {
			Input: `
        namespace: foo
        service: foo
        component: bar
        environment: test

        image:
          repository: foo
          tag: bleh

        ingress:
          enabled: true

        httpRoute:
          enabled: true
      `,
			Asserts: func(t *testing.T, iv schema.InputValues, err error) {
				assert.Error(t, err)
			},
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			input := dedent.Dedent(tc.Input)
			reader := strings.NewReader(strings.TrimSpace(input))

			values, err := parseFromSource(reader)
			tc.Asserts(t, values, err)
		})
	}
}
