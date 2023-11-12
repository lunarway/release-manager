package command

import (
	"github.com/spf13/cobra"

	"github.com/lunarway/release-manager/internal/log"
)

// NewRoot returns a new instance of a daemon command.
func NewRoot(version string) (*cobra.Command, error) {
	var logConfiguration *log.Configuration
	logConfiguration.ParseFromEnvironmnet()
	log.Init(logConfiguration)

	var command = &cobra.Command{
		Use:   "daemon",
		Short: "daemon",
		Run: func(c *cobra.Command, args []string) {
			c.HelpFunc()(c, args)
		},
	}
	command.AddCommand(
		StartDaemon(),
		NewVersion(version),
	)
	return command, nil
}
