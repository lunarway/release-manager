package command

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/lunarway/release-manager/internal/artifact"
	intslack "github.com/lunarway/release-manager/internal/slack"
	"github.com/nlopes/slack"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// NewCommand sets up the move command
func initCommand(options *Options) *cobra.Command {
	var s artifact.Spec
	var users []string

	command := &cobra.Command{
		Use:   "init",
		Short: "",
		Long:  "",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if len(users) == 0 {
				userMappingString := os.Getenv("USER_MAPPINGS")
				users = strings.Split(userMappingString, ",")
			}
			var err error
			options.UserMappings, err = intslack.ParseUserMappings(users)
			if err != nil {
				return err
			}
			isCompanyEmail := func(email string) bool {
				return strings.HasSuffix(email, options.EmailSuffix)
			}
			if !isCompanyEmail(s.Application.AuthorEmail) {
				companyEmail, ok := options.UserMappings[s.Application.AuthorEmail]
				if !ok {
					// Don't break, just continue and use the provided email
					fmt.Printf("user mappings for %s not found", s.Application.AuthorEmail)
				} else {
					s.Application.AuthorEmail = companyEmail
				}
			}

			if !isCompanyEmail(s.Application.CommitterEmail) {
				companyEmail, ok := options.UserMappings[s.Application.CommitterEmail]
				if !ok {
					// Don't break, just continue and use the provided email
					fmt.Printf("user mappings for %s not found", s.Application.CommitterEmail)
				} else {
					s.Application.CommitterEmail = companyEmail
				}
			}

			return nil
		},
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
				client, err := intslack.NewClient(slack.New(options.SlackToken), options.UserMappings, options.EmailSuffix)
				if err != nil {
					fmt.Printf("Error creating Slack client")
					return nil
				}

				// create and post the initial slack message
				title := ":jenkins: Jenkins :sonic-running:"
				titleLink := s.CI.JobURL
				text := fmt.Sprintf("Build for: %s\nBranch: <%s|*%s*>\n", s.Application.Name, s.Application.URL, s.Application.Branch)
				color := intslack.MsgColorYellow
				respChan, timestamp, err := client.PostSlackBuildStarted(s.Application.AuthorEmail, title, titleLink, text, color)
				if err != nil {
					return nil
				}

				// Persist the Slack message to disk for later retrieval and updates
				messageFilePath := path.Join(options.RootPath, options.MessageFileName)
				err = intslack.Persist(messageFilePath, intslack.Message{
					Title:     title,
					TitleLink: titleLink,
					Text:      text,
					Channel:   respChan,
					Timestamp: timestamp,
					Color:     color,
					Service:   s.Application.Name,
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
	command.Flags().StringVar(&s.Namespace, "namespace", "", "the namespace to deploy the service to")
	command.Flags().StringVar(&s.Squad, "squad", "", "the squad who owns the service")

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
	command.Flags().StringSliceVar(&users, "user-mappings", []string{}, "user mappings between emails used by Git and Slack, key-value pair: <email>=<slack-email>")

	// errors are skipped here as the only case they can occour are if thee flag
	// does not exist on the command.
	//nolint:errcheck
	command.MarkFlagRequired("artifact-id")
	//nolint:errcheck
	command.MarkFlagRequired("service")
	//nolint:errcheck
	command.MarkFlagRequired("git-author-name")
	//nolint:errcheck
	command.MarkFlagRequired("git-author-email")
	//nolint:errcheck
	command.MarkFlagRequired("git-message")
	//nolint:errcheck
	command.MarkFlagRequired("git-committer-name")
	//nolint:errcheck
	command.MarkFlagRequired("git-committer-email")
	//nolint:errcheck
	command.MarkFlagRequired("git-sha")
	//nolint:errcheck
	command.MarkFlagRequired("git-branch")

	// Init ci data
	command.Flags().StringVar(&s.CI.JobURL, "ci-job-url", "", "the URL of the Job in CI")

	return command
}
