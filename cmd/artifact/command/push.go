package command

import (
	"context"
	"fmt"
	"path"

	"github.com/lunarway/release-manager/internal/flow"
	"github.com/lunarway/release-manager/internal/git"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/slack"
	"github.com/lunarway/release-manager/internal/tracing"
	"github.com/lunarway/release-manager/internal/try"
	"github.com/spf13/cobra"
)

func pushCommand(options *Options) *cobra.Command {
	gitSvc := git.Service{
		Tracer: tracing.NewNoop(),
	}
	releaseManagerClient := httpinternal.Client{}

	// retries for comitting changes into config repo
	// can be required for racing writes
	const maxRetries = 5
	command := &cobra.Command{
		Use:   "push",
		Short: "push artifact to artifact repository",
		RunE: func(c *cobra.Command, args []string) error {
			var artifactID string
			var err error
			ctx := context.Background()

			if gitSvc.ConfigRepoURL != "" {
				err = try.Do(ctx, gitSvc.Tracer, maxRetries, func(ctx context.Context, attempt int) (bool, error) {
					close, err := gitSvc.InitMasterRepo(ctx)
					if err != nil {
						return false, err
					}
					defer close(ctx)
					artifactID, err = flow.PushArtifact(ctx, &gitSvc, options.FileName, options.RootPath)
					if err != nil {
						return false, err
					}
					return true, nil
				})
				if err != nil {
					return err
				}
			}

			if releaseManagerClient.Metadata.AuthToken != "" {
				artifactID, err = flow.PushArtifactToReleaseManager(ctx, &releaseManagerClient, options.FileName, options.RootPath)
				if err != nil {
					return err
				}
			}
			client, err := slack.NewClient(options.SlackToken, options.UserMappings)
			if err != nil {
				fmt.Printf("Error, not able to create Slack client in successful command: %v", err)
				return nil
			}
			err = client.UpdateMessage(path.Join(options.RootPath, options.MessageFileName), func(m slack.Message) slack.Message {
				m.Text += fmt.Sprintf(":white_check_mark: *Artifact pushed:* %s", artifactID)
				return m
			})
			if err != nil {
				fmt.Printf("Error updating the message file in push")
				return nil
			}
			return nil
		},
	}
	command.Flags().StringVar(&releaseManagerClient.BaseURL, "http-base-url", "https://release-manager.dev.lunarway.com", "address of the http release manager server")
	command.Flags().StringVar(&releaseManagerClient.Metadata.AuthToken, "http-auth-token", "", "auth token for the http service")
	command.Flags().StringVar(&gitSvc.SSHPrivateKeyPath, "ssh-private-key", "", "private key for the config repo")
	command.Flags().StringVar(&gitSvc.ConfigRepoURL, "config-repo", "", "ssh url for the git config repository")

	// TODO: Make the flags Required when we ready to remove old artifact repository
	// // errors are skipped here as the only case they can occour are if thee flag
	// // does not exist on the command.
	// //nolint:errcheck
	// command.MarkFlagRequired("http-base-url")
	// //nolint:errcheck
	// command.MarkFlagRequired("http-auth-token")
	return command
}
