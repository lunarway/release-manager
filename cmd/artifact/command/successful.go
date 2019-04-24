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
			client, err := slack.NewClient(options.SlackToken, options.UserMappings)
			if err != nil {
				fmt.Printf("Error, not able to create Slack client in successful command: %v", err)
				return nil
			}
			err = client.UpdateMessage(path.Join(options.RootPath, options.MessageFileName), func(m slack.Message) slack.Message {
				m.Color = slack.MsgColorGreen
				m.Title = m.Service + " :white_check_mark:"
				return m
			})
			if err != nil {
				fmt.Printf("Error, not able to update slack message with successful message: %v", err)
				return nil
			}

			a, err := artifact.Get(path.Join(options.RootPath, options.FileName))
			if err != nil {
				fmt.Printf("Error, not able to retrieve artifact in successful command: %v", err)
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
				Color:         slack.MsgColorGreen,
			})
			if err != nil {
				fmt.Printf("Error, not able to notify #builds in successful command: %v", err)
				return nil
			}
			return nil
		},
	}
	return command
}
