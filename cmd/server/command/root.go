package command

import (
	"time"

	"github.com/spf13/cobra"
)

type Options struct {
	grpcPort         int
	httpPort         int
	timeout          time.Duration
	configRepo       string
	artifactFileName string
}

// NewCommand returns a new instance of a hamctl command.
func NewCommand() *cobra.Command {
	var options Options
	var command = &cobra.Command{
		Use:   "server",
		Short: "server",
		Run: func(c *cobra.Command, args []string) {
			c.HelpFunc()(c, args)
		},
	}
	command.AddCommand(NewStart(&options))
	command.PersistentFlags().StringVar(&options.configRepo, "config-repo", "git@github.com:lunarway/k8s-cluster-config.git", "ssh url for the git config repository")
	command.PersistentFlags().IntVar(&options.grpcPort, "grpc-port", 7900, "port of the grpc server")
	command.PersistentFlags().IntVar(&options.httpPort, "http-port", 8080, "port of the http server")
	command.PersistentFlags().StringVar(&options.artifactFileName, "artifact-filename", "artifact.json", "the filename of the artifact to be used")
	command.PersistentFlags().DurationVar(&options.timeout, "timeout", 20*time.Second, "timeout of both the grpc and http server")

	return command
}
