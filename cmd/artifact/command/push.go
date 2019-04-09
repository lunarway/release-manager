package command

import (
	"context"
	"fmt"
	"path"

	"github.com/lunarway/release-manager/internal/slack"

	"github.com/lunarway/release-manager/internal/flow"
	"github.com/spf13/cobra"
)

func pushCommand(options *Options) *cobra.Command {
	var configGitRepo, sshPrivateKeyPath string
	command := &cobra.Command{
		Use:   "push",
		Short: "push artifact to a configuration repository",
		RunE: func(c *cobra.Command, args []string) error {
			artifactId, err := flow.PushArtifact(context.Background(), configGitRepo, options.FileName, options.RootPath, sshPrivateKeyPath)
			if err != nil {
				return err
			}
			err = slack.Update(path.Join(options.RootPath, options.MessageFileName), options.SlackToken, func(m slack.Message) slack.Message {
				m.Color = slack.MsgColorGreen
				m.Text += fmt.Sprintf(":white_check_mark: *Artifact pushed:* %s", artifactId)
				return m
			})
			if err != nil {
				fmt.Printf("Error updating the message file in push")
				return nil
			}
			return nil
		},
	}
	command.Flags().StringVar(&sshPrivateKeyPath, "ssh-private-key", "", "private key for the config repo")
	command.MarkFlagRequired("ssh-private-key")
	command.Flags().StringVar(&configGitRepo, "config-repo", "", "ssh url for the git config repository")
	command.MarkFlagRequired("config-repo")
	return command
}
