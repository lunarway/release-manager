package command

import (
	"fmt"
	"path"

	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/slack"
	"github.com/spf13/cobra"
)

func successfulCommand(options *Options) *cobra.Command {
	command := &cobra.Command{
		Use:   "successful",
		Short: "report successful in the pipeline",
		RunE: func(c *cobra.Command, args []string) error {
			err := slack.Update(path.Join(options.RootPath, options.MessageFileName), options.SlackToken, func(m slack.Message) slack.Message {
				m.Color = slack.MsgColorGreen
				return m
			})
			if err != nil {
				fmt.Printf("Error, not able to update slack message with successful message")
			}

			client, err := slack.NewClient(options.SlackToken)
			if err != nil {
				fmt.Printf("Error, not able to create Slack client in successful command")
			}

			a, err := artifact.Get(path.Join(options.RootPath, options.FileName))
			if err != nil {
				fmt.Printf("Error, not able to retrieve artifact in successful command")
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
				Color:         slack.MsgColorGreen,
			})
			if err != nil {
				fmt.Printf("Error, not able to notify #builds in successful command")
			}
			return nil
		},
	}
	return command
}
