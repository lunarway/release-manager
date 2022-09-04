package command

import (
	"github.com/lunarway/release-manager/internal/log"
	"github.com/spf13/cobra"
)

// NewRoot returns a new instance of a daemon command.
func NewRoot(logger *log.Logger, version string) (*cobra.Command, error) {
	var command = &cobra.Command{
		Use:   "daemon",
		Short: "daemon",
		Run: func(c *cobra.Command, args []string) {
			c.HelpFunc()(c, args)
		},
	}
	command.AddCommand(
		StartDaemon(logger),
		NewVersion(version),
	)
	return command, nil
}
