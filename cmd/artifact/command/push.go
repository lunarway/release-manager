package command

import (
	"context"
	"fmt"
	"path"

	"github.com/lunarway/release-manager/internal/flow"
	"github.com/lunarway/release-manager/internal/git"
	"github.com/lunarway/release-manager/internal/slack"
	"github.com/spf13/cobra"
)

func pushCommand(options *Options) *cobra.Command {
	var gitSvc git.Service
	// retries for comitting changes into config repo
	// can be required for racing writes
	const maxRetries = 5
	command := &cobra.Command{
		Use:   "push",
		Short: "push artifact to a configuration repository",
		RunE: func(c *cobra.Command, args []string) error {
			close, err := gitSvc.InitMasterRepo()
			if err != nil {
				return err
			}
			defer close()
			artifactId, err := flow.PushArtifact(context.Background(), &gitSvc, options.FileName, options.RootPath, maxRetries)
			if err != nil {
				return err
			}
			client, err := slack.NewClient(options.SlackToken, options.UserMappings)
			if err != nil {
				fmt.Printf("Error, not able to create Slack client in successful command: %v", err)
				return nil
			}
			err = client.UpdateMessage(path.Join(options.RootPath, options.MessageFileName), func(m slack.Message) slack.Message {
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
	command.Flags().StringVar(&gitSvc.SSHPrivateKeyPath, "ssh-private-key", "", "private key for the config repo")
	command.MarkFlagRequired("ssh-private-key")
	command.Flags().StringVar(&gitSvc.ConfigRepoURL, "config-repo", "", "ssh url for the git config repository")
	command.MarkFlagRequired("config-repo")
	return command
}
