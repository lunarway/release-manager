package command

import (
	"github.com/lunarway/release-manager/cmd/hamctl/command/policy"
	"github.com/spf13/cobra"
)

func NewPolicy(options *Options) *cobra.Command {
	var command = &cobra.Command{
		Use:   "policy",
		Short: "Promote a service to a specific environment following promoting conventions.",
		Run: func(c *cobra.Command, args []string) {
			c.HelpFunc()(c, args)
		},
	}
	command.AddCommand(policy.NewAdd())
	command.AddCommand(policy.NewList())
	command.AddCommand(policy.NewRemove())
	return command
}
