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

type StackConfig struct {
	Name         string
	TemplateURL  string       `json:"template_url"`
	TemplateBody templateBody `json:"template_body"`
	Parameters   map[string]string
	Tags         map[string]string
	Capabilities []string

	// Settings for CreateStack()
	DisableRollback bool `json:"disable_rollback"`

	// Settings for CreateStack()
	EnableTerminationProtection bool `json:"enable_termination_protection"`

	// Settings for CreateStack()
	OnFailure string `json:"on_failure"`
}

func (s *StackConfig) SetBody(body string) {
	s.TemplateBody = templateBody(body)
}

func (s *StackConfig) verifyRequiredFields() error {
	missingFields := []string{}
	if s.Name == "" {
		missingFields = append(missingFields, "name")
	}

	if s.TemplateURL == "" && s.TemplateBody == "" {
		missingFields = append(missingFields, "template_url/template_body")
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

// Custom parser for `template_body` key. Since we're using ghodss/yaml, data
// is actually JSON. Therefore, there was a data transformation that occurred.
// I don't think there's a problem with this, but it might be a problem in the
// future if we wanted to introduce syntax checking.
func (t *templateBody) UnmarshalJSON(data []byte) error {
	*t = templateBody(string(data))
	return nil
}
