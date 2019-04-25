package command

import (
	"fmt"
	"path"

	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/slack"
	"github.com/spf13/cobra"
)

func failureCommand(options *Options) *cobra.Command {
	var errorMessage string
	command := &cobra.Command{
		Use:   "failure",
		Short: "report failure in the pipeline",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if errorMessage == "" {
				errorMessage = "Unexpected error in pipeline"
			}
			return nil
		},
		RunE: func(c *cobra.Command, args []string) error {
			client, err := slack.NewClient(options.SlackToken, options.UserMappings)
			if err != nil {
				fmt.Printf("Error, not able to create Slack client in failure command: %v", err)
				return nil
			}
			err = client.UpdateMessage(path.Join(options.RootPath, options.MessageFileName), func(m slack.Message) slack.Message {
				m.Color = slack.MsgColorRed
				m.Text += fmt.Sprintf(":no_entry: *%s*", errorMessage)
				m.Title = m.Service + " :no_entry:"
				return m
			})
			if err != nil {
				fmt.Printf("Error, not able to update slack message with failure message: %v", err)
				return nil
			}

			a, err := artifact.Get(path.Join(options.RootPath, options.FileName))
			if err != nil {
				fmt.Printf("Error, not able to retrieve artifact in failure command: %v", err)
				return nil
			}

			err = client.NotifySlackBuildsChannel(slack.BuildsOptions{
				Service:       a.Service,
				ArtifactID:    a.ID,
				Branch:        a.Application.Branch,
				CommitSHA:     a.Application.SHA,
				CommitLink:    a.Application.URL,
				CommitAuthor:  a.Application.AuthorName,
				CommitMessage: a.Application.Message,
				CIJobURL:      a.CI.JobURL,
				Color:         slack.MsgColorRed,
			})
			if err != nil {
				fmt.Printf("Error, not able to notify #builds in failure command: %v", err)
				return nil
			}

			return nil
		},
	}
	command.Flags().StringVar(&errorMessage, "error-message", "", "error message to send to slack")
	return command
}
