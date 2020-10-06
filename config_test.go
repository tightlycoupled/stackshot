package stackshot

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
)

func equalErrors(a, b error) bool {
	if a == nil {
		return b == nil
	}
	if b == nil {
		return a == nil
	}
	return a.Error() == b.Error()
}

func TestYAMLParsing(t *testing.T) {
	tests := []struct {
		doc string
		out interface{}
		err error
	}{

		// Defaults
		{
			doc: `---
name: hellobuckets
template: https://cfn-deploy-templates.s3.amazonaws.com/s3bucket-barebones.local.yaml`,
			out: &StackConfig{
				Name:                        "hellobuckets",
				Template:                    "https://cfn-deploy-templates.s3.amazonaws.com/s3bucket-barebones.local.yaml",
				DisableRollback:             false,
				EnableTerminationProtection: false,
				Parameters:                  nil,
				Tags:                        nil,
				Capabilities:                nil,
				OnFailure:                   "",
			},
		},

		// Fully filled out
		{
			doc: `---
name: hellobuckets
template: https://cfn-deploy-templates.s3.amazonaws.com/s3bucket-barebones.local.yaml
disable_rollback: true
enable_termination_protection: true
capabilities:
- CAPABILITY_IAM
- CAPABILITY_AUTO_EXPAND
parameters:
  hello: world
  VpcId: vpc-123abcde789
  SubnetGroupId: subnet-group-id
  MultiAz: true
tags:
  environment: production
  team: alpha`,
			out: &StackConfig{
				Name:                        "hellobuckets",
				Template:                    "https://cfn-deploy-templates.s3.amazonaws.com/s3bucket-barebones.local.yaml",
				DisableRollback:             true,
				EnableTerminationProtection: true,
				Parameters: map[string]string{
					"hello":         "world",
					"VpcId":         "vpc-123abcde789",
					"SubnetGroupId": "subnet-group-id",
					"MultiAz":       "true",
				},
				Tags: map[string]string{
					"environment": "production",
					"team":        "alpha",
				},
				Capabilities: []string{"CAPABILITY_IAM", "CAPABILITY_AUTO_EXPAND"},
			},
		},

		{
			doc: `---
name: hellobuckets
template: https://cfn-deploy-templates.s3.amazonaws.com/s3bucket-barebones.local.yaml
disable_rollback: true
on_failure: DELETE`,
			err: errors.New("disable_rollback and on_failure cannot both be set"),
		},

		{
			doc: "name: *does-not-exist",
			err: errors.New("failed to parse YAML: error converting YAML to JSON: yaml: unknown anchor 'does-not-exist' referenced"),
		},

		{
			doc: `---
name: hellobuckets`,
			err: errors.New("Missing fields from document: template"),
		},

		{
			doc: `---
template: https://example.com/mytemplate.yaml`,
			err: errors.New("Missing fields from document: name"),
		},

		{
			doc: `---
enable_termination_protection: true`,
			err: errors.New("Missing fields from document: name, template"),
		},
	}

	for i, test := range tests {
		t.Run(
			fmt.Sprintf("#%d", i),
			func(t *testing.T) {
				config, err := NewStackFromYAML([]byte(test.doc))
				if test.err != nil && err == nil {
					t.Fatalf("Expected error: %s.\nGot none.", test.err)
				}
				if err != nil && !equalErrors(err, test.err) {
					t.Fatalf("Expected error: %#v, got: %#v", test.err, err)
				}

				if test.out == nil {
					return
				}

				if !cmp.Equal(config, test.out) {
					t.Errorf("Expected:\n%#+v\nGot:\n%#+v\n", test.out, config)
				}
			},
		)
	}

}
