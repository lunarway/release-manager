package command

import (
	"os"
	"time"

	"github.com/spf13/cobra"
)

type Options struct {
	RootPath    string
	httpTimeout time.Duration
	httpBaseURL string
	authToken   string
}

// NewCommand returns a new instance of a hamctl command.
func NewCommand() *cobra.Command {
	var options Options
	var command = &cobra.Command{
		Use:   "hamctl",
		Short: "hamctl controls a release manager server",
		Run: func(c *cobra.Command, args []string) {
			c.HelpFunc()(c, args)
		},
	}
	command.AddCommand(NewPromote(&options))
	command.AddCommand(NewRelease(&options))
	command.AddCommand(NewStatus(&options))
	command.AddCommand(NewPolicy(&options))
	command.PersistentFlags().StringVar(&options.RootPath, "root", ".", "Root from where builds and releases should be found.")
	command.PersistentFlags().DurationVar(&options.httpTimeout, "http-timeout", 20*time.Second, "HTTP request timeout")
	command.PersistentFlags().StringVar(&options.httpBaseURL, "http-base-url", "https://release-manager.dev.lunarway.com", "address of the http release manager server")
	command.PersistentFlags().StringVar(&options.authToken, "http-auth-token", os.Getenv("HAMCTL_AUTH_TOKEN"), "auth token for the http service")
	return command
}
