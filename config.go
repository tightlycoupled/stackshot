package stackshot

import (
	"fmt"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
)

func NewStackFromYAML(doc []byte) (*StackConfig, error) {
	s := StackConfig{}

	if err := yaml.Unmarshal([]byte(doc), &s); err != nil {
		return nil, errors.Wrap(err, "failed to parse YAML")
	}

	if err := s.verifyRequiredFields(); err != nil {
		return nil, err
	}

	return &s, nil
}

type templateReader interface {
	ReadFile(string) ([]byte, error)
}

type StackConfig struct {
	Name         string
	TemplateURL  string
	TemplatePath string
	TemplateBody templateBody
	Parameters   map[string]string
	Tags         map[string]string
	Capabilities []string

	// Settings for CreateStack()
	DisableRollback bool

	// Settings for CreateStack()
	EnableTerminationProtection bool

	// Settings for CreateStack()
	OnFailure string
}

func (s *StackConfig) verifyRequiredFields() error {
	missingFields := []string{}
	if s.Name == "" {
		missingFields = append(missingFields, "name")
	}

	if s.TemplateURL == "" && s.TemplateBody == "" && s.TemplatePath == "" {
		missingFields = append(missingFields, "template_url/template_body/template_path")
	}

	if len(missingFields) != 0 {
		return fmt.Errorf(
			"Missing fields from document: %s",
			strings.Join(missingFields, ", "),
		)
	}

	if s.OnFailure != "" && s.DisableRollback {
		return fmt.Errorf("disable_rollback and on_failure cannot both be set")
	}

	return nil
}

type templateBody string

// Convert data back into YAML to match what a user will write a template in.
func (t *templateBody) UnmarshalJSON(data []byte) error {
	body, err := yaml.JSONToYAML(data)
	if err != nil {
		return err
	}

	*t = templateBody(string(body))
	return nil
}
