package command

import (
	"path"
	"time"

	"github.com/lunarway/release-manager/internal/spec"
	"github.com/spf13/cobra"
)

// NewCommand sets up the move command
func initCommand(options *Options) *cobra.Command {
	var s spec.Spec

	command := &cobra.Command{
		Use:   "init",
		Short: "",
		Long:  "",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Record when this job started
			s.CI.Start = time.Now()

			// Persist the spec to disk
			err := spec.Persist(path.Join(options.RootPath, options.FileName), s)
			if err != nil {
				return err
			}

			return nil
		},
	}

	// Init git data
	command.Flags().StringVar(&s.Application.Author, "git-author", "", "the commit author")
	command.Flags().StringVar(&s.Application.Message, "git-message", "", "the commit message")
	command.Flags().StringVar(&s.Application.Committer, "git-committer", "", "the commit committer")
	command.Flags().StringVar(&s.Application.SHA, "git-sha", "", "the commit sha")
	command.Flags().StringVar(&s.Application.Provider, "provider", "", "the name of the repository provider")
	command.Flags().StringVar(&s.Application.URL, "url", "", "the url to the repository commit")
	command.Flags().StringVar(&s.Application.Name, "name", "", "the name of the repository")
	command.Flags().StringVar(&s.Shuttle.Plan.SHA, "shuttle-plan-sha", "", "the commit sha of the shuttle plan")
	command.Flags().StringVar(&s.Shuttle.Plan.URL, "shuttle-plan-url", "", "the url to the shuttle plan commit")
	command.Flags().StringVar(&s.Shuttle.Plan.Message, "shuttle-plan-message", "", "the shuttle plan commit message")

	command.MarkFlagRequired("git-author")
	command.MarkFlagRequired("git-message")
	command.MarkFlagRequired("git-committer")
	command.MarkFlagRequired("git-sha")

	// Init ci data
	command.Flags().StringVar(&s.CI.JobURL, "ci-job-url", "", "the URL of the Job in CI")

	return command
}
