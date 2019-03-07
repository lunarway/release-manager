package command

import (
	"time"

	"github.com/spf13/cobra"
)

type Options struct {
	RootPath    string
	grpcAddress string
	grpcTimeout time.Duration
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

	command.PersistentFlags().StringVar(&options.RootPath, "root", ".", "Root from where builds and releases should be found.")
	command.PersistentFlags().StringVar(&options.gRPCAddress, "grpc-address", "localhost:7900", "the address of the gRPC release manager server")
	command.PersistentFlags().DurationVar(&options.gRPCTimeout, "grpc-timeout", (20 * time.Second), "gRPC timeout")
	return command
}
