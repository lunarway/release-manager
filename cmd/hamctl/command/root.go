package command

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/lunarway/release-manager/internal/http"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// NewCommand returns a new instance of a hamctl command.
func NewCommand() (*cobra.Command, error) {
	var client http.Client
	var service string
	var command = &cobra.Command{
		Use:   "hamctl",
		Short: "hamctl controls a release manager server",
		PersistentPreRunE: func(c *cobra.Command, args []string) error {
			service = strings.TrimSpace(service)
			if service == "" {
				service = readShuttleService()
			}
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
	command.AddCommand(NewPolicy(&client, &service))
	command.PersistentFlags().DurationVar(&client.Timeout, "http-timeout", 20*time.Second, "HTTP request timeout")
	command.PersistentFlags().StringVar(&client.BaseURL, "http-base-url", "https://release-manager.dev.lunarway.com", "address of the http release manager server")
	command.PersistentFlags().StringVar(&client.AuthToken, "http-auth-token", os.Getenv("HAMCTL_AUTH_TOKEN"), "auth token for the http service")
	command.PersistentFlags().StringVar(&service, "service", "", "service name to execute commands for")
	return command, nil
}

type shuttleSpec struct {
	Vars struct {
		Service string `yaml:"service"`
	}
}

// readShuttleService tries to read the service name from a shuttle
// specification.
// If the file is not found or cannot be parsed, we just fall back to the flag.
func readShuttleService() string {
	f, err := os.Open("shuttle.yaml")
	if err != nil {
		return ""
	}
	var spec shuttleSpec
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&spec)
	if err != nil {
		return ""
	}
	return spec.Vars.Service
}
