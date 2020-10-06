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
	Template     string
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

func (s *StackConfig) verifyRequiredFields() error {
	missingFields := []string{}
	if s.Name == "" {
		missingFields = append(missingFields, "name")
	}

	if s.Template == "" {
		missingFields = append(missingFields, "template")
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
