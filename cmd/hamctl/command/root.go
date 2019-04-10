package command

import (
	"os"
	"time"

	"github.com/lunarway/release-manager/internal/http"
	"github.com/spf13/cobra"
)

// NewCommand returns a new instance of a hamctl command.
func NewCommand() (*cobra.Command, error) {
	var client http.Client
	var service string
	var command = &cobra.Command{
		Use:   "hamctl",
		Short: "hamctl controls a release manager server",
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
	command.MarkFlagRequired("service")
	return command, nil
}
