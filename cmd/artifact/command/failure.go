package command

import (
	"fmt"
	"path"

	"github.com/lunarway/release-manager/internal/artifact"
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
				m.Title = m.Service + " :no_entry:"
				return m
			})
			if err != nil {
				fmt.Printf("Error, not able to update slack message with failure message: %v", err)
				return nil
			}

			client, err := slack.NewClient(options.SlackToken)
			if err != nil {
				fmt.Printf("Error, not able to create Slack client in failure command: %v", err)
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
	return command
}
