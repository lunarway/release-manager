package command

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/lunarway/release-manager/cmd/hamctl/command/actions"
	"github.com/lunarway/release-manager/cmd/hamctl/command/completion"
	"github.com/lunarway/release-manager/internal/http"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// NewRoot returns a new instance of a hamctl command.
func NewRoot(version *string) (*cobra.Command, error) {
	idpURL := os.Getenv("HAMCTL_OAUTH_IDP_URL")
	if idpURL == "" {
		return nil, errors.New("no HAMCTL_OAUTH_IDP_URL env var set")
	}
	clientID := os.Getenv("HAMCTL_OAUTH_CLIENT_ID")
	if clientID == "" {
		return nil, errors.New("no HAMCTL_OAUTH_CLIENT_ID env var set")
	}

	authenticator := http.NewUserAuthenticator(clientID, idpURL)

	var service string
	client := http.Client{
		Metadata: http.Metadata{
			CLIVersion: *version,
		},
	}

	releaseClient := actions.NewReleaseHttpClient(&client)

	var command = &cobra.Command{
		Use:                    "hamctl",
		Short:                  "hamctl controls a release manager server",
		BashCompletionFunction: completion.Hamctl,
		PersistentPreRunE: func(c *cobra.Command, args []string) error {
			if c.Name() == "version" || c.Name() == "completion" || c.Name() == "login" {
				return nil
			}
			defaultShuttleString(shuttleSpecFromFile, &service, func(s *shuttleSpec) string {
				return s.Vars.Service
			})

			if client.BaseURL == "" {
				client.BaseURL = os.Getenv("HAMCTL_URL")
			}

			var missingFlags []string
			if service == "" {
				missingFlags = append(missingFlags, "service")
			}

			if len(missingFlags) != 0 {
				return errors.Errorf(`required flag(s) "%s" not set`, strings.Join(missingFlags, `", "`))
			}
			return nil
		},
		Run: func(c *cobra.Command, args []string) {
			c.HelpFunc()(c, args)
		},
	}
	loggerFunc := func(f string, args ...interface{}) {
		fmt.Printf(f, args...)
	}
	command.AddCommand(
		NewCompletion(command),
		NewDescribe(&client, &service),
		NewPolicy(&client, &service),
		NewPromote(&client, &service, releaseClient),
		NewRelease(&client, &service, loggerFunc, releaseClient),
		NewRollback(&client, &service, loggerFunc, SelectRollbackReleaseFunc, releaseClient),
		NewStatus(&client, &service),
		NewVersion(*version),
		Login(authenticator),
	)
	command.PersistentFlags().DurationVar(&client.Timeout, "http-timeout", 120*time.Second, "HTTP request timeout")
	command.PersistentFlags().StringVar(&client.BaseURL, "http-base-url", "", "address of the http release manager server")
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
