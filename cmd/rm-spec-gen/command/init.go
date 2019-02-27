package command

import (
	"github.com/lunarway/release-manager/cmd/rm-spec-gen/pkg/init"
	"github.com/spf13/cobra"
)

// NewCommand sets up the move command
func initCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "init",
		Short: "",
		Long:  "",
		RunE: func(cmd *cobra.Command, args []string) error {

			return init.Run()
		},
	}

	return c
}
