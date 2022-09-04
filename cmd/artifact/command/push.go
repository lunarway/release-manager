package command

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/lunarway/release-manager/internal/flow"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
	intslack "github.com/lunarway/release-manager/internal/slack"
	"github.com/nlopes/slack"
	"github.com/spf13/cobra"
)

func pushCommand(options *Options, logger *log.Logger) *cobra.Command {
	releaseManagerClient := httpinternal.Client{}

	command := &cobra.Command{
		Use:   "push",
		Short: "push artifact to artifact repository",
		RunE: func(c *cobra.Command, args []string) error {
			var artifactID string
			var err error
			ctx := context.Background()

			if releaseManagerClient.Metadata.AuthToken != "" {
				artifactID, err = flow.PushArtifactToReleaseManager(ctx, logger, &releaseManagerClient, options.FileName, options.RootPath)
				if err != nil {
					return err
				}
			}
			client, err := intslack.NewClient(slack.New(options.SlackToken), logger, options.UserMappings, options.EmailSuffix)
			if err != nil {
				fmt.Printf("Error, not able to create Slack client in successful command: %v", err)
				return nil
			}
			err = client.UpdateMessage(path.Join(options.RootPath, options.MessageFileName), func(m intslack.Message) intslack.Message {
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
	command.Flags().StringVar(&releaseManagerClient.BaseURL, "http-base-url", os.Getenv("ARTIFACT_URL"), "address of the http release manager server")
	command.Flags().StringVar(&releaseManagerClient.Metadata.AuthToken, "http-auth-token", "", "auth token for the http service")

	// errors are skipped here as the only case they can occour are if thee flag
	// does not exist on the command.
	//nolint:errcheck
	command.MarkFlagRequired("http-base-url")
	//nolint:errcheck
	command.MarkFlagRequired("http-auth-token")
	return command
}
