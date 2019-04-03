package command

import (
	"path"
	"time"

	"github.com/lunarway/release-manager/internal/artifact"
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
			err := artifact.Persist(path.Join(options.RootPath, options.FileName), s)
			if err != nil {
				return err
			}

			return nil
		},
	}

	command.Flags().StringVar(&s.ID, "artifact-id", "", "the id of the artifact")

	// Init git data
	command.Flags().StringVar(&s.Application.AuthorName, "git-author-name", "", "the commit author name")
	command.Flags().StringVar(&s.Application.AuthorEmail, "git-author-email", "", "the commit author email")
	command.Flags().StringVar(&s.Application.Message, "git-message", "", "the commit message")
	command.Flags().StringVar(&s.Application.CommitterName, "git-committer-name", "", "the commit committer name")
	command.Flags().StringVar(&s.Application.CommitterEmail, "git-committer-email", "", "the commit committer email")
	command.Flags().StringVar(&s.Application.SHA, "git-sha", "", "the commit sha")
	command.Flags().StringVar(&s.Application.Provider, "provider", "", "the name of the repository provider")
	command.Flags().StringVar(&s.Application.URL, "url", "", "the url to the repository commit")
	command.Flags().StringVar(&s.Application.Name, "name", "", "the name of the repository")
	command.Flags().StringVar(&s.Shuttle.Plan.SHA, "shuttle-plan-sha", "", "the commit sha of the shuttle plan")
	command.Flags().StringVar(&s.Shuttle.Plan.URL, "shuttle-plan-url", "", "the url to the shuttle plan commit")
	command.Flags().StringVar(&s.Shuttle.Plan.Message, "shuttle-plan-message", "", "the shuttle plan commit message")

	command.MarkFlagRequired("artifact-id")
	command.MarkFlagRequired("git-author-name")
	command.MarkFlagRequired("git-author-email")
	command.MarkFlagRequired("git-message")
	command.MarkFlagRequired("git-committer-name")
	command.MarkFlagRequired("git-committer-email")
	command.MarkFlagRequired("git-sha")

	// Init ci data
	command.Flags().StringVar(&s.CI.JobURL, "ci-job-url", "", "the URL of the Job in CI")

	return command
}
