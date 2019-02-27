package command

import (
	"fmt"

	"github.com/lunarway/release-manager/internal/spec"
	"github.com/spf13/cobra"
)

// NewCommand sets up the move command
func initCommand() *cobra.Command {
	var git spec.Git

	command := &cobra.Command{
		Use:   "init",
		Short: "",
		Long:  "",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Git: %s %s %s %s", git.Author, git.Committer, git.Message, git.SHA)
			return nil
		},
	}

	// Init git data
	command.Flags().StringVar(&git.Author, "git-author", "", "the commit author")
	command.Flags().StringVar(&git.Message, "git-message", "", "the commit message")
	command.Flags().StringVar(&git.Committer, "git-committer", "", "the commit committer")
	command.Flags().StringVar(&git.SHA, "git-sha", "", "the commit sha")
	command.MarkFlagRequired("git-author")
	command.MarkFlagRequired("git-message")
	command.MarkFlagRequired("git-committer")
	command.MarkFlagRequired("git-sha")

	return command
}
