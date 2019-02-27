package command

import "github.com/spf13/cobra"

type Options struct {
	RootPath string
}

// NewCommand returns a new instance of a rm-gen-spec command.
func NewCommand() *cobra.Command {
	var options Options
	var command = &cobra.Command{
		Use:   "rm-spec-gen",
		Short: "rm-spec-gen json generate service build specifications",
		Run: func(c *cobra.Command, args []string) {
			c.HelpFunc()(c, args)
		},
	}

	command.AddCommand(initCommand())

	command.PersistentFlags().StringVar(&options.RootPath, "root", ".", "Root from where builds and releases should be found.")
	return command
}
