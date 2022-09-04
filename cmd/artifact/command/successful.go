package command

import (
	"fmt"
	"path"

	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/log"
	intslack "github.com/lunarway/release-manager/internal/slack"
	"github.com/nlopes/slack"
	"github.com/spf13/cobra"
)

func successfulCommand(logger *log.Logger, options *Options) *cobra.Command {
	command := &cobra.Command{
		Use:   "successful",
		Short: "report successful in the pipeline",
		RunE: func(c *cobra.Command, args []string) error {
			client, err := intslack.NewClient(slack.New(options.SlackToken), logger, options.UserMappings, options.EmailSuffix)
			if err != nil {
				fmt.Printf("Error, not able to create Slack client in successful command: %v", err)
				return nil
			}
			err = client.UpdateMessage(path.Join(options.RootPath, options.MessageFileName), func(m intslack.Message) intslack.Message {
				m.Color = intslack.MsgColorGreen
				m.Title = ":jenkins: Jenkins :white_check_mark:"
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

			err = client.NotifySlackBuildsChannel(intslack.BuildsOptions{
				Service:       a.Service,
				ArtifactID:    a.ID,
				Branch:        a.Application.Branch,
				CommitSHA:     a.Application.SHA,
				CommitLink:    a.Application.URL,
				CommitAuthor:  a.Application.AuthorName,
				CommitMessage: a.Application.Message,
				CIJobURL:      a.CI.JobURL,
				Color:         intslack.MsgColorGreen,
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
