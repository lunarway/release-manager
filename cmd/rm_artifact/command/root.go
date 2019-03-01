package command

import "github.com/spf13/cobra"

type Options struct {
	RootPath string
	FileName string
}

// NewCommand returns a new instance of a rm-gen-spec command.
func NewCommand() *cobra.Command {
	var options Options
	var command = &cobra.Command{
		Use:   "rm-artifact",
		Short: "rm-artifact json generate service build specifications",
		Run: func(c *cobra.Command, args []string) {
			c.HelpFunc()(c, args)
		},
	}
	command.PersistentFlags().StringVar(&options.RootPath, "root", ".", "Root from where builds and releases should be found.")
	command.PersistentFlags().StringVar(&options.FileName, "file", "artifact.json", "")
	command.AddCommand(initCommand(&options))
	command.AddCommand(endCommand(&options))
	command.AddCommand(addCommand(&options))
	return command
}
