package command

import (
	"github.com/spf13/cobra"
)

// DaemonCommand returns a new instance of a daemon command.
func DaemonCommand(version *string) (*cobra.Command, error) {
	var command = &cobra.Command{
		Use:   "daemon",
		Short: "daemon",
		Run: func(c *cobra.Command, args []string) {
			c.HelpFunc()(c, args)
		},
	}
	command.AddCommand(StartDaemon(*version))
	return command, nil
}
