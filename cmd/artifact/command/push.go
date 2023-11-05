package command

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/lunarway/release-manager/internal/flow"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	intslack "github.com/lunarway/release-manager/internal/slack"
	"github.com/nlopes/slack"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func pushCommand(options *Options) *cobra.Command {
	releaseManagerClient := httpinternal.Client{}

	command := &cobra.Command{
		Use:   "push",
		Short: "push artifact to artifact repository",
		RunE: func(c *cobra.Command, args []string) error {
			var artifactID string
			var err error
			ctx := context.Background()

			idpURL := os.Getenv("HAMCTL_OAUTH_IDP_URL")
			if idpURL == "" {
				return errors.New("no HAMCTL_OAUTH_IDP_URL env var set")
			}
			clientID := os.Getenv("HAMCTL_OAUTH_CLIENT_ID")
			if clientID == "" {
				return errors.New("no HAMCTL_OAUTH_CLIENT_ID env var set")
			}
			clientSecret := os.Getenv("HAMCTL_OAUTH_CLIENT_SECRET")
			if clientID == "" {
				return errors.New("no HAMCTL_OAUTH_CLIENT_SECRET env var set")
			}
			authenticator := httpinternal.NewClientAuthenticator(clientID, clientSecret, idpURL)
			releaseManagerClient.Auth = &authenticator

			artifactID, err = flow.PushArtifactToReleaseManager(ctx, &releaseManagerClient, options.FileName, options.RootPath)
			if err != nil {
				return err
			}

			client, err := intslack.NewClient(slack.New(options.SlackToken), options.UserMappings, options.EmailSuffix)
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

	// errors are skipped here as the only case they can occour are if thee flag
	// does not exist on the command.
	//nolint:errcheck
	command.MarkFlagRequired("http-base-url")
	//nolint:errcheck
	command.MarkFlagRequired("http-auth-token")
	return command
}
