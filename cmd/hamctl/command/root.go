package command

import (
	"os"
	"strings"
	"time"

	"github.com/lunarway/release-manager/cmd/hamctl/command/completion"
	"github.com/lunarway/release-manager/internal/git"
	"github.com/lunarway/release-manager/internal/http"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// NewCommand returns a new instance of a hamctl command.
func NewCommand(version *string) (*cobra.Command, error) {
	_, email, err := git.CommitterDetails()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to lookup git credentials")
	}
	client := http.Client{
		Metadata: http.Metadata{
			CLIVersion:  *version,
			CallerEmail: email,
		},
	}
	var service string
	var command = &cobra.Command{
		Use:                    "hamctl",
		Short:                  "hamctl controls a release manager server",
		BashCompletionFunction: completion.Hamctl,
		PersistentPreRunE: func(c *cobra.Command, args []string) error {
			// all commands but version and completion requires the "service" flag
			// if this is thee version command, skip the check
			if c.Name() == "version" || c.Name() == "completion" {
				return nil
			}
			defaultShuttleString(shuttleSpecFromFile, &service, func(s *shuttleSpec) string {
				return s.Vars.Service
			})
			if service == "" {
				return errors.New("required flag(s) \"service\" not set")
			}
			return nil
		},
		Run: func(c *cobra.Command, args []string) {
			c.HelpFunc()(c, args)
		},
	}
	command.AddCommand(NewPromote(&client, &service))
	command.AddCommand(NewRelease(&client, &service))
	command.AddCommand(NewStatus(&client, &service))
	command.AddCommand(NewRollback(&client, &service))
	command.AddCommand(NewPolicy(&client, &service))
	command.AddCommand(NewCompletion(command))
	command.PersistentFlags().DurationVar(&client.Timeout, "http-timeout", 30*time.Second, "HTTP request timeout")
	command.PersistentFlags().StringVar(&client.BaseURL, "http-base-url", "https://release-manager.dev.lunarway.com", "address of the http release manager server")
	command.PersistentFlags().StringVar(&client.Metadata.AuthToken, "http-auth-token", os.Getenv("HAMCTL_AUTH_TOKEN"), "auth token for the http service")
	command.PersistentFlags().StringVar(&service, "service", "", "service name to execute commands for")
	return command, nil
}

type shuttleSpec struct {
	Vars shuttleSpecVars
}

type shuttleSpecVars struct {
	Service string `yaml:"service"`
	K8S     struct {
		Namespace string `yaml:"namespace"`
	} `yaml:"k8s"`
}

// shuttleSpecFromFile tries to read a shuttle specification.
func shuttleSpecFromFile() (shuttleSpec, bool) {
	f, err := os.Open("shuttle.yaml")
	if err != nil {
		return shuttleSpec{}, false
	}
	var spec shuttleSpec
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&spec)
	if err != nil {
		return shuttleSpec{}, false
	}
	return spec, true
}

// defaultShuttleString writes a value from a shuttle specification to flagValue
// if the provided flagValue is empty and the value in the spec is set.
func defaultShuttleString(shuttleLocator func() (shuttleSpec, bool), flagValue *string, f func(s *shuttleSpec) string) {
	if flagValue == nil {
		return
	}
	t := strings.TrimSpace(*flagValue)
	if t != "" {
		return
	}
	spec, ok := shuttleLocator()
	if !ok {
		return
	}
	t = f(&spec)
	if t != "" {
		*flagValue = t
	}
}
