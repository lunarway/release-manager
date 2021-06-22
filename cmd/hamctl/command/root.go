package command

import (
	"os"
	"strings"
	"time"

	"github.com/go-openapi/runtime"
	"github.com/lunarway/release-manager/cmd/hamctl/command/completion"
	"github.com/lunarway/release-manager/generated/http/client"
	"github.com/lunarway/release-manager/internal/git"
	"github.com/lunarway/release-manager/internal/http"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// NewRoot returns a new instance of a hamctl command.
func NewRoot(version *string) (*cobra.Command, error) {
	var (
		service      string
		email        string
		clientConfig = http.Config{
			CLIVersion: *version,
		}
		client     = new(client.ReleaseManagerServerAPI)
		clientAuth = new(runtime.ClientAuthInfoWriter)
	)

	var command = &cobra.Command{
		Use:                    "hamctl",
		Short:                  "hamctl controls a release manager server",
		BashCompletionFunction: completion.Hamctl,
		PersistentPreRunE: func(c *cobra.Command, args []string) error {
			// all commands but version and completion requires the "service" flag
			// if this is one of them, skip the check
			if c.Name() == "version" || c.Name() == "completion" {
				return nil
			}
			defaultShuttleString(shuttleSpecFromFile, &service, func(s *shuttleSpec) string {
				return s.Vars.Service
			})

			if clientConfig.BaseURL == "" {
				clientConfig.BaseURL = os.Getenv("HAMCTL_URL")
			}

			if clientConfig.AuthToken == "" {
				clientConfig.AuthToken = os.Getenv("HAMCTL_AUTH_TOKEN")
			}

			var missingFlags []string
			if service == "" {
				missingFlags = append(missingFlags, "service")
			}
			if email == "" {
				_, email, _ = git.CommitterDetails()
				if email == "" {
					missingFlags = append(missingFlags, "user-email")
				}
			}
			clientConfig.CallerEmail = email
			if len(missingFlags) != 0 {
				return errors.Errorf(`required flag(s) "%s" not set`, strings.Join(missingFlags, `", "`))
			}

			localClient, localClientAuth := http.NewClient(&clientConfig)
			// assign the created client to the existing pointer values to ensure
			// references passed to sub commands are updated
			*client = *localClient
			*clientAuth = localClientAuth
			return nil
		},
		Run: func(c *cobra.Command, args []string) {
			c.HelpFunc()(c, args)
		},
	}
	command.AddCommand(
		NewCompletion(command),
		NewDescribe(client, clientAuth, &service),
		// NewPolicy(&client, &service),
		// NewPromote(&client, &service),
		// NewRelease(&client, &service, func(f string, args ...interface{}) {
		// 	fmt.Printf(f, args...)
		// }),
		// NewRollback(&client, &service),
		// NewStatus(&client, &service),
		NewVersion(*version),
	)
	command.PersistentFlags().DurationVar(&clientConfig.Timeout, "http-timeout", 120*time.Second, "HTTP request timeout")
	command.PersistentFlags().StringVar(&clientConfig.BaseURL, "http-base-url", "", "address of the http release manager server")
	command.PersistentFlags().StringVar(&clientConfig.AuthToken, "http-auth-token", "", "auth token for the http service")
	command.PersistentFlags().StringVar(&service, "service", "", "service name to execute commands for")
	command.PersistentFlags().StringVar(&email, "user-email", "", "email of user performing the command (defaults to Git configurated user.email)")

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
