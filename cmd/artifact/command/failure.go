package command

import (
	"fmt"
	"path"

	"github.com/lunarway/release-manager/internal/slack"

	"github.com/spf13/cobra"
)

func failureCommand(options *Options) *cobra.Command {
	command := &cobra.Command{
		Use:   "failure",
		Short: "report failure in the pipeline",
		RunE: func(c *cobra.Command, args []string) error {
			err := slack.Update(path.Join(options.RootPath, options.MessageFileName), options.SlackToken, func(m slack.Message) slack.Message {
				m.Color = slack.MsgColorRed
				m.Text += ":no_entry: *Unexpected error in pipeline*"
				return m
			})
			if err != nil {
				fmt.Printf("Error, not able to update slack message with failure message")
			}
			return nil
		},
	}
	return command
}
