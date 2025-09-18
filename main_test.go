package main

import (
	"strings"
	"testing"

	"github.com/ProRocketeers/yoke-chart/schema"
	"github.com/lithammer/dedent"
)

// meant for testing the parsing mechanism and custom validation logic etc.
func TestMain(t *testing.T) {
	type CaseConfig struct {
		// can contain arbitrary whitespace around it to make it pretty in code
		// but watch tabs/spaces => YAML can't handle tabs that are default indent in Go
		Input   string
		Asserts func(*testing.T, schema.InputValues, error)
	}

	cases := map[string]CaseConfig{}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			input := dedent.Dedent(tc.Input)
			reader := strings.NewReader(strings.TrimSpace(input))

			values, err := parseFromSource(reader)
			tc.Asserts(t, values, err)
		})
	}
}
