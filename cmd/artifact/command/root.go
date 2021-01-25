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
	EmailSuffix     string
}

// NewRoot returns a new instance of an artifact command.
func NewRoot(version string) (*cobra.Command, error) {
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
	command.PersistentFlags().StringVar(&options.EmailSuffix, "email-suffix", "", "company email suffix to expect. E.g.: '@example.com'")
	command.AddCommand(
		addCommand(&options),
		endCommand(&options),
		failureCommand(&options),
		initCommand(&options),
		pushCommand(&options),
		successfulCommand(&options),
		versionCommand(version),
	)
	command.MarkFlagRequired("email-suffix")
	return command, nil
}
