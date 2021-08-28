package command

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

type grafanaConfig struct {
	URL    string
	APIKey string
}

type grafanaOptions map[string]grafanaConfig

func (opts *grafanaOptions) String() string {
	values := opts.GetSlice()
	if len(values) == 0 {
		return "[]"
	}
	return strings.Join(values, ",")
}

func (opts *grafanaOptions) Type() string {
	return "<env>=<api-key>=<url>"
}

func (opts *grafanaOptions) Set(csv string) error {
	for _, split := range strings.Split(csv, ",") {
		err := opts.Append(split)
		if err != nil {
			return fmt.Errorf("flag value '%s': %w", split, err)
		}
	}
	return nil
}

// GetSlice returns the flag value list as an array of strings.
func (opts *grafanaOptions) GetSlice() []string {
	var values []string
	for env, value := range *opts {
		values = append(values, fmt.Sprintf("%s=<redacted>=%s", env, value.URL))
	}
	return values
}

// Append adds the specified value to the end of the flag value list.
func (opts *grafanaOptions) Append(value string) error {
	env, config, err := parseGrafanaConfig(value)
	if err != nil {
		return err
	}
	(*opts)[env] = config
	return nil
}

// Replace will fully overwrite any data currently in the flag value list.
func (opts *grafanaOptions) Replace(values []string) error {
	newOpts := grafanaOptions{}
	for _, value := range values {
		err := newOpts.Append(value)
		if err != nil {
			return err
		}
	}
	*opts = newOpts
	return nil
}

func parseGrafanaConfig(value string) (string, grafanaConfig, error) {
	splits := strings.SplitN(value, "=", 3)

	if len(splits) < 3 {
		return "", grafanaConfig{}, errors.New("value must be formatted as <env>=<api-key>=<url>")
	}

	return splits[0], grafanaConfig{
		APIKey: splits[1],
		URL:    splits[2],
	}, nil
}
