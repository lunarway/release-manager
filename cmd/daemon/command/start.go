package command

import (
	"github.com/spf13/cobra"
)

func StartDaemon() *cobra.Command {
	var command = &cobra.Command{
		Use:   "start",
		Short: "start the release-daemon",
		RunE: func(c *cobra.Command, args []string) error {
			// DO STUFF
			return nil
		},
	}

	return command
}
