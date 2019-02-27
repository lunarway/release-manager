package command

import "github.com/spf13/cobra"

// NewCommand returns a new instance of a rm-gen-spec command.
func NewCommand() *cobra.Command {
	var command = &cobra.Command{
		Use:   "rm-spec-gen",
		Short: "rm-spec-gen json generate service build specifications",
		Run: func(c *cobra.Command, args []string) {
			c.HelpFunc()(c, args)
		},
	}

	command.AddCommand(initCommand())
	return command
}
