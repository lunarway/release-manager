package command

import (
	"os"
	"time"

	"github.com/spf13/cobra"
)

type Options struct {
	RootPath    string
	grpcAddress string
	grpcTimeout time.Duration
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
	command.AddCommand(NewStatus(&options))
	command.AddCommand(NewPolicy(&options))
	command.PersistentFlags().StringVar(&options.RootPath, "root", ".", "Root from where builds and releases should be found.")
	command.PersistentFlags().StringVar(&options.grpcAddress, "grpc-address", "localhost:7900", "address of the gRPC release manager server")
	command.PersistentFlags().DurationVar(&options.grpcTimeout, "grpc-timeout", 20*time.Second, "gRPC timeout")
	command.PersistentFlags().StringVar(&options.httpBaseURL, "http-base-url", "https://release-manager.dev.lunarway.com", "address of the http release manager server")
	command.PersistentFlags().StringVar(&options.authToken, "http-auth-token", os.Getenv("HAMCTL_AUTH_TOKEN"), "auth token for the http service")
	return command
}
