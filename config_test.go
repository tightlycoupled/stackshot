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
Name: hellobuckets
TemplateURL: https://cfn-deploy-templates.s3.amazonaws.com/s3bucket-barebones.local.yaml`,
			out: &StackConfig{
				Name:                        "hellobuckets",
				TemplateBody:                "",
				TemplateURL:                 "https://cfn-deploy-templates.s3.amazonaws.com/s3bucket-barebones.local.yaml",
				DisableRollback:             false,
				EnableTerminationProtection: false,
				Parameters:                  nil,
				Tags:                        nil,
				Capabilities:                nil,
				OnFailure:                   "",
			},
		},

		// Defaults with template_body as valid YAML
		{
			doc: `---
Name: hellobuckets
TemplateBody:
  AWSTemplateFormatVersion: 2010-09-09
  Resources:
    S3Bucket:
      Type: AWS::S3::Bucket`,
			out: &StackConfig{
				Name:                        "hellobuckets",
				TemplateBody:                "AWSTemplateFormatVersion: \"2010-09-09\"\nResources:\n  S3Bucket:\n    Type: AWS::S3::Bucket\n",
				TemplateURL:                 "",
				TemplatePath:                "",
				DisableRollback:             false,
				EnableTerminationProtection: false,
				Parameters:                  nil,
				Tags:                        nil,
				Capabilities:                nil,
				OnFailure:                   "",
			},
		},

		// Defaults with template_path as valid YAML
		{
			doc: `---
Name: hellobuckets
TemplatePath: templates/s3bucket.yaml
`,
			out: &StackConfig{
				Name:                        "hellobuckets",
				TemplatePath:                "templates/s3bucket.yaml",
				TemplateURL:                 "",
				TemplateBody:                "",
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
Name: hellobuckets
TemplateURL: https://cfn-deploy-templates.s3.amazonaws.com/s3bucket-barebones.local.yaml
DisableRollback: true
EnableTerminationProtection: true
Capabilities:
- CAPABILITY_IAM
- CAPABILITY_AUTO_EXPAND
Parameters:
  hello: world
  VpcId: vpc-123abcde789
  SubnetGroupId: subnet-group-id
  MultiAz: true
Tags:
  environment: production
  team: alpha`,
			out: &StackConfig{
				Name:                        "hellobuckets",
				TemplateBody:                "",
				TemplatePath:                "",
				TemplateURL:                 "https://cfn-deploy-templates.s3.amazonaws.com/s3bucket-barebones.local.yaml",
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
Name: hellobuckets
TemplateURL: https://cfn-deploy-templates.s3.amazonaws.com/s3bucket-barebones.local.yaml
DisableRollback: true
OnFailure: DELETE`,
			err: errors.New("disable_rollback and on_failure cannot both be set"),
		},

		{
			doc: "Name: *does-not-exist",
			err: errors.New("failed to parse YAML: error converting YAML to JSON: yaml: unknown anchor 'does-not-exist' referenced"),
		},

		{
			doc: `---
Name: hellobuckets`,
			err: errors.New("Missing fields from document: template_url/template_body/template_path"),
		},

		{
			doc: `---
TemplateURL: https://example.com/mytemplate.yaml`,
			err: errors.New("Missing fields from document: name"),
		},

		{
			doc: `---
EnableTerminationProtection: true`,
			err: errors.New("Missing fields from document: name, template_url/template_body/template_path"),
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
