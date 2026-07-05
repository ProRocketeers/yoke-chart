package main

import (
	"strings"
	"testing"

	"github.com/ProRocketeers/yoke-chart/schema"
	"github.com/lithammer/dedent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		"fails a node port out of its allowed range - lower bound": {
			Input: `
        namespace: foo
        service: foo
        component: bar
        environment: test

        image:
          repository: foo
          tag: bleh

        ports:
          - port: 80
            nodePort: 2500
      `,
			Asserts: func(t *testing.T, iv schema.InputValues, err error) {
				assert.Error(t, err)
			},
		},
		"fails a node port out of its allowed range - upper bound": {
			Input: `
        namespace: foo
        service: foo
        component: bar
        environment: test

        image:
          repository: foo
          tag: bleh

        ports:
          - port: 80
            nodePort: 34000
      `,
			Asserts: func(t *testing.T, iv schema.InputValues, err error) {
				assert.Error(t, err)
			},
		},
		"passes httpRoute (singular) alone": {
			Input: `
        namespace: foo
        service: foo
        component: bar
        environment: test

        image:
          repository: foo
          tag: bleh

        httpRoute:
          parentRefs:
            - name: prod-gateway
          hostnames:
            - myapp.example.com
          rules:
            - backendRefs:
                - name: myapp
                  port: 8080
      `,
			Asserts: func(t *testing.T, iv schema.InputValues, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, iv.HTTPRoute)
				assert.Nil(t, iv.HTTPRoutes)
			},
		},
		"passes httpRoutes (plural) alone": {
			Input: `
        namespace: foo
        service: foo
        component: bar
        environment: test

        image:
          repository: foo
          tag: bleh

        httpRoutes:
          public:
            parentRefs:
              - name: prod-gateway
            hostnames:
              - api.example.com
            rules:
              - backendRefs:
                  - name: myapp
                    port: 8080
      `,
			Asserts: func(t *testing.T, iv schema.InputValues, err error) {
				assert.NoError(t, err)
				assert.Nil(t, iv.HTTPRoute)
				assert.Contains(t, iv.HTTPRoutes, "public")
			},
		},
		"accepts a single volume mount object (backwards compatible shape)": {
			Input: `
        namespace: foo
        service: payments-api
        component: bar
        environment: test

        image:
          repository: foo
          tag: bleh

        volumes:
          app-config:
            type: configMap
            configMapName: payments-api-config
            mounts:
              main:
                containerPath: /etc/payments-api
      `,
			Asserts: func(t *testing.T, iv schema.InputValues, err error) {
				assert.NoError(t, err)
				require.Len(t, iv.Volumes["app-config"].Mounts["main"], 1)
				assert.Equal(t, "/etc/payments-api", iv.Volumes["app-config"].Mounts["main"][0].ContainerPath)
			},
		},
		"accepts multiple volume mounts as a list for the same container": {
			Input: `
        namespace: foo
        service: payments-api
        component: bar
        environment: test

        image:
          repository: foo
          tag: bleh

        volumes:
          app-config:
            type: configMap
            configMapName: payments-api-config
            mounts:
              main:
                - containerPath: /etc/payments-api/app.yaml
                  volumePath: app.yaml
                - containerPath: /etc/payments-api/logging.yaml
                  volumePath: logging.yaml
      `,
			Asserts: func(t *testing.T, iv schema.InputValues, err error) {
				assert.NoError(t, err)
				mounts := iv.Volumes["app-config"].Mounts["main"]
				require.Len(t, mounts, 2)
				assert.Equal(t, "/etc/payments-api/app.yaml", mounts[0].ContainerPath)
				assert.Equal(t, "/etc/payments-api/logging.yaml", mounts[1].ContainerPath)
			},
		},
		"fails when a volume mount inside a list is missing its containerPath": {
			Input: `
        namespace: foo
        service: payments-api
        component: bar
        environment: test

        image:
          repository: foo
          tag: bleh

        volumes:
          app-config:
            type: configMap
            configMapName: payments-api-config
            mounts:
              main:
                - containerPath: /etc/payments-api/app.yaml
                - volumePath: logging.yaml
      `,
			Asserts: func(t *testing.T, iv schema.InputValues, err error) {
				assert.Error(t, err)
			},
		},
		"fails on an invalid mountPropagation value": {
			Input: `
        namespace: foo
        service: payments-api
        component: bar
        environment: test

        image:
          repository: foo
          tag: bleh

        volumes:
          scratch:
            type: local
            mounts:
              main:
                containerPath: /var/scratch
                mountPropagation: Sideways
      `,
			Asserts: func(t *testing.T, iv schema.InputValues, err error) {
				assert.Error(t, err)
			},
		},
		"fails when both httpRoute and httpRoutes are set": {
			Input: `
        namespace: foo
        service: foo
        component: bar
        environment: test

        image:
          repository: foo
          tag: bleh

        httpRoute:
          parentRefs:
            - name: prod-gateway
          hostnames:
            - myapp.example.com
          rules:
            - backendRefs:
                - name: myapp
                  port: 8080
        httpRoutes:
          public:
            parentRefs:
              - name: prod-gateway
            hostnames:
              - api.example.com
            rules:
              - backendRefs:
                  - name: myapp
                    port: 8080
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
