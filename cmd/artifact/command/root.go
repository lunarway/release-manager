package command

import (
	"github.com/spf13/cobra"
)

type Options struct {
	RootPath        string
	FileName        string
	SlackToken      string
	MessageFileName string
	UserMappings    map[string]string
}

// NewCommand returns a new instance of a rm-gen-spec command.
func NewCommand() (*cobra.Command, error) {
	var options Options
	var command = &cobra.Command{
		Use:   "artifact",
		Short: "generates a artifact.json with build status",
		Run: func(c *cobra.Command, args []string) {
			c.HelpFunc()(c, args)
		},
	}

	command.PersistentFlags().StringVar(&options.RootPath, "root", ".", "Root from where builds and releases should be found.")
	command.PersistentFlags().StringVar(&options.FileName, "file", "artifact.json", "")
	command.PersistentFlags().StringVar(&options.SlackToken, "slack-token", "", "slack token to be used for notifications")
	command.PersistentFlags().StringVar(&options.MessageFileName, "message-file", "message.json", "file to store intermediate slack messages")
	command.AddCommand(initCommand(&options))
	command.AddCommand(endCommand(&options))
	command.AddCommand(addCommand(&options))
	command.AddCommand(pushCommand(&options))
	command.AddCommand(failureCommand(&options))
	command.AddCommand(successfulCommand(&options))
	return command, nil
}
