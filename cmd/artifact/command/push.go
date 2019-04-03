package command

import (
	"context"

	"github.com/lunarway/release-manager/internal/flow"
	"github.com/spf13/cobra"
)

func pushCommand(options *Options) *cobra.Command {
	var branch, configGitRepo, service, sshPrivateKeyPath string
	command := &cobra.Command{
		Use:   "push",
		Short: "push artifact to a configuration repository",
		RunE: func(c *cobra.Command, args []string) error {
			return flow.PushArtifact(context.Background(), configGitRepo, options.FileName, options.RootPath, service, branch, sshPrivateKeyPath)
		},
	}
	command.Flags().StringVar(&branch, "git-branch", "", "origin branch of the artifact")
	command.MarkFlagRequired("git-branch")
	command.Flags().StringVar(&service, "service", "", "servce name")
	command.MarkFlagRequired("service")
	command.Flags().StringVar(&sshPrivateKeyPath, "ssh-private-key", "", "private key for the config repo")
	command.MarkFlagRequired("ssh-private-key")
	command.Flags().StringVar(&configGitRepo, "config-repo", "", "ssh url for the git config repository")
	command.MarkFlagRequired("config-repo")
	return command
}
