package command

import (
	"github.com/spf13/cobra"
)

// NewRoot returns a new instance of a daemon command.
func NewRoot(version string) (*cobra.Command, error) {
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
