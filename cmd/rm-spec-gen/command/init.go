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
	command.Flags().StringVar(&s.Git.Author, "git-author", "", "the commit author")
	command.Flags().StringVar(&s.Git.Message, "git-message", "", "the commit message")
	command.Flags().StringVar(&s.Git.Committer, "git-committer", "", "the commit committer")
	command.Flags().StringVar(&s.Git.SHA, "git-sha", "", "the commit sha")
	command.MarkFlagRequired("git-author")
	command.MarkFlagRequired("git-message")
	command.MarkFlagRequired("git-committer")
	command.MarkFlagRequired("git-sha")

	// Init ci data
	command.Flags().StringVar(&s.CI.JobURL, "ci-job-url", "", "the URL of the Job in CI")

	return command
}
