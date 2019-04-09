package command

import (
	"fmt"
	"path"
	"time"

	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/slack"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// NewCommand sets up the move command
func initCommand(options *Options) *cobra.Command {
	var s artifact.Spec

	command := &cobra.Command{
		Use:   "init",
		Short: "",
		Long:  "",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Record when this job started
			s.CI.Start = time.Now()

			// Persist the spec to disk
			filePath := path.Join(options.RootPath, options.FileName)
			err := artifact.Persist(filePath, s)
			if err != nil {
				return errors.WithMessagef(err, "persist to file '%s'", filePath)
			}

			// If we have an email to use for slack, lets inform
			if s.Application.AuthorEmail != "" {
				// Setup Slack client
				client, err := slack.NewClient(options.SlackToken)
				if err != nil {
					fmt.Printf("Error creating Slack client")
					return nil
				}

				// retrieve the slack userId to communicate directly
				userId, err := client.GetSlackIdByEmail(s.Application.AuthorEmail)
				if err != nil {
					fmt.Printf("Error getting Slack Id for author: %s", s.Application.AuthorEmail)
					return nil
				}

				// create and post the initial slack message
				title := fmt.Sprintf("%s", s.Application.Name)
				text := fmt.Sprintf("Build started for branch: *%s*\n", s.Application.Branch)
				color := slack.MsgColorYellow
				respChan, timestamp, err := client.PostSlackBuildStarted(userId, title, text, color)
				if err != nil {
					return nil
				}

				// Persist the Slack message to disk for later retrieval and updates
				messageFilePath := path.Join(options.RootPath, options.MessageFileName)
				err = slack.Persist(messageFilePath, slack.Message{
					Title:     title,
					Text:      text,
					UserID:    userId,
					Channel:   respChan,
					Timestamp: timestamp,
					Color:     color,
				})

				if err != nil {
					fmt.Printf("Error persisting slack message to file")
					return nil
				}
			}

			return nil
		},
	}

	command.Flags().StringVar(&s.ID, "artifact-id", "", "the id of the artifact")
	command.Flags().StringVar(&s.Service, "service", "", "the service name")

	// Init git data
	command.Flags().StringVar(&s.Application.AuthorName, "git-author-name", "", "the commit author name")
	command.Flags().StringVar(&s.Application.AuthorEmail, "git-author-email", "", "the commit author email")
	command.Flags().StringVar(&s.Application.Message, "git-message", "", "the commit message")
	command.Flags().StringVar(&s.Application.CommitterName, "git-committer-name", "", "the commit committer name")
	command.Flags().StringVar(&s.Application.CommitterEmail, "git-committer-email", "", "the commit committer email")
	command.Flags().StringVar(&s.Application.SHA, "git-sha", "", "the commit sha")
	command.Flags().StringVar(&s.Application.Branch, "git-branch", "", "the branch of the repository")
	command.Flags().StringVar(&s.Application.Provider, "provider", "", "the name of the repository provider")
	command.Flags().StringVar(&s.Application.URL, "url", "", "the url to the repository commit")
	command.Flags().StringVar(&s.Application.Name, "name", "", "the name of the repository")
	command.Flags().StringVar(&s.Shuttle.Plan.SHA, "shuttle-plan-sha", "", "the commit sha of the shuttle plan")
	command.Flags().StringVar(&s.Shuttle.Plan.URL, "shuttle-plan-url", "", "the url to the shuttle plan commit")
	command.Flags().StringVar(&s.Shuttle.Plan.Message, "shuttle-plan-message", "", "the shuttle plan commit message")
	command.Flags().StringVar(&s.Shuttle.Plan.Branch, "shuttle-plan-branch", "", "the shuttle plan branch name")

	command.MarkFlagRequired("artifact-id")
	command.MarkFlagRequired("service")
	command.MarkFlagRequired("git-author-name")
	command.MarkFlagRequired("git-author-email")
	command.MarkFlagRequired("git-message")
	command.MarkFlagRequired("git-committer-name")
	command.MarkFlagRequired("git-committer-email")
	command.MarkFlagRequired("git-sha")
	command.MarkFlagRequired("git-branch")

	// Init ci data
	command.Flags().StringVar(&s.CI.JobURL, "ci-job-url", "", "the URL of the Job in CI")

	return command
}
